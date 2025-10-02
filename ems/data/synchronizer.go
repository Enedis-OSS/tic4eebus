// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
Package data is used for manipulating energy management system data model.

The data model can be synchronized with multiple data writers:

  - CSV file

  - InfluxDB database
*/
package data

import (
	"encoding/json"
	"slices"
	"sync"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/Enedis-OSS/tic4eebus/linkymeter"
	"github.com/enbility/spine-go/model"
	"github.com/google/go-cmp/cmp"
	log "github.com/sirupsen/logrus"
)

// Interface for data writers
type DataWriter interface {
	Save(model DataModel)
}

// DataSynchronizer is used to synchronize the data model and save it with multiple data writers
type DataSynchronizer struct {
	model                DataModel
	deviceDiagnosisState model.DeviceDiagnosisStateDataType
	access               sync.Mutex
	writers              []DataWriter
}

// NewDataSynchronizer creates a an instance of DataSynchronizer from data model configuration
func NewDataSynchronizer(dataModelConfig config.DataModelConfig) *DataSynchronizer {
	synchronizer := &DataSynchronizer{}

	synchronizer.model.IsConnected = false
	synchronizer.model.HasMeterData = false
	synchronizer.model.IsOpevSupported = false
	synchronizer.model.Vehicle = make(map[string]interface{})
	synchronizer.model.Wallbox = make(map[string]interface{})
	synchronizer.model.Diagnosis.OperatingState = model.DeviceDiagnosisOperatingStateTypeNormalOperation
	synchronizer.model.Diagnosis.LastErrorCode = model.LastErrorCodeType(DIAGNOSIS_NO_ERROR)
	synchronizer.deviceDiagnosisState.OperatingState = &synchronizer.model.Diagnosis.OperatingState
	synchronizer.deviceDiagnosisState.LastErrorCode = &synchronizer.model.Diagnosis.LastErrorCode

	csvWriter := NewCsvWriter(dataModelConfig.Csv)
	influxDbWriter := NewInfluxDbWriter(dataModelConfig.InfluxDb)
	synchronizer.writers = []DataWriter{csvWriter, influxDbWriter}

	return synchronizer
}

// IsConnected returns true if the EMS is connected to a wallbox
func (s *DataSynchronizer) IsConnected() (isConnected bool) {
	s.access.Lock()
	isConnected = s.model.IsConnected
	s.access.Unlock()

	return isConnected
}

// SetIsConnected sets the connection state of the EMS to a wallbox
//
// If the state has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetIsConnected(isConnected bool) (hasChanged bool) {
	s.access.Lock()
	if isConnected != s.model.IsConnected {
		s.model.IsConnected = isConnected
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// HasMeterData returns true if the Linky meter sends data to the EMS
func (s *DataSynchronizer) HasMeterData() (hasMeter bool) {
	s.access.Lock()
	hasMeter = s.model.HasMeterData
	s.access.Unlock()

	return hasMeter
}

// SetHasMeterData sets the state if the Linky meter sends data to the EMS
//
// If the state has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetHasMeterData(hasMeterData bool) (hasChanged bool) {
	s.access.Lock()
	if hasMeterData != s.model.HasMeterData {
		s.model.HasMeterData = hasMeterData
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// IsOpevSupported returns true if the EEBUS OPEV use case is supported by the wallbox
func (s *DataSynchronizer) IsOpevSupported() (isOpevSupported bool) {
	s.access.Lock()
	isOpevSupported = s.model.IsOpevSupported
	s.access.Unlock()

	return isOpevSupported
}

// SetIsOpevSupported sets the state if the EEBUS OPEV use case is supported by the wallbox
//
// If the state has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetIsOpevSupported(isOpevSupported bool) (hasChanged bool) {
	s.access.Lock()
	if isOpevSupported != s.model.IsOpevSupported {
		s.model.IsOpevSupported = isOpevSupported
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetVehicle returns a copy of the vehicle data map
func (s *DataSynchronizer) GetVehicle() (vehicle map[string]interface{}) {
	vehicle = make(map[string]interface{})
	s.access.Lock()
	for key, value := range s.model.Vehicle {
		vehicle[key] = value
	}
	s.access.Unlock()

	return vehicle
}

// SetVehicle updates the vehicle data map with the provided data
//
// If the data has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetVehicle(vehicle map[string]interface{}) (hasChanged bool) {
	s.access.Lock()
	// compare and update vehicle data
	for k, v := range vehicle {
		oldValue, ok := s.model.Vehicle[k]
		if !ok {
			s.model.Vehicle[k] = v
			hasChanged = true
		} else {
			if !cmp.Equal(v, oldValue) {
				s.model.Vehicle[k] = v
				hasChanged = true
			}
		}
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetWallbox returns a copy of the wallbox data map
func (s *DataSynchronizer) GetWallbox() (wallbox map[string]interface{}) {
	wallbox = make(map[string]interface{})
	s.access.Lock()
	for key, value := range s.model.Wallbox {
		wallbox[key] = value
	}
	s.access.Unlock()

	return wallbox
}

// SetWallbox updates the wallbox data map with the provided data
//
// If the data has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetWallbox(wallbox map[string]interface{}) (hasChanged bool) {
	s.access.Lock()
	for k, v := range wallbox {
		oldValue, ok := s.model.Wallbox[k]
		if !ok {
			s.model.Wallbox[k] = v
			hasChanged = true
		} else {
			if !cmp.Equal(v, oldValue) {
				s.model.Wallbox[k] = v
				hasChanged = true
			}
		}
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetMeter returns a copy of the Linky meter data
func (s *DataSynchronizer) GetMeter() (meter linkymeter.MeterData) {
	s.access.Lock()
	meter = linkymeter.MeterData(s.model.Meter)
	s.access.Unlock()

	return meter
}

// SetMeter updates the Linky meter data with the provided data
//
// If the data has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetMeter(meter linkymeter.MeterData) (hasChanged bool) {
	s.access.Lock()
	if !linkymeter.IsEqual(s.model.Meter, meter) {
		s.model.Meter = linkymeter.MeterData(meter)
		s.model.HasMeterData = true
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetMeterMinAvailableCurrent returns the minimum available current from the Linky meter data
func (s *DataSynchronizer) GetMeterMinAvailableCurrent() (minAvailableCurrent float64) {
	s.access.Lock()
	minAvailableCurrent = slices.Min(s.model.Meter.AvailableCurrentPerPhase)
	s.access.Unlock()

	return minAvailableCurrent
}

// GetOverloadProtection returns a copy of the overload protection data
func (s *DataSynchronizer) GetOverloadProtection() (overloadProtection OverloadProtectionData) {
	s.access.Lock()
	overloadProtection = OverloadProtectionData(s.model.OverloadProtection)
	s.access.Unlock()

	return overloadProtection
}

// GetDiagnosisState returns a pointer to the device diagnosis state data
func (s *DataSynchronizer) GetDiagnosisState() *model.DeviceDiagnosisStateDataType {
	return &s.deviceDiagnosisState
}

// GetDiagnosis returns a copy of the diagnosis data
func (e *DataSynchronizer) GetDiagnosis() (diagnosis DiagnosisData) {
	e.access.Lock()
	diagnosis = e.model.Diagnosis
	e.access.Unlock()

	return diagnosis
}

// SetDiagnosis updates the diagnosis data with the provided data
//
// If the data has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetDiagnosis(operatingState model.DeviceDiagnosisOperatingStateType, lastErrorCode model.LastErrorCodeType) (hasChanged bool) {
	s.access.Lock()
	if lastErrorCode != s.model.Diagnosis.LastErrorCode {
		s.model.Diagnosis.OperatingState = operatingState
		s.model.Diagnosis.LastErrorCode = lastErrorCode
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// DisableOverloadProtectionActive disables the overload protection active state
//
// If the state has changed, it saves the data model with the data writers
func (s *DataSynchronizer) DisableOverloadProtectionActive() {
	s.access.Lock()
	if s.model.OverloadProtection.Active {
		s.model.OverloadProtection.Value = 0
		s.model.OverloadProtection.Active = false
		s.access.Unlock()
		s.save()
		return
	}
	s.access.Unlock()
}

// GetOverloadProtectionValue returns the overload protection value
func (s *DataSynchronizer) GetOverloadProtectionValue() (limitValue float64) {
	s.access.Lock()
	limitValue = s.model.OverloadProtection.Value
	s.access.Unlock()

	return limitValue
}

// SetOverloadProtectionValue sets the overload protection value and activates the overload protection
//
// It saves the data model with the data writers
func (s *DataSynchronizer) SetOverloadProtectionValue(limitValue float64) {
	s.access.Lock()
	s.model.OverloadProtection.Value = limitValue
	s.model.OverloadProtection.Active = true
	s.model.OverloadProtection.Start = time.Now()
	s.access.Unlock()
	s.save()
}

// GetOverloadProtectionResult returns the overload protection result code and description
func (s *DataSynchronizer) SetOverloadProtectionResult(result model.ResultDataType) (hasChanged bool) {
	s.access.Lock()
	if result.ErrorNumber != nil {
		if *result.ErrorNumber != s.model.OverloadProtection.ResultCode {
			s.model.OverloadProtection.ResultCode = *result.ErrorNumber
			hasChanged = true
		}
		if result.Description != nil {
			if *result.Description != s.model.OverloadProtection.ResultDescription {
				s.model.OverloadProtection.ResultDescription = *result.Description
				hasChanged = true
			}
		} else {
			if s.model.OverloadProtection.ResultDescription != "" {
				s.model.OverloadProtection.ResultDescription = ""
				hasChanged = true
			}
		}
	} else {
		if s.model.OverloadProtection.ResultCode != model.ErrorNumberTypeGeneralError {
			s.model.OverloadProtection.ResultCode = model.ErrorNumberTypeGeneralError
			hasChanged = true
		}
		if s.model.OverloadProtection.ResultDescription != "undefined error" {
			s.model.OverloadProtection.ResultDescription = "undefined error"
			hasChanged = true
		}
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetOverloadProtectionLockStart returns the overload protection lock start time
func (s *DataSynchronizer) GetOverloadProtectionLockStart() (lockStart time.Time) {
	s.access.Lock()
	lockStart = s.model.OverloadProtection.LockStart
	s.access.Unlock()

	return lockStart
}

// SetOverloadProtectionLockStart sets the overload protection lock start time and activates the lock
//
// If the lock start time has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetOverloadProtectionLockStart(lockStart time.Time) (hasChanged bool) {
	s.access.Lock()
	if lockStart != s.model.OverloadProtection.Start {
		s.model.OverloadProtection.LockStart = lockStart
		s.model.OverloadProtection.LockActive = true
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetOverloadProtectionLockDuration returns the overload protection lock duration
func (s *DataSynchronizer) GetOverloadProtectionLockDuration() (lockDuration time.Duration) {
	now := time.Now()
	s.access.Lock()
	lockDuration = now.Sub(s.model.OverloadProtection.LockStart)
	s.access.Unlock()

	return lockDuration
}

// SetOverloadProtectionLockActive sets the overload protection lock active state
//
// If the state has changed, it saves the data model with the data writers and returns true
func (s *DataSynchronizer) SetOverloadProtectionLockActive(lockActive bool) (hasChanged bool) {
	s.access.Lock()
	if lockActive != s.model.OverloadProtection.LockActive {
		s.model.OverloadProtection.LockActive = lockActive
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

// GetModel returns a copy of the data model
func (s *DataSynchronizer) GetModel() (model DataModel) {
	model = DataModel{
		IsConnected:        s.IsConnected(),
		HasMeterData:       s.HasMeterData(),
		IsOpevSupported:    s.IsOpevSupported(),
		Vehicle:            s.GetVehicle(),
		Wallbox:            s.GetWallbox(),
		Meter:              s.GetMeter(),
		OverloadProtection: s.GetOverloadProtection(),
		Diagnosis:          s.GetDiagnosis(),
	}

	return model
}

// Print prints the data model as a formatted JSON string
func (s *DataSynchronizer) Print() {
	model := s.GetModel()
	jsonBytes, error := json.MarshalIndent(model, "", "  ")
	if error == nil {
		log.Infof("DataModel : \n%s\n", string(jsonBytes))
	}
}

// save saves the data model with the data writers
func (e *DataSynchronizer) save() {
	model := e.GetModel()
	for _, writer := range e.writers {
		if writer != nil {
			writer.Save(model)
		}
	}
}
