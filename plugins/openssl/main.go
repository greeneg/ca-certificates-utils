package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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

	// process line as JSON
	cfg := configuration.NewConfiguration()
	cfg, err = cfg.FromJson(line)
	if err != nil {
		fmt.Println("ERROR: Cannot process JSON string: " + string(err.Error()))
		os.Exit(1)
	}

	// load our logger
	logger := logger.NewLogger(cfg, "openssl.plugin")

	p := pluginUtils.NewPluginUtils()
	// check that stateDir exists
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

	caDir := filepath.Join(stateDir, "openssl")

	if cfg.Verbose {
		logger.Log(logger.LvlInfo(), "Creating "+caDir)
	}
	// create caDir and all its parents if it does not exist
	fileExists, err = p.FileExists(caDir)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
		os.Exit(1)
	}
	if !fileExists {
		err := os.MkdirAll(caDir, 0755)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot create directory: %v", err))
			os.Exit(1)
		}
	}

	code, err := p.RunTrust(caDir, "openssl", logger)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot run trust command: %v", err))
		os.Exit(code)
	}
}
