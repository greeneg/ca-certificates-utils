module github.com/greeneg/ca-certificates/plugins/java

go 1.26.1

replace github.com/greeneg/ca-certificates/configuration => ../../configuration

replace github.com/greeneg/ca-certificates/pluginUtils => ../../pluginUtils

require (
	github.com/greeneg/ca-certificates/configuration v0.0.0
	github.com/greeneg/ca-certificates/pluginUtils v0.0.0-00010101000000-000000000000
)

require github.com/MakeNowJust/heredoc v1.0.0 // indirect
