<!--
  ~ Copyright (C) 2025 Enedis Smarties team <dt-dsi-nexus-lab-smarties@enedis.fr>
  ~
  ~ SPDX-FileContributor: Jehan BOUSCH
  ~
  ~ SPDX-License-Identifier: Apache-2.0
-->

# Reference

The complete technical documentation of the Go source code is available on [pkg.go.dev](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus)

## Main components

- **[tic4eebus](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/cmd/tic4eebus)** - Main component
- **[config](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/config)** - Configuration
- **[ems](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/ems)** - Energy management system
- **[ems/data](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/ems/data)** - Data model
- **[linkymeter](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/linkymeter)** - Linky meter
- **[evse](https://pkg.go.dev/github.com/Enedis-OSS/tic4eebus/evse)** - Charging station and electric vehicle

## Local documentation

To generate the documentation locally:
```bash
# Install godoc
go install golang.org/x/tools/cmd/godoc@latest

# Start local server
godoc -http=:6060

# Open in browser
open http://localhost:6060/pkg/github.com/Enedis-OSS/tic4eebus/
```