package configuration

import (
	"encoding/json"
	"fmt"
)

type Configuration struct {
	StateDir           string   `json:"stateDir"`
	HooksDirList       []string `json:"pluginDirectories"`
	Verbose            bool     `json:"verbose"`
	DestDir            string   `json:"rootDir"`
	Fresh              bool     `json:"clean"`
	UseSyslog          bool     `json:"useSyslog"`
	LogFile            string   `json:"logFile"`
	SyslogFacility     string   `json:"syslogFacility"`
	DefaultSyslogLevel string   `json:"defaultSyslogLevel"`
}

func NewConfiguration() Configuration {
	c := Configuration{}

	c.StateDir = "var/lib/ca-certificates"
	c.HooksDirList = append(c.HooksDirList, "etc/ca-certificates/update.d")
	c.HooksDirList = append(c.HooksDirList, "usr/lib/ca-certificates/update.d")
	c.Verbose = false
	c.DestDir = "/"
	c.Fresh = false
	c.LogFile = "/var/log/update-ca.log"
	c.UseSyslog = true
	c.SyslogFacility = "DAEMON"
	c.DefaultSyslogLevel = "INFO"

	return c
}

func (c *Configuration) FromJson(s string) (Configuration, error) {
	jsonBytes := []byte(s)
	var cfg Configuration

	err := json.Unmarshal(jsonBytes, &cfg)
	if err != nil {
		fmt.Println("ERROR: Cannot unmarshal JSON string: " + string(err.Error()))
		return Configuration{}, err
	}

	return cfg, nil
}

func (c *Configuration) ToJson(cfg Configuration) (string, error) {
	jsonBytes, err := json.Marshal(cfg)
	if err != nil {
		fmt.Println("ERROR: Cannot marshal struct into JSON: " + string(err.Error()))
		return "", err
	}

	jsonString := string(jsonBytes)

	return jsonString, nil
}
