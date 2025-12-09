// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package linkymeter

import (
	"fmt"
	"strconv"
)

// MeterData represents the Linky meter data
type MeterData struct {
	// The Linky meter serial number
	SerialNumber string `json:"SerialNumber"`
	// The Linky meter date time in format "yy/MM/dd HH:mm:ss"
	DateTime string `json:"DateTime,omitempty"`
	// Whether the Linky meter breaker is opened (true) or closed (false)
	BreakerOpened bool `json:"BreakerOpened"`
	// The Linky meter phase count (1 or 3)
	PhaseCount int `json:"PhaseCount"`
	// The Linky meter overload power limit in VA
	OverLoadPowerLimit int `json:"OverLoadPowerLimit"`
	// The Linky meter overload current limit per phase in A
	OverLoadCurrentLimitPerPhase []float64 `json:"OverLoadCurrentLimitPerPhase"`
	// The Linky meter RMS voltage per phase in V
	RmsVoltagePerPhase []int `json:"RmsVoltagePerPhase"`
	// The Linky meter RMS current per phase in A
	RmsCurrentPerPhase []float64 `json:"RmsCurrentPerPhase"`
	// The Linky meter total apparent import power in VA
	ApparentImportPower int `json:"ApparentImportPower"`
	// The Linky meter apparent import power per phase in VA
	ApparentImportPowerPerPhase []int `json:"ApparentImportPowerPerPhase,omitempty"`
	// The Linky meter available current per phase in A
	AvailableCurrentPerPhase []float64 `json:"AvailableCurrentPerPhase"`
}

// IsEqual checks if two MeterData instances are equal
func IsEqual(meter MeterData, other MeterData) bool {
	if meter.SerialNumber != other.SerialNumber {
		return false
	}
	if meter.DateTime != other.DateTime {
		return false
	}
	if meter.BreakerOpened != other.BreakerOpened {
		return false
	}
	if meter.PhaseCount != other.PhaseCount {
		return false
	}
	if meter.OverLoadPowerLimit != other.OverLoadPowerLimit {
		return false
	}
	if len(meter.OverLoadCurrentLimitPerPhase) != len(other.OverLoadCurrentLimitPerPhase) {
		return false
	}
	for i := 0; i < len(meter.OverLoadCurrentLimitPerPhase); i++ {
		if meter.OverLoadCurrentLimitPerPhase[i] != other.OverLoadCurrentLimitPerPhase[i] {
			return false
		}
	}
	if len(meter.RmsVoltagePerPhase) != len(other.RmsVoltagePerPhase) {
		return false
	}
	for i := 0; i < len(meter.RmsVoltagePerPhase); i++ {
		if meter.RmsVoltagePerPhase[i] != other.RmsVoltagePerPhase[i] {
			return false
		}
	}
	if len(meter.RmsCurrentPerPhase) != len(other.RmsCurrentPerPhase) {
		return false
	}
	for i := 0; i < len(meter.RmsCurrentPerPhase); i++ {
		if meter.RmsCurrentPerPhase[i] != other.RmsCurrentPerPhase[i] {
			return false
		}
	}
	if meter.ApparentImportPower != other.ApparentImportPower {
		return false
	}
	if len(meter.ApparentImportPowerPerPhase) != len(other.ApparentImportPowerPerPhase) {
		return false
	}
	for i := 0; i < len(meter.ApparentImportPowerPerPhase); i++ {
		if meter.ApparentImportPowerPerPhase[i] != other.ApparentImportPowerPerPhase[i] {
			return false
		}
	}
	if len(meter.AvailableCurrentPerPhase) != len(other.AvailableCurrentPerPhase) {
		return false
	}
	for i := 0; i < len(meter.AvailableCurrentPerPhase); i++ {
		if meter.AvailableCurrentPerPhase[i] != other.AvailableCurrentPerPhase[i] {
			return false
		}
	}

	return true
}

// ComputeMeterData computes the Linky meter data from the TIC content
//
// It works for STANDARD and HISTORIC TIC formats (see https://www.enedis.fr/media/2027/download).
// It returns an empty MeterData if the TIC content is nil
func ComputeMeterData(ticContent map[string]string) MeterData {
	var meterData MeterData
	// Check tic content
	if ticContent == nil {
		return MeterData{}
	}
	meterData.SerialNumber = getMeterSerialNumber(ticContent)
	meterData.DateTime = getMeterDateTime(ticContent)
	meterData.BreakerOpened = getMeterBreakerOpened(ticContent)
	meterData.PhaseCount = getMeterPhaseCount(ticContent)
	meterData.OverLoadPowerLimit, meterData.OverLoadCurrentLimitPerPhase = getMeterOverLoadLimit(ticContent)
	meterData.RmsVoltagePerPhase, meterData.RmsCurrentPerPhase = getMeterRmsVoltageAndCurrentPerPhase(ticContent)
	meterData.ApparentImportPower = getMeterApparentImportPower(ticContent)
	meterData.ApparentImportPowerPerPhase = getMeterApparentImportPowerPerPhase(ticContent)
	meterData.AvailableCurrentPerPhase = computeAvailableCurrentPerPhase(meterData)

	return meterData
}

func computeAvailableCurrentPerPhase(meterData MeterData) (availableCurrentPerPhase []float64) {
	availableCurrentPerPhase = make([]float64, meterData.PhaseCount)
	// Compute and update available current for each phase
	for i := 0; i < meterData.PhaseCount; i++ {
		availableCurrentPerPhase[i] = meterData.OverLoadCurrentLimitPerPhase[i] - meterData.RmsCurrentPerPhase[i]
	}

	return availableCurrentPerPhase
}

func getMeterSerialNumber(ticContent map[string]string) (serialNumber string) {
	// Extract ADSC tag (TIC standard)
	adsc, ok := ticContent["ADSC"]
	// ADSC found ?
	if ok {
		// Update meter serial number from ADSC value
		serialNumber = adsc
	} else {
		// Extract ADCO tag (TIC historic)
		adco, ok := ticContent["ADCO"]
		// ADCO found ?
		if ok {
			// Update meter serial number from ADCO value
			serialNumber = adco
		}
	}

	return serialNumber
}

func getMeterDateTime(ticContent map[string]string) (dateTime string) {
	// Extract DATE tag (TIC standard)
	date, ok := ticContent["DATE"]
	// DATE found ?
	if ok {
		// DATE is at least 13 characters ?
		if len(date) >= 13 {
			// Extract date fields and format date time
			year := date[1:3]
			month := date[3:5]
			day := date[5:7]
			hour := date[7:9]
			minute := date[9:11]
			second := date[11:13]
			dateTime = fmt.Sprintf("%s/%s/%s %s:%s:%s", year, month, day, hour, minute, second)
		}
	}

	return dateTime
}

func getMeterBreakerOpened(ticContent map[string]string) (breakerOpened bool) {
	var statusRegister string
	var ok bool
	// Extract STGE tag (TIC standard)
	statusRegister, ok = ticContent["STGE"]
	// STGE found ?
	if !ok {
		// Extract MOTDETAT tag (TIC historic)
		statusRegister, ok = ticContent["MOTDETAT"]
	}
	// STGE or MOTDETAT found ?
	if ok {
		// Convert status register to unsigned integer
		statusRegisterValue, err := strconv.ParseUint(statusRegister, 16, 32)
		// Status register conversion succeed ?
		if err == nil {
			// Get breaker state
			breakerState := (statusRegisterValue & 0x0000000E) >> 1
			// Update breaker opened state
			switch breakerState {
			// Breaker closed
			case 0:
				breakerOpened = false
			// Breaker opened due to overload
			case 1:
				breakerOpened = true
			// Breaker opened due to overvoltage
			case 2:
				breakerOpened = true
			// Breaker opened due to power cut
			case 3:
				breakerOpened = true
			// Breaker opened due to remote order
			case 4:
				breakerOpened = true
			// Breaker opened due to overheat over max current
			case 5:
				breakerOpened = true
			// Breaker opened due to overheat below max current
			case 6:
				breakerOpened = true
			}
		}
	}

	return breakerOpened
}

func getMeterPhaseCount(ticContent map[string]string) (phaseCount int) {
	// Extract URMS1, URMS2 && URMS3 tags (TIC standard)
	_, urms1_Ok := ticContent["URMS1"]
	_, urms2_Ok := ticContent["URMS2"]
	_, urms3_Ok := ticContent["URMS3"]
	// Extract IINST, IINST1, IINST2 && IINST3 tags (TIC historic)
	_, iinst_Ok := ticContent["IINST"]
	_, iinst1_Ok := ticContent["IINST1"]
	_, iinst2_Ok := ticContent["IINST2"]
	_, iinst3_Ok := ticContent["IINST3"]

	if urms1_Ok {
		if urms2_Ok && urms3_Ok {
			phaseCount = 3
		} else {
			phaseCount = 1
		}
	} else {
		if iinst_Ok {
			phaseCount = 1
		} else if iinst1_Ok && iinst2_Ok && iinst3_Ok {
			phaseCount = 3
		}
	}

	return phaseCount
}

func getMeterOverLoadLimit(ticContent map[string]string) (overloadPowerLimit int, overloadCurrentLimitPerPhase []float64) {
	var limit int
	var err error
	// Extract PCOUP tag (TIC standard)
	pcoup, ok := ticContent["PCOUP"]
	// PCOUP found ?
	if ok {
		// Convert PCOUP to integer
		limit, err = strconv.Atoi(pcoup)
		// PCOUP conversion succeed ?
		if err == nil {
			// Update overload power limit in VA
			overloadPowerLimit = limit * 1000
			// Get RMS voltages per phase
			rmsVoltagePerPhase, _ := getMeterRmsVoltageAndCurrentPerPhase(ticContent)
			// Update overload current limit for each phase in A
			overloadCurrentLimitPerPhase = make([]float64, len(rmsVoltagePerPhase))
			for i := 0; i < len(rmsVoltagePerPhase); i++ {
				if rmsVoltagePerPhase[i] > 0 {
					overloadCurrentLimitPerPhase[i] = float64(overloadPowerLimit) / (float64(len(rmsVoltagePerPhase)) * float64(rmsVoltagePerPhase[i]))
				} else {
					overloadCurrentLimitPerPhase[i] = 0.0
				}
			}
		}
	} else {
		// Extract ISOUSC tag (TIC historic)
		isousc, ok := ticContent["ISOUSC"]
		if ok {
			// Convert ISOUSC to integer
			limit, err = strconv.Atoi(isousc)
			// ISOUSC conversion succeed ?
			if err == nil {
				phaseCount := getMeterPhaseCount(ticContent)
				// Update overload power limit in VA
				overloadPowerLimit = limit * 200 * phaseCount
				// Update overload current limit for each phase in A
				overloadCurrentLimitPerPhase = make([]float64, phaseCount)
				for i := 0; i < phaseCount; i++ {
					overloadCurrentLimitPerPhase[i] = float64(limit)
				}
			}
		}
	}

	return overloadPowerLimit, overloadCurrentLimitPerPhase
}

func getMeterRmsVoltageAndCurrentPerPhase(ticContent map[string]string) (rmsVoltagePerPhase []int, rmsCurrentPerPhase []float64) {
	// Extract URMS1, URMS2 & URMS3 tags (TIC standard)
	urms1, urms1_Ok := ticContent["URMS1"]
	urms2, urms2_Ok := ticContent["URMS2"]
	urms3, urms3_Ok := ticContent["URMS3"]
	// URMS1 found ?
	if urms1_Ok {
		// Convert URMS1 to integer
		urms1Value, urms1_Err := strconv.Atoi(urms1)
		// URMS1 conversion succeed ?
		if urms1_Err == nil {
			// URMS2 & URMS3 found ?
			if urms2_Ok && urms3_Ok {
				// Convert URMS2 & URMS3 to integer
				urms2Value, urms2_Err := strconv.Atoi(urms2)
				urms3Value, urms3_Err := strconv.Atoi(urms3)
				// URMS2 & URMS3 conversions succeed ?
				if urms2_Err == nil && urms3_Err == nil {
					// Update RMS voltage for each phase in V
					rmsVoltagePerPhase = []int{urms1Value, urms2Value, urms3Value}
					// Get import apparent power for each phase
					apparentImportPowerPerPhase := getMeterApparentImportPowerPerPhase(ticContent)
					// Compute RMS current for each phase
					rmsCurrentPerPhase = make([]float64, len(rmsVoltagePerPhase))
					for i := 0; i < len(rmsVoltagePerPhase); i++ {
						if rmsVoltagePerPhase[i] > 0 {
							rmsCurrentPerPhase[i] = float64(apparentImportPowerPerPhase[i]) / float64(rmsVoltagePerPhase[i])
						} else {
							rmsCurrentPerPhase[i] = 0.0
						}
					}
				}
			} else {
				// Update RMS voltage for phase 1 in V
				rmsVoltagePerPhase = []int{urms1Value}
				// Get import apparent power
				apparentImportPower := getMeterApparentImportPower(ticContent)
				// Compute RMS current for phase 1
				rmsCurrentPhase1 := float64(apparentImportPower) / float64(urms1Value)
				// Update RMS current for phase 1 in A
				rmsCurrentPerPhase = []float64{rmsCurrentPhase1}
			}
		}
	} else {
		// Extract IINST tag (TIC historic)
		iinst, iinst_Ok := ticContent["IINST"]
		// IINST found ?
		if iinst_Ok {
			// Convert IINST to integer
			iinstValue, iinst_Err := strconv.Atoi(iinst)
			// IINST conversion succeed ?
			if iinst_Err == nil {
				// Update RMS current for phase 1 in A
				rmsCurrentPerPhase = []float64{float64(iinstValue)}
				// Get import apparent power
				apparentImportPower := getMeterApparentImportPower(ticContent)
				// Compute RMS voltage for phase 1
				var rmsVoltagePhase1 int
				if iinstValue > 0 {
					rmsVoltagePhase1 = apparentImportPower / iinstValue
				} else {
					rmsVoltagePhase1 = 0
				}
				// Update RMS voltage for phase 1 in V
				rmsVoltagePerPhase = []int{rmsVoltagePhase1}
			}
		} else {
			// Extract IINST1, IINST2 & IINST3 tags (TIC historic)
			iinst1, iinst1_Ok := ticContent["IINST1"]
			iinst2, iinst2_Ok := ticContent["IINST2"]
			iinst3, iinst3_Ok := ticContent["IINST3"]
			if iinst1_Ok && iinst2_Ok && iinst3_Ok {
				// Convert IINST1, IINST2 && IINST3 to integer
				iinst1Value, iinst1_Err := strconv.Atoi(iinst1)
				iinst2Value, iinst2_Err := strconv.Atoi(iinst2)
				iinst3Value, iinst3_Err := strconv.Atoi(iinst3)
				// IINST1, IINST2 & IINST3 conversions succeed ?
				if iinst1_Err == nil && iinst2_Err == nil && iinst3_Err == nil {
					// Update RMS current for each phase in A
					rmsCurrentPerPhase = []float64{float64(iinst1Value), float64(iinst2Value), float64(iinst3Value)}
					// Get import apparent power
					apparentImportPower := getMeterApparentImportPower(ticContent)
					// Compute RMS voltage for each phase
					rmsVoltagePerPhase = make([]int, len(rmsCurrentPerPhase))
					for i := 0; i < len(rmsCurrentPerPhase); i++ {
						if rmsCurrentPerPhase[i] > 0.0 {
							rmsVoltagePerPhase[i] = apparentImportPower / (3 * int(rmsCurrentPerPhase[i]))
						} else {
							rmsVoltagePerPhase[i] = 0.0
						}
					}
				}
			}
		}
	}

	return rmsVoltagePerPhase, rmsCurrentPerPhase
}

func getMeterApparentImportPower(ticContent map[string]string) (apparentImportPower int) {
	var power string
	var ok bool
	// Extract SINSTS tag (TIC standard)
	power, ok = ticContent["SINSTS"]
	// SINSTS found ?
	if !ok {
		// Extract PAPP tag (TIC historic)
		power, ok = ticContent["PAPP"]
	}
	// SINSTS or PAPP found ?
	if ok {
		// Convert power to integer
		powerValue, err := strconv.Atoi(power)
		// Power conversion succeed ?
		if err == nil {
			// Update total apparent import power in VA
			apparentImportPower = powerValue
		}
	}

	return apparentImportPower
}

func getMeterApparentImportPowerPerPhase(ticContent map[string]string) (apparentImportPowerPerPhase []int) {
	// Extract SINSTS1, SINSTS2 & SINSTS3 tags (TIC standard)
	sinsts1, sinsts1_Ok := ticContent["SINSTS1"]
	sinsts2, sinsts2_Ok := ticContent["SINSTS2"]
	sinsts3, sinsts3_Ok := ticContent["SINSTS3"]
	// SINSTS1 found ?
	if sinsts1_Ok {
		// Convert SINSTS1 to integer
		sinsts1Value, sinsts1_Err := strconv.Atoi(sinsts1)
		// SINSTS1 conversion succeed ?
		if sinsts1_Err == nil {
			// SINSTS2 & SINSTS2 found ?
			if sinsts2_Ok && sinsts3_Ok {
				// Convert SINSTS2 & SINSTS3 to integer
				sinsts2Value, sinsts2_Err := strconv.Atoi(sinsts2)
				sinsts3Value, sinsts3_Err := strconv.Atoi(sinsts3)
				// SINSTS2 & SINSTS3 conversions succeed ?
				if sinsts2_Err == nil && sinsts3_Err == nil {
					// Update apparent import power for each phase in VA
					apparentImportPowerPerPhase = []int{sinsts1Value, sinsts2Value, sinsts3Value}
				}
			} else {
				// Update apparent import power for phase 1 in VA
				apparentImportPowerPerPhase = []int{sinsts1Value}
			}
		}
	}

	return apparentImportPowerPerPhase
}
