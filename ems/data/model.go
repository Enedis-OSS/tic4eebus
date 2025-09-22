// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"time"

	"github.com/Enedis-OSS/tic4eebus/linkymeter"
	"github.com/enbility/spine-go/model"
)

const (
	DIAGNOSIS_NO_ERROR = "No error"
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

type DataModel struct {
	IsConnected        bool                   `json:"IsConnected"`
	HasMeter           bool                   `json:"HasMeter"`
	HasOPEV            bool                   `json:"HasOPEV"`
	Vehicle            map[string]interface{} `json:"Vehicle"`
	Wallbox            map[string]interface{} `json:"Wallbox"`
	Meter              linkymeter.MeterData   `json:"Meter"`
	OverloadProtection OverloadProtectionData `json:"OverloadProtection"`
	Diagnosis          DiagnosisData          `json:"Diagnosis"`
}
