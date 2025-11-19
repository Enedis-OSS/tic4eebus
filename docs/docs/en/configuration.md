<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

# Configuration

## Introduction

The **tic4eebus** application configuration is organized into 7 sections:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *OverloadProtection* | Overload protection parameters | Structure | [Overload protection](#overload-protection) values |
| *Vehicle* | Electric vehicle parameters | Structure | [Electric vehicle](#electric-vehicle) values |
| *Wallbox* | Charging station parameters | Structure | [Charging station](#charging-station) values |
| *Log* | Logging parameters | Structure | [Logging parameters](#logging) values |
| *DataModel* | Data model parameters | Structure | [Data model](#data-model) values |
| *TeleInformationClient* | Tele-information client (TIC) parameters | Structure | [TIC](#tele-information-client-tic) values |
| *Eebus* | EEBUS protocol parameters | Structure | [EEBUS protocol](#eebus-protocol) values |

Each section above is stored in the configuration file in YAML format.

The configuration file is loaded at application startup with the default path "*examples/config.yaml*".

This configuration file path can be modified via the application command line with the "*configFilePath*" argument.

## Description

### Overload Protection

The overload protection configuration includes the following parameters:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *Enable* | Algorithm activation | Boolean | - **true** if the algorithm is active <br>- **false** otherwise |
| *RunningPeriodInSeconds* | Algorithm execution period in seconds | Integer | From 0 to 2^31 - 1 |
| *CurrentLimit* | Current limitation parameters | Structure | [Current limitation](#current-limitation) values |

#### Current Limitation

The current limitation parameters configuration for overload protection includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *ValueInAmps* | EV charging current limit in Amperes | Decimal | From 0.0 to 1.79 × 10^308 |
| *LockDelayInSeconds* | Limitation lock duration in seconds | Decimal | From 0 to 1.79 × 10^308 |

<u>Notes:</u>

- The EV charging current limit (*ValueInAmps*) is only used if the overload protection algorithm is inactive (*Enable*=**false**)
- The limitation lock duration (*LockDelayInSeconds*) is only used if the overload protection algorithm is active (*Enable*=**true**)

### Electric Vehicle

The electric vehicle parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *UpdateDataPeriodInSeconds* | EV data update period in seconds | Integer | From 0 to 2^31 - 1 <br> 0 means no periodic update |
| *DataPersistent* | EV data persistence indicator | Boolean | - **true** if data is persistent <br>- **false** otherwise |

### Charging Station

The charging station parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *UpdateDataPeriodInSeconds* | Station data update period in seconds | Integer | From 0 to 2^31 - 1 <br> 0 means no periodic update |
| *DataPersistent* | Station data persistence indicator | Boolean | - **true** if data is persistent <br>- **false** otherwise |

### Logging

The logging parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *Level* | Logging level used | Enumeration | [Logging level](#logging-level) values |
| *FilePath* | Logging file path | String | Example: "var/log/tic4eebus.log" |
| *Rotation* | Logging file rotation parameters | Structure | [File rotation](#file-rotation) values |

### Data Model

The data model parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *Csv* | CSV file parameters | Structure | [CSV file](#csv-file) values |
| *InfluxDb* | InfluxDB database parameters | Structure | [InfluxDB database](#influxdb-database) values |

<u>Notes:</u>

The *Csv* and *InfluxDb* fields are optional.
If the *Csv* field does not exist, there is no CSV file.
If the *InfluxDb* field does not exist, the InfluxDB database is not populated.

#### CSV File

The CSV file parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *FilePath* | CSV file path | String | Example: "var/data/DataModel.csv" |
| *Rotation* | CSV file rotation parameters | Structure | [File rotation](#file-rotation) values |

#### InfluxDB Database

The InfluxDB database parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *Bucket* | InfluxDB database name (called bucket) | String | Example: "demo-bucket" |
| *Org* | Organization name associated with the database and tables (called measurements) | String | Example: "demo-org" |
| *Token* | Authentication key used for database connection | String | Example: "He_nwgoqXu4XggIzaiUWFHALKnS5JskdTLzGlSYeJMNkjCD-pyR6Yc5Hvl8NWj5Qo5C80mLvuJAS1IuqOcq4GQZZ" |
| *IpAddress* | IP address of the server hosting the InfluxDB database | String | Example: "127.0.0.1" |
| *TcpPort* | TCP port of the server hosting the InfluxDB database | Integer | From 1 to 65535 |

### Tele-Information Client (TIC)

The tele-information client parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *Tic2Websocket* | TIC2WebSocket client parameters | Structure | [Tic2WebSocket](#tic2websocket) values |
| *TicIdentifier* | Linky meter identification parameters | Structure | [TicIdentifier](#ticidentifier) values |

#### Tic2Websocket

The Tic2Websocket client parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *IpAddress* | IP address of the server hosting the Tic2Websocket application | String | Example: "127.0.0.1" |
| *TcpPort* | TCP port of the server hosting the Tic2Websocket application | Integer | From 1 to 65535 |

#### TicIdentifier

The Linky meter identification parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *SerialNumber* | Linky meter serial number | 12-character string | Example: "041976216986" |

<u>Note:</u>

If the *SerialNumber* field is empty, the application will use the first serial number provided by the *Tic2Websocket* application.

### EEBUS Protocol

The EEBUS protocol parameters configuration includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *ServerPort* | TCP port of the charging station EEBUS server | Integer | From 1 to 65535 |
| *RemoteSki* | Identification key of the charging station EEBUS server | 40 hexadecimal character string | Example: "50abfe7714d034b8b15e488b91831047657b9ff2" |
| *CertificateFilePath* | Security certificate path for the energy management system | String | Example: "examples/energy-guard.cert" |
| *PrivateKeyFilePath* | Energy management system encryption private key | String | Example: "examples/energy-guard.key" |
| *VendorCode* | Energy management company identification code compliant with [IANA PEN](https://www.iana.org/assignments/enterprise-numbers/) | String | Example: "i:54076" |
| *DeviceBrand* | Energy management system brand name | String | Example: "Enedis" |
| *DeviceModel* | Energy management system model name | String | Example: "PAC" |
| *SerialNumber* | Energy management system serial number | String | Example: "12345678" |
| *HeartbeatTimeoutInSeconds* | Maximum delay allowed for sending [EEBUS heartbeat](../var/references/EEBus_UC_TS_OverloadProtectionByEVChargingCurrentCurtailment_V1.0.1b.pdf#page=34) to the charging station | Integer | From 0 to 2^32 -1 |

<u>Note:</u>

If the *HeartbeatTimeoutInSeconds* parameter is configured with a value greater than 4 seconds, the charging station, in accordance with [OPEV use case scenario 2](../var/references/EEBus_UC_TS_OverloadProtectionByEVChargingCurrentCurtailment_V1.0.1b.pdf#page=45), considers that the energy management system is not available and sets the charge limitation to its minimum value.

## Types

### Logging Level

The logging level defines the minimum threshold for information to be kept in logs.

It can take the following values, sorted in descending order of severity:

|Enumeration Value|Enumeration Description|
|-----------------------|----------------------------|
| *panic* | Highest level of severity. Logs and then call [panic()](https://go.dev/blog/defer-panic-and-recover). |
| *fatal* | Critical severity level. Logs and then exits the program. |
| *error* | Level used for errors that should definitely be noted. |
| *warn* | Level for non-critical information that deserve eyes |
| *info* | Level for general information about what's going on inside the application. |
| *debug* | Level only enabled when debugging. Very verbose logging. |
| *trace* | Level designating finer-grained informational events than the *debug* |

### File Rotation

File rotation includes the following fields:

|Name|Description|Format|Value|
|---|-----------|------|------|
| *PeriodInHours* | Duration of a rotation period in hours | Integer | From 0 to 2^31 - 1 |
| *PeriodCount* | Maximum number of rotations | Integer | From 0 to 2^31 - 1 |
| *PeriodPattern* | Pattern used for the file name at each new rotation | String | |

## Example

Here is a complete example of the default [configuration file](../var/examples/config.yaml) in YAML format:

```yaml
# Overload protection configuration
OverloadProtection:
  Enable : true
  RunningPeriodInSeconds : 1
  CurrentLimit :
    ValueInAmps : 16.0 # Used only if Enable is false
    LockDelayInSeconds : 10.0 # Used only if Enable is true

# Vehicle configuration
Vehicle:
  UpdateDataPeriodInSeconds : 5 # Unused if 0
  DataPersistent: false

# Wallbox configuration
Wallbox:
  UpdateDataPeriodInSeconds : 5 # Unused if 0
  DataPersistent: false

# Log configuration
Log:
  Level : "trace" # Available levels : panic,fatal,error,warn,info,debug,trace
  FilePath : "var/log/tic4eebus.log"
  Rotation:
    PeriodInHours : 24 # Each day a new file is created
    PeriodCount : 15 # 15 files maximum are created (15 days of logs)
    PeriodPattern: "-%Y-%m-%d" # Each log file name contains the year-month-day of creation

# Data model configuration
DataModel:
  # Csv file configuration (optional)
  Csv: # If provided all fields must be provided
    FilePath : "var/data/EnergyGuardDataModel.csv"
    Rotation:
      PeriodInHours : 24 # Each day a new file is created
      PeriodCount : 7 # 7 files maximum are created (7 days of data model)
      PeriodPattern: "-%Y-%m-%d" # Each csv file name contains the year-month-day of creation

  # InfluxDB configuration (optional)
  InfluxDb: # If provided all fields must be provided
    Bucket : "demo-bucket"
    Org : "demo-org"
    Token : "${TIC4EEBUS_INFLUXDB_TOKEN}" # Example: "He_nwgoqXu4XggIzaiUWFHALKnS5JskdTLzGlSYeJMNkjCD-pyR6Yc5Hvl8NWj5Qo5C80mLvuJAS1IuqOcq4GQZZ"
    IpAddress : "127.0.0.1"
    TcpPort : 8086

# Tele information client (TIC) configuration
TeleInformationClient:
  Tic2Websocket:
    IpAddress : "127.0.0.1"
    TcpPort : 19584
  TicIdentifier:
    SerialNumber : "" # Meter serial number should only be specified if TIC2Websocket handles multiple modems

# EEBUS configuration
Eebus:
  ServerPort : 4817
  RemoteSki : "${TIC4EEBUS_REMOTE_SKI}" # Example: "50abfe7714d034b8b15e488b91831047657b9ff2"
  CertificateFilePath : "examples/energy-guard.cert"
  PrivateKeyFilePath : "examples/energy-guard.key"
  VendorCode : "i:54076"
  DeviceBrand : "Enedis"
  DeviceModel : "PAC"
  SerialNumber : "12345678"
  HeartbeatTimeoutInSeconds : 2
```