module github.com/gizzahub/gzh-cli-package-manager

// Go 1.26 is the supported consumer baseline. The toolchain directive pins the
// latest approved patch release for development and CI reproducibility.
go 1.26.0

toolchain go1.26.7

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
