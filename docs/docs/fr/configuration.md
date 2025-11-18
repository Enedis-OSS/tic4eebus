<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

# Configuration

## Introduction

La configuration de l'application **tic4eebus** est organisée en 7 parties :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *OverloadProtection* | Paramètres de protection des surcharges | Structure | Valeurs des [protection des surcharges](#protection-des-surcharges) |
| *Vehicle* | Paramètres du véhicule électrique | Structure | Valeurs du [véhicule électrique](#vehicule-electrique) |
| *Wallbox* | Paramètres du borne de recharge | Structure | Valeurs de la [borne de recharge](#borne-de-recharge) |
| *Log* | Paramètres de journalisation | Structure | Valeurs des [paramètres de journalisation](#journalisation) |
| *DataModel* | Paramètres du modèle de données | Structure | Valeurs du [modèle de données](#modele-de-donnees) |
| *TeleInformationClient* | Paramètres de la télé information client (TIC) | Structure | Valeurs de la [TIC](#tele-information-client-tic) |
| *Eebus* | Paramètres de protocole EEBUS | Structure | Valeurs du [protocole EEBUS](#protocole-eebus) |

Chaque partie ci-dessus est stockée dans le fichier de configuration au format YAML.

Le fichier de configuration est chargé au démarrage de l'application avec le chemin par défaut "*examples/config.yaml*".

Ce chemin du fichier de configuration est modifiable par la ligne de commande de l'application avec l'argument "*configFilePath*".

## Description

### Protection des surcharges

La configuration de la protection des surcharges comporte les paramètres suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *Enable* | Activation de l'algorithme | Booléen | - **true** si l'algorithme est actif <br>- **false** sinon |
| *RunningPeriodInSeconds* | Période d'exécution de l'algorithme en secondes | Entier | De 0 à 2^31 - 1 |
| *CurrentLimit* | Paramètres liés à la limitation de courant | Structure | Valeurs de la [limitation de courant](#limitation-de-courant) |

#### Limitation de courant

La configuration des paramètres de limitation de courant de la protection des surcharges comporte les champ suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *ValueInAmps* | Limitation du courant de charge du VE en Ampère | Décimal | De 0.0 à 1.79 × 10^308 |
| *LockDelayInSeconds* | Durée de verrouillage de la limitation en secondes | Décimal | De 0 à 1.79 × 10^308 |

<u>Remarques:</u>

- La limitation du courant de charge du VE (*ValueInAmps*) est utilisé uniquement si l'algorithme de protection des surcharges est inactif (*Enable*=**false**)
- La durée de verrouiullage de la limitation (*LockDelayInSeconds*) n'est utilisé que si l'algorithme de protection des surcharges est actif (*Enable*=**true**)

### Véhicule électrique

La configuration des paramètres liés au véhicule électrique comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *UpdateDataPeriodInSeconds* | Période de mise à jour des données de VE en secondes | Entier | De 0 à 2^31 - 1 <br> 0 signifie pas de mise à jour périodique |
| *DataPersistent* | Indicateur de persistence des données du VE | Booléen | - **true** si les données sont persistantes <br>- **false** sinon |

### Borne de recharge

La configuration des paramètres liés à la borne de recharge comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *UpdateDataPeriodInSeconds* | Période de mise à jour des données de la borne en secondes | Entier | De 0 à 2^31 - 1 <br> 0 signifie pas de mise à jour périodique |
| *DataPersistent* | Indicateur de persistence des données de la borne | Booléen | - **true** si les données sont persistantes <br>- **false** sinon |

### Journalisation

La configuration des paramètres de journalisation comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *Level* | Niveau de journalisation utilisé | Énumération | Valeurs de [niveau de journalisation](#niveau-de-journalisation)  |
| *FilePath* | Chemin d'accès du fichier de journalisation | Chaine de caractères | Exemple : "var/log/tic4eebus.log" |
| *Rotation* | Paramètres de rotation des fichiers de journalisation | Structure | Valeurs de [rotation de fichier](#rotation-des-fichiers) |

### Modèle de données

La configuration des paramètres du modèle de données comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *Csv* | Paramètres du fichier CSV | Structure | Valeurs du [fichier CSV](#fichier-csv)  |
| *InfluxDb* | Paramètres de la base de données InfluxDB | Structure | Valeurs de [la base de données InfluxDB](#base-de-donnees-influxdb)|

<u>Remarques:</u>

Les champs *Csv* et *InfluxDb* sont optionnels.
Si le champs *Csv* n'existe pas, il n'y a pas de fichier CSV.
Si le champs *InfluxDb* n'existe pas, la base de données InfluxDB n'est pas alimentée.

#### Fichier CSV

La configuration des paramètres du fichier CSV comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *FilePath* | Chemin d'accès du fichier CSV | Chaine de caractères | Exemple : "var/data/DataModel.csv" |
| *Rotation* | Paramètres de rotation des fichiers CSV | Structure | Valeurs de [rotation de fichier](#rotation-des-fichiers) |

#### Base de données InfluxDB

La configuration des paramètres de la base de données InfluxDB comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *Bucket* | Nom de la base (appelée bucket) InfluxDB | Chaine de caractères | Exemple : "demo-bucket" |
| *Org* | Nom de l'organisation associée à la base et aux tables (appelées measurment) | Chaine de caractères | Exemple : "demo-org" |
| *Token* | Clef d'authentification utilisée pour la dconnexion à la base | Chaine de caractères | Exemple: "He_nwgoqXu4XggIzaiUWFHALKnS5JskdTLzGlSYeJMNkjCD-pyR6Yc5Hvl8NWj5Qo5C80mLvuJAS1IuqOcq4GQZZ" |
| *IpAddress* | Adresse IP du serveur hébergeant la base de données InfluxDB | Chaine de caractères | Exemple : "127.0.0.1" |
| *TcpPort* | Port TCP du serveur hébergeant la base de données InfluxDB | Entier | De 1 à 65535 |

### Télé information client (TIC)

La configuration des paramètres de la télé information client comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *Tic2Websocket* | Paramètres du client TIC2WebSocket | Structure | Valeurs de [Tic2WebSocket](#tic2websocket) |
| *TicIdentifier* | Paramètres d'identification du compteur Linky | Structure | Valeurs de [TicIdentifier](#ticidentifier) |

#### Tic2Websocket

La configuration des paramètres du client Tic2Websocket comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *IpAddress* | Adresse IP du serveur hébergeant l'application Tic2Websocket | Chaine de caractères | Exemple : "127.0.0.1" |
| *TcpPort* | Port TCP du serveur hébergeant l'application Tic2Websocket | Entier | De 1 à 65535 |

#### TicIdentifier

La configuration des paramètres d'identification du compteur Linky comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *SerialNumber* | Numéro de série du compteur Linky | Chaine de 12 caractères | Exemple: "041976216986" |

<u>Remarque:</u>

Si le champs *SerialNumber* est vide alors l'application utilisera le premier numéro de série fournit pas l'application *Tic2Websocket*.

### Protocole EEBUS

La configuration des paramètres du protocole EEBUS comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *ServerPort* | Port TCP du serveur EEBUS de la borne de recharge | Entier | De 1 à 65535 |
| *RemoteSki* | Clef d'identification du serveur EEBUS de la borne de recharge | Chaine de 40 caractères hexadécimaux | Exemple : "50abfe7714d034b8b15e488b91831047657b9ff2" |
| *CertificateFilePath* | Chemin du certificat de sécurité du gestionnaire d'énergie | Chaine de caractères | Exemple : "examples/energy-guard.cert" |
| *PrivateKeyFilePath* | Clef privée de chiffrement du gestionnaire d'énergie | Chaine de caractères | Exemple : "examples/energy-guard.key" |
| *VendorCode* | Code d'identification de l'entrepise du gestionnaire d'énergie conforme [IANA PEN](https://www.iana.org/assignments/enterprise-numbers/)| Chaine de caractères | Exemple : "i:54076" |
| *DeviceBrand* | Nom de la marque du gestionnaire d'énergie | Chaine de caractères | Exemple : "Enedis" |
| *DeviceModel* | Nom du modèle du gestionnaire d'énergie | Chaine de caractères | Exemple : "PAC" |
| *SerialNumber* | Numéro de série du gestionnaire d'énergie | Chaine de caractères | Exemple : "12345678" |
| *HeartbeatTimeoutInSeconds* | Délai max imparti pour l'envoi de [heartbeat EEBUS](../var/references/EEBus_UC_TS_OverloadProtectionByEVChargingCurrentCurtailment_V1.0.1b.pdf#page=34) à la borne de recharge | Entier | De 0 à 2^32 -1 |

<u>Remarque:</u>

Si le paramètre *HeartbeatTimeoutInSeconds* est configuré avec une valeur supérieur à 4 secondes, la borne de recharge, conformément au [scénario 2 du cas d'utilisation OPEV](../var/references/EEBus_UC_TS_OverloadProtectionByEVChargingCurrentCurtailment_V1.0.1b.pdf#page=45), considère que le gestionnaire d'énergie n'est pas disponible et configure la limitation de charge à sa valeur minimale.

## Types

### Niveau de journalisation

Le niveau de journalisation permet de définir le seuil minimum des informations à conserver dans les journaux de bord.

Il peut prendre les valeurs suivantes classées ordre décroissant de gravité :

|Valeur de l'énumération|Description de l'énumération|
|-----------------------|----------------------------|
| *panic* |  Niveau de gravité le plus élevé. Effectue la journalisation puis appelle [panic()](https://go.dev/blog/defer-panic-and-recover). |
| *fatal* | Niveau de gravité critique. Effectue la journalisation puis quitte le programme. |
| *error* | Niveau utilisé pour les erreurs qui doivent absolument être signalées. |
| *warn* | Niveau pour les informations non critiques mais qui méritent une attention particulière |
| *info* | Niveau pour les informations générales décrivant le fonctionnement courant de l’application. |
| *debug* | Niveau généralement activé uniquement pour le débogage. Journalisation très détaillée. |
| *trace* | Niveau qui désigne des événements informationnels encore plus détaillés que *debug* |

### Rotation des fichiers

La rotation des fichiers comporte les champs suivants :

|Nom|Description|Format|Valeur|
|---|-----------|------|------|
| *PeriodInHours* | Durée d'une période de rotation en nombre d'heures | Entier | De 0 à 2^31 - 1 |
| *PeriodCount* | Nombre de rotation max | Entier | De 0 à 2^31 - 1 |
| *PeriodPattern* | Motif utilisé pour le nom du fichier à chaque nouvelle rotation | Chaine de caractères |  |

## Exemple

Voici un exemple complet du [fichier de configuration](../var/examples/config.yaml) par défaut au format YAML :

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