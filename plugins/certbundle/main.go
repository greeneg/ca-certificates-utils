package main

import (
	"bufio"
	"fmt"
	"os"
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

	caFile := cfg.DestDir + "/" + cfg.StateDir + "/ca-bundle.pem"
	caDir := cfg.DestDir + "/" + cfg.StateDir + "/pem"

	// first get the stat info for the above
	p := pluginUtils.NewPluginUtils()
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
	etcCaPemFile := cfg.DestDir + "etc/ssl/ca-bundle.pem"
	fileExists, err := p.FileExists(etcCaPemFile)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if fileExists && !p.IsSymLink(etcCaPemFile) {
		err = p.ConfigureEtcSslCaBundlePem(etcCaPemFile)
	}
}
