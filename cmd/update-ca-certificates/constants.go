package main

const appName string = "update-ca-certificates"
const appVersion string = "0.1"

type Configuration struct {
	StateDir string
	HooksDirList []string
	Verbose bool
	DestDir string
	Fresh bool
	UseSyslog bool
	LogFile string
	SyslogFacility string
	DefaultSyslogLevel string
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
