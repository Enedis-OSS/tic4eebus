<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

[🇫🇷 Français](README.md) | [🇺🇸 English](README.en.md)

# Configuration pour les développeurs

1- Exécuter depuis le dossier docs à la racine du projet : 
 - Via [Docker](https://squidfunk.github.io/mkdocs-material/getting-started/ "des méthodes et documentations plus récentes sont probablement disponibles ici")
   - exécuter `docker-compose up --build` pour démarrer les conteneurs (serveur PlantUML et serveur MkDoc)
   - exécuter `docker-compose down` pour arrêter les conteneurs (serveur PlantUML et serveur MkDoc)
 
2- Visiter [http://localhost:8000/](http://localhost:8000/)

3- Déployer la [documentation officielle](https://enedis-oss.github.io/tic4eebus/)
 - Via Docker
   - exécuter `docker-compose up -d plantuml` pour démarrer le serveur PlantUML
   - exécuter `docker-compose run --rm mkdocs gh-deploy` pour démarrer le serveur MkDoc, générer les pages et deployer sur GitHub

# Documentation utile
[https://www.mkdocs.org/getting-started/](https://www.mkdocs.org/getting-started/)  
[https://squidfunk.github.io/mkdocs-material/getting-started/](https://squidfunk.github.io/mkdocs-material/getting-started/)