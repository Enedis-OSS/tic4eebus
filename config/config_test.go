// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
// SPDX-FileContributor: Mathieu SABARTHES
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"os"
	"path/filepath"
	"testing"

	log "github.com/sirupsen/logrus"
)

func TestLoadBasicConfiguration(t *testing.T) {
	// Create temporary config file
	configContent := `
OverloadProtection:
  Enable: true
  RunningPeriodInSeconds: 1
  CurrentLimit:
    ValueInAmps: 16.0
    LockDelayInSeconds: 10.0
Vehicle:
  UpdateDataPeriodInSeconds: 5
  DataPersistent: false
Wallbox:
  UpdateDataPeriodInSeconds: 5
  DataPersistent: false
Log:
  Level: "trace"
  FilePath: "var/log/tic4eebus.log"
  Rotation:
    PeriodInHours: 24
    PeriodCount: 15
    PeriodPattern: "-%Y-%m-%d"
DataModel:
  Csv:
    FilePath: "var/data/EnergyGuardDataModel.csv"
    Rotation:
      PeriodInHours: 24
      PeriodCount: 7
      PeriodPattern: "-%Y-%m-%d"
TeleInformationClient:
  TIC2Websocket:
    IPAddress: "127.0.0.1"
    TCPPort: 19584
  TICIdentifier:
    SerialNumber: ""
EEBUS:
  ServerPort: 4817
  RemoteSKI: "0123456789abcdef01234567890abcdef0123456"
  CertificateFilePath: "examples/energy-guard.cert"
  PrivateKeyFilePath: "examples/energy-guard.key"
  VendorCode: "i:12345"
  DeviceBrand: "Enedis"
  DeviceModel: "PAC"
  SerialNumber: "12345678"
  HeartbeatTimeoutInSeconds: 2
`

	// Create temporary config file
	// "var/data/EnergyGuardDataModel.csv", "examples/energy-guard.cert", and "examples/energy-guard.key" are required to exist
	// for the configuration to be valid. Create empty files for testing purposes.
	// Ensure the directory for the certificate and key files exists

	requiredFiles := []string{
		"examples/energy-guard.cert",
		"examples/energy-guard.key",
		"var/data/EnergyGuardDataModel.csv",
	}
	for _, file := range requiredFiles {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(file)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		defer os.Remove(file)
	}

	// Create temporary config file
	tmpfile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load configuration
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error while loading configuration file (%v): %v", tmpfile.Name(), err)
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
		{"EEBUS Server Port", config.EEBUS.ServerPort, 4817},
		{"EEBUS Remote SKI", config.EEBUS.RemoteSKI, "0123456789abcdef01234567890abcdef0123456"},
		{"EEBUS Certificate Path", config.EEBUS.CertificateFilePath, "examples/energy-guard.cert"},
		{"EEBUS Private Key Path", config.EEBUS.PrivateKeyFilePath, "examples/energy-guard.key"},
		{"EEBUS Vendor Code", config.EEBUS.VendorCode, "i:12345"},
		{"TIC IP Address", config.TeleInformationClient.TIC2Websocket.IPAddress, "127.0.0.1"},
		{"TIC TCP Port", config.TeleInformationClient.TIC2Websocket.TCPPort, 19584},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("%s : expected %v, actual %v", tt.name, tt.expected, tt.actual)
			}
		})
	}
}

func TestLoadInvalidConfiguration(t *testing.T) {
	// Test with a non-existent file
	_, err := LoadConfig("non-existing-file.yaml")
	if err == nil {
		t.Error("LoadConfig should fail for non-existing file")
	}

	// Test with invalid YAML content
	tmpfile, err := os.CreateTemp("", "invalid_config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	invalidContent := `
		Log:
		Level: "invalid_level"
		FilePath: "/var/log/tic4eebus.log"
	`
	if _, err := tmpfile.Write([]byte(invalidContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	_, err = LoadConfig(tmpfile.Name())
	if err == nil {
		t.Error("LoadConfig should fail for invalid log level")
	}
}

func TestLoadOptionalConfiguration(t *testing.T) {
	// Create temporary config file with minimal configuration (only required fields)
	configContent := `
OverloadProtection:
  Enable: true
  RunningPeriodInSeconds: 1
  CurrentLimit:
    ValueInAmps: 16.0
    LockDelayInSeconds: 10.0
Vehicle:
  UpdateDataPeriodInSeconds: 5
  DataPersistent: false
Wallbox:
  UpdateDataPeriodInSeconds: 5
  DataPersistent: false
Log:
  Level: "trace"
  FilePath: "var/log/tic4eebus.log"
  Rotation:
    PeriodInHours: 24
    PeriodCount: 15
    PeriodPattern: "-%Y-%m-%d"
DataModel:
TeleInformationClient:
  TIC2Websocket:
    IPAddress: "127.0.0.1"
    TCPPort: 19584
  TICIdentifier:
    SerialNumber: ""
EEBUS:
  ServerPort: 4817
  RemoteSKI: "0123456789abcdef01234567890abcdef0123456"
  CertificateFilePath: "examples/energy-guard.cert"
  PrivateKeyFilePath: "examples/energy-guard.key"
  VendorCode: "i:12345"
  DeviceBrand: "Enedis"
  DeviceModel: "PAC"
  SerialNumber: "12345678"
  HeartbeatTimeoutInSeconds: 2
`
	// Create temporary config file
	// "var/data/EnergyGuardDataModel.csv", "examples/energy-guard.cert", and "examples/energy-guard.key" are required to exist
	// for the configuration to be valid. Create empty files for testing purposes.
	// Ensure the directory for the certificate and key files exists

	requiredFiles := []string{
		"examples/energy-guard.cert",
		"examples/energy-guard.key",
		"var/data/EnergyGuardDataModel.csv",
	}
	for _, file := range requiredFiles {
		if err := os.MkdirAll(filepath.Dir(file), 0755); err != nil {
			t.Fatal(err)
		}
		f, err := os.Create(file)
		if err != nil {
			t.Fatal(err)
		}
		f.Close()
		defer os.Remove(file)
	}

	// Create temporary config file
	tmpfile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(configContent)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	// Load configuration
	config, err := LoadConfig(tmpfile.Name())
	if err != nil {
		t.Fatalf("Error while loading configuration file (%v): %v", tmpfile.Name(), err)
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
		{"EEBUS Server Port", config.EEBUS.ServerPort, 4817},
		{"EEBUS Remote SKI", config.EEBUS.RemoteSKI, "0123456789abcdef01234567890abcdef0123456"},
		{"EEBUS Certificate Path", config.EEBUS.CertificateFilePath, "examples/energy-guard.cert"},
		{"EEBUS Private Key Path", config.EEBUS.PrivateKeyFilePath, "examples/energy-guard.key"},
		{"EEBUS Vendor Code", config.EEBUS.VendorCode, "i:12345"},
		{"TIC IP Address", config.TeleInformationClient.TIC2Websocket.IPAddress, "127.0.0.1"},
		{"TIC TCP Port", config.TeleInformationClient.TIC2Websocket.TCPPort, 19584},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.actual != test.expected {
				t.Errorf("%s : expected %v, actual %v", test.name, test.expected, test.actual)
			}
		})
	}
}