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
		// cannot use the logger here since we haven't initialized it yet, so just print the error
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
	logger := logger.NewLogger(cfg, "java")

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

	caFile := filepath.Join(stateDir, "java-cacerts")

	if cfg.Verbose {
		logger.Log(logger.LvlInfo(), "Creating "+caFile)
	}
	code, err := p.RunTrust(caFile, "java-cacerts", logger)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot extract java cacerts: %v", err))
		os.Exit(code)
	}
	// now move the generated file to the correct location and name
	err = os.Rename(caFile+".new", caFile)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot rename file: %v", err))
		os.Exit(1)
	}
}
