package logger

import (
	"fmt"
	"log/syslog"

	"github.com/greeneg/ca-certificates-utils/configuration"
)

type Logger struct {
	LogFile            string `json:"logFile"`
	UseSyslog          bool   `json:"useSyslog"`
	UseLogFile         bool   `json:"useLogFile"`
	UseConsoleLog      bool   `json:"useConsoleLog"`
	SyslogFacility     string `json:"syslogFacility"`
	DefaultSyslogLevel string `json:"defaultSyslogLevel"`
	SyslogWriter       *syslog.Writer
}

type LogLevel int

const (
	INFO LogLevel = iota
	NOTICE
	WARNING
	ERROR
)

func (l LogLevel) String() string {
	return [...]string{"INFO", "NOTICE", "WARNING", "ERROR"}[l]
}

func NewLogger(cfg configuration.Configuration, appName string) Logger {
	l := Logger{}

	l.LogFile = cfg.LogFile
	l.UseSyslog = cfg.UseSyslog
	l.UseLogFile = cfg.UseLogFile
	l.UseConsoleLog = cfg.UseConsoleLog
	l.SyslogFacility = cfg.SyslogFacility
	l.DefaultSyslogLevel = cfg.DefaultSyslogLevel

	if l.UseSyslog {
		var facility syslog.Priority
		var loglevel syslog.Priority
		switch l.DefaultSyslogLevel {
		case "INFO":
			loglevel = syslog.LOG_INFO
		case "NOTICE":
			loglevel = syslog.LOG_NOTICE
		case "WARNING":
			loglevel = syslog.LOG_WARNING
		case "ERROR":
			loglevel = syslog.LOG_ERR
		default:
			loglevel = syslog.LOG_INFO
		}
		switch l.SyslogFacility {
		case "DAEMON":
			facility = syslog.LOG_DAEMON
		default:
			facility = syslog.LOG_DAEMON
		}
		sysLog, err := syslog.New(loglevel|facility, appName)
		if err != nil {
			loglevel = syslog.LOG_INFO
			facility = syslog.LOG_DAEMON
			sysLog, err = syslog.New(loglevel|facility, appName)
			if err != nil {
				// if we can't create a syslog writer, disable syslog logging and log the error to stdout
				l.UseSyslog = false
				fmt.Println(fmt.Errorf("ERROR: %w", err))
			} else {
				l.SyslogWriter = sysLog
			}
		} else {
			l.SyslogWriter = sysLog
		}
	}

	return l
}

func (l Logger) GetDefaultLogLevel() LogLevel {
	switch l.DefaultSyslogLevel {
	case "INFO":
		return INFO
	case "NOTICE":
		return NOTICE
	case "WARNING":
		return WARNING
	case "ERROR":
		return ERROR
	default:
		return INFO
	}
}

func (l Logger) GetSyslogFacility() string {
	switch l.SyslogFacility {
	case "DAEMON":
		return "DAEMON"
	default:
		return "DAEMON"
	}
}

func (l Logger) Log(level LogLevel, message string) {
	if l.UseSyslog && l.SyslogWriter != nil {
		switch level {
		case INFO:
			l.SyslogWriter.Info("I: " + message)
		case NOTICE:
			l.SyslogWriter.Notice("N: " + message)
		case WARNING:
			l.SyslogWriter.Warning("W: " + message)
		case ERROR:
			l.SyslogWriter.Err("E: " + message)
		default:
			l.SyslogWriter.Info("I: " + message)
		}
	}
	if l.UseConsoleLog {
		fmt.Printf("%s: %s\n", level.String(), message)
	}
}

func (l Logger) LvlInfo() LogLevel {
	return INFO
}

func (l Logger) LvlNotice() LogLevel {
	return NOTICE
}

func (l Logger) LvlWarning() LogLevel {
	return WARNING
}

func (l Logger) LvlError() LogLevel {
	return ERROR
}
