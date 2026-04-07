package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/greeneg/ca-certificates-utils/configuration"
	"github.com/greeneg/ca-certificates-utils/logger"
	"github.com/greeneg/ca-certificates-utils/pluginUtils"
)

func main() {
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	// ReadString returns data even with io.EOF if stdin doesn't end with newline
	// Only fail if we have no data at all
	if err != nil && err != io.EOF {
		fmt.Println("ERROR: Cannot read string: " + string(err.Error()))
		os.Exit(1)
	}
	if line == "" && err != nil {
		fmt.Println("ERROR: No input received from stdin")
		os.Exit(1)
	}
	// Trim any trailing newline for consistent processing
	line = strings.TrimRight(line, "\n")

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

	// load our logger
	logger := logger.NewLogger(cfg, "certbundle")

	// ensure that destDir ends with a slash
	p := pluginUtils.NewPluginUtils()
	cfg.DestDir = p.EnsureVarEndsWithSlash(cfg.DestDir)
	stateDir := filepath.Join(cfg.DestDir, cfg.StateDir)
	fileExists, err := p.FileExists(stateDir)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
		os.Exit(1)
	}
	if !fileExists {
		logger.Log(logger.LvlError(), fmt.Sprintf("State directory does not exist: %s", stateDir))
		os.Exit(1)
	}
	caFile := filepath.Join(stateDir, "ca-bundle.pem")
	caDir := filepath.Join(stateDir, "pem")

	// first get the stat info for the above
	fileTimeStamp = p.StatInfo(caFile, cfg, logger)
	dirTimeStamp = p.StatInfo(caDir, cfg, logger)

	if !cfg.Fresh && fileTimeStamp.After(dirTimeStamp) {
		os.Exit(0)
	}

	// now execute trust to get the pem file generated
	code, err := p.RunTrust(caFile, "bundle", logger)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Could not run command: %v", err))
		os.Exit(code)
	}

	if cfg.Verbose {
		logger.Log(logger.LvlInfo(), "Creating "+caFile)
	}
	err = p.GeneratePemFile(caFile, logger)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot generate pem file: %v", err))
		os.Exit(1)
	}
	etcCaPemFile := filepath.Join(cfg.DestDir, "etc/ssl/ca-bundle.pem")
	fileExists, err = p.FileExists(etcCaPemFile)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
		os.Exit(1)
	}
	if fileExists && !p.IsSymLink(etcCaPemFile) {
		err := os.Remove(etcCaPemFile)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot remove existing /etc/ssl/ca-bundle.pem: %v", err))
			os.Exit(1)
		}
		err = p.ConfigureEtcSslCaBundlePem(etcCaPemFile, logger)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot configure /etc/ssl/ca-bundle.pem: %v", err))
			os.Exit(1)
		}
	}
}
