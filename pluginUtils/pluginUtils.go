package pluginUtils

import (
	"errors"
	"fmt"
	"log/syslog"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"time"

	"github.com/greeneg/ca-certificates/configuration"

	"github.com/MakeNowJust/heredoc"
)

func (p PluginUtils) RunTrust(f, t string) (int, error) {
	var format string
	var target string
	switch t {
	case "bundle":
		format = "pem-bundle"
		target = f + ".tmp"
	case "directory-hash":
		format = "pem-directory-hash"
		target = f
	case "java-cacerts":
		format = "java-cacerts"
		target = f + ".new"
	case "openssl":
		format = "openssl-directory"
		target = f
	default:
		return 1, fmt.Errorf("unsupported trust extraction type: %s", t)
	}

	cmd := exec.Command("/usr/bin/trust", "extract", "--format="+format, "--purpose=server-auth", "--filter=ca-anchors", "--overwrite", target)
	err := cmd.Run()
	if err != nil {
		exitCode := -1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		fmt.Printf("ERROR: Could not run command %q: %v", cmd.Args, err)
		return exitCode, err
	}

	return 0, nil
}

func (p PluginUtils) GeneratePemFile(f string) error {
	// first create our heredoc header
	header := heredoc.Doc(`
	#
	# automatically created by $0. Do not edit!
	#
	# Use of this file is deprecated and should only be used as last
	# resort by applications that do not support p11-kit or reading /etc/ssl/certs.
	# You should avoid hardcoding any paths in applications anyways though. Use
	# functions that know the operating system defaults instead:
	#
	# - openssl: SSL_CTX_set_default_verify_paths()
	# - gnutls: gnutls_certificate_set_x509_system_trust(cred)
	#`)

	// read in our new file and then prepend the header to it
	content, err := os.ReadFile(f + ".tmp")
	if err != nil {
		fmt.Println("ERROR: Cannot read file: " + string(err.Error()))
		return err
	}

	text := string(content)
	fileText := header + text
	fileBytes := []byte(fileText)

	// write file back out
	err = os.WriteFile(f, fileBytes, 0644)
	if err != nil {
		fmt.Println("ERROR: Cannot write file: " + string(err.Error()))
		return err
	}

	return nil
}

func (p PluginUtils) IsSymLink(f string) bool {
	s, err := os.Lstat(f)
	if err == nil {
		if s.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func (p PluginUtils) ConfigureEtcSslCaBundlePem(f string) error {
	baseDir := path.Base(f)

	if !p.FileExists(baseDir) {
		err := os.Mkdir(baseDir, 0755)
		if err != nil {
			return err
		}
	}
	err := os.Symlink("../../var/lib/ca-certificates/ca-bundle.pem", f)
	if err != nil {
		return err
	}

	return nil
}

func (p PluginUtils) StatInfo(f string, c configuration.Configuration) time.Time {
	var t time.Time
	s, err := os.Stat(f)
	if err == nil {
		t = s.ModTime()
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			fmt.Println("ERROR: Cannot stat file! " + string(err.Error()))
			if !c.Fresh {
				os.Exit(2)
			} else {
				fmt.Println("NOTICE: file does not exist. 'fresh' option selected. Continuing")
			}
		} else if errors.Is(err, os.ErrNotExist) {
			// set our fileTimeStamp to default
			t = time.Time{}
		}
	}

	return t
}

func (p PluginUtils) CheckSymlinkTarget(f, target string) bool {
	if p.IsSymLink(f) {
		linkTarget, err := os.Readlink(f)
		if err != nil {
			fmt.Println("ERROR: Cannot read symlink: " + string(err.Error()))
			os.Exit(2)
		}
		if linkTarget == target {
			return true
		}
	}
	return false
}

func (p PluginUtils) FileExists(f string) bool {
	_, err := os.Stat(f)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	fmt.Println("ERROR: Cannot stat file! " + string(err.Error()))
	os.Exit(2)
	return false
}

func (p PluginUtils) FindPlugins(c configuration.Configuration, s *syslog.Writer) ([]string, error) {
	var plugins []string

	for _, p := range c.HooksDirList {
		dir := c.DestDir + p
		err := os.Chdir(dir)
		if err != nil {
			if dir == "/etc/ca-certificates/update.d" {
				if errors.Is(err, os.ErrNotExist) {
					// we can effectively skip this case as
					// the /etc/ca-certificates/update.d is
					// an override location
					fmt.Printf("Location %s does not exist\n", dir)
					if c.UseSyslog {
						s.Notice("N: Location /etc/ca-certificates/update.d does not exist. Skipping")
					}
					continue
				}
			}
			fmt.Println(fmt.Errorf("ERROR: %w", err))
			if c.UseSyslog {
				s.Err("E: " + string(err.Error()))
				os.Exit(1)
			}
		}
		fmt.Printf("Checking location %s for plugins\n", dir)
		if c.UseSyslog {
			s.Info("I: Checking location " + dir + " for plugins")
		}
		matches, err := filepath.Glob("*.plugin")
		if err != nil {
			fmt.Println(fmt.Errorf("ERROR: %w", err))
			if c.UseSyslog {
				s.Err("E: " + string(err.Error()))
				os.Exit(1)
			}
		}
		if len(matches) > 0 {
			fmt.Printf("Found: %v", matches)
			if c.UseSyslog {
				s.Info("I: Found " + fmt.Sprintf("%v", matches))
			}
			plugins = append(plugins, matches...)
		} else {
			fmt.Printf("No plugins found in %s\n", dir)
			if c.UseSyslog {
				s.Err("E: No plugins in " + dir)
			}
		}
	}

	return plugins, nil
}

func (p PluginUtils) RunPlugins(plugins []string, c configuration.Configuration) error {
	for _, plugin := range plugins {
		fmt.Printf("plugin: %s", plugin)
	}
	return nil
}
