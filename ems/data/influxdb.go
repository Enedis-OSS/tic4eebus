// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
// SPDX-FileContributor: Mathieu SABARTHES
//
// SPDX-License-Identifier: Apache-2.0

package data

import (
	"context"
	"fmt"
	"time"

	"github.com/Enedis-OSS/tic4eebus/config"
	"github.com/Enedis-OSS/tic4eebus/evse"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	log "github.com/sirupsen/logrus"
)

type InfluxDbWriter struct {
	client influxdb2.Client
	config config.InfluxDbConfig
}

func NewInfluxDbWriter(config *config.InfluxDbConfig) *InfluxDbWriter {
	if config == nil {
		return nil
	}
	handler := &InfluxDbWriter{}

	url := fmt.Sprintf("%s:%d", config.IpAddress, config.TcpPort)
	handler.client = influxdb2.NewClient(url, config.Token)
	log.Info("InfluxDb client created for URL: ", url)
	handler.config = *config

	return handler
}

func (h *InfluxDbWriter) Save(model DataModel) {
	if h == nil {
		return
	}
	health, err := h.client.Health(context.Background())
	if err != nil {
		log.Errorf("Cannot check influxDB health: %v", err)
		return
	}

	if health.Status != "pass" {
		log.Errorf("InfluxDB health check failed: status=%s", health.Status)
		return
	}
	writeAPI := h.client.WriteAPIBlocking(h.config.Org, h.config.Bucket)

	p := createPoint(model)
	if p == nil {
		log.Error("Cannot create influxDB point")
		return
	}

	if err := writeAPI.WritePoint(context.Background(), p); err != nil {
		log.Errorf("Cannot write influxDB point: %v", err)
		return
	}
}

func createPoint(model DataModel) (p *write.Point) {
	var currentPerPhase1, currentPerPhase2, currentPerPhase3 float64
	currentPerPhase, ok := model.Vehicle[evse.VEHICLE_CURRENT_PER_PHASE]
	if ok {
		currentPerPhaseTable, ok := currentPerPhase.([]float64)
		if ok {
			if len(currentPerPhaseTable) > 0 {
				currentPerPhase1 = currentPerPhaseTable[0]
			}
			if len(currentPerPhaseTable) > 1 {
				currentPerPhase2 = currentPerPhaseTable[1]
			}
			if len(currentPerPhaseTable) > 2 {
				currentPerPhase3 = currentPerPhaseTable[2]
			}
		}
	}

	var overLoadCurrentLimit1, overLoadCurrentLimit2, overLoadCurrentLimit3 float64
	if len(model.Meter.OverLoadCurrentLimitPerPhase) > 0 {
		overLoadCurrentLimit1 = model.Meter.OverLoadCurrentLimitPerPhase[0]
	}
	if len(model.Meter.OverLoadCurrentLimitPerPhase) > 1 {
		overLoadCurrentLimit2 = model.Meter.OverLoadCurrentLimitPerPhase[1]
	}
	if len(model.Meter.OverLoadCurrentLimitPerPhase) > 2 {
		overLoadCurrentLimit3 = model.Meter.OverLoadCurrentLimitPerPhase[2]
	}

	var rmsCurrent1, rmsCurrent2, rmsCurrent3 float64
	if len(model.Meter.RmsCurrentPerPhase) > 0 {
		rmsCurrent1 = model.Meter.RmsCurrentPerPhase[0]
	}
	if len(model.Meter.RmsCurrentPerPhase) > 1 {
		rmsCurrent2 = model.Meter.RmsCurrentPerPhase[1]
	}
	if len(model.Meter.RmsCurrentPerPhase) > 2 {
		rmsCurrent3 = model.Meter.RmsCurrentPerPhase[2]
	}

	p = write.NewPoint("EnergyGuard",
		map[string]string{
			"tag": "EnergyGuard",
		},
		map[string]interface{}{
			"Meter_DateTime":              model.Meter.DateTime,
			"Meter_BreakerOpened":         model.Meter.BreakerOpened,
			"IsConnected":                 model.IsConnected,
			"HasMeter":                    model.HasMeter,
			"HasOPEV":                     model.HasOPEV,
			"Diagnosis_LastErrorCode":     model.Diagnosis.LastErrorCode,
			"Meter_OverLoadCurrentLimit1": overLoadCurrentLimit1,
			"Meter_OverLoadCurrentLimit2": overLoadCurrentLimit2,
			"Meter_OverLoadCurrentLimit3": overLoadCurrentLimit3,
			"Meter_RmsCurrent1":           rmsCurrent1,
			"Meter_RmsCurrent2":           rmsCurrent2,
			"Meter_RmsCurrent3":           rmsCurrent3,
			"EV_CurrentPerPhase1":         currentPerPhase1,
			"EV_CurrentPerPhase2":         currentPerPhase2,
			"EV_CurrentPerPhase3":         currentPerPhase3,
			"OverloadProtection_Value":    model.OverloadProtection.Value,
		},
		time.Now())

	return p
}
