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
	cfg.DestDir = p.EnsureVarEndsWithSlash(cfg.DestDir)
	stateDir := cfg.DestDir + cfg.StateDir
	fileExists, err := p.FileExists(stateDir)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if !fileExists {
		fmt.Println("ERROR: State directory does not exist: " + stateDir)
		os.Exit(1)
	}

	caFile := cfg.DestDir + cfg.StateDir + "/java-cacerts"

	if cfg.Verbose {
		fmt.Println("Creating " + caFile)
	}
	code, err := p.RunTrust(caFile, "java-cacerts")
	if err != nil {
		fmt.Println("ERROR: Cannot extract java cacerts: " + string(err.Error()))
		os.Exit(code)
	}
	// now move the generated file to the correct location and name
	err = os.Rename(caFile+".new", caFile)
	if err != nil {
		fmt.Println("ERROR: Cannot rename file: " + string(err.Error()))
		os.Exit(1)
	}
}
