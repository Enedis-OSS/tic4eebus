// SPDX-FileCopyrightText: 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
// SPDX-FileContributor: Jehan BOUSCH
//
// SPDX-License-Identifier: Apache-2.0

package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type CurrentLimitConfig struct {
	ValueInAmps        float64
	LockDelayInSeconds float64
}

type FileRotationConfig struct {
	PeriodInHours int
	PeriodCount   int
	PeriodPattern string
}

type CsvConfig struct {
	FilePath string
	Rotation FileRotationConfig
}

type InfluxDbConfig struct {
	Bucket    string
	Org       string
	Token     string
	IpAddress string
	TcpPort   int
}

type TIC2WebsocketConfig struct {
	IPAddress string
	TCPPort   int
}

type TICIdentifierConfig struct {
	SerialNumber string
}

type OverloadProtectionConfig struct {
	Enable                 bool
	RunningPeriodInSeconds int
	CurrentLimit           CurrentLimitConfig
}

type VehicleConfig struct {
	UpdateDataPeriodInSeconds int
	DataPersistent            bool
}

type WallboxConfig struct {
	UpdateDataPeriodInSeconds int
	DataPersistent            bool
}

type LogConfig struct {
	Level    log.Level
	FilePath string
	Rotation FileRotationConfig
}

type DataModelConfig struct {
	Csv      *CsvConfig
	InfluxDb *InfluxDbConfig
}

type TeleInformationClientConfig struct {
	TIC2Websocket TIC2WebsocketConfig
	TICIdentifier TICIdentifierConfig
}

type EEBUSConfig struct {
	ServerPort                int
	RemoteSKI                 string
	CertificateFilePath       string
	PrivateKeyFilePath        string
	VendorCode                string
	DeviceBrand               string
	DeviceModel               string
	SerialNumber              string
	HeartbeatTimeoutInSeconds int
}

type Config struct {
	OverloadProtection    OverloadProtectionConfig
	Vehicle               VehicleConfig
	Wallbox               WallboxConfig
	Log                   LogConfig
	DataModel             DataModelConfig
	TeleInformationClient TeleInformationClientConfig
	EEBUS                 EEBUSConfig
}

const (
	INVALID_PARAMETER = "invalid config parameter"
	REMOTE_SKI_REGEXP = "^[a-fA-F0-9]{40}$" // SHA-1 hash is 20 bytes, represented as 40 hexadecimal digits
)

func LoadConfig(configFilePath string) (Config, error) {
	var config Config
	configMap := make(map[interface{}]interface{})
	data, err := os.ReadFile(configFilePath)
	if err != nil {
		return config, fmt.Errorf("error reading YAML file: %v", err)
	}
	dataWithEnv := os.ExpandEnv(string(data))
	err = yaml.Unmarshal([]byte(dataWithEnv), &configMap)
	if err != nil {
		return config, fmt.Errorf("error unmarshalling YAML: %v", err)
	}
	config.OverloadProtection.Enable,
		config.OverloadProtection.RunningPeriodInSeconds,
		config.OverloadProtection.CurrentLimit.ValueInAmps,
		config.OverloadProtection.CurrentLimit.LockDelayInSeconds,
		err = loadOverloadProtection(configMap)
	if err != nil {
		return config, err
	}
	config.Vehicle.UpdateDataPeriodInSeconds, config.Vehicle.DataPersistent, err = loadVehicle(configMap)
	if err != nil {
		return config, err
	}
	config.Wallbox.UpdateDataPeriodInSeconds, config.Wallbox.DataPersistent, err = loadWallbox(configMap)
	if err != nil {
		return config, err
	}
	config.Log.Level,
		config.Log.FilePath,
		config.Log.Rotation.PeriodInHours,
		config.Log.Rotation.PeriodCount,
		config.Log.Rotation.PeriodPattern,
		err = loadLog(configMap)
	if err != nil {
		return config, err
	}
	config.DataModel.Csv,
		config.DataModel.InfluxDb,
		err = loadDataModel(configMap)
	if err != nil {
		return config, err
	}
	config.TeleInformationClient.TIC2Websocket.IPAddress,
		config.TeleInformationClient.TIC2Websocket.TCPPort,
		config.TeleInformationClient.TICIdentifier.SerialNumber,
		err = loadTeleInformationClient(configMap)
	if err != nil {
		return config, err
	}
	config.EEBUS.ServerPort,
		config.EEBUS.RemoteSKI,
		config.EEBUS.CertificateFilePath,
		config.EEBUS.PrivateKeyFilePath,
		config.EEBUS.VendorCode,
		config.EEBUS.DeviceBrand,
		config.EEBUS.DeviceModel,
		config.EEBUS.SerialNumber,
		config.EEBUS.HeartbeatTimeoutInSeconds,
		err = loadEEBUS(configMap)
	if err != nil {
		return config, err
	}

	return config, err
}

func DumpConfig(config Config) (string, error) {
	buffer, err := yaml.Marshal((&config))

	if err != nil {
		return "", err
	}

	return string(buffer), err
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
	tic2WebsocketMap, err := loadParameterAsMap(teleInformationClientMap, paramParentName, "TIC2Websocket", true)
	if err != nil {
		return "", 0, "", err
	}
	paramParentName = paramParentName + "." + paramName
	tic2WebsocketIPAddress, err = loadParameterAsString(tic2WebsocketMap, paramParentName, "IPAddress", true)
	if err != nil {
		return "", 0, "", err
	}
	tic2WebsocketTCPPort, err = loadParameterAsInt(tic2WebsocketMap, paramParentName, "TCPPort", true, 1, true, 65535)
	if err != nil {
		return tic2WebsocketIPAddress, 0, "", err
	}
	ticIdentifierMap, err := loadParameterAsMap(teleInformationClientMap, paramName, "TICIdentifier", true)
	if err != nil {
		return tic2WebsocketIPAddress, tic2WebsocketTCPPort, "", err
	}
	ticIdentifierSerialNumber, err = loadParameterAsString(ticIdentifierMap, paramParentName, "SerialNumber", false)

	return tic2WebsocketIPAddress, tic2WebsocketTCPPort, ticIdentifierSerialNumber, err
}

func loadEEBUS(configMap map[interface{}]interface{}) (serverPort int,
	remoteSKI string,
	certificateFilePath string,
	privateKeyFilePath string,
	vendorCode string,
	deviceBrand string,
	deviceModel string,
	serialNumber string,
	heartbeatTimeoutInSeconds int,
	err error) {

	paramParentName, paramName := "", "EEBUS"
	eebusMap, err := loadParameterAsMap(configMap, paramParentName, paramName, true)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	paramParentName = paramName
	serverPort, err = loadParameterAsInt(eebusMap, paramParentName, "ServerPort", true, 1, true, 65535)
	if err != nil {
		return 0, "", "", "", "", "", "", "", 0, err
	}
	remoteSKI, err = loadParameterAsSKI(eebusMap, paramParentName, "RemoteSKI")
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
				return nil, fmt.Errorf("%s: %s not found", INVALID_PARAMETER, paramName)
			} else {
				return nil, fmt.Errorf("%s: %s.%s not found", INVALID_PARAMETER, paramParentName, paramName)
			}
		} else {
			return nil, nil
		}
	}
	parameterValue, ok := parameter.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("%s: %s.%s is not a map (%v)", INVALID_PARAMETER, paramParentName, paramName, reflect.TypeOf(parameter))
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
			return "", fmt.Errorf("%s: %s.%s file '%s' not found", INVALID_PARAMETER, paramParentName, paramName, filePath)
		} else {
			return "", fmt.Errorf("%s: %s.%s file '%s' error (%s)", INVALID_PARAMETER, paramParentName, paramName, filePath, err.Error())
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
		return 0, fmt.Errorf("%s: %s.%s unknown (%s)", INVALID_PARAMETER, paramParentName, paramName, err.Error())
	}

	return logLevel, nil
}

func loadParameterAsSKI(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (string, error) {
	ski, err := loadParameterAsString(parameterMap, paramParentName, paramName, true)
	if err != nil {
		return "", err
	}
	re, err := regexp.Compile(REMOTE_SKI_REGEXP)
	if err != nil {
		return "", fmt.Errorf("%s: %s.%s regex error (%s)", INVALID_PARAMETER, paramParentName, paramName, err.Error())
	}
	if len(ski) != 40 { // SHA-1 hash is 20 bytes, represented as 40 hexadecimal digits
		return "", fmt.Errorf("%s: %s.%s must be 40 hexadecimal digits", INVALID_PARAMETER, paramParentName, paramName)
	}
	// Check if the string matches the regex for a SHA-1 hash
	if !re.MatchString(ski) {
		return "", fmt.Errorf("%s: %s.%s is not a SHA-1 hash with 20 hexadecimal digits", INVALID_PARAMETER, paramParentName, paramName)
	}

	return ski, nil
}

func loadParameterAsBoolean(parameterMap map[interface{}]interface{}, paramParentName string, paramName string) (bool, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return false, fmt.Errorf("%s: %s.%s not found", INVALID_PARAMETER, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(bool)
	if !ok {
		return false, fmt.Errorf("%s: %s.%s '%v' is not a boolean", INVALID_PARAMETER, paramParentName, paramName, parameter)
	}

	return parameterValue, nil
}

func loadParameterAsString(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, notEmpty bool) (string, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return "", fmt.Errorf("%s: %s.%s not found", INVALID_PARAMETER, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(string)
	if !ok {
		return "", fmt.Errorf("%s: %s.%s '%v' is not a string", INVALID_PARAMETER, paramParentName, paramName, parameter)
	}
	if notEmpty {
		if len(parameterValue) == 0 {
			return "", fmt.Errorf("%s: %s.%s cannot be empty", INVALID_PARAMETER, paramParentName, paramName)
		}
	}

	return parameterValue, nil
}

func loadParameterAsInt(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, hasMin bool, minValue int, hasMax bool, maxValue int) (int, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s not found", INVALID_PARAMETER, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(int)
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s '%v' is not an integer", INVALID_PARAMETER, paramParentName, paramName, parameter)
	}
	if hasMin {
		if hasMax {
			if parameterValue < minValue || parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  out of range [%d, %d]", INVALID_PARAMETER, paramParentName, paramName, minValue, maxValue)
			}
		} else {
			if parameterValue < minValue {
				return 0, fmt.Errorf("%s: %s.%s  must be >= %d", INVALID_PARAMETER, paramParentName, paramName, minValue)
			}
		}
	} else {
		if hasMax {
			if parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  must be < %d", INVALID_PARAMETER, paramParentName, paramName, maxValue)
			}
		}
	}

	return parameterValue, nil
}

func loadParameterAsFloat(parameterMap map[interface{}]interface{}, paramParentName string, paramName string, hasMin bool, minValue float64, hasMax bool, maxValue float64) (float64, error) {
	parameter, ok := parameterMap[paramName]
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s not found", INVALID_PARAMETER, paramParentName, paramName)
	}
	parameterValue, ok := parameter.(float64)
	if !ok {
		return 0, fmt.Errorf("%s: %s.%s '%v' is not a float", INVALID_PARAMETER, paramParentName, paramName, parameter)
	}
	if hasMin {
		if hasMax {
			if parameterValue < minValue || parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  out of range [%f, %f]", INVALID_PARAMETER, paramParentName, paramName, minValue, maxValue)
			}
		} else {
			if parameterValue < minValue {
				return 0, fmt.Errorf("%s: %s.%s  must be >= %f", INVALID_PARAMETER, paramParentName, paramName, minValue)
			}
		}
	} else {
		if hasMax {
			if parameterValue > maxValue {
				return 0, fmt.Errorf("%s: %s.%s  must be < %f", INVALID_PARAMETER, paramParentName, paramName, maxValue)
			}
		}
	}

	return parameterValue, nil
}
