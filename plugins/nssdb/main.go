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

	// show the configuration in the logs
	fmt.Printf("Configuration: %+v\n", cfg)

	// load our logger
	logger := logger.NewLogger(cfg, "nssdb.plugin")

	// ensure that the tools we need are available (certutil)
	_, err = pluginUtils.NewPluginUtils().RunCommand("which", []string{"certutil"}, logger)
	if err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Required tool 'certutil' is not available: %v", err))
		os.Exit(1)
	}

	p := pluginUtils.NewPluginUtils()
	// check that stateDir exists
	cfg.DestDir = p.EnsureVarEndsWithSlash(cfg.DestDir)
	stateDir := filepath.Join(cfg.DestDir, cfg.StateDir)
	nssdbWorkDir := filepath.Join(stateDir, "nssdb")
	etcPkiDir := filepath.Join(cfg.DestDir, "etc/pki/nssdb")
	logger.Log(logger.LvlInfo(), fmt.Sprintf("State directory: %s", stateDir))
	logger.Log(logger.LvlInfo(), fmt.Sprintf("NSSDB working directory: %s", nssdbWorkDir))
	logger.Log(logger.LvlInfo(), fmt.Sprintf("Symlink target directory: %s", etcPkiDir))
	for _, dir := range []string{stateDir, nssdbWorkDir, etcPkiDir} {
		fileExists, err := p.FileExists(dir)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
			os.Exit(1)
		}
		// if the directory doesn't exist, create it
		if !fileExists {
			err := os.MkdirAll(dir, 0755)
			if err != nil {
				logger.Log(logger.LvlError(), fmt.Sprintf("Cannot create directory: %v", err))
				os.Exit(1)
			}
		}
	}

	// now check if the database files exist in the nssdbWorkDir, if not, create an empty database using certutil
	certDBFiles := []string{"cert9.db", "key4.db", "pkcs11.txt"}
	var createDB bool
	for _, file := range certDBFiles {
		filePath := filepath.Join(nssdbWorkDir, file)
		logger.Log(logger.LvlInfo(), fmt.Sprintf("Checking if NSS database file exists: %s", filePath))
		fileExists, err := p.FileExists(filePath)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
			os.Exit(1)
		}
		if !fileExists {
			createDB = true
		} else {
			logger.Log(logger.LvlInfo(), fmt.Sprintf("NSS database file exists: %s", filePath))
		}
	}

	if createDB {
		logger.Log(logger.LvlInfo(), "Creating NSS database")
		cmd := "certutil"
		args := []string{"-N", "-d", "sql:" + nssdbWorkDir, "--empty-password"}
		_, err := p.RunCommand(cmd, args, logger)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot create NSS database: %v", err))
			os.Exit(1)
		}
	} else {
		logger.Log(logger.LvlInfo(), "NSS database already exists, skipping creation")
	}

	// get updated certdata.txt from upstream and import it into the NSS database using certutil
	// download certdata.txt from https://hg-edge.mozilla.org/projects/nss/raw-file/tip/lib/ckfw/builtins/certdata.txt
	if cfg.UpdateCertDataFile {
		logger.Log(logger.LvlInfo(), "Updating certdata.txt and importing certificates into NSS database")
		certdataURL := "https://hg-edge.mozilla.org/projects/nss/raw-file/tip/lib/ckfw/builtins/certdata.txt"
		certdataPath := filepath.Join(stateDir, "certdata.txt")
		err = p.DownloadFile(certdataURL, certdataPath, logger)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot download certdata.txt: %v", err))
			os.Exit(1)
		}

		// process certdata.txt and import the certificates into the NSS database
		logger.Log(logger.LvlInfo(), "Importing certificates into NSS database using updated certdata.txt (updateCertificateDataFile is true)")
		err = p.ProcessCertdataAndImportToNSSDB(certdataPath, nssdbWorkDir, logger)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot process certdata.txt and import to NSS database: %v", err))
			os.Exit(1)
		}
		os.Exit(0) // intentional. Trying to debug the certdata stuff
	} else {
		// check if certdata.txt exists in the stateDir, if not, log a warning and continue with an empty NSS database
		certdataPath := filepath.Join(stateDir, "certdata.txt")
		fileExists, err := p.FileExists(certdataPath)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
			os.Exit(1)
		}
		if !fileExists {
			logger.Log(logger.LvlWarning(), fmt.Sprintf("certdata.txt does not exist in state directory: %s. Continuing with an empty NSS database.", certdataPath))
			os.Exit(0)
		}
		logger.Log(logger.LvlInfo(), "Importing certificates into NSS database using existing certdata.txt (updateCertificateDataFile is false)")
		err = p.ProcessCertdataAndImportToNSSDB(certdataPath, nssdbWorkDir, logger)
		if err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot process certdata.txt and import to NSS database: %v", err))
			os.Exit(1)
		}
	}

	// now clean up the temporary files created during certdata.txt processing
	//	err = p.CleanupTempFiles(os.TempDir()+"/certdata_processing", logger)
	//	if err != nil {
	//		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot clean up temporary files: %v", err))
	//		os.Exit(1)
	//	}

	// now create a symlink from the nssdbWorkDir to /etc/pki/nssdb
	// if the symlink already exists, remove it and create a new one
	symlinkPath := etcPkiDir
	if _, err := os.Lstat(symlinkPath); err == nil {
		if err := os.Remove(symlinkPath); err != nil {
			logger.Log(logger.LvlError(), fmt.Sprintf("Cannot remove existing symlink: %v", err))
			os.Exit(1)
		}
	}
	if err := os.Symlink(nssdbWorkDir, symlinkPath); err != nil {
		logger.Log(logger.LvlError(), fmt.Sprintf("Cannot create symlink: %v", err))
		os.Exit(1)
	}
}
