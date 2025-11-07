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
   - run `docker-compose up --build` to start containers
   - run `docker-compose down` to stop containers
 
2- visit  [http://localhost:8000/](http://localhost:8000/)

# Useful docs
[https://www.mkdocs.org/getting-started/](https://www.mkdocs.org/getting-started/)  
[https://squidfunk.github.io/mkdocs-material/getting-started/](https://squidfunk.github.io/mkdocs-material/getting-started/)