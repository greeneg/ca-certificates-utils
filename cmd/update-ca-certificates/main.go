package main

import (
	"errors"
	"fmt"
	"log/syslog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

func showHelp() {
	fmt.Println(appName + " - An application for updating system certificate stores")
	div1 := strings.Repeat("-", 79)
	div2 := strings.Repeat("-", 7)
	fmt.Println(div1)
	fmt.Printf("\nOPTIONS:\n%s\n", div2)
	fmt.Println("  -f|--fresh           Create a brand new set of certificate stores")
	fmt.Println("  -r|--root DIRECTORY  The directory to treat as the filesystem root")
	fmt.Println("  -v|--verbose         Whether to be verbose, or to only output to syslog")
	fmt.Println("  -h|--help            Show this help message")
	fmt.Println("  -V|--version         Show the application version")
}

func showVersion() {
	fmt.Println(appName + " - An application for updating system certificate stores")
	div1 := strings.Repeat("-", 79)
	fmt.Println(div1)
	fmt.Println("version:   " + appVersion)
	fmt.Println("Author:    Gary L. Greene Jr.")
	fmt.Println("Copyright: ©2026 YggdrasilSoft, LLC.")
	fmt.Println("License:   GPL v3.0 or later")
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func ProcessArgs(args []string, c Configuration) (Configuration, error) {
	rootDir := "/"
	hasRootDir := false
	for position, a := range args {
		if a == "-f" || a == "--fresh" {
			c.Fresh = true
		}
		if a == "-v" || a == "--verbose" {
			c.Verbose = true
		}
		if a == "-h" || a == "--help" {
			showHelp()
			os.Exit(0)
		}
		if a == "-V" || a == "--version" {
			showVersion()
			os.Exit(0)
		}
		// finally process the root flag
		if a == "-r" || a == "--root" {
			_tDir := ""
			if position+1 < len(os.Args) {
				_tDir = os.Args[position+1]
			} else {
				fmt.Println("No directory specified! Exiting")
				os.Exit(1)
			}
			// now test if the directory exists
			dirExistence, err := Exists(c.DestDir)
			if err != nil {
				return c, fmt.Errorf("Cannot use directory! %w", err)
			}
			if dirExistence {
				c.DestDir = _tDir
				hasRootDir = true
			}
		}
	}
	if !hasRootDir {
		c.DestDir = rootDir
	}
	return c, nil
}

func findPlugins(c Configuration, s *syslog.Writer) ([]string, error) {
	var plugins []string

	for _, p := range(c.HooksDirList) {
		dir := c.DestDir + p
		err := os.Chdir(dir)
		if err != nil {
			if dir == "/etc/ca-certificates/update.d" {
				if errors.Is(err, os.ErrNotExist) {
					// we can effectively skip this case as
					// the /etc/ca-certificates/update.d is
					// an override location
					fmt.Printf("Location %s does not exist\n", dir)
					if c.UseSyslog {
						s.Notice("N: Location /etc/ca-certificates/update.d does not exist. Skipping")
					}
					continue
				}
			}
			fmt.Println(fmt.Errorf("ERROR: %w", err))
			if c.UseSyslog {
				s.Err("E: " + string(err.Error()))
				os.Exit(1)
			}
		}
		fmt.Printf("Checking location %s for plugins\n", dir)
		if c.UseSyslog {
			s.Info("I: Checking location " + dir + " for plugins")
		}
		matches, err := filepath.Glob("*.plugin")
		if err != nil {
			fmt.Println(fmt.Errorf("ERROR: %w", err))
			if c.UseSyslog {
				s.Err("E: " + string(err.Error()))
				os.Exit(1)
			}
		}
		if len(matches) > 0 {
			fmt.Printf("Found: %v", matches)
			if c.UseSyslog {
				s.Info("I: Found " + fmt.Sprintf("%v", matches))
			}
			plugins = append(plugins, matches...)
		} else {
			fmt.Printf("No plugins found in %s\n", dir)
			if c.UseSyslog {
				s.Err("E: No plugins in " + dir)
			}
		}
	}

	return plugins, nil
}

func runPlugins(p []string, c Configuration) error {
	for _, plugin := range(p) {
		fmt.Printf("plugin: %s", plugin)
	}
	return nil
}

func isRoot() bool {
	return os.Geteuid() == 0
}

func main() {
	c := NewConfiguration()

	// lets deal with our flags
	c, err := ProcessArgs(os.Args, c)
	if err != nil {
		e := fmt.Errorf("ERROR: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}

	// check that we're running as root
	if !isRoot() {
		fmt.Println(fmt.Errorf("ERROR: %s", "Application not running as root. Exiting"))
		os.Exit(2)
	}

	// if we want to use syslog, create our logger
	var sysLog *syslog.Writer
	if c.UseSyslog {
		var facility syslog.Priority
		var loglevel syslog.Priority
		switch c.DefaultSyslogLevel {
			case "INFO":
				loglevel = syslog.LOG_INFO
			case "NOTICE":
				loglevel = syslog.LOG_NOTICE
			case "WARNING":
				loglevel = syslog.LOG_WARNING
			default:
				loglevel = syslog.LOG_INFO
		}
		switch c.SyslogFacility {
			case "DAEMON":
				facility = syslog.LOG_DAEMON
			default:
				facility = syslog.LOG_DAEMON
		}
		sysLog, err = syslog.New(loglevel|facility, appName)
		if err != nil {
			fmt.Println(fmt.Errorf("ERROR: %w", err))
		}
	}

	// now that our args are parsed, set our umask
	oldUmask := syscall.Umask(0x022) // owner write, all others read

	// Create a lock file if the system is doing a transactional update to avoid running any plugins, which can
	// cause RPM transactional scripts to fail
	v, exists := os.LookupEnv("TRANSACTIONAL_UPDATE")
	if exists {
		switch v {
			case "true", "yes", "1":
				if c.Verbose {
					fmt.Println("transactional update in progress, not running any plugins")
					if c.UseSyslog {
						sysLog.Info("I: Transactional update in progress. not running any plugins.")
					}
				}
				_, err := os.Create("/etc/pki/trust/.updated")
				if err != nil {
					fmt.Println(fmt.Errorf("ERROR: %w", err))
					if c.UseSyslog {
						sysLog.Err("E: " + string(err.Error()))
					}
					os.Exit(1)
				}
				os.Exit(0)
			default:
				if c.Verbose {
					fmt.Println("transactional updates are not running. Continuing")
					if c.UseSyslog {
						sysLog.Info("I: Transactional updates are not running. Continuing")
					}
				}
		}
	}
	// check if our update lock file exists
	fileExists, err := Exists("/etc/pki/trust/.updated")
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: %w", err))
		if c.UseSyslog {
			sysLog.Err("E: " + string(err.Error()))
		}
		os.Exit(1)
	}
	if fileExists {
		err = os.Remove("/etc/pki/trust/.updated")
		if err != nil {
			fmt.Println("ERROR: %w", err)
			if c.UseSyslog {
				sysLog.Err("E: " + string(err.Error()))
			}
			os.Exit(1)
		}
	}

	// find all installed plugins
	plugins, err := findPlugins(c, sysLog)
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: %w", err))
		if c.UseSyslog {
			sysLog.Err("E: " + string(err.Error()))
		}
		os.Exit(1)
	}

	// now execute the plugins
	err = runPlugins(plugins, c)
	if err != nil {
		fmt.Println(fmt.Errorf("ERROR: %w", err))
		if c.UseSyslog {
			sysLog.Err("E: " + string(err.Error()))
		}
		os.Exit(1)
	}

	syscall.Umask(oldUmask) // restore our previous umask
}
