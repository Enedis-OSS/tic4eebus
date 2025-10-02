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
	// Diagnosis last error code when there is no error
	DIAGNOSIS_NO_ERROR = "No error"
)

// OverloadProtectionData represents the overload protection data
type OverloadProtectionData struct {
	Active            bool                  `json:"Active"`            // Whether overload protection algortihm is active
	Value             float64               `json:"Value"`             // The electrical vehicle charge limitation value in Amps
	Start             time.Time             `json:"Start"`             // The last time when the overload protection was activated
	ResultCode        model.ErrorNumberType `json:"ResultCode"`        // The last electrical vehicle charge limitation update result code
	ResultDescription model.DescriptionType `json:"ResultDescription"` // The last electrical vehicle charge limitation update result description
	LockActive        bool                  `json:"LockActive"`        // Whether the overload protection is currently locking the electrical vehicle charge limitation
	LockStart         time.Time             `json:"LockStart"`         // The last time when the overload protection lock started
}

// DiagnosisData represents the diagnosis data send to the wallbox
type DiagnosisData struct {
	OperatingState model.DeviceDiagnosisOperatingStateType `json:"OperatingState"` // The current operating state of the energy management system
	LastErrorCode  model.LastErrorCodeType                 `json:"LastErrorCode"`  // The last error code of the energy management system
}

type DataModel struct {
	IsConnected        bool                   `json:"IsConnected"`        // Whether the energy management system is connected (EEBUS) to the wallbox
	HasMeterData       bool                   `json:"HasMeterData"`       // Whether the energy management system is receiving linky meter data
	IsOpevSupported    bool                   `json:"IsOpevSupported"`    // Whether the wallbox supports the EEBUS OPEV use case
	Vehicle            map[string]interface{} `json:"Vehicle"`            // The vehicle data (if available, otherwise empty)
	Wallbox            map[string]interface{} `json:"Wallbox"`            // The wallbox data (if available, otherwise empty)
	Meter              linkymeter.MeterData   `json:"Meter"`              // The linky meter data
	OverloadProtection OverloadProtectionData `json:"OverloadProtection"` // The overload protection data
	Diagnosis          DiagnosisData          `json:"Diagnosis"`          // The diagnosis data
}
