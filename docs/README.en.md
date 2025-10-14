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
 - Via Python
   - create a virtual environment: 
     - unix/macOS `python3 -m venv .venv`
     - windows `py -m venv .venv`
   - activate the virtual environment:
     - unix/macOS: `source .venv/bin/activate`
     - windows: `.venv\Scripts\activate`
   - install required packages: `pip install -r requirements.txt`
   - start the server: `mkdocs serve`
   - visit: [http://localhost:8000/](http://localhost:8000/)
 - Via [Docker](https://squidfunk.github.io/mkdocs-material/getting-started/ "more recent methods and docs may very well be available there") (in case you are not a big python fan)
   - build an image from de provided [Dockerfile](Dockerfile "you'll only need to do this once") `docker build -t tic4eebus/doc .`
   - run `docker run --rm -p 8000:8000 -v ${PWD}:/docs tic4eebus/doc`
 
2- visit  [http://localhost:8000/](http://localhost:8000/)

# Useful docs
[https://www.mkdocs.org/getting-started/](https://www.mkdocs.org/getting-started/)  
[https://squidfunk.github.io/mkdocs-material/getting-started/](https://squidfunk.github.io/mkdocs-material/getting-started/)