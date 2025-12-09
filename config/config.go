// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

/*
Package config implements utility routines for manipulating application configuration.

The application configuration are grouped in the following categories:

  - overload protection parameters

  - vehicle parameters

  - wallbox parameters

  - log parameters

  - data model parameters

  - teleinformation client parameters

  - eebus parameters
*/
package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

// Current limitation configuration for electric vehicle charge
type CurrentLimitConfig struct {
	ValueInAmps        float64 // Electric vehicle charge current limitation in Amps
	LockDelayInSeconds float64 // Electric vehicle charge lock delay in seconds
}

// File rotation configuration for log and data model CSV files
type FileRotationConfig struct {
	PeriodInHours int    // Rotation period in hours
	PeriodCount   int    // Number of files to keep
	PeriodPattern string // Rotation pattern, e.g. ".%Y%m%d%H"
}

// Data model CSV file configuration
type CsvConfig struct {
	FilePath string             // CSV file path
	Rotation FileRotationConfig // CSV file rotation configuration
}

// Data model InfluxDb configuration
type InfluxDbConfig struct {
	Bucket    string // InfluxDb bucket
	Org       string // InfluxDb organization
	Token     string // InfluxDb token
	IpAddress string // InfluxDb server IP address
	TcpPort   int    // InfluxDb server TCP port
}

// TIC2WebSocket client configuration
type Tic2WebsocketConfig struct {
	IpAddress string // TIC2WebSocket server IP address
	TcpPort   int    // TIC2WebSocket server TCP port
}

// TIC identifier configuration used for TIC2WebSocket client subscription
type TicIdentifierConfig struct {
	SerialNumber string // Linky meter serial number
}

// Overload protection configuration
type OverloadProtectionConfig struct {
	Enable                 bool               // Enable/disable overload protection algorithm
	RunningPeriodInSeconds int                // Overload protection algorithm running period in seconds
	CurrentLimit           CurrentLimitConfig // Electric vehicle charge current limitation configuration
}

// Vehicle configuration
type VehicleConfig struct {
	UpdateDataPeriodInSeconds int  // Vehicle data update period in seconds
	DataPersistent            bool // Vehicle data persistent enable/disable
}

// Wallbox configuration
type WallboxConfig struct {
	UpdateDataPeriodInSeconds int  // Wallbox data update period in seconds
	DataPersistent            bool // Wallbox data persistent enable/disable
}

// Log configuration
type LogConfig struct {
	Level    log.Level          // Log level
	FilePath string             // Log file path
	Rotation FileRotationConfig // Log file rotation configuration
}

// Data model configuration
type DataModelConfig struct {
	Csv      *CsvConfig      // CSV file configuration or nil if not configured
	InfluxDb *InfluxDbConfig // InfluxDb configuration or nil if not configured
}

// Teleinformation client configuration
type TeleInformationClientConfig struct {
	Tic2Websocket Tic2WebsocketConfig // TIC2WebSocket client configuration
	TicIdentifier TicIdentifierConfig // TIC identifier configuration
}

// EEBUS configuration
type EebusConfig struct {
	ServerPort                int    // EEBUS server TCP port
	RemoteSki                 string // EEBUS remote SKI (SHA-1 hash, 40 hexadecimal digits)
	CertificateFilePath       string // EEBUS certificate file path
	PrivateKeyFilePath        string // EEBUS private key file path
	VendorCode                string // EEBUS vendor code
	DeviceBrand               string // EEBUS device brand
	DeviceModel               string // EEBUS device model
	SerialNumber              string // EEBUS device serial number
	HeartbeatTimeoutInSeconds int    // EEBUS heartbeat timeout in seconds
}

// Application configuration structure
// It groups all configuration categories
// The configuration is loaded from a YAML file using LoadConfig() function
type Config struct {
	OverloadProtection    OverloadProtectionConfig    // Overload protection configuration
	Vehicle               VehicleConfig               // Vehicle configuration
	Wallbox               WallboxConfig               // Wallbox configuration
	Log                   LogConfig                   // Log configuration
	DataModel             DataModelConfig             // Data model configuration
	TeleInformationClient TeleInformationClientConfig // Teleinformation client configuration
	Eebus                 EebusConfig                 // EEBUS configuration
}

const (
	invalid_parameter = "invalid config parameter"
	remote_ski_regexp = "^[a-fA-F0-9]{40}$" // SHA-1 hash is 20 bytes, represented as 40 hexadecimal digits
)

// Load configuration from configFilePath. copies from src to dst until either EOF is reached
// on src or an error occurs. It returns the configuration loaded and the error encountered.
//
// A successful load returns err == nil.
func LoadConfig(configFilePath string) (Config, error) {
	var config Config
	// Create empty configuration map
	configMap := make(map[any]any)
	// Read YAML file bytes
	data, err := os.ReadFile(configFilePath)
	// Check file reading
	if err != nil {
		return config, fmt.Errorf("error reading YAML file: %v", err)
	}
	// Replace environment variables in the file bytes by its value
	dataWithEnv := os.ExpandEnv(string(data))
	// Load YAML file bytes into configuration map
	err = yaml.Unmarshal([]byte(dataWithEnv), &configMap)
	// Check configuration map loading
	if err != nil {
		return config, fmt.Errorf("error unmarshalling YAML: %v", err)
	}
	// Load overload protection configuration
	config.OverloadProtection.Enable,
		config.OverloadProtection.RunningPeriodInSeconds,
		config.OverloadProtection.CurrentLimit.ValueInAmps,
		config.OverloadProtection.CurrentLimit.LockDelayInSeconds,
		err = loadOverloadProtection(configMap)
	if err != nil {
		return config, err
	}
	// Load vehicle configuration
	config.Vehicle.UpdateDataPeriodInSeconds,
		config.Vehicle.DataPersistent,
		err = loadVehicle(configMap)
	if err != nil {
		return config, err
	}
	// Load wallbox configuration
	config.Wallbox.UpdateDataPeriodInSeconds,
		config.Wallbox.DataPersistent,
		err = loadWallbox(configMap)
	if err != nil {
		return config, err
	}
	// Load log configuration
	config.Log.Level,
		config.Log.FilePath,
		config.Log.Rotation.PeriodInHours,
		config.Log.Rotation.PeriodCount,
		config.Log.Rotation.PeriodPattern,
		err = loadLog(configMap)
	if err != nil {
		return config, err
	}
	// Load data model configuration
	config.DataModel.Csv,
		config.DataModel.InfluxDb,
		err = loadDataModel(configMap)
	if err != nil {
		return config, err
	}
	// Load tele information configuration
	config.TeleInformationClient.Tic2Websocket.IpAddress,
		config.TeleInformationClient.Tic2Websocket.TcpPort,
		config.TeleInformationClient.TicIdentifier.SerialNumber,
		err = loadTeleInformationClient(configMap)
	if err != nil {
		return config, err
	}
	// Load EEBUS configuration
	config.Eebus.ServerPort,
		config.Eebus.RemoteSki,
		config.Eebus.CertificateFilePath,
		config.Eebus.PrivateKeyFilePath,
		config.Eebus.VendorCode,
		config.Eebus.DeviceBrand,
		config.Eebus.DeviceModel,
		config.Eebus.SerialNumber,
		config.Eebus.HeartbeatTimeoutInSeconds,
		err = loadEebus(configMap)
	if err != nil {
		return config, err
	}

	return config, err
}

func loadOverloadProtection(configMap map[interface{}]interface{}) (enable bool, runningPeriodInSeconds int, valueInAmps float64, lockDelayInSeconds float64, err error) {
	paramParentName, paramName := "", "OverloadProtection"
	overloadProtectionMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return false, 0, 0.0, 0.0, err
	}
	paramParentName = paramName
	enable, err = loadParameterAsBoolean(overloadProtectionMap, paramParentName, "Enable")
	if err != nil {
		return enable, 0, 0.0, 0.0, err
	}
	runningPeriodInSeconds, err = loadParameterAsInt(overloadProtectionMap, paramParentName, "RunningPeriodInSeconds", true, 0, false, 0)
	if err != nil {
		return enable, runningPeriodInSeconds, 0.0, 0.0, err
	}
	paramName = "CurrentLimit"
	currentLimitMap, err := loadParameterAsMap(overloadProtectionMap, paramParentName, paramName, true)
	if err != nil {
		return enable, runningPeriodInSeconds, 0.0, 0.0, err
	}
	paramParentName = paramParentName + "." + paramName
	valueInAmps, err = loadParameterAsFloat(currentLimitMap, paramParentName, "ValueInAmps", true, 0.0, false, 0.0)
	if err != nil {
		return enable, runningPeriodInSeconds, 0.0, 0.0, err
	}
	lockDelayInSeconds, err = loadParameterAsFloat(currentLimitMap, paramParentName, "LockDelayInSeconds", true, 0.0, false, 0.0)

	return enable, runningPeriodInSeconds, valueInAmps, lockDelayInSeconds, err
}

func loadVehicle(configMap map[interface{}]interface{}) (updateDataPeriodInSeconds int, dataPersistent bool, err error) {
	paramParentName, paramName := "", "Vehicle"
	vehicleMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return 0, false, err
	}
	paramParentName = paramName
	updateDataPeriodInSeconds, err = loadParameterAsInt(vehicleMap, paramParentName, "UpdateDataPeriodInSeconds", true, 0, false, 0)
	if err != nil {
		return 0, false, err
	}
	dataPersistent, err = loadParameterAsBoolean(vehicleMap, paramParentName, "DataPersistent")

	return updateDataPeriodInSeconds, dataPersistent, err
}

func loadWallbox(configMap map[interface{}]interface{}) (updateDataPeriodInSeconds int, dataPersistent bool, err error) {
	paramParentName, paramName := "", "Wallbox"
	wallboxMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return 0, false, err
	}
	paramParentName = paramName
	updateDataPeriodInSeconds, err = loadParameterAsInt(wallboxMap, paramParentName, "UpdateDataPeriodInSeconds", true, 0, false, 0)
	if err != nil {
		return 0, false, err
	}
	dataPersistent, err = loadParameterAsBoolean(wallboxMap, paramParentName, "DataPersistent")

	return updateDataPeriodInSeconds, dataPersistent, err
}

func loadLog(configMap map[interface{}]interface{}) (level log.Level, filePath string, periodInHours int, periodCount int, periodPattern string, err error) {
	paramParentName, paramName := "", "Log"
	logMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return 0, "", 0, 0, "", err
	}
	paramParentName = paramName
	level, err = loadParameterAsLogLevel(logMap, paramParentName, "Level")
	if err != nil {
		return 0, "", 0, 0, "", err
	}
	filePath, err = loadParameterAsString(logMap, paramParentName, "FilePath", true)
	if err != nil {
		return level, "", 0, 0, "", err
	}
	periodInHours, periodCount, periodPattern, err = loadParameterAsFileRotationMap(logMap, paramParentName, "Rotation")
	if err != nil {
		return level, filePath, 0, 0, "", err
	}

	return level, filePath, periodInHours, periodCount, periodPattern, err
}

func loadDataModel(configMap map[interface{}]interface{}) (csvConfig *CsvConfig, influxDbConfig *InfluxDbConfig, err error) {
	paramParentName, paramName := "", "DataModel"
	dataModelMap, err := loadParameterAsMap(configMap, "", "DataModel", true)
	if err != nil {
		return nil, nil, err
	}
	paramParentName = paramName
	paramName = "Csv"
	csvMap, err := loadParameterAsMap(dataModelMap, paramParentName, paramName, false)
	if err != nil {
		return nil, nil, err
	}
	if csvMap != nil {
		paramParentName = paramParentName + "." + paramName
		filePath, err := loadParameterAsString(csvMap, paramParentName, "FilePath", true)
		if err != nil {
			return nil, nil, err
		}
		periodInHours, periodCount, periodPattern, err := loadParameterAsFileRotationMap(csvMap, paramParentName, "Rotation")
		if err != nil {
			return nil, nil, err
		}
		csvConfig = &CsvConfig{filePath, FileRotationConfig{periodInHours, periodCount, periodPattern}}
	}
	paramParentName = "DataModel"
	paramName = "InfluxDb"
	influxDbMap, err := loadParameterAsMap(dataModelMap, paramParentName, paramName, false)
	if err != nil {
		return nil, nil, err
	}
	if len(influxDbMap) > 0 {
		paramParentName = paramParentName + "." + paramName
		bucket, err := loadParameterAsString(influxDbMap, paramParentName, "Bucket", true)
		if err != nil {
			return nil, nil, err
		}
		org, err := loadParameterAsString(influxDbMap, paramParentName, "Org", true)
		if err != nil {
			return nil, nil, err
		}
		token, err := loadParameterAsString(influxDbMap, paramParentName, "Token", true)
		if err != nil {
			return nil, nil, err
		}
		ipAddress, err := loadParameterAsString(influxDbMap, paramParentName, "IpAddress", true)
		if err != nil {
			return nil, nil, err
		}
		tcpPort, err := loadParameterAsInt(influxDbMap, paramParentName, "TcpPort", true, 1, true, 65535)
		if err != nil {
			return nil, nil, err
		}
		influxDbConfig = &InfluxDbConfig{bucket, org, token, ipAddress, tcpPort}
	}

	return csvConfig, influxDbConfig, err
}

func loadTeleInformationClient(configMap map[interface{}]interface{}) (tic2WebsocketIPAddress string, tic2WebsocketTCPPort int, ticIdentifierSerialNumber string, err error) {
	paramParentName, paramName := "", "TeleInformationClient"
	teleInformationClientMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return "", 0, "", err
	}
	paramParentName = paramName
	tic2WebsocketMap, err := loadParameterAsMap(teleInformationClientMap, paramParentName, "Tic2Websocket", true)
	if err != nil {
		return "", 0, "", err
	}
	paramParentName = paramParentName + "." + paramName
	tic2WebsocketIPAddress, err = loadParameterAsString(tic2WebsocketMap, paramParentName, "IpAddress", true)
	if err != nil {
		return "", 0, "", err
	}
	tic2WebsocketTCPPort, err = loadParameterAsInt(tic2WebsocketMap, paramParentName, "TcpPort", true, 1, true, 65535)
	if err != nil {
		return tic2WebsocketIPAddress, 0, "", err
	}
	ticIdentifierMap, err := loadParameterAsMap(teleInformationClientMap, paramName, "TicIdentifier", true)
	if err != nil {
		return tic2WebsocketIPAddress, tic2WebsocketTCPPort, "", err
	}
	ticIdentifierSerialNumber, err = loadParameterAsString(ticIdentifierMap, paramParentName, "SerialNumber", false)

	return tic2WebsocketIPAddress, tic2WebsocketTCPPort, ticIdentifierSerialNumber, err
}

func loadEebus(configMap map[interface{}]interface{}) (serverPort int,
	remoteSKI string,
	certificateFilePath string,
	privateKeyFilePath string,
	vendorCode string,
	deviceBrand string,
	deviceModel string,
	serialNumber string,
	heartbeatTimeoutInSeconds int,
	err error) {

	paramParentName, paramName := "", "Eebus"
	eebusMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	paramParentName = paramName
	serverPort, err = loadParameterAsInt(eebusMap, paramParentName, "ServerPort", true, 1, true, 65535)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	remoteSKI, err = loadParameterAsSKI(eebusMap, paramParentName, "RemoteSki")
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	certificateFilePath, err = loadParameterAsExistingFilePath(eebusMap, paramParentName, "CertificateFilePath")
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	privateKeyFilePath, err = loadParameterAsExistingFilePath(eebusMap, paramParentName, "PrivateKeyFilePath")
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	vendorCode, err = loadParameterAsString(eebusMap, paramParentName, "VendorCode", true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	deviceBrand, err = loadParameterAsString(eebusMap, paramParentName, "DeviceBrand", true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	deviceModel, err = loadParameterAsString(eebusMap, paramParentName, "DeviceModel", true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	serialNumber, err = loadParameterAsString(eebusMap, paramParentName, "SerialNumber", true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	heartbeatTimeoutInSeconds, err = loadParameterAsInt(eebusMap, paramParentName, "HeartbeatTimeoutInSeconds", true, 0, false, 0)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}

	return serverPort, remoteSKI, certificateFilePath, privateKeyFilePath, vendorCode, deviceBrand, deviceModel, serialNumber, heartbeatTimeoutInSeconds, nil
}

func loadParameterAsMap(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, mandatory bool) (map[interface{}]interface{}, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		if mandatory {
			if len(paramParentName) == 0 {
				return nil, fmt.Errorf("%s: %s not found", invalid_parameter, paramName)
			} else {
				return nil, fmt.Errorf("%s: %s.%s not found", invalid_parameter, paramParentName, paramName)
			}
		} else {
			return nil, nil
		}
	}
	parameterValue, ok := parameter.(map[any]any)
	if parameterValue == nil {
		return nil, nil
	}

	if !ok {
		return nil, fmt.Errorf("%s: %s.%s is not a map (%v)", invalid_parameter, paramParentName, paramName, reflect.TypeOf(parameter))
	}

	return parameterValue, nil
}

func loadParameterAsExistingFilePath(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (string, error) {
	filePath, err := loadParameterAsString(parameterMap, paramParentName, paramName, true)
	if err != nil {
		return "", err
	}
	_, err = os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("%s: %s.%s file '%s' not found", invalid_parameter, paramParentName, paramName, filePath)
		} else {
			return "", fmt.Errorf("%s: %s.%s file '%s' error (%s)", invalid_parameter, paramParentName, paramName, filePath, err.Error())
		}
	}

	return filePath, nil
}

func loadParameterAsFileRotationMap(parentMap map[interface{}]interface{}, paramParentName string, paramName string) (periodInHours int, periodCount int, periodPattern string, err error) {
	rotationMap, err := loadParameterAsMap(parentMap, paramParentName, paramName, true)
	if err != nil {
		return 0, 0, "", err
	}
	newParentKey := paramParentName + "." + paramName
	periodInHours, err = loadParameterAsInt(rotationMap, newParentKey, "PeriodInHours", true, 0, false, 0)
	if err != nil {
		return 0, 0, "", err
	}
	periodCount, err = loadParameterAsInt(rotationMap, newParentKey, "PeriodCount", true, 0, false, 0)
	if err != nil {
		return 0, 0, "", err
	}
	periodPattern, err = loadParameterAsString(rotationMap, newParentKey, "PeriodPattern", true)
	if err != nil {
		return 0, 0, "", err
	}

	return periodInHours, periodCount, periodPattern, nil
}

func loadParameterAsLogLevel(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (log.Level, error) {
	level, err := loadParameterAsString(parameterMap, paramParentName, paramName, true)
	if err != nil {
		return 0, err
	}
	logLevel, err := log.ParseLevel(level)
	if err != nil {
		return 0, fmt.Errorf("%s: %s.%s unknown (%s)", invalid_parameter, paramParentName, paramName, err.Error())
	}

	return logLevel, nil
}

func loadParameterAsSKI(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (string, error) {
	ski, err := loadParameterAsString(parameterMap, paramParentName, paramName, true)
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(remote_ski_regexp)
	if err != nil {
		return "", fmt.Errorf("%s: %s.%s regex error (%s)", invalid_parameter, paramParentName, paramName, err.Error())
	}
	if len(ski) != 40 { // SHA-1 hash is 20 bytes, represented as 40 hexadecimal digits
		return "", fmt.Errorf("%s: %s.%s must be 40 hexadecimal digits", invalid_parameter, paramParentName, paramName)
	}
	// Check if the string matches the regex for a SHA-1 hash
	if !re.MatchString(ski) {
		return "", fmt.Errorf("%s: %s.%s is not a SHA-1 hash with 20 hexadecimal digits", invalid_parameter, paramParentName, paramName)
	}

	return ski, nil
}

func loadParameterAsBoolean(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (bool, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return false, fmt.Errorf("%s: %s.%s not found", invalid_parameter, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(bool)
	if !ok {
		return false, fmt.Errorf("%s: %s.%s '%v' is not a boolean", invalid_parameter, paramParentName, paramName, parameter)
	}

	return parameterValue, nil
}

func loadParameterAsString(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, notEmpty bool) (string, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return "", fmt.Errorf("%s: %s.%s not found", invalid_parameter, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(string)
	if !ok {
		return "", fmt.Errorf("%s: %s.%s '%v' is not a string", invalid_parameter, paramParentName, paramName, parameter)
	}
	if notEmpty {
		if len(parameterValue) == 0 {
			return "", fmt.Errorf("%s: %s.%s cannot be empty", invalid_parameter, paramParentName, paramName)
		}
	}

	return parameterValue, nil
}

func loadParameterAsInt(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, hasMin bool, minValue int, hasMax bool, maxValue int) (int, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s not found", invalid_parameter, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(int)
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s '%v' is not an integer", invalid_parameter, paramParentName, paramName, parameter)
	}
	if hasMin {
		if hasMax {
			if parameterValue < minValue || parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  out of range [%d, %d]", invalid_parameter, paramParentName, paramName, minValue, maxValue)
			}
		} else {
			if parameterValue < minValue {
				return 0, fmt.Errorf("%s: %s.%s  must be >= %d", invalid_parameter, paramParentName, paramName, minValue)
			}
		}
	} else {
		if hasMax {
			if parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  must be < %d", invalid_parameter, paramParentName, paramName, maxValue)
			}
		}
	}

	return parameterValue, nil
}

func loadParameterAsFloat(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, hasMin bool, minValue float64, hasMax bool, maxValue float64) (float64, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s not found", invalid_parameter, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(float64)
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s '%v' is not a float", invalid_parameter, paramParentName, paramName, parameter)
	}
	if hasMin {
		if hasMax {
			if parameterValue < minValue || parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  out of range [%f, %f]", invalid_parameter, paramParentName, paramName, minValue, maxValue)
			}
		} else {
			if parameterValue < minValue {
				return 0, fmt.Errorf("%s: %s.%s  must be >= %f", invalid_parameter, paramParentName, paramName, minValue)
			}
		}
	} else {
		if hasMax {
			if parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  must be < %f", invalid_parameter, paramParentName, paramName, maxValue)
			}
		}
	}

	return parameterValue, nil
}
