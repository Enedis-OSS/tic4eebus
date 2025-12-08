<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

# tic4eebus
*Cas d'utilisation OPEV EEBUS OPEV avec la Télé Information Client (TIC) d'un compteur Linky pour gérer la charge d'un véhicule électrique en fonction de l'énergie disponible (PCOUP)*

[![statut REUSE](https://api.reuse.software/badge/git.fsfe.org/reuse/api)](https://api.reuse.software/info/git.fsfe.org/reuse/api)

[🇫🇷 Français](README.md) | [🇺🇸 English](README.en.md)

## Sommaire

* [Introduction](#introduction)
* [Installation](#installation)
* [Documentation](#documentation)
* [Contribuer](#contrib)
* [Support](#support)
* [Contributeurs](#contributors)

## <a name="introduction"></a> Introduction

L'application **tic4eebus** est utilisée comme gestionnaire d'energie (EnergyGuard dans la documentation EEBUS) utilisant la Télé Information Client (TIC)
du compteur Linky comme source de métrologie pour piloter la charge d'un véhicule électrique via une borne de recharge (interface EEBUS).

## <a name="installation"></a> Installation

### Prérequis

Pour générer l'application, vous avez besoin d'installer :

- [Go](https://fr.education-wiki.com/2324783-how-to-install-go)

Pour le bon fonctionnement de l'application, vous avez besoin d'installer :

- [TIC2WebSocket](https://github.com/Enedis-OSS/TIC2WebSocket)

### Générer le fichier exécutable

Lorsque vous êtes à la racine du projet, tapez simplement la commande suivante :

```bash 
go build -o tic4eebus cmd/tic4eebus/main.go
```

En conséquence, le fichier exécutable *tic4eebus* est créé.

### Démarrage de l'Application

Pour démarrer l'application **tic4eebus**, exécutez la commande :

```bash 
./tic4eebus
```
*Remarque: Pour le bon fonctionnement de l'application **tic4eebus** l'application **TIC2WebSocket** doit être démarrée.*

### Aide

Ajoutez l'option `--help` pour obtenir des informations de base sur l'utilisation de l'application.
```bash 
./tic4eebus.sh --help
```

### Version

Ajoutez l'option `--version` pour afficher la version de l'application.
```bash 
./tic4eebus --version
```

## <a name="documentation"></a> Documentation

Pour plus d'informations sur le fonctionnement de tic4eebus, consultez la [documentation officielle](https://enedis-oss.github.io/tic4eebus/).

## <a name="contrib"></a> Contribuer ?

![PRs Bienvenues](https://img.shields.io/badge/PRs-welcome-brightgreen.svg?style=flat-square)

Vous n'avez pas besoin d'être développeur pour contribuer, ni de faire beaucoup, vous pouvez simplement :
* Améliorer la documentation,
* Corriger une faute d'orthographe,
* [Signaler un bug](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose)
* [Demander une fonctionnalité](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose)
* [Nous donner des conseils ou des idées](https://github.com/Enedis-OSS/tic4eebus/issues/new/choose),
* etc.

Pour vous aider à démarrer, nous vous invitons à lire [Contributing](CONTRIBUTING.fr.md), qui vous donne les règles et conventions de code à respecter

Pour contribuer à cette documentation (README, CONTRIBUTING, etc.), nous nous conformons à la [CommonMark Spec](https://spec.commonmark.org/)

## <a name="contributors"></a> Contributeurs

Contributeurs principaux :
* **Jehan BOUSCH** (<jehan-externe.bousch@enedis.fr>)

Nous nous efforçons de fournir un environnement bienveillant et de soutenir toute [contribution](#contrib).
