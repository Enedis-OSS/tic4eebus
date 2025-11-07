<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ SPDX-FileContributor: Mathieu SABARTHES
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->
# Journal des modifications

[🇫🇷 Français](CHANGELOG.fr.md) | [🇺🇸 English](CHANGELOG.md)

## [v1.1.0](https://github.com/Enedis-OSS/tic4eebus/tree/v1.1.0)
### ✨ Nouvelles fonctionnalités:
* Écriture du modèle de données vers InfluxDB pour la visualisation avec Grafana
### 🔧 Améliorations techniques:
* Configuration optionnelle pour le modèle de données (fichier CSV, InfluxDB)

## [v1.0.0](https://github.com/Enedis-OSS/tic4eebus/tree/v1.0.0)
### ✨ Nouvelles fonctionnalités :
* Cas d'utilisation OPEV EEBUS (scénario 1) : Le gestionnaire d'énergie limite le courant de charge du VE
* Cas d'utilisation OPEV EEBUS (scénario 2) : Le VE vérifie la disponibilité du gestionnaire d'énergie
* Cas d'utilisation OPEV EEBUS (scénario 3) : Le gestionnaire d'énergie envoie un état d'erreur
* Fichier de configuration avec paramètres pour la protection contre les surcharges, VE, logs, modèle de données, télé-information client compteur Linky, EEBUS
* Fichier de log avec rotation
* Interpréteur de ligne de commande avec aide, version et chemin du fichier de configuration
* Modèle de données exporté en fichier CSV avec rotation