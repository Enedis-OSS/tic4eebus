// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
// SPDX-FileContributor: Mathieu SABARTHES
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"path/filepath"
	"reflect"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestLoadBasicConfiguration(t *testing.T) {
	configPath := filepath.Join("testdata", "basic_config.yaml")

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error while loading configuration file (%v): %v", configPath, err)
	}

	// Check values
	tests := []struct {
		name     string
		actual   any
		expected any
	}{
		{"Log Level", config.Log.Level, log.TraceLevel},
		{"Log FilePath", config.Log.FilePath, "var/log/tic4eebus.log"},
		{"Log Rotation Period", config.Log.Rotation.PeriodInHours, 24},
		{"Log Rotation Count", config.Log.Rotation.PeriodCount, 15},
		{"Overload Protection Enabled", config.OverloadProtection.Enable, true},
		{"Overload Protection Running Period", config.OverloadProtection.RunningPeriodInSeconds, 1},
		{"Overload Protection Current Limit", config.OverloadProtection.CurrentLimit.ValueInAmps, 16.0},
		{"Overload Protection Lock Delay", config.OverloadProtection.CurrentLimit.LockDelayInSeconds, 10.0},
		{"Vehicle Update Period", config.Vehicle.UpdateDataPeriodInSeconds, 5},
		{"Vehicle Data Persistent", config.Vehicle.DataPersistent, false},
		{"EEBUS Server Port", config.Eebus.ServerPort, 4817},
		{"EEBUS Remote SKI", config.Eebus.RemoteSki, "0123456789abcdef01234567890abcdef0123456"},
		{"EEBUS Certificate Path", config.Eebus.CertificateFilePath, "../examples/energy-guard.cert"},
		{"EEBUS Private Key Path", config.Eebus.PrivateKeyFilePath, "../examples/energy-guard.key"},
		{"EEBUS Vendor Code", config.Eebus.VendorCode, "i:12345"},
		{"TIC IP Address", config.TeleInformationClient.Tic2Websocket.IpAddress, "127.0.0.1"},
		{"TIC TCP Port", config.TeleInformationClient.Tic2Websocket.TcpPort, 19584},
		{"Data Model CSV FilePath", config.DataModel.Csv.FilePath, "testdata/EnergyGuardDataModel.csv"},
		{"Data Model CSV Rotation Period", config.DataModel.Csv.Rotation.PeriodInHours, 24},
		{"Data Model CSV Rotation Count", config.DataModel.Csv.Rotation.PeriodCount, 7},
		{"Data Model InfluxDB Bucket", config.DataModel.InfluxDb.Bucket, "demo-bucket"},
		{"Data Model InfluxDB Org", config.DataModel.InfluxDb.Org, "demo-org"},
		{"Data Model InfluxDB Token", config.DataModel.InfluxDb.Token, "He_nwgoqXu4XggIzaiUWFHALKnS5JskdTLzGlSYeJMNkjCD-pyR6Yc5Hvl8NWj5Qo5C80mLvuJAS1IuqOcq4GQZZ"},
		{"Data Model InfluxDB IP Address", config.DataModel.InfluxDb.IpAddress, "127.0.0.1"},
		{"Data Model InfluxDB TCP Port", config.DataModel.InfluxDb.TcpPort, 8086},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedNil := test.expected == nil || (reflect.ValueOf(test.expected).Kind() == reflect.Ptr && reflect.ValueOf(test.expected).IsNil())
			actualNil := test.actual == nil || (reflect.ValueOf(test.actual).Kind() == reflect.Ptr && reflect.ValueOf(test.actual).IsNil())
			if expectedNil && actualNil {
				return
			}
			if test.actual != test.expected {
				t.Errorf("%s : expected %v, actual %v", test.name, test.expected, test.actual)
			}
		})
	}
}

func TestLoadInvalidConfiguration_nonExistingFile(t *testing.T) {
	// Test with a non-existent file
	_, err := LoadConfig("non-existing-file.yaml")
	if err == nil {
		t.Error("LoadConfig should fail for non-existing file")
	}

	// Test with invalid YAML content
	configPath := filepath.Join("testdata", "invalid_config.yaml")
	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("LoadConfig should fail for invalid log level")
	}
}

func TestLoadOptionalConfiguration_emptyDataModel(t *testing.T) {
	configPath := filepath.Join("testdata", "optional_config.yaml")

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Error while loading configuration file (%v): %v", configPath, err)
	}

	// Check values
	tests := []struct {
		name     string
		actual   any
		expected any
	}{
		{"Log Level", config.Log.Level, log.TraceLevel},
		{"Log FilePath", config.Log.FilePath, "var/log/tic4eebus.log"},
		{"Log Rotation Period", config.Log.Rotation.PeriodInHours, 24},
		{"Log Rotation Count", config.Log.Rotation.PeriodCount, 15},
		{"Overload Protection Enabled", config.OverloadProtection.Enable, true},
		{"Overload Protection Running Period", config.OverloadProtection.RunningPeriodInSeconds, 1},
		{"Overload Protection Current Limit", config.OverloadProtection.CurrentLimit.ValueInAmps, 16.0},
		{"Overload Protection Lock Delay", config.OverloadProtection.CurrentLimit.LockDelayInSeconds, 10.0},
		{"Vehicle Update Period", config.Vehicle.UpdateDataPeriodInSeconds, 5},
		{"Vehicle Data Persistent", config.Vehicle.DataPersistent, false},
		{"EEBUS Server Port", config.Eebus.ServerPort, 4817},
		{"EEBUS Remote SKI", config.Eebus.RemoteSki, "0123456789abcdef01234567890abcdef0123456"},
		{"EEBUS Certificate Path", config.Eebus.CertificateFilePath, "../examples/energy-guard.cert"},
		{"EEBUS Private Key Path", config.Eebus.PrivateKeyFilePath, "../examples/energy-guard.key"},
		{"EEBUS Vendor Code", config.Eebus.VendorCode, "i:12345"},
		{"TIC IP Address", config.TeleInformationClient.Tic2Websocket.IpAddress, "127.0.0.1"},
		{"TIC TCP Port", config.TeleInformationClient.Tic2Websocket.TcpPort, 19584},
		{"Data Model CSV is nil", config.DataModel.Csv, nil},
		{"Data Model InfluxDB is nil", config.DataModel.InfluxDb, nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			expectedNil := test.expected == nil || (reflect.ValueOf(test.expected).Kind() == reflect.Ptr && reflect.ValueOf(test.expected).IsNil())
			actualNil := test.actual == nil || (reflect.ValueOf(test.actual).Kind() == reflect.Ptr && reflect.ValueOf(test.actual).IsNil())
			if expectedNil && actualNil {
				return
			}
			if test.actual != test.expected {
				t.Errorf("%s : expected %v, actual %v", test.name, test.expected, test.actual)
			}
		})
	}
}
