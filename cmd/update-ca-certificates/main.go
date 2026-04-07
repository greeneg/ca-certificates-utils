package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/greeneg/ca-certificates-utils/configuration"
	"github.com/greeneg/ca-certificates-utils/logger"
	"github.com/greeneg/ca-certificates-utils/pluginUtils"
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

func ProcessArgs(args []string, c configuration.Configuration, p *pluginUtils.PluginUtils) (configuration.Configuration, error) {
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
			dirExistence, err := p.FileExists(c.DestDir)
			if err != nil {
				fmt.Println(fmt.Errorf("ERROR: %w", err))
				os.Exit(1)
			}
			// ensure that the directory ends with a slash
			_tDir = p.EnsureVarEndsWithSlash(_tDir)
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

func isRoot() bool {
	return os.Geteuid() == 0
}

func main() {
	c := configuration.NewConfiguration()
	p := pluginUtils.NewPluginUtils()

	logger := logger.NewLogger(c, appName)

	// first, check that we have the required tools used by our plugins
	logger.Log(logger.LvlInfo(), "Checking for required tools")
	err := p.CheckRequiredTools()
	if err != nil {
		e := fmt.Errorf("%w", err)
		logger.Log(logger.LvlError(), e.Error())
		os.Exit(1)
	}

	// lets deal with our flags
	logger.Log(logger.LvlInfo(), "Processing command line arguments")
	c, err = ProcessArgs(os.Args, c, &p)
	if err != nil {
		e := fmt.Errorf("%w", err)
		logger.Log(logger.LvlError(), e.Error())
		os.Exit(1)
	}

	// check that we're running as root
	logger.Log(logger.LvlInfo(), "Checking if application is running as root")
	if !isRoot() {
		errMsg := fmt.Errorf("%s", "Application not running as root. Exiting")
		logger.Log(logger.LvlError(), errMsg.Error())
		os.Exit(2)
	}

	// now that our args are parsed, set our umask
	logger.Log(logger.LvlInfo(), "Setting umask to 0022")
	oldUmask := syscall.Umask(0x022) // owner write, all others read

	// Create a lock file if the system is doing a transactional update to avoid running any plugins, which can
	// cause RPM transactional scripts to fail
	logger.Log(logger.LvlInfo(), "Checking for transactional update environment variable")
	v, exists := os.LookupEnv("TRANSACTIONAL_UPDATE")
	if exists {
		switch v {
		case "true", "yes", "1":
			if c.Verbose {
				logger.Log(logger.LvlInfo(), "transactional update in progress, not running any plugins")
			}
			err := os.WriteFile("/etc/pki/trust/.updated", []byte(""), 0644)
			if err != nil {
				errMsg := fmt.Errorf("%w", err)
				logger.Log(logger.LvlError(), errMsg.Error())
				os.Exit(1)
			}
			os.Exit(0)
		default:
			if c.Verbose {
				logger.Log(logger.LvlInfo(), "transactional updates are not running. Continuing")
			}
		}
	}
	// check if our update lock file exists
	logger.Log(logger.LvlInfo(), "Checking for transactional update lock file")
	fileExists, err := p.FileExists("/etc/pki/trust/.updated")
	if err != nil {
		errMsg := fmt.Errorf("%w", err)
		logger.Log(logger.LvlError(), errMsg.Error())
		os.Exit(1)
	}
	if fileExists {
		err = os.Remove("/etc/pki/trust/.updated")
		if err != nil {
			errMsg := fmt.Errorf("%w", err)
			logger.Log(logger.LvlError(), errMsg.Error())
			os.Exit(1)
		}
	}

	// find all installed plugins
	logger.Log(logger.LvlInfo(), "Finding plugins")
	plugins, err := p.FindPlugins(c, logger)
	if err != nil {
		errMsg := fmt.Errorf("%w", err)
		logger.Log(logger.LvlError(), errMsg.Error())
		os.Exit(1)
	}

	// now execute the plugins
	if len(plugins) > 0 {
		logger.Log(logger.LvlInfo(), "Running plugins")
		p.RunPlugins(plugins, c, logger)
	}

	logger.Log(logger.LvlInfo(), "Finished running plugins and restoring umask")
	syscall.Umask(oldUmask) // restore our previous umask
}
