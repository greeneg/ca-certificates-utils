package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/greeneg/ca-certificates/configuration"
	"github.com/greeneg/ca-certificates/pluginUtils"
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

	p := pluginUtils.NewPluginUtils()
	// check that stateDir exists
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

	etcCertsDir := filepath.Join(cfg.DestDir, "etc/ssl/certs")
	pemDir := filepath.Join(stateDir, "pem")

	// ensure that the pemDir exists
	fileExists, err = p.FileExists(pemDir)
	if err != nil {
		fmt.Println("ERROR: Cannot check if file exists: " + string(err.Error()))
		os.Exit(1)
	}
	if !fileExists {
		err := os.Mkdir(pemDir, 0755)
		if err != nil {
			fmt.Println("ERROR: Cannot create directory: " + string(err.Error()))
			os.Exit(1)
		}
	}

	code, err := p.RunTrust(pemDir, "directory-hash")
	if err != nil {
		fmt.Println("ERROR: Could not run command: " + string(err.Error()))
		os.Exit(code)
	}

	fmt.Println("Creating " + etcCertsDir)
	// check that /etc/ssl/certs exists and is a symlink to the pemDir
	if !p.IsSymLink(etcCertsDir) || !p.CheckSymlinkTarget(etcCertsDir, pemDir) {
		// if not, remove the existing path and create the symlink
		if cfg.Verbose {
			fmt.Println("NOTICE: Restoring symlink for " + etcCertsDir)
		}
		info, err := os.Lstat(etcCertsDir)
		if err != nil {
			if !os.IsNotExist(err) {
				fmt.Println("ERROR: Cannot inspect existing /etc/ssl/certs: " + string(err.Error()))
				os.Exit(1)
			}
		} else {
			if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
				err = os.RemoveAll(etcCertsDir)
			} else {
				err = os.Remove(etcCertsDir)
			}
			if err != nil {
				fmt.Println("ERROR: Cannot remove existing /etc/ssl/certs: " + string(err.Error()))
				os.Exit(1)
			}
		}
		err = os.Symlink(pemDir, etcCertsDir)
		if err != nil {
			fmt.Println("ERROR: Cannot create symlink: " + string(err.Error()))
			os.Exit(1)
		}
	}
}
