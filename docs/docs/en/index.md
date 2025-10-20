<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

# Overview

## Introduction

The **tic4eebus** application is used as a energy management system (referred as EnergyGuard in EEBUS documentation) using Linky customer interface (known as Télé Information Client or TIC)
as metering source to control electrical vehicle charge through a wallbox (i.e the EEBUS interface).

![Overview diagram](../img/overview_diagram.png)

## References

### Linky Meter Integration

The integration of a Linky meter into the EEBUS ecosystem is described in a document available [here](../ref/Integration-of-the-Linky-Smart-Meter-within-EEBUS-ecosystem.pdf).

### OPEV Use Case

The OPEV (Overload Protection by EV Charging Current Curtailment) use case from the EEBUS standard, designed to prevent
the circuit breaker upstream of the electrical installation from tripping, can be downloaded [here](../ref/EEBus_UC_TS_OverloadProtectionByEVChargingCurrentCurtailment_V1.0.1b.pdf).

## Purpose

The purpose of this documentation is to provide the necessary information for users, developers, and contributors to the project:

- [Installation](installation.md)
- [Configuration](configuration.md)
- [Software Architecture](architecture.md)
- [Data Model](data_model.md)