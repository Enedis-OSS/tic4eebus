// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
Package ems implements energy management system routines for EEBus Overload Protection by EV Charging Current Curtailment (OPEV) use case.

The energy management system referred to as "Energy Guard" in EEBus protocol is responsible for following scenarios:

  - curtails changing current of connected electric vehicles to avoid overload of electrical installation

  - provide heartbeat mechanism to remote EEBus node (typically wallbox) to check its availability

  - sends error states to remote EEBus node (typically wallbox)
*/
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
	entity_type                                          = model.EntityTypeTypeCEM
	tic2websocket_client_run_and_watch_period_in_seconds = 1
	tic_read_timeout                                     = "TIC read timeout"
	opev_use_case_name                                   = model.UseCaseNameTypeOverloadProtectionByEVChargingCurrentCurtailment
	opev_use_case_version                                = "1.0.1"
	opev_use_case_document_sub_revision                  = "release"
)

var (
	opev_use_case_scenario = []model.UseCaseScenarioSupportType{1, 2, 3}
)

// Callback used for data subscription
type onData func(data data.DataModel)

type subscriber struct {
	onData onData
}

// EnergyGuard handler for EEBUS node service
type EnergyGuard struct {
	data                       *data.DataSynchronizer
	config                     config.Config
	subscriberAccess           sync.Mutex
	subscriberMap              map[string]subscriber
	service                    *service.Service
	diagnosis                  *server.DeviceDiagnosis
	vehicle                    *evse.Vehicle
	wallbox                    *evse.Wallbox
	tic2WebsocketAccess        sync.Mutex
	tic2WebsocketSubcriptionId string
	tic2WebsocketAvailableTic  linkymeter.TicIdentifier
	tic2WebsocketClient        *linkymeter.TIC2WebsocketClient
	scheduler                  *gocron.Scheduler
	overloadProtectionJob      *gocron.Job
	tic2WebsocketClientJob     *gocron.Job
}

// NewEnergyGuard creates an instance of EnergyGuard from configuration data.
func NewEnergyGuard(
	config config.Config,
) *EnergyGuard {
	energyGuard := &EnergyGuard{}
	// Save configuration
	energyGuard.config = config
	// Create data synchronizer
	energyGuard.data = data.NewDataSynchronizer(energyGuard.config.DataModel)
	// Create TIC2WebSocket client
	energyGuard.createTic2WebsocketClient()
	// Load certificate and configure EEBUS node service
	certificate := energyGuard.loadCertificate()
	energyGuard.configureService(certificate)
	// Create wallbox, vehicle and diagnosis
	localEntity := energyGuard.service.LocalDevice().EntityForType(entity_type)
	energyGuard.createWallbox(localEntity)
	energyGuard.createVehicle(localEntity)
	energyGuard.createDiagnosis(localEntity)
	// Create scheduler for periodic overload protection algorithm
	energyGuard.scheduler = gocron.NewScheduler(time.UTC)

	return energyGuard
}

// Starts the EnergyGuard.
//
// This includes starting the TIC2WebSocket client, registering to the remote EEBUS node, starting the EEBUS node service, and starting the overload protection algorithm.
func (e *EnergyGuard) Start() {
	e.startTic2WebsocketClient()

	e.Info("Registering to remote EEBUS node")
	e.service.RegisterRemoteSKI(e.config.Eebus.RemoteSki)

	e.Info("Starting EEBUS node service")
	e.service.Start()

	e.startOverloadProtection()
}

// Stops the EnergyGuard.
//
// This includes stopping the overload protection algorithm, stopping the EEBUS node service, disabling remote connection to wallbox and vehicle, and stopping the TIC2WebSocket client.
func (e *EnergyGuard) Stop() {
	e.stopOverloadProtection()

	e.Info("Stopping EEBUS node service")
	e.service.Shutdown()
	e.vehicle.DisableRemoteConnection()
	e.wallbox.DisableRemoteConnection()

	e.stopTic2WebsocketClient()
	e.scheduler.Stop()
}

func (e *EnergyGuard) startTic2WebsocketClient() {
	e.Info("Start TIC2Websocket client")

	if !e.scheduler.IsRunning() {
		e.scheduler.StartAsync()
	}
	var err error
	if e.tic2WebsocketClientJob == nil {
		e.tic2WebsocketClientJob, err = e.scheduler.Every(tic2websocket_client_run_and_watch_period_in_seconds).Seconds().Do(e.runAndWatchTic2WebsocketClient)
	}
	if err != nil {
		log.Fatalf("Cannot start TIC2Websocket client job : %s", err.Error())
	}
}

func (e *EnergyGuard) stopTic2WebsocketClient() {
	var err error

	e.Info("Stopping TIC2Websocket client")
	e.scheduler.Job(e.tic2WebsocketClientJob).Stop()
	if e.tic2WebsocketClient.IsConnected() {
		if e.tic2WebsocketClient.CheckSubscriber(e.getTic2WebsocketSubscriptionIdAndAvailableTic()) {
			err := e.tic2WebsocketClient.UnsubscribeTic(e.getTic2WebsocketSubscriptionId())
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

func (e *EnergyGuard) runAndWatchTic2WebsocketClient() {
	if !e.data.HasMeterData() {
		if !e.tic2WebsocketClient.IsConnected() {
			error := e.connectTic2WebsocketClient()
			if error != nil {
				return
			}
		}
		if linkymeter.IsEmptyIdentifier(e.tic2WebsocketAvailableTic) {
			error := e.findTic2WebsocketClientAvailableTic()
			if error != nil {
				return
			}
		}
		if !e.tic2WebsocketClient.CheckSubscriber(e.getTic2WebsocketSubscriptionIdAndAvailableTic()) {
			error := e.subscribeTic2WebsocketClient()
			if error != nil {
				return
			}
		}
	}
}

func (e *EnergyGuard) connectTic2WebsocketClient() error {
	tic2WebsocketHost := fmt.Sprintf("%s:%d", e.config.TeleInformationClient.Tic2Websocket.IpAddress, e.config.TeleInformationClient.Tic2Websocket.TcpPort)
	err := e.tic2WebsocketClient.Connect(tic2WebsocketHost)

	if err != nil {
		return e.onTic2WebsocketErrorConnectionFailure(tic2WebsocketHost, err)
	}
	log.Infof("TIC2WebsocketClient connected with host '%s'", tic2WebsocketHost)

	return nil
}

func (e *EnergyGuard) findTic2WebsocketClientAvailableTic() error {

	availableTics, err := e.tic2WebsocketClient.GetAvailableTics()
	if err != nil {
		return e.onTic2WebsocketErrorCannotGetAvailableTic(err)
	}
	var availableTic linkymeter.TicIdentifier
	meterSerialNumber := e.config.TeleInformationClient.TicIdentifier.SerialNumber
	if len(meterSerialNumber) > 0 {
		serialNumberFound := false
		for i := 0; i < len(availableTics); i++ {
			if availableTics[i].SerialNumber == meterSerialNumber {
				availableTic = availableTics[i]
				serialNumberFound = true
				break
			}
		}
		if !serialNumberFound {
			return e.onTic2WebsocketErrorCannotFindMeterSerialNumber(meterSerialNumber)
		}
	} else {
		for i := 0; i < len(availableTics); i++ {
			if availableTics[i].SerialNumber != "" {
				availableTic = availableTics[i]
				break
			}
		}
		if len(availableTic.SerialNumber) == 0 {
			return e.onTic2WebsocketErrorNoMeterSerialNumberAvailable()
		}
	}
	log.Infof("TIC2Websocket find available TIC '%s'", availableTic)
	e.setTic2WebsocketAvailableTic(availableTic)

	return nil
}

func (e *EnergyGuard) subscribeTic2WebsocketClient() error {
	availableTic := e.getTic2WebsocketAvailableTic()
	subcriptionId, err := e.tic2WebsocketClient.SubscribeTic(e.onTicData, e.onTicError, e.onTIC2WebsocketErrorAbnormalClosure, availableTic)
	if err != nil {
		return e.onTic2WebsocketErrorSubscriptionFailure(availableTic, err)
	}
	log.Infof("TIC2WebsocketClient subscribed with available TIC '%+v'", availableTic)
	e.setTic2WebsocketSubscriptionId(subcriptionId)

	return err
}

func (e *EnergyGuard) getTic2WebsocketSubscriptionIdAndAvailableTic() (subscriptionId string, availableTic linkymeter.TicIdentifier) {
	e.tic2WebsocketAccess.Lock()
	subscriptionId = e.tic2WebsocketSubcriptionId
	availableTic = linkymeter.TicIdentifier(e.tic2WebsocketAvailableTic)
	e.tic2WebsocketAccess.Unlock()

	return subscriptionId, availableTic
}

func (e *EnergyGuard) clearTic2WebsocketSubscriptionIdAndAvailableTic() {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketSubcriptionId = ""
	e.tic2WebsocketAvailableTic = linkymeter.TicIdentifier{}
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) getTic2WebsocketAvailableTic() (availableTic linkymeter.TicIdentifier) {
	e.tic2WebsocketAccess.Lock()
	availableTic = linkymeter.TicIdentifier(e.tic2WebsocketAvailableTic)
	e.tic2WebsocketAccess.Unlock()

	return availableTic
}

func (e *EnergyGuard) setTic2WebsocketAvailableTic(availableTic linkymeter.TicIdentifier) {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketAvailableTic = availableTic
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) clearTIC2WebsocketSubscriptionId() {
	e.tic2WebsocketAccess.Lock()
	e.tic2WebsocketSubcriptionId = ""
	e.tic2WebsocketAccess.Unlock()
}

func (e *EnergyGuard) getTic2WebsocketSubscriptionId() (subscriptionId string) {
	e.tic2WebsocketAccess.Lock()
	subscriptionId = e.tic2WebsocketSubcriptionId
	e.tic2WebsocketAccess.Unlock()

	return subscriptionId
}

func (e *EnergyGuard) setTic2WebsocketSubscriptionId(subscriptionId string) {
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
	if !e.data.IsOpevSupported() {
		return
	}
	// Vehicle not connected ?
	if !e.vehicle.IsConnected() {
		return
	}
	// Get vehicle current limits
	currentLimits, currentLimitsError := e.getVehicleCurrentLimits()
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
		if e.data.HasMeterData() {
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
				minAvailableCurrent := e.data.GetMeterMinAvailableCurrent()
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
	localEntity := localDevice.EntityForType(entity_type)
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
	remoteDevice := localDevice.RemoteDeviceForSki(e.config.Eebus.RemoteSki)
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

// SubscribeData subscribes to data model updates and returns a subscription ID
func (e *EnergyGuard) SubscribeData(onData onData) (id string) {
	subscriber := subscriber{onData: onData}
	id = uuid.New().String()
	e.subscriberAccess.Lock()
	e.subscriberMap[id] = subscriber
	e.subscriberAccess.Unlock()

	return id
}

// UnsubscribeData unsubscribes from data model updates using given subscription ID.
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

func (e *EnergyGuard) getVehicleCurrentLimits() (evse.CurrentLimits, error) {
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

func (e *EnergyGuard) notifyData() {
	e.subscriberAccess.Lock()
	for _, subscriber := range e.subscriberMap {
		go subscriber.onData(e.data.GetModel())
	}
	e.subscriberAccess.Unlock()
}

func (e *EnergyGuard) loadCertificate() tls.Certificate {

	e.Info("Loading certificate")
	certificate, error := tls.LoadX509KeyPair(e.config.Eebus.CertificateFilePath, e.config.Eebus.PrivateKeyFilePath)
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
		e.config.Eebus.VendorCode,
		e.config.Eebus.DeviceBrand,
		e.config.Eebus.DeviceModel,
		e.config.Eebus.SerialNumber,
		[]shipapi.DeviceCategoryType{shipapi.DeviceCategoryTypeEnergyManagementSystem},
		model.DeviceTypeTypeEnergyManagementSystem,
		[]model.EntityTypeType{entity_type},
		e.config.Eebus.ServerPort,
		certificate,
		time.Duration(float64(e.config.Eebus.HeartbeatTimeoutInSeconds)*float64(time.Second)),
	)
	if error != nil {
		log.Fatal(error)
	}
	configuration.SetAlternateIdentifier(e.config.Eebus.DeviceBrand + "-" + e.config.Eebus.DeviceModel + "-" + e.config.Eebus.SerialNumber)

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

func (e *EnergyGuard) createTic2WebsocketClient() {
	e.Info("Creating TIC2Websocket client")
	e.tic2WebsocketClient = linkymeter.NewTIC2WebsocketClient()
}

// Logging interface for Trace level
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Trace
func (e *EnergyGuard) Trace(args ...interface{}) {
	log.Trace(args...)
}

// Logging interface for Trace level with formatting
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Tracef
func (e *EnergyGuard) Tracef(format string, args ...interface{}) {
	log.Tracef(format, args...)
}

// Logging interface for Debug level
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Debug
func (e *EnergyGuard) Debug(args ...interface{}) {
	log.Debug(args...)
}

// Logging interface for Debug level with formatting
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Debugf
func (e *EnergyGuard) Debugf(format string, args ...interface{}) {
	log.Debugf(format, args...)
}

// Logging interface for Info level
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Info
func (e *EnergyGuard) Info(args ...interface{}) {
	log.Info(args...)
}

// Logging interface for Info level with formatting
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Infof
func (e *EnergyGuard) Infof(format string, args ...interface{}) {
	log.Infof(format, args...)
}

// Logging interface for Error level
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Error
func (e *EnergyGuard) Error(args ...interface{}) {
	log.Error(args...)
}

// Logging interface for Error level with formatting
//
// See https://pkg.go.dev/github.com/sirupsen/logrus#Logger.Errorf
func (e *EnergyGuard) Errorf(format string, args ...interface{}) {
	log.Errorf(format, args...)
}

// EEBUS Service interface used to notify a connection with a remote EEBUS node.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.RemoteSKIConnected
func (e *EnergyGuard) RemoteSKIConnected(service api.ServiceInterface, ski string) {
	e.Infof("Connected with EEBUS remote service (ski=%s)", ski)
	hasChanged := e.data.SetIsConnected(true)
	e.wallbox.EnableRemoteConnection()
	e.vehicle.EnableRemoteConnection()
	if hasChanged {
		e.notifyData()
	}
}

// EEBUS Service interface used to notify a disconnection with a remote EEBUS node.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.RemoteSKIDisconnected
func (e *EnergyGuard) RemoteSKIDisconnected(service api.ServiceInterface, ski string) {
	errorMsg := fmt.Errorf("disconnected from EEBUS remote service (ski=%s)", ski)
	e.Error(errorMsg)
	hasChanged := e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(errorMsg.Error()))
	e.data.SetIsConnected(false)
	e.data.SetIsOpevSupported(false)
	e.wallbox.DisableRemoteConnection()
	e.vehicle.DisableRemoteConnection()
	if hasChanged {
		e.notifyData()
	}
}

// EEBUS Service interface used to notify when a remote EEBUS node is detected.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.VisibleRemoteServicesUpdated
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

// EEBUS Service interface used to provide the remote EEBUS node SHIP ID.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.ServiceShipIDUpdate
func (e *EnergyGuard) ServiceShipIDUpdate(ski string, shipdID string) {
	e.Infof("ServiceShipIDUpdate with ski=%s and shipID=%s", ski, shipdID)
}

// EEBUS Service interface used to provide the pairing state for the remote EEBUS node.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.ServicePairingDetailUpdate
func (e *EnergyGuard) ServicePairingDetailUpdate(ski string, detail *shipapi.ConnectionStateDetail) {
	if ski == e.config.Eebus.RemoteSki && detail.State() == shipapi.ConnectionStateRemoteDeniedTrust {
		e.Error("The remote service denied trust. Exiting.")
		e.service.CancelPairingWithSKI(ski)
		e.service.UnregisterRemoteSKI(ski)
		e.service.Shutdown()
		os.Exit(0)
	}
}

// EEBUS Service interface used to check the remote EEBUS node Subject Key Identifier (SKI) with configuration data.
//
// See https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/service#Service.AllowWaitingForTrust
func (e *EnergyGuard) AllowWaitingForTrust(ski string) bool {
	return ski == e.config.Eebus.RemoteSki
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

func (e *EnergyGuard) onTicData(ticData linkymeter.TicData) {
	meterData := linkymeter.ComputeMeterData(ticData.Content)
	hasChanged := e.onTicSuccess(meterData)
	if hasChanged {
		e.notifyData()
	}
}

func (e *EnergyGuard) onTicError(ticError linkymeter.TicError) {
	notificationNeeded := false
	if ticError.ErrorMessage == tic_read_timeout {
		notificationNeeded = e.onTicErrorReadTimeout()
	} else {
		notificationNeeded = e.onTicErrorCritical(ticError.ErrorMessage)
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

func (e *EnergyGuard) onTicSuccess(meterData linkymeter.MeterData) (hasChanged bool) {
	hasChanged = e.data.SetMeter(meterData)
	if hasChanged && e.data.IsConnected() && e.data.IsOpevSupported() {
		hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, model.LastErrorCodeType(data.DIAGNOSIS_NO_ERROR))
	}
	return hasChanged
}

func (e *EnergyGuard) onTicErrorReadTimeout() (hasChanged bool) {
	e.data.SetHasMeterData(false)
	hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(tic_read_timeout))
	return hasChanged
}

func (e *EnergyGuard) onTicErrorCritical(errMsg string) (hasChanged bool) {
	e.data.SetHasMeterData(false)
	hasChanged = e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(errMsg))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
	return hasChanged
}

func (e *EnergyGuard) onTIC2WebsocketErrorAbnormalClosure() {
	e.data.SetHasMeterData(false)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType("TIC2Websocket abnormal closure"))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
}

func (e *EnergyGuard) onTic2WebsocketErrorConnectionFailure(host string, err error) error {
	e.data.SetHasMeterData(false)
	err = fmt.Errorf("TIC2WebsocketClient cannot connect with host '%s' : %s", host, err.Error())
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
	return err
}

func (e *EnergyGuard) onTic2WebsocketErrorCannotGetAvailableTic(err error) error {
	e.data.SetHasMeterData(false)
	err = fmt.Errorf("TIC2WebsocketClient cannot get available TICs : %s", err.Error())
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
	return err
}

func (e *EnergyGuard) onTic2WebsocketErrorCannotFindMeterSerialNumber(meterSerialNumber string) error {
	e.data.SetHasMeterData(false)
	err := fmt.Errorf("TIC2WebsocketClient cannot find meter serial number '%s'", meterSerialNumber)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
	return err
}

func (e *EnergyGuard) onTic2WebsocketErrorNoMeterSerialNumberAvailable() error {
	e.data.SetHasMeterData(false)
	err := fmt.Errorf("TIC2WebsocketClient has no meter serial number available")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
	e.clearTic2WebsocketSubscriptionIdAndAvailableTic()
	return err
}

func (e *EnergyGuard) onTic2WebsocketErrorSubscriptionFailure(availableTIC linkymeter.TicIdentifier, err error) error {
	e.data.SetHasMeterData(false)
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
	e.onOpevUseCaseSupported()
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
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("wallbox has been disconnected")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

// OnWallboxSupported
func (e *EnergyGuard) onWallboxSupported() {
	log.Info("Wallbox is supported")
	go e.readUsecasesInfos()
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoLocalDeviceFound() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : no local device found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedLocalEntityFound() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : expected local entity '%s' not found", entity_type)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedLocalFeatureNotFound(featureType model.FeatureTypeType) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : expected local feature '%s' not found", featureType)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoRemoteDeviceFound() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : no remote device found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorExpectedRemoteFeatureNotFound(featureType model.FeatureTypeType) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : expected remote feature '%s' not found", featureType)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorNoRemoteDeviceSenderFound() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : no remote device sender found")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorSendRequestFailed(function model.FunctionType, err error) {
	e.data.SetIsOpevSupported(false)
	err = fmt.Errorf("read use case info failure : send request %v failed (%v)", function, err)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorSendRequestMsgCounterNotDefined(function model.FunctionType) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("read use case info failure : send request %v message counter not defined", function)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReadUsecaseInfosErrorAddCallbackResponseFailed(function model.FunctionType, err error) {
	e.data.SetIsOpevSupported(false)
	err = fmt.Errorf("read use case info failure : cannot add response callback for function %v (%v)", function, err)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onReceiveUsecasesInfos(msg spineapi.ResponseMessage) {
	e.Debugf("onReceiveUsecasesInfos : %+v", msg)
	if msg.Data == nil {
		e.onOpevUsecaseErrorDataNotProvided()
		return
	}
	data, ok := msg.Data.(*model.NodeManagementUseCaseDataType)
	if !ok {
		e.onOpevUsecaseErrorDataTypeUnexpected(msg.Data)
		return
	}
	for _, info := range data.UseCaseInformation {
		for _, support := range info.UseCaseSupport {
			if support.UseCaseName == nil {
				continue
			}
			if *support.UseCaseName == opev_use_case_name {
				if support.UseCaseAvailable == nil {
					e.onOpevUsecaseErrorAvailableNotDefined()
					return
				}
				if !*support.UseCaseAvailable {
					e.onOpevUsecaseErrorNotAvailable()
					return
				}
				if support.UseCaseVersion == nil {
					e.onOpevUsecaseErrorVersionNotDefined()
					return
				}
				if *support.UseCaseVersion != opev_use_case_version {
					e.onOpevUsecaseErrorVersionUnexpected(*support.UseCaseVersion)
					return
				}
				if len(support.ScenarioSupport) == 0 {
					e.onOpevUsecaseErrorScenarioNotDefined()
					return
				}
				if !cmp.Equal(support.ScenarioSupport, opev_use_case_scenario) {
					e.onOpevUsecaseErrorScenarioUnexpected(support.ScenarioSupport)
					return
				}
				if support.UseCaseDocumentSubRevision == nil {
					e.onOpevUsecaseErrorDocumentSubRevisionNotDefined()
					return
				}
				if *support.UseCaseDocumentSubRevision != opev_use_case_document_sub_revision {
					e.onOpevUsecaseErrorDocumentSubRevisionUnexpected(*support.UseCaseDocumentSubRevision)
					return
				}
				e.onOpevUseCaseSupported()
			}
		}
	}
}

func (e *EnergyGuard) onOpevUseCaseSupported() {
	e.data.SetIsOpevSupported(true)
	if e.data.HasMeterData() && e.data.IsConnected() {
		e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, model.LastErrorCodeType(data.DIAGNOSIS_NO_ERROR))
	}
}

func (e *EnergyGuard) onOpevUsecaseErrorDataNotProvided() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case data not provided")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorDataTypeUnexpected(data any) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case data type '%v' unexpected (should be NodeManagementUseCaseDataType)", reflect.TypeOf(data))
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorAvailableNotDefined() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case available not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorNotAvailable() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case not available")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorVersionNotDefined() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case version not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorVersionUnexpected(version model.SpecificationVersionType) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case version '%s' unexpected (should be '%s')", version, opev_use_case_version)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorScenarioNotDefined() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case scenario not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorScenarioUnexpected(scenario []model.UseCaseScenarioSupportType) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case scenario '%v' unexpected (should be '%v')", scenario, opev_use_case_scenario)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorDocumentSubRevisionNotDefined() {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case document sub revision not defined")
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}

func (e *EnergyGuard) onOpevUsecaseErrorDocumentSubRevisionUnexpected(subRevision string) {
	e.data.SetIsOpevSupported(false)
	err := fmt.Errorf("OPEV use case document sub revision '%s' unexpected (should be '%s')", subRevision, opev_use_case_document_sub_revision)
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
	if e.data.HasMeterData() && e.data.IsConnected() {
		e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeNormalOperation, data.DIAGNOSIS_NO_ERROR)
	}
}

func (e *EnergyGuard) onVehicleLoadControlLimitsFailure(errorNumber model.ErrorNumberType, errorDescription model.DescriptionType) {
	err := fmt.Errorf("cannot write vehicle load control limit : ErrorNumber=%d, Description=%s", errorNumber, errorDescription)
	e.updateDiagnosis(model.DeviceDiagnosisOperatingStateTypeFailure, model.LastErrorCodeType(err.Error()))
}
