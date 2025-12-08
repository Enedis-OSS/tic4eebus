<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ SPDX-FileContributor: Mathieu SABARTHES
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->
# Changelog

[🇫🇷 Français](CHANGELOG.fr.md) | [🇺🇸 English](CHANGELOG.md)

## [v1.1.1](https://github.com/Enedis-OSS/tic4eebus/tree/v1.1.1)
### 📖 Documentation:
* Application configuration
* Application software architecture
* Application data model
* Application source code (Go doc)
### 🐛 Fixed bugs:
* CSV file containing the data model being erased at each application restart

## [v1.1.0](https://github.com/Enedis-OSS/tic4eebus/tree/v1.1.0)
### ✨ New features:
* Write data model to InfluxDB for Grafana visualization
### 🔧 Technical enhancements:
* Optional configuration for data model (CSV file, InfluxDB)

## [v1.0.0](https://github.com/Enedis-OSS/tic4eebus/tree/v1.0.0)
### ✨ New features:
* EEBUS OPEV use case (scenario 1) : Energy Guard curtails charging current of EV
* EEBUS OPEV use case (scenario 2) : EV checks Energy Guard availability
* EEBUS OPEV use case (scenario 3) : Energy Guard sends error state
* Configuration file with settings for Overload protection, VE, log, data model, TIC Linky meter, EEBUS
* Log file with rotation
* Command line parser with help, version and configuration file path
* Data model exported as CSV file with rotation