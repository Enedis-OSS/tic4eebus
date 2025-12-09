<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~ 
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~ 
  ~ SPDX-License-Identifier: Apache-2.0
-->

[🇫🇷 Français](README.fr.md) | [🇺🇸 English](README.md)

# Developer setup

1- Run in the project docs root folder: 
 - Via [Docker](https://squidfunk.github.io/mkdocs-material/getting-started/ "more recent methods and docs may very well be available there")
   - build an image from de provided [Dockerfile](Dockerfile "you'll only need to do this once") `docker build -t tic4eebus/doc .`
   - run `docker-compose up --build` to start containers (PlantUML server and MkDoc server)
   - run `docker-compose down` to stop containers (PlantUML server and MkDoc server)
 
2- Visit  [http://localhost:8000/](http://localhost:8000/)

3- Deploy [official documentation](https://enedis-oss.github.io/tic4eebus/)
 - Via Docker
   - run `docker-compose up -d plantuml` to start PlantUML server
   - run `docker-compose run --rm mkdocs gh-deploy` to start MkDoc server, build pages and deploy on GitHub

# Useful docs
[https://www.mkdocs.org/getting-started/](https://www.mkdocs.org/getting-started/)  
[https://squidfunk.github.io/mkdocs-material/getting-started/](https://squidfunk.github.io/mkdocs-material/getting-started/)