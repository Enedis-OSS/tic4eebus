// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
Package evse implements electrical vehicle and wallbox routines for data update and charge limitation modification.

The electrical vehicle referred to as "EV" in EEBus protocol is responsible for following scenarios:

  - notifying wallbox when it is connected or disconnected

  - provide data to wallbox (charge state, communication standard, identifications, manufacturer data, charging power limits, etc.)

  - apply charge limitations received from wallbox (current limits, load control limits)

The wallbox referred to as "EVSE" in EEBus protocol is responsible for following scenarios:

  - establishing and maintaining EEBus connection with EV

  - providing data to energy management system (charge state, communication standard, identifications, manufacturer data, charging power limits, phases connected, current per phase, power per phase, energy charged, current limits, load control limits)

  - sending charge limitations to EV (current limits, load control limits)

  - notifying energy management system when EV is connected or disconnected

  - notifying energy management system when OPEV use case is supported by EV

  - checking energy management system availability and applying default charge limitations when necessary

  - checking energy management system error state and applying default charge limitations when necessary
*/
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
	vehicle_use_case_supported = "UseCaseSupported"
	vehicle_use_case_evcc      = "EVCC"
	vehicle_use_case_evcem     = "EVCEM"
	vehicle_use_case_opev      = "OPEV"
)

// EVCC vehicle data key used for connection state (true/false)
const VEHICLE_IS_CONNECTED = "IsConnected"

// EVCC vehicle data key used for charge state
//
// See also: https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/usecases/api#EVChargeStateType
const VEHICLE_CHARGE_STATE = "ChargeState"

// EVCC vehicle data key used for communication standard
//
// See also: https://pkg.go.dev/github.com/enbility/spine-go@v0.7.0/model#DeviceConfigurationKeyValueStringType
const VEHICLE_COMMUNICATION_STANDARD = "CommunicationStandard"

// EVCC vehicle data key used for asymetric charging support (true/false)
const VEHICLE_ASYMETRIC_CHARGING_SUPPORT = "AsymetricChargingSupport"

// EVCC vehicle data key used for a list of IdentificationItem
//
// See also: https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/usecases/api#IdentificationItem
const VEHICLE_IDENTIFICATIONS = "Identifications"

// EVCC vehicle data key used for manufacturer data
//
// See also: https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/api#ManufacturerData
const VEHICLE_MANUFACTURER_DATA = "ManufacturerData"

// EVCC vehicle data key used for charging power limits
//
// See [ChargingPowerLimits]
const VEHICLE_CHARGING_POWER_LIMITS = "ChargingPowerLimits"

// EVCC vehicle data key used for sleep mode (true/false)
const VEHICLE_IS_IN_SLEEP_MODE = "IsInSleepMode"

// EVCEM vehicle data key used for number of phases connected (1, 2 or 3)
const VEHICLE_PHASES_CONNECTED = "PhasesConnected"

// EVCEM vehicle data key used for a list of current per phase in Amps
const VEHICLE_CURRENT_PER_PHASE = "CurrentPerPhase"

// EVCEM vehicle data key used for a list power per phase in Watts
const VEHICLE_POWER_PER_PHASE = "PowerPerPhase"

// EVCEM vehicle data key used for energy charged in kWh
const VEHICLE_ENERGY_CHARGED = "EnergyCharged"

// OPEV vehicle data key used for current limits
//
// See [CurrentLimits]
const VEHICLE_CURRENT_LIMITS = "CurrentLimits"

// OPEV vehicle data key used for load control limits
//
// See also: https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/usecases/api#LoadLimitsPhase
const VEHICLE_LOAD_CONTROL_LIMITS = "LoadControlLimits"

// ChargingPowerLimits represents the charging power limits of the vehicle
type ChargingPowerLimits struct {
	Min     float64 // minimum charging power in Watts
	Max     float64 // maximum charging power in Watts
	Standby float64 // standby power in Watts
}

// CurrentLimits represents the charging current limits of the vehicle
type CurrentLimits struct {
	Min     []float64 // minimum current limits in Amps
	Max     []float64 // maximum current limits in Amps
	Default []float64 // default current limits in Amps
}

type onVehicleData func(vehicleData map[string]interface{})
type onVehicleResult func(result model.ResultDataType)
type onVehicleConnected func()
type onVehicleDisconnected func()
type onVehicleOpevSupported func()

type vehicleEventSubscriber struct {
	onData          onVehicleData
	onConnected     onVehicleConnected
	onDisconnected  onVehicleDisconnected
	onOpevSupported onVehicleOpevSupported
}

// Vehicle handle an electrical vehicle and its data
type Vehicle struct {
	data             map[string]interface{}
	dataAccess       sync.Mutex
	config           config.VehicleConfig
	subscriberAccess sync.Mutex
	subscriberMap    map[string]vehicleEventSubscriber
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

// NewVehicle creates a new Vehicle instance with given EEBUS service, local entity and vehicle configuration
func NewVehicle(
	service api.ServiceInterface,
	localEntity spineapi.EntityLocalInterface,
	config config.VehicleConfig,
) *Vehicle {
	vehicle := &Vehicle{}

	vehicle.data = make(map[string]interface{})
	vehicle.config = config
	vehicle.subscriberMap = make(map[string]vehicleEventSubscriber)

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

// EnableRemoteConnection enables the remote connection to the vehicle
//
// The remote connection is enable by the energy management system when the EEBUS connection with the wallbox is established.
// When the remote connection is enabled, all data available are read from the vehicle.
func (v *Vehicle) EnableRemoteConnection() {
	v.remoteConnection.Store(true)
}

// DisableRemoteConnection disables the remote connection to the vehicle
//
// The remote connection is disabled by the energy management system when the EEBUS connection with the wallbox is closed.
// When the remote connection is disabled, all data are cleared (if not persistent) and no data is read from the vehicle.
func (v *Vehicle) DisableRemoteConnection() {
	v.remoteConnection.Store(false)
}

// SubscribeData subscribes to vehicle data updates, connection and disconnection events, OPEV supported notification and returns a subscription ID
func (v *Vehicle) SubscribeData(onData onVehicleData, onConnected onVehicleConnected, onDisconnected onVehicleDisconnected, onOPEVSupported onVehicleOpevSupported) (id string) {
	subscriber := vehicleEventSubscriber{onData: onData, onConnected: onConnected, onDisconnected: onDisconnected, onOpevSupported: onOPEVSupported}
	id = uuid.New().String()
	v.subscriberAccess.Lock()
	v.subscriberMap[id] = subscriber
	v.subscriberAccess.Unlock()

	return id
}

// UnsubscribeData unsubscribes from vehicle data updates, connection and disconnection events, OPEV supported notification using the subscription ID
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
		go subscriber.onOpevSupported()
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

// IsConnected returns true if the vehicle is connected with the wallbox, false otherwise
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

// GetData returns a copy of the vehicle data map
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

// WriteLoadControlLimits send the load limits request to the vehicle with a given callback called on vehicle result and returns a message counter or an error
func (v *Vehicle) WriteLoadControlLimits(limits []ucapi.LoadLimitsPhase, onResult onVehicleResult) (*model.MsgCounterType, error) {
	return v.useCase.opev.handler.WriteLoadControlLimits(v.remoteEntity, limits, onResult)
}

// EVCC Event Handler
func (v *Vehicle) onEvent_EVCC(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	var dataName string
	var data interface{}
	var error error

	if !v.remoteConnection.Load() {
		return
	}
	useCaseName := vehicle_use_case_evcc
	error = nil
	v.remoteEntity = entity

	switch event {
	case evcc.UseCaseSupportUpdate:
		v.useCase.evcc.supported.Store(true)
		dataName = vehicle_use_case_supported
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
	useCaseName := vehicle_use_case_evcem
	error = nil

	switch event {
	case evcem.UseCaseSupportUpdate:
		v.useCase.evcc.supported.Store(true)
		dataName = vehicle_use_case_supported
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
	useCaseName := vehicle_use_case_opev
	error = nil

	switch event {
	case opev.UseCaseSupportUpdate:
		v.useCase.opev.supported.Store(true)
		dataName = vehicle_use_case_supported
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
