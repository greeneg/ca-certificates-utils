module github.com/greeneg/ca-certificates-utils/pluginUtils

go 1.26.1

require github.com/MakeNowJust/heredoc v1.0.0

require github.com/greeneg/ca-certificates-utils/configuration v0.0.0

require github.com/greeneg/ca-certificates-utils/logger v0.0.0

replace github.com/greeneg/ca-certificates-utils/configuration => ../configuration
replace github.com/greeneg/ca-certificates-utils/logger => ../logger
