// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package ems

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/Enedis-OSS/tic4eebus/evse"
	"github.com/Enedis-OSS/tic4eebus/linkymeter"
	ucapi "github.com/enbility/eebus-go/usecases/api"
	"github.com/enbility/spine-go/model"
	"github.com/google/go-cmp/cmp"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	log "github.com/sirupsen/logrus"
)

const (
	CSV_COLUMN_SEPARATOR   = ';'
	CSV_FLOATING_PRECISION = 3
	DIAGNOSIS_NO_ERROR     = "No error"
)

var (
	CSV_COLUMNS = []string{
		"Timestamp",
		"IsConnected",
		"HasMeter",
		"HasOPEV",
		"EV_IsConnected",
		"EV_AsymetricChargingSupport",
		"EV_ChargeState",
		"EV_ChargingPowerLimits_Min",
		"EV_ChargingPowerLimits_Max",
		"EV_ChargingPowerLimits_Standby",
		"EV_CurrentLimits_Min1",
		"EV_CurrentLimits_Min2",
		"EV_CurrentLimits_Min3",
		"EV_CurrentLimits_Max1",
		"EV_CurrentLimits_Max2",
		"EV_CurrentLimits_Max3",
		"EV_CurrentLimits_Default1",
		"EV_CurrentLimits_Default2",
		"EV_CurrentLimits_Default3",
		"EV_CurrentPerPhase1",
		"EV_CurrentPerPhase2",
		"EV_CurrentPerPhase3",
		"EV_EnergyCharged",
		"EV_Identifications",
		"EV_IsInSleepMode",
		"EV_LoadControlLimits1_IsChangeable",
		"EV_LoadControlLimits1_IsActive",
		"EV_LoadControlLimits1_Value",
		"EV_LoadControlLimits2_IsChangeable",
		"EV_LoadControlLimits2_IsActive",
		"EV_LoadControlLimits2_Value",
		"EV_LoadControlLimits3_IsChangeable",
		"EV_LoadControlLimits3_IsActive",
		"EV_LoadControlLimits3_Value",
		"EV_ManufacturerData_DeviceName",
		"EV_ManufacturerData_DeviceCode",
		"EV_ManufacturerData_SerialNumber",
		"EV_ManufacturerData_SoftwareRevision",
		"EV_ManufacturerData_HardwareRevision",
		"EV_ManufacturerData_VendorName",
		"EV_ManufacturerData_VendorCode",
		"EV_ManufacturerData_BrandName",
		"EV_ManufacturerData_PowerSource",
		"EV_ManufacturerData_ManufacturerNodeIdentification",
		"EV_ManufacturerData_ManufacturerLabel",
		"EV_ManufacturerData_ManufacturerDescription",
		"EV_PhasesConnected",
		"EV_PowerPerPhase1",
		"EV_PowerPerPhase1",
		"EV_PowerPerPhase3",
		"EVSE_IsConnected",
		"EVSE_ManufacturerData_DeviceName",
		"EVSE_ManufacturerData_DeviceCode",
		"EVSE_ManufacturerData_SerialNumber",
		"EVSE_ManufacturerData_SoftwareRevision",
		"EVSE_ManufacturerData_HardwareRevision",
		"EVSE_ManufacturerData_VendorName",
		"EVSE_ManufacturerData_VendorCode",
		"EVSE_ManufacturerData_BrandName",
		"EVSE_ManufacturerData_PowerSource",
		"EVSE_ManufacturerData_ManufacturerNodeIdentification",
		"EVSE_ManufacturerData_ManufacturerLabel",
		"EVSE_ManufacturerData_ManufacturerDescription",
		"EVSE_OperatingState",
		"Meter_SerialNumber",
		"Meter_DateTime",
		"Meter_BreakerOpened",
		"Meter_PhaseCount",
		"Meter_OverLoadPowerLimit",
		"Meter_OverLoadCurrentLimit1",
		"Meter_OverLoadCurrentLimit2",
		"Meter_OverLoadCurrentLimit3",
		"Meter_RmsVoltage1",
		"Meter_RmsVoltage2",
		"Meter_RmsVoltage3",
		"Meter_RmsCurrent1",
		"Meter_RmsCurrent2",
		"Meter_RmsCurrent3",
		"Meter_ApparentImportPower",
		"Meter_ApparentImportPower1",
		"Meter_ApparentImportPower2",
		"Meter_ApparentImportPower3",
		"Meter_AvailableCurrent1",
		"Meter_AvailableCurrent2",
		"Meter_AvailableCurrent3",
		"OverloadProtection_Active",
		"OverloadProtection_Value",
		"OverloadProtection_Start",
		"OverloadProtection_ResultCode",
		"OverloadProtection_ResultDescription",
		"OverloadProtection_LockActive",
		"OverloadProtection_LockStart",
		"Diagnosis_OperatingState",
		"Diagnosis_LastErrorCode",
	}
)

type OverloadProtectionData struct {
	Active            bool                  `json:"Active"`
	Value             float64               `json:"Value"`
	Start             time.Time             `json:"Start"`
	ResultCode        model.ErrorNumberType `json:"ResultCode"`
	ResultDescription model.DescriptionType `json:"ResultDescription"`
	LockActive        bool                  `json:"LockActive"`
	LockStart         time.Time             `json:"LockStart"`
}

type DiagnosisData struct {
	OperatingState model.DeviceDiagnosisOperatingStateType `json:"OperatingState"`
	LastErrorCode  model.LastErrorCodeType                 `json:"LastErrorCode"`
}

type EnergyGuardDataModel struct {
	IsConnected        bool                   `json:"IsConnected"`
	HasMeter           bool                   `json:"HasMeter"`
	HasOPEV            bool                   `json:"HasOPEV"`
	Vehicle            map[string]interface{} `json:"Vehicle"`
	Wallbox            map[string]interface{} `json:"Wallbox"`
	Meter              linkymeter.MeterData   `json:"Meter"`
	OverloadProtection OverloadProtectionData `json:"OverloadProtection"`
	Diagnosis          DiagnosisData          `json:"Diagnosis"`
}

type EnergyGuardData struct {
	model                EnergyGuardDataModel
	deviceDiagnosisState model.DeviceDiagnosisStateDataType
	access               sync.Mutex
	logger               *log.Logger
}

type CsvFormatter struct {
	TimestampFormat string
	ColumnSeparator string
}

func (f *CsvFormatter) Format(entry *log.Entry) ([]byte, error) {
	timestamp := entry.Time.Format(f.TimestampFormat)
	return []byte(fmt.Sprintf("%s%s%s\n", timestamp, f.ColumnSeparator, entry.Message)), nil
}

func NewEnergyGuardData(dataModelConfig config.DataModelConfig) *EnergyGuardData {
	data := &EnergyGuardData{}

	data.model.IsConnected = false
	data.model.HasMeter = false
	data.model.HasMeter = false
	data.model.Vehicle = make(map[string]interface{})
	data.model.Wallbox = make(map[string]interface{})
	data.model.Diagnosis.OperatingState = model.DeviceDiagnosisOperatingStateTypeNormalOperation
	data.model.Diagnosis.LastErrorCode = model.LastErrorCodeType(DIAGNOSIS_NO_ERROR)
	data.deviceDiagnosisState.OperatingState = &data.model.Diagnosis.OperatingState
	data.deviceDiagnosisState.LastErrorCode = &data.model.Diagnosis.LastErrorCode

	// Create a CSV logger
	data.logger = log.New()

	csvFileExtension := filepath.Ext(dataModelConfig.Csv.FilePath)
	csvFileWithoutExtension := strings.TrimSuffix(dataModelConfig.Csv.FilePath, csvFileExtension)
	csvFilePathWithPattern := csvFileWithoutExtension + dataModelConfig.Csv.Rotation.PeriodPattern + csvFileExtension
	csvRotationTime := time.Duration(float64(dataModelConfig.Csv.Rotation.PeriodInHours) * float64(time.Hour))
	csvMaxAge := time.Duration(float64(dataModelConfig.Csv.Rotation.PeriodCount) * float64(dataModelConfig.Csv.Rotation.PeriodInHours) * float64(time.Hour))
	// Configure csv with rotation matching with config
	writer, _ := rotatelogs.New(
		csvFilePathWithPattern,
		rotatelogs.WithRotationTime(csvRotationTime),
		rotatelogs.WithMaxAge(csvMaxAge),
		rotatelogs.WithHandler(rotatelogs.HandlerFunc(func(e rotatelogs.Event) {
			if e.Type() != rotatelogs.FileRotatedEventType {
				return
			}
			writeCSVTitle((e.(*rotatelogs.FileRotatedEvent).CurrentFile()))
		})),
	)
	data.logger.SetFormatter(&CsvFormatter{
		TimestampFormat: "2006-01-02 15:04:05.000",
		ColumnSeparator: string(CSV_COLUMN_SEPARATOR),
	})
	data.logger.SetOutput(writer)
	writer.Rotate()

	return data
}

func writeCSVTitle(fileName string) {
	file, err := os.Create(fileName)
	if err != nil {
		log.Fatal(err)
	}
	writer := csv.NewWriter(file)
	writer.Comma = CSV_COLUMN_SEPARATOR
	writer.Write(CSV_COLUMNS)
	writer.Flush()
	defer file.Close()
}

func IsOverloadProtectionEqual(overloadProtection OverloadProtectionData, other OverloadProtectionData) bool {
	if overloadProtection.Active != other.Active {
		return false
	}
	if overloadProtection.Value != other.Value {
		return false
	}
	if overloadProtection.Start != other.Start {
		return false
	}
	if overloadProtection.ResultCode != other.ResultCode {
		return false
	}
	if overloadProtection.ResultDescription != other.ResultDescription {
		return false
	}
	if overloadProtection.LockStart != other.LockStart {
		return false
	}
	if overloadProtection.LockActive != other.LockActive {
		return false
	}

	return true
}

func (e *EnergyGuardData) IsConnected() (isConnected bool) {
	e.access.Lock()
	isConnected = e.model.IsConnected
	e.access.Unlock()

	return isConnected
}

func (e *EnergyGuardData) SetIsConnected(isConnected bool) (hasChanged bool) {
	e.access.Lock()
	if isConnected != e.model.IsConnected {
		e.model.IsConnected = isConnected
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) HasMeter() (hasMeter bool) {
	e.access.Lock()
	hasMeter = e.model.HasMeter
	e.access.Unlock()

	return hasMeter
}

func (e *EnergyGuardData) SetHasMeter(hasMeter bool) (hasChanged bool) {
	e.access.Lock()
	if hasMeter != e.model.HasMeter {
		e.model.HasMeter = hasMeter
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) HasOPEV() (hasOPEV bool) {
	e.access.Lock()
	hasOPEV = e.model.HasOPEV
	e.access.Unlock()

	return hasOPEV
}

func (e *EnergyGuardData) SetHasOPEV(hasOPEV bool) (hasChanged bool) {
	e.access.Lock()
	if hasOPEV != e.model.HasOPEV {
		e.model.HasOPEV = hasOPEV
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GetVehicle() (vehicle map[string]interface{}) {
	vehicle = make(map[string]interface{})
	e.access.Lock()
	for key, value := range e.model.Vehicle {
		vehicle[key] = value
	}
	e.access.Unlock()

	return vehicle
}

func (e *EnergyGuardData) SetVehicle(vehicle map[string]interface{}) (hasChanged bool) {
	e.access.Lock()
	// compare and update vehicle data
	for k, v := range vehicle {
		oldValue, ok := e.model.Vehicle[k]
		if !ok {
			e.model.Vehicle[k] = v
			hasChanged = true
		} else {
			if !cmp.Equal(v, oldValue) {
				e.model.Vehicle[k] = v
				hasChanged = true
			}
		}
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GetWallbox() (wallbox map[string]interface{}) {
	wallbox = make(map[string]interface{})
	e.access.Lock()
	for key, value := range e.model.Wallbox {
		wallbox[key] = value
	}
	e.access.Unlock()

	return wallbox
}

func (e *EnergyGuardData) SetWallbox(wallbox map[string]interface{}) (hasChanged bool) {
	e.access.Lock()
	for k, v := range wallbox {
		oldValue, ok := e.model.Wallbox[k]
		if !ok {
			e.model.Wallbox[k] = v
			hasChanged = true
		} else {
			if !cmp.Equal(v, oldValue) {
				e.model.Wallbox[k] = v
				hasChanged = true
			}
		}
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GeMeter() (meter linkymeter.MeterData) {
	e.access.Lock()
	meter = linkymeter.MeterData(e.model.Meter)
	e.access.Unlock()

	return meter
}

func (e *EnergyGuardData) SetMeter(meter linkymeter.MeterData) (hasChanged bool) {
	e.access.Lock()
	if !linkymeter.IsEqual(e.model.Meter, meter) {
		e.model.Meter = linkymeter.MeterData(meter)
		e.model.HasMeter = true
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GeMeterMinAvailableCurrent() (minAvailableCurrent float64) {
	e.access.Lock()
	minAvailableCurrent = slices.Min(e.model.Meter.AvailableCurrentPerPhase)
	e.access.Unlock()

	return minAvailableCurrent
}

func (e *EnergyGuardData) GeOverloadProtection() (overloadProtection OverloadProtectionData) {
	e.access.Lock()
	overloadProtection = OverloadProtectionData(e.model.OverloadProtection)
	e.access.Unlock()

	return overloadProtection
}

func (e *EnergyGuardData) SetOverloadProtection(overloadprotection OverloadProtectionData) (hasChanged bool) {
	e.access.Lock()
	if !IsOverloadProtectionEqual(e.model.OverloadProtection, overloadprotection) {
		e.model.OverloadProtection = OverloadProtectionData(overloadprotection)
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}
func (e *EnergyGuardData) GetDiagnosisState() *model.DeviceDiagnosisStateDataType {
	return &e.deviceDiagnosisState
}

func (e *EnergyGuardData) GetDiagnosis() (diagnosis DiagnosisData) {
	e.access.Lock()
	diagnosis = e.model.Diagnosis
	e.access.Unlock()

	return diagnosis
}

func (e *EnergyGuardData) SetDiagnosis(operatingState model.DeviceDiagnosisOperatingStateType, lastErrorCode model.LastErrorCodeType) (hasChanged bool) {
	e.access.Lock()
	if lastErrorCode != e.model.Diagnosis.LastErrorCode {
		e.model.Diagnosis.OperatingState = operatingState
		e.model.Diagnosis.LastErrorCode = lastErrorCode
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) DisableOverloadProtectionActive() {
	e.access.Lock()
	if e.model.OverloadProtection.Active {
		e.model.OverloadProtection.Value = 0
		e.model.OverloadProtection.Active = false
		e.access.Unlock()
		e.save()
		return
	}
	e.access.Unlock()
}

func (e *EnergyGuardData) GetOverloadProtectionValue() (limitValue float64) {
	e.access.Lock()
	limitValue = e.model.OverloadProtection.Value
	e.access.Unlock()

	return limitValue
}

func (e *EnergyGuardData) SetOverloadProtectionValue(limitValue float64) {
	e.access.Lock()
	e.model.OverloadProtection.Value = limitValue
	e.model.OverloadProtection.Active = true
	e.model.OverloadProtection.Start = time.Now()
	e.access.Unlock()
	e.save()
}

func (e *EnergyGuardData) SetOverloadProtectionResult(result model.ResultDataType) (hasChanged bool) {
	e.access.Lock()
	if result.ErrorNumber != nil {
		if *result.ErrorNumber != e.model.OverloadProtection.ResultCode {
			e.model.OverloadProtection.ResultCode = *result.ErrorNumber
			hasChanged = true
		}
		if result.Description != nil {
			if *result.Description != e.model.OverloadProtection.ResultDescription {
				e.model.OverloadProtection.ResultDescription = *result.Description
				hasChanged = true
			}
		} else {
			if e.model.OverloadProtection.ResultDescription != "" {
				e.model.OverloadProtection.ResultDescription = ""
				hasChanged = true
			}
		}
	} else {
		if e.model.OverloadProtection.ResultCode != model.ErrorNumberTypeGeneralError {
			e.model.OverloadProtection.ResultCode = model.ErrorNumberTypeGeneralError
			hasChanged = true
		}
		if e.model.OverloadProtection.ResultDescription != "undefined error" {
			e.model.OverloadProtection.ResultDescription = "undefined error"
			hasChanged = true
		}
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GetOverloadProtectionLockStart() (lockStart time.Time) {
	e.access.Lock()
	lockStart = e.model.OverloadProtection.LockStart
	e.access.Unlock()

	return lockStart
}

func (e *EnergyGuardData) SetOverloadProtectionLockStart(lockStart time.Time) (hasChanged bool) {
	e.access.Lock()
	if lockStart != e.model.OverloadProtection.Start {
		e.model.OverloadProtection.LockStart = lockStart
		e.model.OverloadProtection.LockActive = true
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GetOverloadProtectionLockDuration() (lockDuration time.Duration) {
	now := time.Now()
	e.access.Lock()
	lockDuration = now.Sub(e.model.OverloadProtection.LockStart)
	e.access.Unlock()

	return lockDuration
}

func (e *EnergyGuardData) SetOverloadProtectionLockActive(lockActive bool) (hasChanged bool) {
	e.access.Lock()
	if lockActive != e.model.OverloadProtection.LockActive {
		e.model.OverloadProtection.LockActive = lockActive
		hasChanged = true
	}
	e.access.Unlock()
	if hasChanged {
		e.save()
	}

	return hasChanged
}

func (e *EnergyGuardData) GetModel() (model EnergyGuardDataModel) {
	model = EnergyGuardDataModel{
		IsConnected:        e.IsConnected(),
		HasMeter:           e.HasMeter(),
		HasOPEV:            e.HasOPEV(),
		Vehicle:            e.GetVehicle(),
		Wallbox:            e.GetWallbox(),
		Meter:              e.GeMeter(),
		OverloadProtection: e.GeOverloadProtection(),
		Diagnosis:          e.GetDiagnosis(),
	}

	return model
}

func (e *EnergyGuardData) Print() {
	model := e.GetModel()
	jsonBytes, error := json.MarshalIndent(model, "", "  ")
	if error == nil {
		log.Infof("EnergyGuardDataModel : \n%s\n", string(jsonBytes))
	}
}

func (e *EnergyGuardData) save() {
	model := e.GetModel()
	columns := extractColumns(model)
	message := strings.Join(columns, string(CSV_COLUMN_SEPARATOR))
	e.logger.Info(message)
}

func extractColumns(model EnergyGuardDataModel) []string {
	IsConnected := strconv.FormatBool(model.IsConnected)
	HasMeter := strconv.FormatBool(model.HasMeter)
	HasOPEV := strconv.FormatBool(model.HasOPEV)
	EV_IsConnected := extractColumn_EV_IsConnected(model.Vehicle)
	EV_AsymetricChargingSupport := extractColumn_EV_AsymetricChargingSupport(model.Vehicle)
	EV_ChargeState := extractColumn_EV_ChargeState(model.Vehicle)
	EV_ChargingPowerLimits_Min, EV_ChargingPowerLimits_Max, EV_ChargingPowerLimits_Standby := extractColumn_EV_ChargingPowerLimits(model.Vehicle)
	EV_CurrentLimits_Min1,
		EV_CurrentLimits_Min2,
		EV_CurrentLimits_Min3,
		EV_CurrentLimits_Max1,
		EV_CurrentLimits_Max2,
		EV_CurrentLimits_Max3,
		EV_CurrentLimits_Default1,
		EV_CurrentLimits_Default2,
		EV_CurrentLimits_Default3 := extractColumn_EV_CurrentLimits(model.Vehicle)
	EV_CurrentPerPhase1, EV_CurrentPerPhase2, EV_CurrentPerPhase3 := extractColumn_CurrentPerPhase(model.Vehicle)
	EV_EnergyCharged := extractColumn_EV_EnergyCharged(model.Vehicle)
	EV_Identifications := extractColumn_EV_Identifications(model.Vehicle)
	EV_IsInSleepMode := extractColumn_EV_IsInSleepMode(model.Vehicle)
	EV_LoadControlLimits1_IsChangeable,
		EV_LoadControlLimits1_IsActive,
		EV_LoadControlLimits1_Value,
		EV_LoadControlLimits2_IsChangeable,
		EV_LoadControlLimits2_IsActive,
		EV_LoadControlLimits2_Value,
		EV_LoadControlLimits3_IsChangeable,
		EV_LoadControlLimits3_IsActive,
		EV_LoadControlLimits3_Value := extractColumn_EV_LoadControlLimits(model.Vehicle)
	EV_ManufacturerData_DeviceName,
		EV_ManufacturerData_DeviceCode,
		EV_ManufacturerData_SerialNumber,
		EV_ManufacturerData_SoftwareRevision,
		EV_ManufacturerData_HardwareRevision,
		EV_ManufacturerData_VendorName,
		EV_ManufacturerData_VendorCode,
		EV_ManufacturerData_BrandName,
		EV_ManufacturerData_PowerSource,
		EV_ManufacturerData_ManufacturerNodeIdentification,
		EV_ManufacturerData_ManufacturerLabel,
		EV_ManufacturerData_ManufacturerDescription := extractColumn_EV_ManufacturerData(model.Vehicle)
	EV_PhasesConnected := extractColumn_EV_PhasesConnected(model.Vehicle)
	EV_PowerPerPhase1, EV_PowerPerPhase2, EV_PowerPerPhase3 := extractColumn_EV_PowerPerPhase(model.Vehicle)
	EVSE_IsConnected := extractColumn_EVSE_IsConnected(model.Wallbox)
	EVSE_ManufacturerData_DeviceName,
		EVSE_ManufacturerData_DeviceCode,
		EVSE_ManufacturerData_SerialNumber,
		EVSE_ManufacturerData_SoftwareRevision,
		EVSE_ManufacturerData_HardwareRevision,
		EVSE_ManufacturerData_VendorName,
		EVSE_ManufacturerData_VendorCode,
		EVSE_ManufacturerData_BrandName,
		EVSE_ManufacturerData_PowerSource,
		EVSE_ManufacturerData_ManufacturerNodeIdentification,
		EVSE_ManufacturerData_ManufacturerLabel,
		EVSE_ManufacturerData_ManufacturerDescription := extractColumn_EVSE_ManufacturerData(model.Wallbox)
	EVSE_OperatingState := extractColumn_EVSE_OperatingState(model.Wallbox)
	Meter_SerialNumber := model.Meter.SerialNumber
	Meter_DateTime := model.Meter.DateTime
	Meter_BreakerOpened := strconv.FormatBool(model.Meter.BreakerOpened)
	Meter_PhaseCount := strconv.FormatInt(int64(model.Meter.PhaseCount), 10)
	Meter_OverLoadPowerLimit := strconv.FormatInt(int64(model.Meter.OverLoadPowerLimit), 10)
	Meter_OverLoadCurrentLimit1, Meter_OverLoadCurrentLimit2, Meter_OverLoadCurrentLimit3 := extractColumn_Meter_OverLoadCurrentLimitPerPhase(model.Meter)
	Meter_RmsVoltage1, Meter_RmsVoltage2, Meter_RmsVoltage3 := extractColumn_Meter_RmsVoltagePerPhase(model.Meter)
	Meter_RmsCurrent1, Meter_RmsCurrent2, Meter_RmsCurrent3 := extractColumn_Meter_RmsCurrentPerPhase(model.Meter)
	Meter_ApparentImportPower := strconv.FormatInt(int64(model.Meter.ApparentImportPower), 10)
	Meter_ApparentImportPower1, Meter_ApparentImportPower2, Meter_ApparentImportPower3 := extractColumn_Meter_ApparentImportPowerPerPhase(model.Meter)
	Meter_AvailableCurrent1, Meter_AvailableCurrent2, Meter_AvailableCurrent3 := extractColumn_Meter_AvailableCurrentPerPhase(model.Meter)
	OverloadProtection_Active := strconv.FormatBool(model.OverloadProtection.Active)
	OverloadProtection_Value := strconv.FormatFloat(model.OverloadProtection.Value, 'f', CSV_FLOATING_PRECISION, 64)
	OverloadProtection_Start := model.OverloadProtection.Start.String()
	OverloadProtection_ResultCode := strconv.FormatInt(int64(model.OverloadProtection.ResultCode), 10)
	OverloadProtection_ResultDescription := string(model.OverloadProtection.ResultDescription)
	OverloadProtection_LockActive := strconv.FormatBool(model.OverloadProtection.LockActive)
	OverloadProtection_LockStart := model.OverloadProtection.LockStart.String()
	Diagnosis_OperatingState := string(model.Diagnosis.OperatingState)
	Diagnosis_LastErrorCode := string(model.Diagnosis.LastErrorCode)

	colums := []string{
		IsConnected,
		HasMeter,
		HasOPEV,
		EV_IsConnected,
		EV_AsymetricChargingSupport,
		EV_ChargeState,
		EV_ChargingPowerLimits_Min,
		EV_ChargingPowerLimits_Max,
		EV_ChargingPowerLimits_Standby,
		EV_CurrentLimits_Min1,
		EV_CurrentLimits_Min2,
		EV_CurrentLimits_Min3,
		EV_CurrentLimits_Max1,
		EV_CurrentLimits_Max2,
		EV_CurrentLimits_Max3,
		EV_CurrentLimits_Default1,
		EV_CurrentLimits_Default2,
		EV_CurrentLimits_Default3,
		EV_CurrentPerPhase1,
		EV_CurrentPerPhase2,
		EV_CurrentPerPhase3,
		EV_EnergyCharged,
		EV_Identifications,
		EV_IsInSleepMode,
		EV_LoadControlLimits1_IsChangeable,
		EV_LoadControlLimits1_IsActive,
		EV_LoadControlLimits1_Value,
		EV_LoadControlLimits2_IsChangeable,
		EV_LoadControlLimits2_IsActive,
		EV_LoadControlLimits2_Value,
		EV_LoadControlLimits3_IsChangeable,
		EV_LoadControlLimits3_IsActive,
		EV_LoadControlLimits3_Value,
		EV_ManufacturerData_DeviceName,
		EV_ManufacturerData_DeviceCode,
		EV_ManufacturerData_SerialNumber,
		EV_ManufacturerData_SoftwareRevision,
		EV_ManufacturerData_HardwareRevision,
		EV_ManufacturerData_VendorName,
		EV_ManufacturerData_VendorCode,
		EV_ManufacturerData_BrandName,
		EV_ManufacturerData_PowerSource,
		EV_ManufacturerData_ManufacturerNodeIdentification,
		EV_ManufacturerData_ManufacturerLabel,
		EV_ManufacturerData_ManufacturerDescription,
		EV_PhasesConnected,
		EV_PowerPerPhase1,
		EV_PowerPerPhase2,
		EV_PowerPerPhase3,
		EVSE_IsConnected,
		EVSE_ManufacturerData_DeviceName,
		EVSE_ManufacturerData_DeviceCode,
		EVSE_ManufacturerData_SerialNumber,
		EVSE_ManufacturerData_SoftwareRevision,
		EVSE_ManufacturerData_HardwareRevision,
		EVSE_ManufacturerData_VendorName,
		EVSE_ManufacturerData_VendorCode,
		EVSE_ManufacturerData_BrandName,
		EVSE_ManufacturerData_PowerSource,
		EVSE_ManufacturerData_ManufacturerNodeIdentification,
		EVSE_ManufacturerData_ManufacturerLabel,
		EVSE_ManufacturerData_ManufacturerDescription,
		EVSE_OperatingState,
		Meter_SerialNumber,
		Meter_DateTime,
		Meter_BreakerOpened,
		Meter_PhaseCount,
		Meter_OverLoadPowerLimit,
		Meter_OverLoadCurrentLimit1,
		Meter_OverLoadCurrentLimit2,
		Meter_OverLoadCurrentLimit3,
		Meter_RmsVoltage1,
		Meter_RmsVoltage2,
		Meter_RmsVoltage3,
		Meter_RmsCurrent1,
		Meter_RmsCurrent2,
		Meter_RmsCurrent3,
		Meter_ApparentImportPower,
		Meter_ApparentImportPower1,
		Meter_ApparentImportPower2,
		Meter_ApparentImportPower3,
		Meter_AvailableCurrent1,
		Meter_AvailableCurrent2,
		Meter_AvailableCurrent3,
		OverloadProtection_Active,
		OverloadProtection_Value,
		OverloadProtection_Start,
		OverloadProtection_ResultCode,
		OverloadProtection_ResultDescription,
		OverloadProtection_LockActive,
		OverloadProtection_LockStart,
		Diagnosis_OperatingState,
		Diagnosis_LastErrorCode,
	}

	return colums
}

func extractColumn_EV_IsConnected(vehicle map[string]interface{}) (isConnected string) {
	value, ok := vehicle[evse.VEHICLE_IS_CONNECTED]

	if ok {
		boolValue := value.(bool)
		isConnected = strconv.FormatBool(boolValue)
	}

	return isConnected
}

func extractColumn_EV_AsymetricChargingSupport(vehicle map[string]interface{}) (asymetricChargingSupport string) {
	value, ok := vehicle[evse.VEHICLE_ASYMETRIC_CHARGING_SUPPORT]

	if ok {
		boolValue := value.(bool)
		asymetricChargingSupport = strconv.FormatBool(boolValue)
	}

	return asymetricChargingSupport
}

func extractColumn_EV_ChargeState(vehicle map[string]interface{}) (chargeState string) {
	value, ok := vehicle[evse.VEHICLE_CHARGE_STATE]

	if ok {
		chargeStateValue := value.(ucapi.EVChargeStateType)
		chargeState = string(chargeStateValue)
	}

	return chargeState
}

func extractColumn_EV_ChargingPowerLimits(vehicle map[string]interface{}) (min string, max string, standby string) {
	value, ok := vehicle[evse.VEHICLE_CHARGING_POWER_LIMITS]

	if ok {
		chargingPowerLimits := value.(evse.ChargingPowerLimits)
		min = strconv.FormatFloat(chargingPowerLimits.Min, 'f', CSV_FLOATING_PRECISION, 64)
		max = strconv.FormatFloat(chargingPowerLimits.Max, 'f', CSV_FLOATING_PRECISION, 64)
		standby = strconv.FormatFloat(chargingPowerLimits.Standby, 'f', CSV_FLOATING_PRECISION, 64)
	}

	return min, max, standby
}

func extractColumn_EV_CurrentLimits(vehicle map[string]interface{}) (min1 string,
	min2 string,
	min3 string,
	max1 string,
	max2 string,
	max3 string,
	default1 string,
	default2 string,
	default3 string) {
	value, ok := vehicle[evse.VEHICLE_CURRENT_LIMITS]

	if ok {
		currentLimits := value.(evse.CurrentLimits)
		if len(currentLimits.Min) > 0 {
			min1 = strconv.FormatFloat(currentLimits.Min[0], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Min) > 1 {
			min2 = strconv.FormatFloat(currentLimits.Min[1], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Min) > 2 {
			min3 = strconv.FormatFloat(currentLimits.Min[2], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Max) > 0 {
			max1 = strconv.FormatFloat(currentLimits.Max[0], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Max) > 1 {
			max2 = strconv.FormatFloat(currentLimits.Max[1], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Max) > 2 {
			max3 = strconv.FormatFloat(currentLimits.Max[2], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Default) > 0 {
			max1 = strconv.FormatFloat(currentLimits.Default[0], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Default) > 1 {
			max2 = strconv.FormatFloat(currentLimits.Default[1], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentLimits.Default) > 2 {
			max3 = strconv.FormatFloat(currentLimits.Default[2], 'f', CSV_FLOATING_PRECISION, 64)
		}
	}
	return min1, min2, min3, max1, max2, max3, default1, default2, default3
}

func extractColumn_CurrentPerPhase(vehicle map[string]interface{}) (currentPhase1 string, currentPhase2 string, currentPhase3 string) {
	value, ok := vehicle[evse.VEHICLE_CURRENT_PER_PHASE]

	if ok {
		currentPerPhase := value.([]float64)
		if len(currentPerPhase) > 0 {
			currentPhase1 = strconv.FormatFloat(currentPerPhase[0], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentPerPhase) > 1 {
			currentPhase2 = strconv.FormatFloat(currentPerPhase[1], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(currentPerPhase) > 2 {
			currentPhase3 = strconv.FormatFloat(currentPerPhase[2], 'f', CSV_FLOATING_PRECISION, 64)
		}
	}

	return currentPhase1, currentPhase2, currentPhase3
}

func extractColumn_EV_EnergyCharged(vehicle map[string]interface{}) (energyCharged string) {
	value, ok := vehicle[evse.VEHICLE_ENERGY_CHARGED]

	if ok {
		energyChargedValue := value.(float64)
		energyCharged = strconv.FormatFloat(energyChargedValue, 'f', CSV_FLOATING_PRECISION, 64)
	}

	return energyCharged
}

func extractColumn_EV_Identifications(vehicle map[string]interface{}) (identifications string) {
	value, ok := vehicle[evse.VEHICLE_IDENTIFICATIONS]

	if ok {
		identificationsValue := value.([]ucapi.IdentificationItem)
		if len(identificationsValue) > 0 {
			identifications = identificationsValue[0].Value
		}
	}

	return identifications
}

func extractColumn_EV_IsInSleepMode(vehicle map[string]interface{}) (sleepMode string) {
	value, ok := vehicle[evse.VEHICLE_IS_IN_SLEEP_MODE]

	if ok {
		boolValue := value.(bool)
		sleepMode = strconv.FormatBool(boolValue)
	}

	return sleepMode
}

func extractColumn_EV_LoadControlLimits(vehicle map[string]interface{}) (isChangeable1 string,
	isActive1 string,
	limit1 string,
	isChangeable2 string,
	isActive2 string,
	limit2 string,
	isChangeable3 string,
	isActive3 string,
	limit3 string) {
	value, ok := vehicle[evse.VEHICLE_LOAD_CONTROL_LIMITS]

	if ok {
		limits := value.([]ucapi.LoadLimitsPhase)
		for i := 0; i < len(limits); i++ {
			switch limits[i].Phase {
			case model.ElectricalConnectionPhaseNameTypeA:
				isChangeable1 = strconv.FormatBool(limits[i].IsChangeable)
				isActive1 = strconv.FormatBool(limits[i].IsActive)
				limit1 = strconv.FormatFloat(limits[i].Value, 'f', CSV_FLOATING_PRECISION, 64)
			case model.ElectricalConnectionPhaseNameTypeB:
				isChangeable2 = strconv.FormatBool(limits[i].IsChangeable)
				isActive2 = strconv.FormatBool(limits[i].IsActive)
				limit2 = strconv.FormatFloat(limits[i].Value, 'f', CSV_FLOATING_PRECISION, 64)
			case model.ElectricalConnectionPhaseNameTypeC:
				isChangeable3 = strconv.FormatBool(limits[i].IsChangeable)
				isActive3 = strconv.FormatBool(limits[i].IsActive)
				limit3 = strconv.FormatFloat(limits[i].Value, 'f', CSV_FLOATING_PRECISION, 64)
			}
		}
	}

	return isChangeable1, isActive1, limit1, isChangeable2, isActive2, limit2, isChangeable3, isActive3, limit3
}

func extractColumn_EV_ManufacturerData(vehicle map[string]interface{}) (deviceName string,
	deviceCode string,
	serialNumber string,
	softwareRevision string,
	hardwareRevision string,
	vendorName string,
	vendorCode string,
	brandName string,
	powerSource string,
	manufacturerNodeIdentification string,
	manufacturerLabel string,
	manufacturerDescription string) {
	value, ok := vehicle[evse.VEHICLE_MANUFACTURER_DATA]

	if ok {
		manufacturerData := value.(ucapi.ManufacturerData)
		deviceName = manufacturerData.DeviceName
		deviceCode = manufacturerData.DeviceCode
		serialNumber = manufacturerData.SerialNumber
		softwareRevision = manufacturerData.SoftwareRevision
		hardwareRevision = manufacturerData.HardwareRevision
		powerSource = manufacturerData.PowerSource
		manufacturerNodeIdentification = manufacturerData.ManufacturerNodeIdentification
		vendorName = manufacturerData.VendorName
		vendorCode = manufacturerData.VendorCode
		brandName = manufacturerData.BrandName
		manufacturerLabel = manufacturerData.ManufacturerLabel
		manufacturerDescription = manufacturerData.ManufacturerDescription
	}

	return deviceName,
		deviceCode,
		serialNumber,
		softwareRevision,
		hardwareRevision,
		vendorName,
		vendorCode,
		brandName,
		powerSource,
		manufacturerNodeIdentification,
		manufacturerLabel,
		manufacturerDescription
}

func extractColumn_EV_PhasesConnected(vehicle map[string]interface{}) (phasesConnected string) {
	value, ok := vehicle[evse.VEHICLE_PHASES_CONNECTED]

	if ok {
		uintValue := value.(uint)
		phasesConnected = strconv.FormatUint(uint64(uintValue), 10)
	}

	return phasesConnected
}

func extractColumn_EV_PowerPerPhase(vehicle map[string]interface{}) (powerPhase1 string, powerPhase2 string, powerPhase3 string) {
	value, ok := vehicle[evse.VEHICLE_POWER_PER_PHASE]

	if ok {
		powerPerPhase := value.([]float64)
		if len(powerPerPhase) > 0 {
			powerPhase1 = strconv.FormatFloat(powerPerPhase[0], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(powerPerPhase) > 1 {
			powerPhase2 = strconv.FormatFloat(powerPerPhase[1], 'f', CSV_FLOATING_PRECISION, 64)
		}
		if len(powerPerPhase) > 2 {
			powerPhase3 = strconv.FormatFloat(powerPerPhase[2], 'f', CSV_FLOATING_PRECISION, 64)
		}
	}

	return powerPhase1, powerPhase2, powerPhase3
}

func extractColumn_EVSE_IsConnected(wallbox map[string]interface{}) (isConnected string) {
	value, ok := wallbox[evse.WALLBOX_IS_CONNECTED]

	if ok {
		boolValue := value.(bool)
		isConnected = strconv.FormatBool(boolValue)
	}

	return isConnected
}

func extractColumn_EVSE_ManufacturerData(wallbox map[string]interface{}) (deviceName string,
	deviceCode string,
	serialNumber string,
	softwareRevision string,
	hardwareRevision string,
	vendorName string,
	vendorCode string,
	brandName string,
	powerSource string,
	manufacturerNodeIdentification string,
	manufacturerLabel string,
	manufacturerDescription string) {
	value, ok := wallbox[evse.WALLBOX_MANUFACTURER_DATA]

	if ok {
		manufacturerData := value.(ucapi.ManufacturerData)
		deviceName = manufacturerData.DeviceName
		deviceCode = manufacturerData.DeviceCode
		serialNumber = manufacturerData.SerialNumber
		softwareRevision = manufacturerData.SoftwareRevision
		hardwareRevision = manufacturerData.HardwareRevision
		powerSource = manufacturerData.PowerSource
		manufacturerNodeIdentification = manufacturerData.ManufacturerNodeIdentification
		vendorName = manufacturerData.VendorName
		vendorCode = manufacturerData.VendorCode
		brandName = manufacturerData.BrandName
		manufacturerLabel = manufacturerData.ManufacturerLabel
		manufacturerDescription = manufacturerData.ManufacturerDescription
	}

	return deviceName,
		deviceCode,
		serialNumber,
		softwareRevision,
		hardwareRevision,
		vendorName,
		vendorCode,
		brandName,
		powerSource,
		manufacturerNodeIdentification,
		manufacturerLabel,
		manufacturerDescription
}

func extractColumn_EVSE_OperatingState(wallbox map[string]interface{}) (operatingState string) {
	value, ok := wallbox[evse.WALLBOX_OPERATING_STATE]

	if ok {
		operatingStateValue := value.(model.DeviceDiagnosisOperatingStateType)
		operatingState = string(operatingStateValue)
	}

	return operatingState
}

func extractColumn_Meter_OverLoadCurrentLimitPerPhase(meter linkymeter.MeterData) (limitPhase1 string, limitPhase2 string, limitPhase3 string) {

	if len(meter.OverLoadCurrentLimitPerPhase) > 0 {
		limitPhase1 = strconv.FormatFloat(meter.OverLoadCurrentLimitPerPhase[0], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.OverLoadCurrentLimitPerPhase) > 1 {
		limitPhase2 = strconv.FormatFloat(meter.OverLoadCurrentLimitPerPhase[1], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.OverLoadCurrentLimitPerPhase) > 2 {
		limitPhase3 = strconv.FormatFloat(meter.OverLoadCurrentLimitPerPhase[2], 'f', CSV_FLOATING_PRECISION, 64)
	}

	return limitPhase1, limitPhase2, limitPhase3
}

func extractColumn_Meter_RmsVoltagePerPhase(meter linkymeter.MeterData) (voltagePhase1 string, voltagePhase2 string, voltagePhase3 string) {

	if len(meter.RmsVoltagePerPhase) > 0 {
		voltagePhase1 = strconv.FormatInt(int64(meter.RmsVoltagePerPhase[0]), 10)
	}
	if len(meter.RmsVoltagePerPhase) > 1 {
		voltagePhase2 = strconv.FormatInt(int64(meter.RmsVoltagePerPhase[1]), 10)
	}
	if len(meter.RmsVoltagePerPhase) > 2 {
		voltagePhase3 = strconv.FormatInt(int64(meter.RmsVoltagePerPhase[2]), 10)
	}

	return voltagePhase1, voltagePhase2, voltagePhase3
}

func extractColumn_Meter_RmsCurrentPerPhase(meter linkymeter.MeterData) (currentPhase1 string, currentPhase2 string, currentPhase3 string) {

	if len(meter.RmsCurrentPerPhase) > 0 {
		currentPhase1 = strconv.FormatFloat(meter.RmsCurrentPerPhase[0], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.RmsCurrentPerPhase) > 1 {
		currentPhase2 = strconv.FormatFloat(meter.RmsCurrentPerPhase[1], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.RmsCurrentPerPhase) > 2 {
		currentPhase3 = strconv.FormatFloat(meter.RmsCurrentPerPhase[2], 'f', CSV_FLOATING_PRECISION, 64)
	}

	return currentPhase1, currentPhase2, currentPhase3
}

func extractColumn_Meter_ApparentImportPowerPerPhase(meter linkymeter.MeterData) (apparentImportPower1 string, apparentImportPower2 string, apparentImportPower3 string) {

	if len(meter.ApparentImportPowerPerPhase) > 0 {
		apparentImportPower1 = strconv.FormatInt(int64(meter.ApparentImportPowerPerPhase[0]), 10)
	}
	if len(meter.ApparentImportPowerPerPhase) > 1 {
		apparentImportPower2 = strconv.FormatInt(int64(meter.ApparentImportPowerPerPhase[1]), 10)
	}
	if len(meter.ApparentImportPowerPerPhase) > 2 {
		apparentImportPower3 = strconv.FormatInt(int64(meter.ApparentImportPowerPerPhase[2]), 10)
	}

	return apparentImportPower1, apparentImportPower2, apparentImportPower3
}

func extractColumn_Meter_AvailableCurrentPerPhase(meter linkymeter.MeterData) (availableCurrent1 string, availableCurrent2 string, availableCurrent3 string) {

	if len(meter.AvailableCurrentPerPhase) > 0 {
		availableCurrent1 = strconv.FormatFloat(meter.AvailableCurrentPerPhase[0], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.AvailableCurrentPerPhase) > 1 {
		availableCurrent2 = strconv.FormatFloat(meter.AvailableCurrentPerPhase[1], 'f', CSV_FLOATING_PRECISION, 64)
	}
	if len(meter.AvailableCurrentPerPhase) > 2 {
		availableCurrent3 = strconv.FormatFloat(meter.AvailableCurrentPerPhase[2], 'f', CSV_FLOATING_PRECISION, 64)
	}

	return availableCurrent1, availableCurrent2, availableCurrent3
}
