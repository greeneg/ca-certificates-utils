package pluginUtils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
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
	}

	cmd := exec.Command("/usr/bin/trust", "extract", "--format="+format, "--purpose=server-auth", "--filter=ca-anchors", "--overwrite", target)
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Could not run command '%s': "+string(err.Error()), cmd.Args)
		return cmd.ProcessState.ExitCode(), err
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
