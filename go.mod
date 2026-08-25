module github.com/gizzahub/gzh-cli-package-manager

// The module remains consumable by Go 1.24.0+. The toolchain below recommends
// the development compiler when a developer's default Go version is older.
go 1.24.0

toolchain go1.26.6

require (
	github.com/spf13/cobra v1.10.2
	github.com/stretchr/testify v1.12.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/spf13/pflag v1.0.9 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
)
