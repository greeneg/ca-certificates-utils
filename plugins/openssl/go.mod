module github.com/greeneg/ca-certificates-utils/plugins/openssl

go 1.26.1

require (
	github.com/greeneg/ca-certificates-utils/configuration v0.0.0
	github.com/greeneg/ca-certificates-utils/logger v0.0.0
	github.com/greeneg/ca-certificates-utils/pluginUtils v0.0.0
)

require github.com/MakeNowJust/heredoc v1.0.0 // indirect

replace github.com/greeneg/ca-certificates-utils/configuration => ../../configuration

replace github.com/greeneg/ca-certificates-utils/logger => ../../logger

replace github.com/greeneg/ca-certificates-utils/pluginUtils => ../../pluginUtils
