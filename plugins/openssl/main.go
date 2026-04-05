package main

import (
	"bufio"
	"fmt"
	"os"

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

	// process line as JSON
	cfg := configuration.NewConfiguration()
	cfg, err = cfg.FromJson(line)
	if err != nil {
		fmt.Println("ERROR: Cannot process JSON string: " + string(err.Error()))
		os.Exit(1)
	}

	p := pluginUtils.NewPluginUtils()
	// check that stateDir exists
	stateDir := cfg.DestDir + "/" + cfg.StateDir
	fileExists, err := p.FileExists(stateDir)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if !fileExists {
		fmt.Println("ERROR: State directory does not exist: " + stateDir)
		os.Exit(1)
	}

	caDir := cfg.DestDir + "/" + cfg.StateDir + "/openssl"

	if cfg.Verbose {
		fmt.Println("Creating " + caDir)
	}
	// create caDir and all its parents if it does not exist
	fileExists, err = p.FileExists(caDir)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if !fileExists {
		err := os.MkdirAll(caDir, 0755)
		if err != nil {
			fmt.Println("ERROR: Cannot create directory: " + string(err.Error()))
			os.Exit(1)
		}
	}

	code, err := p.RunTrust(caDir, "openssl")
	if err != nil {
		fmt.Println("ERROR: Cannot run trust command: " + string(err.Error()))
		os.Exit(code)
	}
}
