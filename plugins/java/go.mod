module github.com/greeneg/ca-certificates-utils/plugins/java

go 1.26.1

replace github.com/greeneg/ca-certificates-utils/configuration => ../../configuration

replace github.com/greeneg/ca-certificates-utils/logger => ../../logger

replace github.com/greeneg/ca-certificates-utils/pluginUtils => ../../pluginUtils

require (
	github.com/greeneg/ca-certificates-utils/configuration v0.0.0
	github.com/greeneg/ca-certificates-utils/logger v0.0.0
	github.com/greeneg/ca-certificates-utils/pluginUtils v0.0.0
)

require (
	github.com/MakeNowJust/heredoc v1.0.0 // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jlaffaye/ftp v0.2.0 // indirect
)
