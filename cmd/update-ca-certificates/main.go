package main

import (
	"fmt"
	"os"
	"strings"
	"errors"
)

func showHelp() {
	fmt.Println(appName + " - An application for updating system certificate stores")
	div1 := strings.Repeat("-", 79)
	div2 := strings.Repeat("-", 7)
	fmt.Println(div1)
	fmt.Printf("\nOPTIONS:\n%s\n", div2)
	fmt.Println("  -f|--fresh           Create a brand new set of certificate stores")
	fmt.Println("  -r|--root DIRECTORY  The directory to treat as the filesystem root")
	fmt.Println("  -v|--verbose         Whether to be verbose, or to only output to syslog")
	fmt.Println("  -h|--help            Show this help message")
	fmt.Println("  -V|--version         Show the application version")
}

func showVersion() {
	fmt.Println(appName + " - An application for updating system certificate stores")
	div1 := strings.Repeat("-", 79)
	fmt.Println(div1)
	fmt.Println("version:   " + appVersion)
	fmt.Println("Author:    Gary L. Greene Jr.")
	fmt.Println("Copyright: ©2026 YggdrasilSoft, LLC.")
	fmt.Println("License:   GPL v3.0 or later")
}

func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func ProcessArgs(args []string, c Configuration) (Configuration, error) {
	rootDir := "/"
	hasRootDir := false
	for position, a := range args {
		if a == "-f" || a == "--fresh" {
			c.Fresh = true
		}
		if a == "-v" || a == "--verbose" {
			c.Verbose = true
		}
		if a == "-h" || a == "--help" {
			showHelp()
			os.Exit(0)
		}
		if a == "-V" || a == "--version" {
			showVersion()
			os.Exit(0)
		}
		// finally process the root flag
		if a == "-r" || a == "--root" {
			_tDir := ""
			if position+1 < len(os.Args) {
				_tDir = os.Args[position+1]
			} else {
				fmt.Println("No directory specified! Exiting")
				os.Exit(1)
			}
			// now test if the directory exists
			dirExistence, err := Exists(c.DestDir)
			if err != nil {
				return c, fmt.Errorf("Cannot use directory! %w", err)
			}
			if dirExistence {
				c.DestDir = _tDir
				hasRootDir = true
			}
		}
	}
	if !hasRootDir {
		c.DestDir = rootDir
	}
	return c, nil
}

func main() {
	c := NewConfiguration()

	// lets deal with our flags
	c, err := ProcessArgs(os.Args, c)
	if err != nil {
		e := fmt.Errorf("ERROR: %w", err)
		fmt.Println(e)
		os.Exit(1)
	}

	// now that our args are parsed,
}
