package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

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
	cfg := pluginUtils.NewConfiguration()
	cfg, err = cfg.FromJson(line)

	caFile := cfg.DestDir + "/" + cfg.StateDir + "/ca-bundle.pem"
	caDir  := cfg.DestDir + "/" + cfg.StateDir + "/pem"

	// first get the stat info for the above
	fileTimeStamp = pluginUtils.statInfo(caFile, cfg)
	dirTimeStamp = pluginUtils.statInfo(caDir, cfg)

	if !cfg.Fresh && fileTimeStamp.After(dirTimeStamp) {
		os.Exit(0)
	}

	// now execute trust to get the pem file generated
	code, err := runTrust(caFile)
	if err != nil {
		fmt.Println("ERROR: Could not run command: " + string(err.Error()))
		os.Exit(code)
	}

	if cfg.Verbose { fmt.Println("Creating " + caFile) }
	err = generatePemFile(caFile)
	if err != nil {
		fmt.Println("ERROR: Cannot generate pem file: " + string(err.Error()))
		os.Exit(1)
	}
	etcCaPemFile := cfg.DestDir + "etc/ssl/ca-bundle.pem"
	if fileExists(etcCaPemFile) && !isSymLink(etcCaPemFile) {
		err = configureEtcSslCaBundlePem(etcCaPemFile)
	}
}
