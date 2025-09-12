// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package evse

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/enbility/eebus-go/api"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/eebus-go/usecases/cem/evcc"
	"github.com/enbility/eebus-go/usecases/cem/evcem"
	"github.com/enbility/eebus-go/usecases/cem/opev"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	VEHICLE_USE_CASE_SUPPORTED = "UseCaseSupported"
	VEHICLE_USE_CASE_EVCC      = "EVCC"
	VEHICLE_USE_CASE_EVCEM     = "EVCEM"
	VEHICLE_USE_CASE_OPEV      = "OPEV"
	// EVCC datas
	VEHICLE_IS_CONNECTED               = "IsConnected"
	VEHICLE_CHARGE_STATE               = "ChargeState"
	VEHICLE_COMMUNICATION_STANDARD     = "CommunicationStandard"
	VEHICLE_ASYMETRIC_CHARGING_SUPPORT = "AsymetricChargingSupport"
	VEHICLE_IDENTIFICATIONS            = "Identifications"
	VEHICLE_MANUFACTURER_DATA          = "ManufacturerData"
	VEHICLE_CHARGING_POWER_LIMITS      = "ChargingPowerLimits"
	VEHICLE_IS_IN_SLEEP_MODE           = "IsInSleepMode"
	// EVCEM datas
	VEHICLE_PHASES_CONNECTED  = "PhasesConnected"
	VEHICLE_CURRENT_PER_PHASE = "CurrentPerPhase"
	VEHICLE_POWER_PER_PHASE   = "PowerPerPhase"
	VEHICLE_ENERGY_CHARGED    = "EnergyCharged"
	// OPEV datas
	VEHICLE_CURRENT_LIMITS      = "CurrentLimits"
	VEHICLE_LOAD_CONTROL_LIMITS = "LoadControlLimits"
)

type ChargingPowerLimits struct {
	Min     float64
	Max     float64
	Standby float64
}

type CurrentLimits struct {
	Min     []float64
	Max     []float64
	Default []float64
}

type OnVehicleData func(vehicleData map[string]interface{})
type OnVehicleResult func(result model.ResultDataType)
type OnVehicleConnected func()
type OnVehicleDisconnected func()
type OnVehicleOPEVSupported func()

type VehicleDataSubscriber struct {
	onData          OnVehicleData
	onConnected     OnVehicleConnected
	onDisconnected  OnVehicleDisconnected
	onOPEVSupported OnVehicleOPEVSupported
}

type Vehicle struct {
	data             map[string]interface{}
	dataAccess       sync.Mutex
	config           config.VehicleConfig
	subscriberAccess sync.Mutex
	subscriberMap    map[string]VehicleDataSubscriber
	useCase          struct {
		evcc struct {
			supported atomic.Bool
			handler   *evcc.EVCC
		}
		evcem struct {
			supported atomic.Bool
			handler   *evcem.EVCEM
		}
		opev struct {
			supported atomic.Bool
			handler   *opev.OPEV
		}
	}
	remoteConnection atomic.Bool
	remoteEntity     spineapi.EntityRemoteInterface
	scheduler        *gocron.Scheduler
	job              *gocron.Job
}

func NewVehicle(
	service api.ServiceInterface,
	localEntity spineapi.EntityLocalInterface,
	config config.VehicleConfig,
) *Vehicle {
	vehicle := &Vehicle{}

	vehicle.data = make(map[string]interface{})
	vehicle.config = config
	vehicle.subscriberMap = make(map[string]VehicleDataSubscriber)

	vehicle.useCase.evcc.handler = evcc.NewEVCC(service, localEntity, vehicle.onEvent_EVCC)
	vehicle.useCase.evcc.supported.Store(false)
	vehicle.useCase.evcem.handler = evcem.NewEVCEM(service, localEntity, vehicle.onEvent_EVCEM)
	vehicle.useCase.evcem.supported.Store(false)
	vehicle.useCase.opev.handler = opev.NewOPEV(localEntity, vehicle.onEvent_OPEV)
	vehicle.useCase.opev.supported.Store(false)

	service.AddUseCase(vehicle.useCase.evcc.handler)
	service.AddUseCase(vehicle.useCase.evcem.handler)
	service.AddUseCase(vehicle.useCase.opev.handler)

	vehicle.remoteConnection.Store(false)
	vehicle.remoteEntity = nil
	vehicle.scheduler = gocron.NewScheduler(time.UTC)

	return vehicle
}

func (v *Vehicle) EnableRemoteConnection() {
	v.remoteConnection.Store(true)
}

func (v *Vehicle) DisableRemoteConnection() {
	v.remoteConnection.Store(false)
}

func (v *Vehicle) SubscribeData(onData OnVehicleData, onConnected OnVehicleConnected, onDisconnected OnVehicleDisconnected, onOPEVSupported OnVehicleOPEVSupported) (id string) {
	subscriber := VehicleDataSubscriber{onData: onData, onConnected: onConnected, onDisconnected: onDisconnected, onOPEVSupported: onOPEVSupported}
	id = uuid.New().String()
	v.subscriberAccess.Lock()
	v.subscriberMap[id] = subscriber
	v.subscriberAccess.Unlock()

	return id
}

func (v *Vehicle) UnsubscribeData(id string) error {
	_, ok := v.subscriberMap[id]
	if !ok {
		return fmt.Errorf("subscriber id '%s' not found", id)
	}
	v.subscriberAccess.Lock()
	delete(v.subscriberMap, id)
	v.subscriberAccess.Unlock()

	return nil
}

func (v *Vehicle) notifyData() {
	data := v.GetData()
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onData(data)
	}
	v.subscriberAccess.Unlock()
}

func (v *Vehicle) notifyConnected() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onConnected()
	}
	v.subscriberAccess.Unlock()
}

func (v *Vehicle) notifyDisconnected() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onDisconnected()
	}
	v.subscriberAccess.Unlock()
}

func (v *Vehicle) notifyOPEVSupported() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onOPEVSupported()
	}
	v.subscriberAccess.Unlock()
}

func (v *Vehicle) reset() {
	if !v.config.DataPersistent {
		v.dataAccess.Lock()
		for key := range v.data {
			delete(v.data, key)
		}
		v.dataAccess.Unlock()
	}

	v.useCase.evcc.supported.Store(false)
	v.useCase.evcem.supported.Store(false)
	v.useCase.opev.supported.Store(false)
}

func (v *Vehicle) IsConnected() bool {
	var isConnected bool

	v.dataAccess.Lock()
	value, ok := v.data[VEHICLE_IS_CONNECTED]
	v.dataAccess.Unlock()

	if !ok {
		isConnected = false
	} else {
		isConnected = value.(bool)
	}

	return isConnected
}

func (v *Vehicle) setIsConnected(isConnected bool) {
	v.dataAccess.Lock()
	v.data[VEHICLE_IS_CONNECTED] = isConnected
	v.dataAccess.Unlock()
}

func (v *Vehicle) GetData() map[string]interface{} {
	data := make(map[string]interface{})

	v.dataAccess.Lock()
	for key, value := range v.data {
		data[key] = value
	}
	v.dataAccess.Unlock()

	return data
}

func (v *Vehicle) updateData() {
	if !v.remoteConnection.Load() {
		return
	}
	// EVCC
	_, err := v.readAndSetChargeState()
	if err != nil {
		log.Errorf("Failed to read vehicle charge state: %s", err.Error())
	}
	_, err = v.readAndSetCommunicationStandard()
	if err != nil {
		log.Errorf("Failed to read vehicle communication standard: %s", err.Error())
	}
	_, err = v.readAndSetAsymmetricChargingSupport()
	if err != nil {
		log.Errorf("Failed to read vehicle asymmetric charging support: %s", err.Error())
	}
	_, err = v.readAndSetIdentifications()
	if err != nil {
		log.Errorf("Failed to read vehicle identifications: %s", err.Error())
	}
	_, err = v.readAndSetManufacturerData()
	if err != nil {
		log.Errorf("Failed to read vehicle manufacturer data: %s", err.Error())
	}
	_, err = v.readAndSetChargingPowerLimits()
	if err != nil {
		log.Errorf("Failed to read vehicle charging power limits: %s", err.Error())
	}
	_, err = v.readAndSetIsInSleepMode()
	if err != nil {
		log.Errorf("Failed to read vehicle sleep mode status: %s", err.Error())
	}
	// EVCEM
	_, err = v.readAndSetPhasesConnected()
	if err != nil {
		log.Errorf("Failed to read vehicle phases connected: %s", err.Error())
	}
	_, err = v.readAndSetCurrentPerPhase()
	if err != nil {
		log.Errorf("Failed to read vehicle current per phase: %s", err.Error())
	}
	_, err = v.readAndSetPowerPerPhase()
	if err != nil {
		log.Errorf("Failed to read vehicle power per phase: %s", err.Error())
	}
	_, err = v.readAndSetEnergyCharged()
	if err != nil {
		log.Errorf("Failed to read vehicle energy charged: %s", err.Error())
	}
	// OPEV
	_, err = v.readAndSetCurrentLimits()
	if err != nil {
		log.Errorf("Failed to read vehicle current limits: %s", err.Error())
	}
	_, err = v.readAndSetLoadControlLimits()
	if err != nil {
		log.Errorf("Failed to read vehicle load control limits: %s", err.Error())
	}
	// Notify subscribers
	v.notifyData()
}

func (v *Vehicle) startUpdateData() {
	if !v.scheduler.IsRunning() {
		v.scheduler.StartAsync()
	}
	var err error
	if v.job == nil {
		if v.config.UpdateDataPeriodInSeconds > 0 {
			v.job, err = v.scheduler.Every(v.config.UpdateDataPeriodInSeconds).Seconds().Do(v.updateData)
			if err != nil {
				log.Errorf("Failed to start vehicle data update: %s", err.Error())
			}
		}
	}
}

func (v *Vehicle) stopUpdateData() {
	v.scheduler.Stop()
}

func (v *Vehicle) readAndSetChargeState() (ucapi.EVChargeStateType, error) {
	chargeState, error := v.useCase.evcc.handler.ChargeState(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_CHARGE_STATE] = chargeState
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_CHARGE_STATE)
		}
	}
	v.dataAccess.Unlock()

	return chargeState, error
}

func (v *Vehicle) readAndSetCommunicationStandard() (model.DeviceConfigurationKeyValueStringType, error) {
	communicationStandard, error := v.useCase.evcc.handler.CommunicationStandard(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_COMMUNICATION_STANDARD] = communicationStandard
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_COMMUNICATION_STANDARD)
		}
	}
	v.dataAccess.Unlock()

	return communicationStandard, error
}

func (v *Vehicle) readAndSetAsymmetricChargingSupport() (bool, error) {
	asymetricChargingSupport, error := v.useCase.evcc.handler.AsymmetricChargingSupport(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_ASYMETRIC_CHARGING_SUPPORT] = asymetricChargingSupport
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_ASYMETRIC_CHARGING_SUPPORT)
		}
	}
	v.dataAccess.Unlock()

	return asymetricChargingSupport, error
}

func (v *Vehicle) readAndSetIdentifications() ([]ucapi.IdentificationItem, error) {
	identifications, error := v.useCase.evcc.handler.Identifications(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_IDENTIFICATIONS] = identifications
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_IDENTIFICATIONS)
		}
	}
	v.dataAccess.Unlock()

	return identifications, error
}

func (v *Vehicle) readAndSetManufacturerData() (ucapi.ManufacturerData, error) {
	manufacturerData, error := v.useCase.evcc.handler.ManufacturerData(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_MANUFACTURER_DATA] = manufacturerData
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_MANUFACTURER_DATA)
		}
	}
	v.dataAccess.Unlock()

	return manufacturerData, error
}

func (v *Vehicle) readAndSetChargingPowerLimits() (ChargingPowerLimits, error) {
	var chargingPowerLimits ChargingPowerLimits
	var error error

	chargingPowerLimits.Min, chargingPowerLimits.Max, chargingPowerLimits.Standby, error = v.useCase.evcc.handler.ChargingPowerLimits(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_CHARGING_POWER_LIMITS] = chargingPowerLimits
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_CHARGING_POWER_LIMITS)
		}
	}
	v.dataAccess.Unlock()

	return chargingPowerLimits, error
}

func (v *Vehicle) readAndSetIsInSleepMode() (bool, error) {
	isInSleepMode, error := v.useCase.evcc.handler.IsInSleepMode(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_IS_IN_SLEEP_MODE] = isInSleepMode
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_IS_IN_SLEEP_MODE)
		}
	}
	v.dataAccess.Unlock()

	return isInSleepMode, error
}

func (v *Vehicle) readAndSetPhasesConnected() (uint, error) {
	phasesConnected, error := v.useCase.evcem.handler.PhasesConnected(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_PHASES_CONNECTED] = phasesConnected
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_PHASES_CONNECTED)
		}
	}
	v.dataAccess.Unlock()

	return phasesConnected, error
}

func (v *Vehicle) readAndSetCurrentPerPhase() ([]float64, error) {
	currentPerPhase, error := v.useCase.evcem.handler.CurrentPerPhase(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_CURRENT_PER_PHASE] = currentPerPhase
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_CURRENT_PER_PHASE)
		}
	}
	v.dataAccess.Unlock()

	return currentPerPhase, error
}

func (v *Vehicle) readAndSetPowerPerPhase() ([]float64, error) {
	powerPerPhase, error := v.useCase.evcem.handler.PowerPerPhase(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_POWER_PER_PHASE] = powerPerPhase
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_POWER_PER_PHASE)
		}
	}
	v.dataAccess.Unlock()

	return powerPerPhase, error
}

func (v *Vehicle) readAndSetEnergyCharged() (float64, error) {
	energyCharged, error := v.useCase.evcem.handler.EnergyCharged(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_ENERGY_CHARGED] = energyCharged
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_ENERGY_CHARGED)
		}
	}
	v.dataAccess.Unlock()

	return energyCharged, error
}

func (v *Vehicle) readAndSetCurrentLimits() (CurrentLimits, error) {
	var currentLimits CurrentLimits
	var error error

	currentLimits.Min, currentLimits.Max, currentLimits.Default, error = v.useCase.opev.handler.CurrentLimits(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_CURRENT_LIMITS] = currentLimits
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_CURRENT_LIMITS)
		}
	}
	v.dataAccess.Unlock()

	return currentLimits, error
}

func (v *Vehicle) readAndSetLoadControlLimits() ([]ucapi.LoadLimitsPhase, error) {
	loadControlLimits, error := v.useCase.opev.handler.LoadControlLimits(v.remoteEntity)

	v.dataAccess.Lock()
	if error == nil {
		v.data[VEHICLE_LOAD_CONTROL_LIMITS] = loadControlLimits
	} else {
		if !v.config.DataPersistent {
			delete(v.data, VEHICLE_LOAD_CONTROL_LIMITS)
		}
	}
	v.dataAccess.Unlock()

	return loadControlLimits, error
}

func (v *Vehicle) WriteLoadControlLimits(limits []ucapi.LoadLimitsPhase, onResult OnVehicleResult) (*model.MsgCounterType, error) {
	return v.useCase.opev.handler.WriteLoadControlLimits(v.remoteEntity, limits, onResult)
}

func (v *Vehicle) onEvent_EVCC(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	var dataName string
	var data interface{}
	var error error

	if !v.remoteConnection.Load() {
		return
	}
	useCaseName := VEHICLE_USE_CASE_EVCC
	error = nil
	v.remoteEntity = entity

	switch event {
	case evcc.UseCaseSupportUpdate:
		v.useCase.evcc.supported.Store(true)
		dataName = VEHICLE_USE_CASE_SUPPORTED
		data = true
	case evcc.EvConnected:
		v.setIsConnected(true)
		dataName = VEHICLE_IS_CONNECTED
		data = true
		v.startUpdateData()
		v.notifyConnected()
	case evcc.EvDisconnected:
		v.stopUpdateData()
		v.reset()
		v.setIsConnected(false)
		dataName = VEHICLE_IS_CONNECTED
		data = false
		v.notifyDisconnected()
	case evcc.DataUpdateChargeState:
		data, error = v.readAndSetChargeState()
		dataName = VEHICLE_CHARGE_STATE
	case evcc.DataUpdateCommunicationStandard:
		data, error = v.readAndSetCommunicationStandard()
		dataName = VEHICLE_COMMUNICATION_STANDARD
	case evcc.DataUpdateAsymmetricChargingSupport:
		data, error = v.readAndSetAsymmetricChargingSupport()
		dataName = VEHICLE_ASYMETRIC_CHARGING_SUPPORT
	case evcc.DataUpdateIdentifications:
		data, error = v.readAndSetIdentifications()
		dataName = VEHICLE_IDENTIFICATIONS
	case evcc.DataUpdateManufacturerData:
		data, error = v.readAndSetManufacturerData()
		dataName = VEHICLE_MANUFACTURER_DATA
	case evcc.DataUpdateCurrentLimits:
		data, error = v.readAndSetChargingPowerLimits()
		dataName = VEHICLE_CHARGING_POWER_LIMITS
	case evcc.DataUpdateIsInSleepMode:
		data, error = v.readAndSetIsInSleepMode()
		dataName = VEHICLE_IS_IN_SLEEP_MODE
	}

	if error == nil {
		log.Infof("%s %s=%+v", useCaseName, dataName, data)
		v.notifyData()
	} else {
		log.Errorf("%s cannot read %s : %s", useCaseName, dataName, error.Error())
	}
}

// EVCEM Event Handler

func (v *Vehicle) onEvent_EVCEM(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	var dataName string
	var data interface{}
	var error error

	if !v.remoteConnection.Load() {
		return
	}
	useCaseName := VEHICLE_USE_CASE_EVCEM
	error = nil

	switch event {
	case evcem.UseCaseSupportUpdate:
		v.useCase.evcc.supported.Store(true)
		dataName = VEHICLE_USE_CASE_SUPPORTED
		data = true
	case evcem.DataUpdatePhasesConnected:
		data, error = v.readAndSetPhasesConnected()
		dataName = VEHICLE_PHASES_CONNECTED
	case evcem.DataUpdateCurrentPerPhase:
		data, error = v.readAndSetCurrentPerPhase()
		dataName = VEHICLE_CURRENT_PER_PHASE
	case evcem.DataUpdatePowerPerPhase:
		data, error = v.readAndSetPowerPerPhase()
		dataName = VEHICLE_POWER_PER_PHASE
	case evcem.DataUpdateEnergyCharged:
		data, error = v.readAndSetEnergyCharged()
		dataName = VEHICLE_ENERGY_CHARGED
	}

	if error == nil {
		log.Infof("%s %s=%+v", useCaseName, dataName, data)
		v.notifyData()
	} else {
		log.Errorf("%s cannot read %s : %s", useCaseName, dataName, error.Error())
	}
}

// OPEV Event Handler

func (v *Vehicle) onEvent_OPEV(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	var dataName string
	var data interface{}
	var error error

	if !v.remoteConnection.Load() {
		return
	}
	useCaseName := VEHICLE_USE_CASE_OPEV
	error = nil

	switch event {
	case opev.UseCaseSupportUpdate:
		v.useCase.opev.supported.Store(true)
		dataName = VEHICLE_USE_CASE_SUPPORTED
		data = true
		v.notifyOPEVSupported()
	case opev.DataUpdateCurrentLimits:
		data, error = v.readAndSetCurrentLimits()
		dataName = VEHICLE_CURRENT_LIMITS
	case opev.DataUpdateLimit:
		data, error = v.readAndSetLoadControlLimits()
		dataName = VEHICLE_LOAD_CONTROL_LIMITS
	}

	if error == nil {
		log.Infof("%s %s=%+v", useCaseName, dataName, data)
		v.notifyData()
	} else {
		log.Errorf("%s cannot read %s : %s", useCaseName, dataName, error.Error())
	}
}
