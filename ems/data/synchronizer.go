// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

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

type DataWriter interface {
	Save(model DataModel)
}

type DataSynchronizer struct {
	model                DataModel
	deviceDiagnosisState model.DeviceDiagnosisStateDataType
	access               sync.Mutex
	writers              []DataWriter
}

func NewDataSynchronizer(dataModelConfig config.DataModelConfig) *DataSynchronizer {
	synchronizer := &DataSynchronizer{}

	synchronizer.model.IsConnected = false
	synchronizer.model.HasMeter = false
	synchronizer.model.HasMeter = false
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

func (s *DataSynchronizer) IsConnected() (isConnected bool) {
	s.access.Lock()
	isConnected = s.model.IsConnected
	s.access.Unlock()

	return isConnected
}

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

func (s *DataSynchronizer) HasMeter() (hasMeter bool) {
	s.access.Lock()
	hasMeter = s.model.HasMeter
	s.access.Unlock()

	return hasMeter
}

func (s *DataSynchronizer) SetHasMeter(hasMeter bool) (hasChanged bool) {
	s.access.Lock()
	if hasMeter != s.model.HasMeter {
		s.model.HasMeter = hasMeter
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

func (s *DataSynchronizer) HasOPEV() (hasOPEV bool) {
	s.access.Lock()
	hasOPEV = s.model.HasOPEV
	s.access.Unlock()

	return hasOPEV
}

func (s *DataSynchronizer) SetHasOPEV(hasOPEV bool) (hasChanged bool) {
	s.access.Lock()
	if hasOPEV != s.model.HasOPEV {
		s.model.HasOPEV = hasOPEV
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

func (s *DataSynchronizer) GetVehicle() (vehicle map[string]interface{}) {
	vehicle = make(map[string]interface{})
	s.access.Lock()
	for key, value := range s.model.Vehicle {
		vehicle[key] = value
	}
	s.access.Unlock()

	return vehicle
}

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

func (s *DataSynchronizer) GetWallbox() (wallbox map[string]interface{}) {
	wallbox = make(map[string]interface{})
	s.access.Lock()
	for key, value := range s.model.Wallbox {
		wallbox[key] = value
	}
	s.access.Unlock()

	return wallbox
}

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

func (s *DataSynchronizer) GeMeter() (meter linkymeter.MeterData) {
	s.access.Lock()
	meter = linkymeter.MeterData(s.model.Meter)
	s.access.Unlock()

	return meter
}

func (s *DataSynchronizer) SetMeter(meter linkymeter.MeterData) (hasChanged bool) {
	s.access.Lock()
	if !linkymeter.IsEqual(s.model.Meter, meter) {
		s.model.Meter = linkymeter.MeterData(meter)
		s.model.HasMeter = true
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}

func (s *DataSynchronizer) GeMeterMinAvailableCurrent() (minAvailableCurrent float64) {
	s.access.Lock()
	minAvailableCurrent = slices.Min(s.model.Meter.AvailableCurrentPerPhase)
	s.access.Unlock()

	return minAvailableCurrent
}

func (s *DataSynchronizer) GeOverloadProtection() (overloadProtection OverloadProtectionData) {
	s.access.Lock()
	overloadProtection = OverloadProtectionData(s.model.OverloadProtection)
	s.access.Unlock()

	return overloadProtection
}

func (s *DataSynchronizer) SetOverloadProtection(overloadprotection OverloadProtectionData) (hasChanged bool) {
	s.access.Lock()
	if !cmp.Equal(s.model.OverloadProtection, overloadprotection) {
		s.model.OverloadProtection = OverloadProtectionData(overloadprotection)
		hasChanged = true
	}
	s.access.Unlock()
	if hasChanged {
		s.save()
	}

	return hasChanged
}
func (s *DataSynchronizer) GetDiagnosisState() *model.DeviceDiagnosisStateDataType {
	return &s.deviceDiagnosisState
}

func (e *DataSynchronizer) GetDiagnosis() (diagnosis DiagnosisData) {
	e.access.Lock()
	diagnosis = e.model.Diagnosis
	e.access.Unlock()

	return diagnosis
}

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

func (s *DataSynchronizer) GetOverloadProtectionValue() (limitValue float64) {
	s.access.Lock()
	limitValue = s.model.OverloadProtection.Value
	s.access.Unlock()

	return limitValue
}

func (s *DataSynchronizer) SetOverloadProtectionValue(limitValue float64) {
	s.access.Lock()
	s.model.OverloadProtection.Value = limitValue
	s.model.OverloadProtection.Active = true
	s.model.OverloadProtection.Start = time.Now()
	s.access.Unlock()
	s.save()
}

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

func (s *DataSynchronizer) GetOverloadProtectionLockStart() (lockStart time.Time) {
	s.access.Lock()
	lockStart = s.model.OverloadProtection.LockStart
	s.access.Unlock()

	return lockStart
}

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

func (s *DataSynchronizer) GetOverloadProtectionLockDuration() (lockDuration time.Duration) {
	now := time.Now()
	s.access.Lock()
	lockDuration = now.Sub(s.model.OverloadProtection.LockStart)
	s.access.Unlock()

	return lockDuration
}

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

func (s *DataSynchronizer) GetModel() (model DataModel) {
	model = DataModel{
		IsConnected:        s.IsConnected(),
		HasMeter:           s.HasMeter(),
		HasOPEV:            s.HasOPEV(),
		Vehicle:            s.GetVehicle(),
		Wallbox:            s.GetWallbox(),
		Meter:              s.GeMeter(),
		OverloadProtection: s.GeOverloadProtection(),
		Diagnosis:          s.GetDiagnosis(),
	}

	return model
}

func (s *DataSynchronizer) Print() {
	model := s.GetModel()
	jsonBytes, error := json.MarshalIndent(model, "", "  ")
	if error == nil {
		log.Infof("DataModel : \n%s\n", string(jsonBytes))
	}
}

func (e *DataSynchronizer) save() {
	model := e.GetModel()
	for _, writer := range e.writers {
		if writer != nil {
			writer.Save(model)
		}
	}
}
