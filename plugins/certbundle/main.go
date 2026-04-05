package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/greeneg/ca-certificates/configuration"
	"github.com/greeneg/ca-certificates/pluginUtils"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println("ERROR: Cannot read string: " + string(err.Error()))
		os.Exit(1)
	}

	// variables
	var fileTimeStamp time.Time
	var dirTimeStamp time.Time

	// process line as JSON
	cfg := configuration.NewConfiguration()
	cfg, err = cfg.FromJson(line)
	if err != nil {
		fmt.Println("ERROR: Cannot process JSON string: " + string(err.Error()))
		os.Exit(1)
	}

	// ensure that destDir ends with a slash
	p := pluginUtils.NewPluginUtils()
	cfg.DestDir = p.EnsureVarEndsWithSlash(cfg.DestDir)
	stateDir := filepath.Join(cfg.DestDir, cfg.StateDir)
	fileExists, err := p.FileExists(stateDir)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if !fileExists {
		fmt.Println("ERROR: State directory does not exist: " + stateDir)
		os.Exit(1)
	}
	caFile := filepath.Join(stateDir, "ca-bundle.pem")
	caDir := filepath.Join(stateDir, "pem")

	// first get the stat info for the above
	fileTimeStamp = p.StatInfo(caFile, cfg)
	dirTimeStamp = p.StatInfo(caDir, cfg)

	if !cfg.Fresh && fileTimeStamp.After(dirTimeStamp) {
		os.Exit(0)
	}

	// now execute trust to get the pem file generated
	code, err := p.RunTrust(caFile, "bundle")
	if err != nil {
		fmt.Println("ERROR: Could not run command: " + string(err.Error()))
		os.Exit(code)
	}

	if cfg.Verbose {
		fmt.Println("Creating " + caFile)
	}
	err = p.GeneratePemFile(caFile)
	if err != nil {
		fmt.Println("ERROR: Cannot generate pem file: " + string(err.Error()))
		os.Exit(1)
	}
	etcCaPemFile := filepath.Join(cfg.DestDir, "etc/ssl/ca-bundle.pem")
	fileExists, err = p.FileExists(etcCaPemFile)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if fileExists && !p.IsSymLink(etcCaPemFile) {
		err := os.Remove(etcCaPemFile)
		if err != nil {
			fmt.Println("ERROR: Cannot remove existing /etc/ssl/ca-bundle.pem: " + string(err.Error()))
			os.Exit(1)
		}
		err = p.ConfigureEtcSslCaBundlePem(etcCaPemFile)
		if err != nil {
			fmt.Println("ERROR: Cannot configure /etc/ssl/ca-bundle.pem: " + string(err.Error()))
			os.Exit(1)
		}
	}
}
