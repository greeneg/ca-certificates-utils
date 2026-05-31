package configuration

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type Configuration struct {
	StateDir            string   `json:"stateDir"`
	HooksDirList        []string `json:"pluginDirectories"`
	Verbose             bool     `json:"verbose"`
	DestDir             string   `json:"rootDir"`
	Fresh               bool     `json:"clean"`
	UpdateCertDataFile  bool     `json:"updateCertificateDataFile"`
	UseSyslog           bool     `json:"useSyslog"`
	UseLogFile          bool     `json:"useLogFile"`
	UseConsoleLog       bool     `json:"useConsoleLog"`
	LogFile             string   `json:"logFile"`
	EnableNssDbPlugin   bool     `json:"enableNssDbPlugin"`
	EnableJavaP12Plugin bool     `json:"enableJavaP12Plugin"`
	SyslogFacility      string   `json:"syslogFacility"`
	DefaultSyslogLevel  string   `json:"defaultSyslogLevel"`
}

func NewConfiguration() Configuration {
	c := Configuration{}

	// find certutil and keytool. If not found, disable NssDb and Java p12 plugins, respectively
	certutilPath, err := exec.LookPath("certutil")
	if err != nil {
		fmt.Printf("certutil not found, disabling NSS DB plugin: %v\n", err)
		c.EnableNssDbPlugin = false
	} else {
		fmt.Printf("certutil found at: %s\n", certutilPath)
		c.EnableNssDbPlugin = true
	}

	keytoolPath, err := exec.LookPath("keytool")
	if err != nil {
		fmt.Printf("keytool not found, disabling Java p12 plugin: %v\n", err)
		c.EnableJavaP12Plugin = false
	} else {
		fmt.Printf("keytool found at: %s\n", keytoolPath)
		c.EnableJavaP12Plugin = true
	}

	c.StateDir = "var/lib/ca-certificates"
	c.HooksDirList = append(c.HooksDirList, "etc/ca-certificates/update.d")
	c.HooksDirList = append(c.HooksDirList, "usr/lib/ca-certificates/update.d")
	c.Verbose = false
	c.DestDir = "/"
	c.Fresh = false
	c.LogFile = "/var/log/update-ca.log"
	c.UpdateCertDataFile = false
	c.UseSyslog = true
	c.UseLogFile = false
	c.UseConsoleLog = true
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
