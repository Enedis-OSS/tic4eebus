// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package ems

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"slices"
	"sync"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/Enedis-OSS/tic4eebus/ems/data"
	"github.com/Enedis-OSS/tic4eebus/evse"
	"github.com/Enedis-OSS/tic4eebus/linkymeter"
	"github.com/enbility/eebus-go/api"
	"github.com/enbility/eebus-go/features/server"
	"github.com/enbility/eebus-go/service"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	shipapi "github.com/enbility/ship-go/api"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/go-co-op/gocron"
	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	ENTITY_TYPE                                          = model.EntityTypeTypeCEM
	TIC2WEBSOCKET_CLIENT_RUN_AND_WATCH_PERIOD_IN_SECONDS = 1
	TIC_READ_TIMEOUT                                     = "TIC read timeout"
	OPEV_USE_CASE_NAME                                   = model.UseCaseNameTypeOverloadProtectionByEVChargingCurrentCurtailment
	OPEV_USE_CASE_VERSION                                = "1.0.1"
	OPEV_USE_CASE_DUCUMENT_SUB_REVISION                  = "release"
)

var (
	OPEV_USE_CASE_SCENARIO = []model.UseCaseScenarioSupportType{1, 2, 3}
)

type OnEnergyGuardData func(data data.DataModel)

type EnergyGuardDataSubscriber struct {
	onData OnEnergyGuardData
}

type EnergyGuard struct {
	data                       *data.DataSynchronizer
	config                     config.Config
	subscriberAccess           sync.Mutex
	subscriberMap              map[string]EnergyGuardDataSubscriber
	service                    *service.Service
	diagnosis                  *server.DeviceDiagnosis
	vehicle                    *evse.Vehicle
	wallbox                    *evse.Wallbox
	tic2WebsocketAccess        sync.Mutex
	tic2WebsocketSubcriptionId string
	tic2WebsocketAvailableTIC  linkymeter.TICIdentifier
	tic2WebsocketClient        *linkymeter.TIC2WebsocketClient
	scheduler                  *gocron.Scheduler
	overloadProtectionJob      *gocron.Job
	tic2WebsocketClientJob     *gocron.Job
}

func NewEnergyGuard(
	config config.Config,
) *EnergyGuard {
	energyGuard := &EnergyGuard{}

	energyGuard.config = config

	energyGuard.data = data.NewDataSynchronizer(energyGuard.config.DataModel)

	energyGuard.createTIC2WebsocketClient()

	certificate := energyGuard.loadCertificate()
	energyGuard.configureService(certificate)

	localEntity := energyGuard.service.LocalDevice().EntityForType(ENTITY_TYPE)
	energyGuard.createWallbox(localEntity)
	energyGuard.createVehicle(localEntity)
	energyGuard.createDiagnosis(localEntity)

	energyGuard.scheduler = gocron.NewScheduler(time.UTC)

	return energyGuard
}

func (e *EnergyGuard) Start() {
	e.startTIC2WebsocketClient()

	e.Info("Registering to remote EEBUS node")
	e.service.RegisterRemoteSKI(e.config.EEBUS.RemoteSKI)

	e.Info("Starting EEBUS node service")
	e.service.Start()

	e.startOverloadProtection()
}

func (e *EnergyGuard) Stop() {
	e.stopOverloadProtection()

	e.Info("Stopping EEBUS node service")
	e.service.Shutdown()
	e.vehicle.DisableRemoteConnection()
	e.wallbox.DisableRemoteConnection()

	e.stopTIC2WebsocketClient()
	e.scheduler.Stop()
}

func (e *EnergyGuard) startTIC2WebsocketClient() {
	e.Info("Start TIC2Websocket client")

	if !e.scheduler.IsRunning() {
		e.scheduler.StartAsync()
	}
	var err error
	if e.tic2WebsocketClientJob == nil {
		e.tic2WebsocketClientJob, err = e.scheduler.Every(TIC2WEBSOCKET_CLIENT_RUN_AND_WATCH_PERIOD_IN_SECONDS).Seconds().Do(e.runAndWatchTIC2WebsocketClient)
	}
	if err != nil {
		log.Fatalf("Cannot start TIC2Websocket client job : %s", err.Error())
	}
}

func (e *EnergyGuard) stopTIC2WebsocketClient() {
	var err error

	e.Info("Stopping TIC2Websocket client")
	e.scheduler.Job(e.tic2WebsocketClientJob).Stop()
	if e.tic2WebsocketClient.IsConnected() {
		if e.tic2WebsocketClient.CheckSubscriber(e.getTIC2WebsocketSubscriptionIdAndAvailableTIC()) {
			err := e.tic2WebsocketClient.UnsubscribeTIC(e.getTIC2WebsocketSubscriptionId())
			if err != nil {
				e.Errorf("Cannot unsubscribe from TIC: %s", err.Error())
			}
		}
		err = e.tic2WebsocketClient.Close()
		if err != nil {
			e.Errorf("Cannot stop TIC2Websocket properly : %s", err.Error())
		}
	}
}

func (e *EnergyGuard) runAndWatchTIC2WebsocketClient() {
	if !e.data.HasMeter() {
		if !e.tic2WebsocketClient.IsConnected() {
			error := e.connectTIC2WebsocketClient()
			if error != nil {
				return
			}
		}
		if linkymeter.IsEmptyIdentifier(e.tic2WebsocketAvailableTIC) {
			error := e.findTIC2WebsocketClientAvailableTIC()
			if error != nil {
				return
			}
		}
		if !e.tic2WebsocketClient.CheckSubscriber(e.getTIC2WebsocketSubscriptionIdAndAvailableTIC()) {
			error := e.subscribeTIC2WebsocketClient()
			if error != nil {
				return
			}
		}
	}
}

func (e *EnergyGuard) connectTIC2WebsocketClient() error {
	tic2WebsocketHost := fmt.Sprintf("%s:%d", e.config.TeleInformationClient.TIC2Websocket.IPAddress, e.config.TeleInformationClient.TIC2Websocket.TCPPort)
	err := e.tic2WebsocketClient.Connect(tic2WebsocketHost)

	if err != nil {
		return e.onTIC2WebsocketErrorConnectionFailure(tic2WebsocketHost, err)
	}
	log.Infof("TIC2WebsocketClient connected with host '%s'", tic2WebsocketHost)

	return nil
}

func (e *EnergyGuard) findTIC2WebsocketClientAvailableTIC() error {

	availableTICs, err := e.tic2WebsocketClient.GetAvailableTICs()
	if err != nil {
		return e.onTIC2WebsocketErrorCannotGetAvailableTIC(err)
	}
	var availableTIC linkymeter.TICIdentifier
	meterSerialNumber := e.config.TeleInformationClient.TICIdentifier.SerialNumber
	if len(meterSerialNumber) > 0 {
		serialNumberFound := false
		for i := 0; i < len(availableTICs); i++ {
			if availableTICs[i].SerialNumber == meterSerialNumber {
				availableTIC = availableTICs[i]
				serialNumberFound = true
				break
			}
		}
		if !serialNumberFound {
			return e.onTIC2WebsocketErrorCannotFindMeterSerialNumber(meterSerialNumber)
		}
	} else {
		for i := 0; i < len(availableTICs); i++ {
			if availableTICs[i].SerialNumber != "" {
				availableTIC = availableTICs[i]
				break
			}
		}
		if len(availableTIC.SerialNumber) == 0 {
			return e.onTIC2WebsocketErrorNoMeterSerialNumberAvailable()
		}
	}
	log.Infof("TIC2Websocket find available TIC '%s'", availableTIC)
	e.setTIC2WebsocketAvailableTIC(availableTIC)

	return nil
}

func (e *EnergyGuard) subscribeTIC2WebsocketClient() error {
	availableTIC := e.getTIC2WebsocketAvailableTIC()
	subcriptionId, err := e.tic2WebsocketClient.SubscribeTIC(e.onTICData, e.onTICError, e.onTIC2WebsocketErrorAbnormalClosure, availableTIC)
	if err != nil {
		return e.onTIC2WebsocketErrorSubscriptionFailure(availableTIC, err)
	}
	log.Infof("TIC2WebsocketClient subscribed with available TIC '%+v'", availableTIC)
	e.setTIC2WebsocketSubscriptionId(subcriptionId)

	return err
}

func (e *EnergyGuard) getTIC2WebsocketSubscriptionIdAndAvailableTIC() (subscriptionId string, availableTIC linkymeter.TICIdentifier) {
	e.tic2WebsocketAccess.Lock()
	subscriptionId = e.tic2WebsocketSubcriptionId
	availableTIC = linkymeter.TICIdentifier(e.tic2WebsocketAvailableTIC)
	e.tic2WebsocketAccess.Unlock()

	return subscriptionId, availableTIC
}

func (e *EnergyGuard) clearTIC2WebsocketSubscriptionIdAndAvailableTIC() {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketSubcriptionId = ""
	e.tic2WebsocketAvailableTIC = linkymeter.TICIdentifier{}
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) getTIC2WebsocketAvailableTIC() (availableTIC linkymeter.TICIdentifier) {
	e.tic2WebsocketAccess.Lock()
	availableTIC = linkymeter.TICIdentifier(e.tic2WebsocketAvailableTIC)
	e.tic2WebsocketAccess.Unlock()

	return availableTIC
}

func (e *EnergyGuard) setTIC2WebsocketAvailableTIC(availableTIC linkymeter.TICIdentifier) {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketAvailableTIC = availableTIC
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) clearTIC2WebsocketSubscriptionId() {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketSubcriptionId = ""
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) getTIC2WebsocketSubscriptionId() (subscriptionId string) {
	e.tic2WebsocketAccess.Lock()
	subscriptionId = e.tic2WebsocketSubcriptionId
	e.tic2WebsocketAccess.Unlock()

	return subscriptionId
}

func (e *EnergyGuard) setTIC2WebsocketSubscriptionId(subscriptionId string) {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketSubcriptionId = subscriptionId
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) startOverloadProtection() {
	e.Info("Start overload protection")

	if !e.scheduler.IsRunning() {
		e.scheduler.StartAsync()
	}
	var err error
	if e.overloadProtectionJob == nil {
		e.overloadProtectionJob, err = e.scheduler.Every(e.config.OverloadProtection.RunningPeriodInSeconds).Seconds().Do(e.runOverloadProtection)
	}
	if err != nil {
		log.Fatalf("Cannot start overload protection : %s", err.Error())
	}
}

func (e *EnergyGuard) stopOverloadProtection() {
	e.scheduler.Job(e.overloadProtectionJob).Stop()
}

func (e *EnergyGuard) runOverloadProtection() {
	log.Info("OverloadProtection : running")
	e.data.Print()
	// EEBUS connection not established ?
	if !e.data.IsConnected() {
		return
	}
	// EEBUS OPEV usa case available ?
	if !e.data.HasOPEV() {
		return
	}
	// Vehicle not connected ?
	if !e.vehicle.IsConnected() {
		return
	}
	// Get vehicle current limits
	currentLimits, currentLimitsError := e.GetVehicleCurrentLimits()
	if currentLimitsError != nil {
		log.Errorf("OverloadProtection cannot get vehicle current limits : %s", currentLimitsError)
		return
	}
	maxCurrentLimit := slices.Min(currentLimits.Max)
	minCurrentLimit := slices.Min(currentLimits.Min)
	var currentLimit float64
	// Overload protection algorithm not used ?
	if !e.config.OverloadProtection.Enable {
		// Current limit specified is over vehicle max charging current ?
		if e.config.OverloadProtection.CurrentLimit.ValueInAmps > maxCurrentLimit {
			log.Warnf("OverloadProtection current limit specified (%f) is over vehicle max current limit (%f)", e.config.OverloadProtection.CurrentLimit.ValueInAmps, maxCurrentLimit)
			currentLimit = maxCurrentLimit
			// Current limit specified is below vehicle min charging current ?
		} else if e.config.OverloadProtection.CurrentLimit.ValueInAmps < minCurrentLimit {
			log.Warnf("OverloadProtection current limit specified (%f) is below vehicle max current limit (%f)", e.config.OverloadProtection.CurrentLimit.ValueInAmps, minCurrentLimit)
			currentLimit = minCurrentLimit
		} else {
			currentLimit = e.config.OverloadProtection.CurrentLimit.ValueInAmps
		}
	} else {
		// Linky meter data available ?
		if e.data.HasMeter() {
			loadControlLimit := e.data.GetOverloadProtectionValue()
			if loadControlLimit == 0.0 {
				loadControlLimit = maxCurrentLimit
			}
			// Compute safety duration
			now := time.Now()
			lockDuration := e.data.GetOverloadProtectionLockDuration()
			// Safety delay elapsed ?
			if lockDuration.Seconds() > e.config.OverloadProtection.CurrentLimit.LockDelayInSeconds {
				log.Info("OverloadProtection : safety delay is over")
				e.data.SetOverloadProtectionLockActive(false)
				// Get minimum available current
				minAvailableCurrent := e.data.GeMeterMinAvailableCurrent()
				// Overload occurs ?
				if minAvailableCurrent < 0.0 {
					log.Infof("OverloadProtection : decrease current limit from %f A", math.Abs(minAvailableCurrent))
					// Decrease vehicle max current limit
					currentLimit = loadControlLimit - math.Abs(minAvailableCurrent)
					if currentLimit < minCurrentLimit {
						currentLimit = minCurrentLimit
					}
					// No overload
				} else {
					log.Infof("OverloadProtection : inccrease current limit from %f A", minAvailableCurrent)
					// Increase vehcicle max current limit
					currentLimit = loadControlLimit + math.Abs(minAvailableCurrent)
					if currentLimit > maxCurrentLimit {
						currentLimit = maxCurrentLimit
					}
				}
				// Update safety parameters
				e.data.SetOverloadProtectionLockStart(now)
			} else {
				// Maintain current limit
				currentLimit = loadControlLimit
			}
		}
	}
	// Update vehicle max current limit
	e.writeVehicleLoadControlLimits(currentLimit)
}

func (e *EnergyGuard) readUsecasesInfos() {
	e.Debug("Read use cases infos")
	localDevice := e.service.LocalDevice()
	if localDevice == nil {
		e.onReadUsecaseInfosErrorNoLocalDeviceFound()
		return
	}
	localEntity := localDevice.EntityForType(ENTITY_TYPE)
	if localEntity == nil {
		e.onReadUsecaseInfosErrorExpectedLocalEntityFound()
		return
	}
	localFeature := localEntity.GetOrAddFeature(model.FeatureTypeTypeNodeManagement, model.RoleTypeClient)
	if localFeature == nil {
		e.onReadUsecaseInfosErrorExpectedLocalFeatureNotFound(model.FeatureTypeTypeNodeManagement)
		return
	}
	localFeatureAddress := localFeature.Address()
	var remoteFeatureAddress *model.FeatureAddressType
	remoteFeatureAddress = nil
	remoteDevice := localDevice.RemoteDeviceForSki(e.config.EEBUS.RemoteSKI)
	if remoteDevice == nil {
		e.onReadUsecaseInfosErrorNoRemoteDeviceFound()
		return
	}
	remoteEntities := remoteDevice.Entities()
	for _, remoteEntity := range remoteEntities {
		remoteFeatures := remoteEntity.Features()
		for _, remoteFeature := range remoteFeatures {
			if remoteFeature.Type() == model.FeatureTypeTypeNodeManagement {
				remoteFeatureAddress = remoteFeature.Address()
			}
		}
	}
	if remoteFeatureAddress == nil {
		e.onReadUsecaseInfosErrorExpectedRemoteFeatureNotFound(model.FeatureTypeTypeNodeManagement)
		return
	}
	remoteSender := remoteDevice.Sender()
	if remoteSender == nil {
		e.onReadUsecaseInfosErrorNoRemoteDeviceSenderFound()
		return
	}
	remoteFunction := model.FunctionTypeNodeManagementUseCaseData
	remoteUsecaseDatas := model.NodeManagementUseCaseDataType{}
	remoteCmd := []model.CmdType{{
		Function:                  &remoteFunction,
		NodeManagementUseCaseData: &remoteUsecaseDatas,
	}}
	remoteAckRequest := false
	msgCounter, err := remoteSender.Request(model.CmdClassifierTypeRead, localFeatureAddress, remoteFeatureAddress, remoteAckRequest, remoteCmd)
	if err != nil {
		e.onReadUsecaseInfosErrorSendRequestFailed(remoteFunction, err)
		return
	}
	if msgCounter == nil {
		e.onReadUsecaseInfosErrorSendRequestMsgCounterNotDefined(remoteFunction)
		return
	}
	err = localFeature.AddResponseCallback(*msgCounter, e.onReceiveUsecasesInfos)
	if err != nil {
		e.onReadUsecaseInfosErrorAddCallbackResponseFailed(remoteFunction, err)
	}
}

func (e *EnergyGuard) SubscribeData(onData OnEnergyGuardData) (id string) {
	subscriber := EnergyGuardDataSubscriber{onData: onData}
	id = uuid.New().String()
	e.subscriberAccess.Lock()
	e.subscriberMap[id] = subscriber
	e.subscriberAccess.Unlock()

	return id
}

func (e *EnergyGuard) UnsubscribeData(id string) error {
	_, ok := e.subscriberMap[id]
	if !ok {
		return fmt.Errorf("subscriber id '%s' not found", id)
	}
	e.subscriberAccess.Lock()
	delete(e.subscriberMap, id)
	e.subscriberAccess.Unlock()

	return nil
}

func (e *EnergyGuard) GetVehicleCurrentLimits() (evse.CurrentLimits, error) {
	// EEBUS remote device not connected ?
	if !e.data.IsConnected() {
		return evse.CurrentLimits{}, fmt.Errorf("EEBUS remote device not connected")
	}
	// Vehicle data not available ?
	vehicle := e.data.GetVehicle()
	if vehicle == nil {
		return evse.CurrentLimits{}, fmt.Errorf("vehicle data not available")
	}
	currentLimits, ok := vehicle[evse.VEHICLE_CURRENT_LIMITS]
	// Load control limits available ?
	if !ok {
		return evse.CurrentLimits{}, fmt.Errorf("vehicle current limits not available")
	}
	return currentLimits.(evse.CurrentLimits), nil
}

func (e *EnergyGuard) GetVehicleLoadControlLimits() ([]ucapi.LoadLimitsPhase, error) {
	// EEBUS remote device not connected ?
	if !e.data.IsConnected() {
		return nil, fmt.Errorf("EEBUS remote device not connected")
	}
	// Vehicle data not available ?
	vehicle := e.data.GetVehicle()
	if vehicle == nil {
		return nil, fmt.Errorf("vehicle data not available")
	}
	loadControlLimits, ok := vehicle[evse.VEHICLE_LOAD_CONTROL_LIMITS]
	// Load control limits available ?
	if !ok {
		return nil, fmt.Errorf("vehicle load control limits data not available")
	}
	return loadControlLimits.([]ucapi.LoadLimitsPhase), nil
}

func (e *EnergyGuard) notifyData() {
	e.subscriberAccess.Lock()
	for _, subscriber := range e.subscriberMap {
		go subscriber.onData(e.data.GetModel())
	}
	e.subscriberAccess.Unlock()
}

func (e *EnergyGuard) loadCertificate() tls.Certificate {

	e.Info("Loading certificate")
	certificate, error := tls.LoadX509KeyPair(e.config.EEBUS.CertificateFilePath, e.config.EEBUS.PrivateKeyFilePath)
	if error != nil {
		log.Fatal(error)
	}
	x509Certificate, error := x509.ParseCertificate(certificate.Certificate[0])
	if error != nil {
		log.Fatal(error)
	}
	localSki := fmt.Sprintf("%x", x509Certificate.SubjectKeyId)
	e.Infof("localSKI=%s\n", localSki)

	return certificate
}

func (e *EnergyGuard) configureService(certificate tls.Certificate) {

	e.Info("Creating EEBUS node configuration")
	configuration, error := api.NewConfiguration(
		e.config.EEBUS.VendorCode,
		e.config.EEBUS.DeviceBrand,
		e.config.EEBUS.DeviceModel,
		e.config.EEBUS.SerialNumber,
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeEnergyManagementSystem},
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{ENTITY_TYPE},
		e.config.EEBUS.ServerPort,
		certificate,
		time.Duration(float64(e.config.EEBUS.HeartbeatTimeoutInSeconds)*float64(time.Second)),
	)
	if error != nil {
		log.Fatal(error)
	}
	configuration.SetAlternateIdentifier(e.config.EEBUS.DeviceBrand + "-" + e.config.EEBUS.DeviceModel + "-" + e.config.EEBUS.SerialNumber)

	e.Info("Creating EEBUS node service")
	e.service = service.NewService(configuration, e)
	e.service.SetLogging(e)

	e.Info("Setting up EEBUS node service")
	error = e.service.Setup()
	if error != nil {
		log.Fatal(error)
	}
}

func (e *EnergyGuard) createWallbox(localEntity spineapi.EntityLocalInterface) {
	e.Info("Creating Wallbox")
	e.wallbox = evse.NewWallbox(e.service, localEntity, e.config.Wallbox)
	e.wallbox.SubscribeData(e.onWallboxData, e.onWallboxConnected, e.onWallboxDisconnected, e.onWallboxSupported)
}

func (e *EnergyGuard) createVehicle(localEntity spineapi.EntityLocalInterface) {
	e.Info("Creating Vehicle")
	e.vehicle = evse.NewVehicle(e.service, localEntity, e.config.Vehicle)
	e.vehicle.SubscribeData(e.onVehicleData, e.onVehicleConnected, e.onVehicleDisconnected, e.onVehicleOPEVSupported)
}

func (e *EnergyGuard) createDiagnosis(localEntity spineapi.EntityLocalInterface) {
	e.Info("Creating Diagnosis")
	diagnosis, error := server.NewDeviceDiagnosis(localEntity)
	if error != nil {
		log.Fatal(error)
	}
	e.diagnosis = diagnosis
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, model.LastErrorCodeType(data.DIAGNOSIS_NO_ERROR))
}

func (e *EnergyGuard) updateDiagnosis(operatingState model.DeviceDiagnosisOperatingStateType, lastErrorCode model.LastErrorCodeType) (hasChanged bool) {
	hasChanged = e.data.SetDiagnosis(operatingState, lastErrorCode)

	if hasChanged {
		if operatingState == model.DeviceDiagnosisOperatingStateTypeNormalOperation {
			e.Infof("Update Diagnosis : operatingState=%s,lastErrorCode=%s", operatingState, lastErrorCode)
		} else {
			e.Errorf("Update Diagnosis : operatingState=%s,lastErrorCode=%s", operatingState, lastErrorCode)
		}
	}
	diagnosisState := e.data.GetDiagnosisState()
	e.diagnosis.SetLocalState(diagnosisState)

	return hasChanged
}

func (e *EnergyGuard) createTIC2WebsocketClient() {
	e.Info("Creating TIC2Websocket client")
	e.tic2WebsocketClient = linkymeter.NewTIC2WebsocketClient()
}

// Logging interface

func (e *EnergyGuard) Trace(args ...interface{}) {
	log.Trace(args...)
}

func (e *EnergyGuard) Tracef(format string, args ...interface{}) {
	log.Tracef(format, args...)
}

func (e *EnergyGuard) Debug(args ...interface{}) {
	log.Debug(args...)
}

func (e *EnergyGuard) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

func (e *EnergyGuard) Info(args ...interface{}) {
	log.Info(args...)
}

func (e *EnergyGuard) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

func (e *EnergyGuard) Error(args ...interface{}) {
	log.Error(args...)
}

func (e *EnergyGuard) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// EEBUSServiceHandler

func (e *EnergyGuard) RemoteSKIConnected(service api.ServiceInterface, ski string) {
	e.Infof("Connected with EEBUS remote service (ski=%s)", ski)
	hasChanged := e.data.SetIsConnected(true)
	e.wallbox.EnableRemoteConnection()
	e.vehicle.EnableRemoteConnection()
	if hasChanged {
		e.notifyData()
	}
}

func (e *EnergyGuard) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {
	errorMsg := fmt.Errorf("disconnected from EEBUS remote service (ski=%s)", ski)
	e.Error(errorMsg)
	hasChanged := e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(errorMsg.Error()))
	e.data.SetIsConnected(false)
	e.data.SetHasOPEV(false)
	e.wallbox.DisableRemoteConnection()
	e.vehicle.DisableRemoteConnection()
	if hasChanged {
		e.notifyData()
	}
}

func (e *EnergyGuard) VisibleRemoteServicesUpdated(service api.ServiceInterface, entries []shipapi.RemoteService) {
	for i := 0; i < len(entries); i++ {
		e.Infof(
			"Remote service detected : name=%q, ski=%q, id=%q, brand=%q, type=%q, model=%q\n",
			entries[i].Name,
			entries[i].Ski,
			entries[i].Identifier,
			entries[i].Brand,
			entries[i].Type,
			entries[i].Model)
	}
}

func (e *EnergyGuard) ServiceShipIDUpdate(ski string, shipdID string) {
	e.Infof("ServiceShipIDUpdate with ski=%s and shipID=%s", ski, shipdID)
}

func (e *EnergyGuard) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	if ski == e.config.EEBUS.RemoteSKI && detail.State() == shipapi.ConnectionStateRemoteDeniedTrust {
		e.Error("The remote service denied trust. Exiting.")
		e.service.CancelPairingWithSKI(ski)
		e.service.UnregisterRemoteSKI(ski)
		e.service.Shutdown()
		os.Exit(0)
	}
}

func (e *EnergyGuard) AllowWaitingForTrust(ski string) bool {
	return ski == e.config.EEBUS.RemoteSKI
}

func (e *EnergyGuard) writeVehicleLoadControlLimits(targetCurrentLimit float64) {
	actualCurrentLimit := e.data.GetOverloadProtectionValue()

	if actualCurrentLimit != targetCurrentLimit {
		loadControlLimits := []ucapi.LoadLimitsPhase{
			{Phase: model.ElectricalConnectionPhaseNameTypeA, Value: targetCurrentLimit, IsActive: true},
			{Phase: model.ElectricalConnectionPhaseNameTypeB, Value: targetCurrentLimit, IsActive: true},
			{Phase: model.ElectricalConnectionPhaseNameTypeC, Value: targetCurrentLimit, IsActive: true},
		}
		msgCounter, err := e.vehicle.WriteLoadControlLimits(loadControlLimits, e.onVehicleLoadControlLimitsWritten)
		if err != nil {
			e.Errorf("Cannot write load control limits : error=%s", err)
		} else {
			e.data.SetOverloadProtectionValue(targetCurrentLimit)
			e.Infof("Writing load control limits : value=%+v, msgCounter=%d", loadControlLimits, *msgCounter)
		}
	}
}

func (e *EnergyGuard) onTICData(ticData linkymeter.TICData) {
	meterData := linkymeter.ComputeMeterData(ticData.Content)
	hasChanged := e.onTICSuccess(meterData)
	if hasChanged {
		e.notifyData()
	}
}

func (e *EnergyGuard) onTICError(ticError linkymeter.TICError) {
	notificationNeeded := false
	if ticError.ErrorMessage == TIC_READ_TIMEOUT {
		notificationNeeded = e.onTICErrorReadTimeout()
	} else {
		notificationNeeded = e.onTICErrorCritical(ticError.ErrorMessage)
	}
	ticErrorBytes, error := json.MarshalIndent(ticError, "", "  ")
	if error != nil {
		log.Errorf("Cannot encode ticError to JSON : %s", error)
	} else {
		log.Infof("TICError : \n%s\n", string(ticErrorBytes))
	}
	if notificationNeeded {
		e.notifyData()
	}
}

func (e *EnergyGuard) onTICSuccess(meterData linkymeter.MeterData) (hasChanged bool) {
	hasChanged = e.data.SetMeter(meterData)
	if hasChanged && e.data.IsConnected() && e.data.HasOPEV() {
		hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, model.LastErrorCodeType(data.DIAGNOSIS_NO_ERROR))
	}
	return hasChanged
}

func (e *EnergyGuard) onTICErrorReadTimeout() (hasChanged bool) {
	e.data.SetHasMeter(false)
	hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(TIC_READ_TIMEOUT))
	return hasChanged
}

func (e *EnergyGuard) onTICErrorCritical(errMsg string) (hasChanged bool) {
	e.data.SetHasMeter(false)
	hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(errMsg))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
	return hasChanged
}

func (e *EnergyGuard) onTIC2WebsocketErrorAbnormalClosure() {
	e.data.SetHasMeter(false)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType("TIC2Websocket abnormal closure"))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
}

func (e *EnergyGuard) onTIC2WebsocketErrorConnectionFailure(host string, err error) error {
	e.data.SetHasMeter(false)
	err = fmt.Errorf("TIC2WebsocketClient cannot connect with host '%s' : %s", host, err.Error())
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
	return err
}

func (e *EnergyGuard) onTIC2WebsocketErrorCannotGetAvailableTIC(err error) error {
	e.data.SetHasMeter(false)
	err = fmt.Errorf("TIC2WebsocketClient cannot get available TICs : %s", err.Error())
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
	return err
}

func (e *EnergyGuard) onTIC2WebsocketErrorCannotFindMeterSerialNumber(meterSerialNumber string) error {
	e.data.SetHasMeter(false)
	err := fmt.Errorf("TIC2WebsocketClient cannot find meter serial number '%s'", meterSerialNumber)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
	return err
}

func (e *EnergyGuard) onTIC2WebsocketErrorNoMeterSerialNumberAvailable() error {
	e.data.SetHasMeter(false)
	err := fmt.Errorf("TIC2WebsocketClient has no meter serial number available")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTIC2WebsocketSubscriptionIdAndAvailableTIC()
	return err
}

func (e *EnergyGuard) onTIC2WebsocketErrorSubscriptionFailure(availableTIC linkymeter.TICIdentifier, err error) error {
	e.data.SetHasMeter(false)
	err = fmt.Errorf("TIC2WebsocketClient cannot subscribe with available TIC '%+v' : %s", availableTIC, err.Error())
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTIC2WebsocketSubscriptionId()
	return err
}

// OnVehicleData
func (e *EnergyGuard) onVehicleData(vehicleData map[string]interface{}) {
	notificationNeeded := e.data.SetVehicle(vehicleData)
	if notificationNeeded {
		e.data.SetVehicle(vehicleData)
		e.notifyData()
	}
}

// onVehicleConnected
func (e *EnergyGuard) onVehicleConnected() {
	log.Info("Vehicle is connected")
}

// onVehicleDisconnected
func (e *EnergyGuard) onVehicleDisconnected() {
	log.Info("Vehicle is disconnected")
	e.data.DisableOverloadProtectionActive()
}

// onVehicleOPEVSupported
func (e *EnergyGuard) onVehicleOPEVSupported() {
	log.Info("Vehicle OPEV is supported")
	e.onUsecaseSuccess()
}

// OnWallboxData
func (e *EnergyGuard) onWallboxData(wallboxData map[string]interface{}) {
	notificationNeeded := e.data.SetWallbox(wallboxData)
	if notificationNeeded {
		e.notifyData()
	}
}

// OnWallboxConnected
func (e *EnergyGuard) onWallboxConnected() {
	log.Info("Wallbox is connected")
}

// OnWallboxDisconnected
func (e *EnergyGuard) onWallboxDisconnected() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("wallbox has been disconnected")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

// OnWallboxSupported
func (e *EnergyGuard) onWallboxSupported() {
	log.Info("Wallbox is supported")
	go e.readUsecasesInfos()
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoLocalDeviceFound() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : no local device found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedLocalEntityFound() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : expected local entity '%s' not found", ENTITY_TYPE)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedLocalFeatureNotFound(featureType model.FeatureTypeType) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : expected local feature '%s' not found", featureType)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoRemoteDeviceFound() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : no remote device found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedRemoteFeatureNotFound(featureType model.FeatureTypeType) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : expected remote feature '%s' not found", featureType)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoRemoteDeviceSenderFound() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : no remote device sender found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorSendRequestFailed(function model.FunctionType, err error) {
	e.data.SetHasOPEV(false)
	err = fmt.Errorf("read use case info failure : send request %v failed (%v)", function, err)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorSendRequestMsgCounterNotDefined(function model.FunctionType) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("read use case info failure : send request %v message counter not defined", function)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorAddCallbackResponseFailed(function model.FunctionType, err error) {
	e.data.SetHasOPEV(false)
	err = fmt.Errorf("read use case info failure : cannot add response callback for function %v (%v)", function, err)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReceiveUsecasesInfos(msg spineapi.ResponseMessage) {
	e.Debugf("onReceiveUsecasesInfos : %+v", msg)
	if msg.Data == nil {
		e.onUsecaseErrorDataNotProvided()
		return
	}
	data, ok := msg.Data.(*model.NodeManagementUseCaseDataType)
	if !ok {
		e.onUsecaseErrorDataTypeUnexpected(msg.Data)
		return
	}
	for _, info := range data.UseCaseInformation {
		for _, support := range info.UseCaseSupport {
			if support.UseCaseName == nil {
				continue
			}
			if *support.UseCaseName == OPEV_USE_CASE_NAME {
				if support.UseCaseAvailable == nil {
					e.onUsecaseErrorAvailableNotDefined()
					return
				}
				if !*support.UseCaseAvailable {
					e.onUsecaseErrorNotAvailable()
					return
				}
				if support.UseCaseVersion == nil {
					e.onUsecaseErrorVersionNotDefined()
					return
				}
				if *support.UseCaseVersion != OPEV_USE_CASE_VERSION {
					e.onUsecaseErrorVersionUnexpected(*support.UseCaseVersion)
					return
				}
				if len(support.ScenarioSupport) == 0 {
					e.onUsecaseErrorScenarioNotDefined()
					return
				}
				if !cmp.Equal(support.ScenarioSupport, OPEV_USE_CASE_SCENARIO) {
					e.onUsecaseErrorScenarioUnexpected(support.ScenarioSupport)
					return
				}
				if support.UseCaseDocumentSubRevision == nil {
					e.onUsecaseErrorDocumentSubRevisionNotDefined()
					return
				}
				if *support.UseCaseDocumentSubRevision != OPEV_USE_CASE_DUCUMENT_SUB_REVISION {
					e.onUsecaseErrorDocumentSubRevisionUnexpected(*support.UseCaseDocumentSubRevision)
					return
				}
				e.onUsecaseSuccess()
			}
		}
	}
}

func (e *EnergyGuard) onUsecaseSuccess() {
	e.data.SetHasOPEV(true)
	if e.data.HasMeter() && e.data.IsConnected() {
		e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, model.LastErrorCodeType(data.DIAGNOSIS_NO_ERROR))
	}
}

func (e *EnergyGuard) onUsecaseErrorDataNotProvided() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case data not provided")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorDataTypeUnexpected(data any) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case data type '%v' unexpected (should be NodeManagementUseCaseDataType)", reflect.TypeOf(data))
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorAvailableNotDefined() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case available not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorNotAvailable() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case not available")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorVersionNotDefined() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case version not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorVersionUnexpected(version model.SpecificationVersionType) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case version '%s' unexpected (should be '%s')", version, OPEV_USE_CASE_VERSION)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorScenarioNotDefined() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case scenario not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorScenarioUnexpected(scenario []model.UseCaseScenarioSupportType) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case scenario '%v' unexpected (should be '%v')", scenario, OPEV_USE_CASE_SCENARIO)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorDocumentSubRevisionNotDefined() {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case document sub revision not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onUsecaseErrorDocumentSubRevisionUnexpected(subRevision string) {
	e.data.SetHasOPEV(false)
	err := fmt.Errorf("OPEV use case document sub revision '%s' unexpected (should be '%s')", subRevision, OPEV_USE_CASE_DUCUMENT_SUB_REVISION)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

// onVehicleLoadControlLimitsWritten
func (e *EnergyGuard) onVehicleLoadControlLimitsWritten(result model.ResultDataType) {
	// Update overload protection data
	e.data.SetOverloadProtectionResult(result)
	// Log request result
	if result.ErrorNumber != nil {
		if result.Description == nil {
			e.Infof("Vehicle load control limit written result : ErrorNumber=%d", *result.ErrorNumber)
			if *result.ErrorNumber == model.ErrorNumberTypeNoError {
				e.onVehicleLoadControlLimitsSuccess()
			}
		} else {
			e.onVehicleLoadControlLimitsFailure(*result.ErrorNumber, *result.Description)
		}
	}
}

func (e *EnergyGuard) onVehicleLoadControlLimitsSuccess() {
	if e.data.HasMeter() && e.data.IsConnected() {
		e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, data.DIAGNOSIS_NO_ERROR)
	}
}

func (e *EnergyGuard) onVehicleLoadControlLimitsFailure(errorNumber model.ErrorNumberType, errorDescription model.DescriptionType) {
	err := fmt.Errorf("cannot write vehicle load control limit : ErrorNumber=%d, Description=%s", errorNumber, errorDescription)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}
