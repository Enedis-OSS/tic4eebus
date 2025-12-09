<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

# tic4eebus
*EEBUS OPEV use case with Linky customer interface (TIC) aiming to handle Electrical Vehicle charge according to available energy (PCOUP)*

[![REUSE status](https://api.reuse.software/badge/git.fsfe.org/reuse/api)](https://api.reuse.software/info/git.fsfe.org/reuse/api)

[🇫🇷 Français](README.md) | [🇺🇸 English](README.en.md)

## Summary

* [Introduction](#introduction)
* [Installation](#installation)
* [Documentation](#documentation)
* [Contributing](#contrib)
* [Support](#support)
* [Contributors](#contributors)

## <a name="introduction"></a> Introduction

The **tic4eebus** application is used as a energy management system (referred as EnergyGuard in EEBUS documentation) using Linky customer interface (known as Télé Information Client or TIC)
as metering source to control electrical vehicle charge through a wallbox (i.e the EEBUS interface).

## <a name="installation"></a> Installation

### Prerequisites

To generate the application, you need to install :

- [Go](https://go.dev/doc/install)

For application proper functioning, you need to install :

- [TIC2WebSocket](https://github.com/Enedis-OSS/TIC2WebSocket)

### Build executable file

When you are at the project root, simply type the following command:

```bash 
go build -o tic4eebus cmd/tic4eebus/main.go
```

As a result, the *tic4eebus* executable file is created.

### Starting the Application

To start the **tic4eebus** application, execute the commnand :

```bash 
./tic4eebus
```
*Note: For application **tic4eebus** proper functioning, application **TIC2WebSocket** must be started.*

### Help

Add `--help` option to get basic information on how to use the application.
```bash 
./tic4eebus --help
```

### Version

Add `--version` option to display the application version.
```bash 
./tic4eebus --version
```

## <a name="documentation"></a> Documentation

Get the [official documentation](https://enedis-oss.github.io/tic4eebus/) for more information about how tic4eebus works.

## <a name="contrib"></a> Contributing ?

![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)

You don't need to be a developer to contribute, nor do much, you can simply:
* Enhance documentation,
* Correct a spelling,
* [Report a bug](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose)
* [Ask a feature](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose)
* [Give us advices or ideas](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose),
* etc.

To help you start, we invite you to read [Contributing](CONTRIBUTING.md), which gives you rules and code conventions to respect

To contribute to this documentation (README, CONTRIBUTING, etc.), we conform to the [CommonMark Spec](https://spec.commonmark.org/)

## <a name="contributors"></a> Contributors

Core contributors :
* **Jehan BOUSCH** (<jehan-externe.bousch@enedis.fr>)

We strive to provide a benevolent environment and support any [contribution](#contrib).
