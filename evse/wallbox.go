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
	"github.com/enbility/eebus-go/usecases/cem/evsecc"
	spineapi "github.com/enbility/spine-go/api"
	"github.com/enbility/spine-go/model"
	"github.com/go-co-op/gocron"
	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

const (
	wallbox_use_case_evsecc    = "EVSECC"
	wallbox_use_case_supported = "UseCaseSupported"
)

// EVSECC wallbox data key used for connection state (true/false)
const WALLBOX_IS_CONNECTED = "IsConnected"

// EVSECC wallbox data key used for operating state
//
// See also: https://pkg.go.dev/github.com/enbility/spine-go@v0.7.0/model#DeviceDiagnosisOperatingStateType
const WALLBOX_OPERATING_STATE = "OperatingState"

// EVSECC wallbox data key used for manufacturer data
//
// See also: https://pkg.go.dev/github.com/enbility/eebus-go@v0.7.0/api#ManufacturerData
const WALLBOX_MANUFACTURER_DATA = "ManufacturerData"

type onWallboxData func(wallboxData map[string]interface{})
type onWallboxConnected func()
type onWallboxDisconnected func()
type onWallboxSupported func()

type wallboxDataSubscriber struct {
	onData         onWallboxData
	onConnected    onWallboxConnected
	onDisconnected onWallboxDisconnected
	onSupported    onWallboxSupported
}

// Wallbox handle a wallbox and its data
type Wallbox struct {
	data             map[string]interface{}
	dataAccess       sync.Mutex
	config           config.WallboxConfig
	subscriberAccess sync.Mutex
	subscriberMap    map[string]wallboxDataSubscriber
	useCase          struct {
		supported atomic.Bool
		handler   *evsecc.EVSECC
	}
	remoteConnection atomic.Bool
	remoteEntity     spineapi.EntityRemoteInterface
	scheduler        *gocron.Scheduler
	job              *gocron.Job
}

// NewWallbox creates a new Wallbox instance
func NewWallbox(
	service api.ServiceInterface,
	localEntity spineapi.EntityLocalInterface,
	config config.WallboxConfig,
) *Wallbox {

	wallbox := &Wallbox{}

	wallbox.data = make(map[string]interface{})
	wallbox.config = config
	wallbox.subscriberMap = make(map[string]wallboxDataSubscriber)

	wallbox.useCase.handler = evsecc.NewEVSECC(localEntity, wallbox.onEvent_EVSECC)
	wallbox.useCase.supported.Store(false)
	service.AddUseCase(wallbox.useCase.handler)

	wallbox.remoteConnection.Store(false)
	wallbox.remoteEntity = nil
	wallbox.scheduler = gocron.NewScheduler(time.UTC)

	return wallbox
}

// EnableRemoteConnection enables the remote connection to the wallbox
//
// The remote connection is enable by the energy management system when the EEBUS connection is established.
// When the remote connection is enabled, all datas available are read from the wallbox.
func (w *Wallbox) EnableRemoteConnection() {
	w.remoteConnection.Store(true)
}

// DisableRemoteConnection disables the remote connection to the wallbox
//
// The remote connection is disabled by the energy management system when the EEBUS connection is closed.
// When the remote connection is disabled, no data are read from the wallbox.
func (w *Wallbox) DisableRemoteConnection() {
	w.remoteConnection.Store(false)
}

// SubscribeData subscribes to wallbox data updates, connection and disconnection events, use case supported notification and returns a subscription ID
func (v *Wallbox) SubscribeData(onData onWallboxData, onConnected onWallboxConnected, onDisconnected onWallboxDisconnected, onSupported onWallboxSupported) (id string) {
	subscriber := wallboxDataSubscriber{onData: onData, onConnected: onConnected, onDisconnected: onDisconnected, onSupported: onSupported}
	id = uuid.New().String()
	v.subscriberAccess.Lock()
	v.subscriberMap[id] = subscriber
	v.subscriberAccess.Unlock()

	return id
}

// UnsubscribeData unsubscribes from wallbox data updates, connection and disconnection events, use case supported notification using the subscription ID
func (v *Wallbox) UnsubscribeData(id string) error {
	_, ok := v.subscriberMap[id]
	if !ok {
		return fmt.Errorf("subscriber id '%s' not found", id)
	}
	v.subscriberAccess.Lock()
	delete(v.subscriberMap, id)
	v.subscriberAccess.Unlock()

	return nil
}

func (v *Wallbox) notifyData() {
	data := v.GetData()
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onData(data)
	}
	v.subscriberAccess.Unlock()
}

func (v *Wallbox) notifyConnected() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onConnected()
	}
	v.subscriberAccess.Unlock()
}

func (v *Wallbox) notifyDisconnected() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onDisconnected()
	}
	v.subscriberAccess.Unlock()
}

func (v *Wallbox) notifySupported() {
	v.subscriberAccess.Lock()
	for _, subscriber := range v.subscriberMap {
		go subscriber.onSupported()
	}
	v.subscriberAccess.Unlock()
}

func (w *Wallbox) reset() {
	if !w.config.DataPersistent {
		w.dataAccess.Lock()
		for key := range w.data {
			delete(w.data, key)
		}
		w.dataAccess.Unlock()
	}

	w.useCase.supported.Store(false)
}

// IsConnected returns true if the connection with the wallbox is established, false otherwise
func (w *Wallbox) IsConnected() bool {
	var isConnected bool

	w.dataAccess.Lock()
	value, ok := w.data[WALLBOX_IS_CONNECTED]
	w.dataAccess.Unlock()

	if !ok {
		isConnected = false
	} else {
		isConnected = value.(bool)
	}

	return isConnected
}

func (w *Wallbox) setIsConnected(isConnected bool) {
	w.dataAccess.Lock()
	w.data[WALLBOX_IS_CONNECTED] = isConnected
	w.dataAccess.Unlock()
}

// GetData returns a copy of the wallbox data map
func (w *Wallbox) GetData() map[string]interface{} {
	data := make(map[string]interface{})

	w.dataAccess.Lock()
	for key, value := range w.data {
		data[key] = value
	}
	w.dataAccess.Unlock()

	return data
}

func (w *Wallbox) updateData() {
	if !w.remoteConnection.Load() {
		return
	}
	if w.useCase.supported.Load() {
		_, err := w.readAndSetManufacturerData()
		if err != nil {
			log.Errorf("Failed to read wallbox manufacturer data: %s", err.Error())
		}
		_, err = w.readAndSetOperatingState()
		if err != nil {
			log.Errorf("Failed to read wallbox operating state: %s", err.Error())
		}
	}
	w.notifyData()
}

func (w *Wallbox) startUpdateData() {
	if !w.scheduler.IsRunning() {
		w.scheduler.StartAsync()
	}
	var err error
	if w.job == nil {
		if w.config.UpdateDataPeriodInSeconds != 0 {
			w.job, err = w.scheduler.Every(w.config.UpdateDataPeriodInSeconds).Seconds().Do(w.updateData)
			if err != nil {
				log.Errorf("Failed to start wallbox data update: %s", err.Error())
			}
		}
	}
}

func (w *Wallbox) stopUpdateData() {
	w.scheduler.Stop()
}

func (w *Wallbox) readAndSetManufacturerData() (ucapi.ManufacturerData, error) {
	manufacturerData, error := w.useCase.handler.ManufacturerData(w.remoteEntity)

	w.dataAccess.Lock()
	if error == nil {
		w.data[WALLBOX_MANUFACTURER_DATA] = manufacturerData
	} else {
		if !w.config.DataPersistent {
			delete(w.data, WALLBOX_MANUFACTURER_DATA)
		}
	}
	w.dataAccess.Unlock()

	return manufacturerData, error
}

func (w *Wallbox) readAndSetOperatingState() (model.DeviceDiagnosisOperatingStateType, error) {
	operatingState, _, error := w.useCase.handler.OperatingState(w.remoteEntity)

	w.dataAccess.Lock()
	if error == nil {
		w.data[WALLBOX_OPERATING_STATE] = operatingState
	} else {
		if !w.config.DataPersistent {
			delete(w.data, WALLBOX_OPERATING_STATE)
		}
	}
	w.dataAccess.Unlock()

	return operatingState, error
}

// EVSECC Event Handler
func (w *Wallbox) onEvent_EVSECC(ski string, device spineapi.DeviceRemoteInterface, entity spineapi.EntityRemoteInterface, event api.EventType) {
	var dataName string
	var data interface{}
	var error error

	if !w.remoteConnection.Load() {
		return
	}
	useCaseName := wallbox_use_case_evsecc
	error = nil
	w.remoteEntity = entity

	switch event {
	case evsecc.UseCaseSupportUpdate:
		w.useCase.supported.Store(true)
		dataName = wallbox_use_case_supported
		data = true
		w.notifySupported()
	case evsecc.EvseConnected:
		w.setIsConnected(true)
		dataName = WALLBOX_IS_CONNECTED
		data = true
		w.startUpdateData()
		w.notifyConnected()
	case evsecc.EvseDisconnected:
		w.stopUpdateData()
		w.reset()
		w.setIsConnected(false)
		dataName = WALLBOX_IS_CONNECTED
		data = false
		w.notifyDisconnected()
	case evsecc.DataUpdateManufacturerData:
		data, error = w.readAndSetManufacturerData()
		dataName = WALLBOX_MANUFACTURER_DATA
	case evsecc.DataUpdateOperatingState:
		data, error = w.readAndSetOperatingState()
		dataName = WALLBOX_OPERATING_STATE
	}

	if error == nil {
		log.Infof("%s %s=%+v", useCaseName, dataName, data)
		w.notifyData()
	} else {
		log.Errorf("%s cannot read %s : %s", useCaseName, dataName, error.Error())
	}
}
