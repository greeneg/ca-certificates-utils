package pluginUtils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/MakeNowJust/heredoc"

)

func runTrust(f string) (int, error) {
	cmd := exec.Command("/usr/bin/trust", "extract", "--format=pem-bundle", "--purpose=server-auth", "--filter=ca-anchors", "--overwrite", f + ".tmp")
	err := cmd.Run()
	if err != nil {
		fmt.Printf("ERROR: Could not run command '%s': " + string(err.Error()), cmd.Args)
		return cmd.ProcessState.ExitCode(), err
	}

	return 0, nil
}

func generatePemFile(f string) error {
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

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if err == nil {
		return true
	}
	if errors.Is(err, os.ErrNotExist) {
		return false
	}
	return false
}

func isSymLink(f string) bool {
	s, err := os.Lstat(f)
	if err == nil {
		if s.Mode()&os.ModeSymlink != 0 {
			return true
		}
	}
	return false
}

func configureEtcSslCaBundlePem(p string) error {
	baseDir := path.Base(p)

	if !fileExists(baseDir) {
		err := os.Mkdir(baseDir, 0755)
		if err != nil {
			return err
		}
	}
	err := os.Symlink("../../var/lib/ca-certificates/ca-bundle.pem", p)
	if err != nil {
		return err
	}

	return nil
}

func statInfo(f string, c Configuration) time.Time {
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
