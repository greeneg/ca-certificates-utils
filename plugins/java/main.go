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
	if !p.FileExists(stateDir) {
		fmt.Println("ERROR: State directory does not exist: " + stateDir)
		os.Exit(1)
	}

	caFile := cfg.DestDir + "/" + cfg.StateDir + "/java-cacerts"

	if cfg.Verbose {
		fmt.Println("Creating " + caFile)
	}
	p.RunTrust(caFile, "java-cacerts")
	// now move the generated file to the correct location and name
	err = os.Rename(caFile+".new", caFile)
	if err != nil {
		fmt.Println("ERROR: Cannot rename file: " + string(err.Error()))
		os.Exit(1)
	}
}
