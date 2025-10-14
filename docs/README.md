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
 - Via Python
   - créer un environnement virtuel : 
     - unix/macOS `python3 -m venv .venv`
     - windows `py -m venv .venv`
   - activer l'environnement virtuel :
     - unix/macOS : `source .venv/bin/activate`
     - windows : `.venv\Scripts\activate`
   - installer les paquets requis : `pip install -r requirements.txt`
   - démarrer le serveur : `mkdocs serve`
   - visiter : [http://localhost:8000/](http://localhost:8000/)
 - Via [Docker](https://squidfunk.github.io/mkdocs-material/getting-started/ "des méthodes et documentations plus récentes sont probablement disponibles ici") (au cas où vous n'êtes pas un grand fan de Python)
   - construire une image depuis le [Dockerfile](Dockerfile "vous n'aurez besoin de le faire qu'une seule fois") fourni `docker build -t tic4eebus/doc .`
   - exécuter `docker run --rm -p 8000:8000 -v ${PWD}:/docs tic4eebus/doc`
 
2- visiter [http://localhost:8000/](http://localhost:8000/)

# Documentation utile
[https://www.mkdocs.org/getting-started/](https://www.mkdocs.org/getting-started/)  
[https://squidfunk.github.io/mkdocs-material/getting-started/](https://squidfunk.github.io/mkdocs-material/getting-started/)