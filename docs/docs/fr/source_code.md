<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~
  ~ SPDX-License-Identifier: Apache-2.0
-->

# Référence

La documentation technique complète du code source Go est disponible sur [pkg.go.dev](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus)

## Principaux composants

- **[tic4eebus](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/cmd/tic4eebus)** - Application
- **[config](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/config)** - Configuration
- **[ems](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/ems)** - Gestionnaire d'énergie
- **[ems/data](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/ems/data)** - Modèle de données
- **[linkymeter](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/linkymeter)** - Compteur Linky
- **[evse](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/evse)** - Borne de recharge et véhicule électrique

## Documentation locale

Pour générer la documentation localement :
```bash
# Installer godoc
go install golang.org/x/tools/cmd/godoc@latest

# Lancer le serveur local
godoc -http=:6060

# Ouvrir dans le navigateur
open http://localhost:6060/pkg/github.com/Enedis-OSS/tic4eebus/
```