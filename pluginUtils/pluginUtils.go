package pluginUtils

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/greeneg/ca-certificates-utils/configuration"
	"github.com/greeneg/ca-certificates-utils/logger"

	"github.com/MakeNowJust/heredoc"
)

func (p PluginUtils) RunTrust(f, t string, l logger.Logger) (int, error) {
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
		exitCode := 1
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}
		l.Log(l.LvlError(), fmt.Sprintf("Could not run command %q: %v", cmd.Args, err))
		return exitCode, err
	}

	return 0, nil
}

func (p PluginUtils) GeneratePemFile(f string, l logger.Logger) error {
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
		l.Log(l.LvlError(), fmt.Sprintf("Cannot read file: %v", err))
		return err
	}

	text := string(content)
	fileText := header + text
	fileBytes := []byte(fileText)

	// write file back out
	err = os.WriteFile(f, fileBytes, 0644)
	if err != nil {
		l.Log(l.LvlError(), fmt.Sprintf("Cannot write file: %v", err))
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

func (p PluginUtils) ConfigureEtcSslCaBundlePem(f string, l logger.Logger) error {
	baseDir := filepath.Dir(f)

	e, err := p.FileExists(f)
	if err != nil {
		l.Log(l.LvlError(), fmt.Sprintf("Cannot check if file exists: %v", err))
		return err
	}

	if !e {
		err := os.MkdirAll(baseDir, 0755)
		if err != nil {
			l.Log(l.LvlError(), fmt.Sprintf("Cannot create directory: %v", err))

			return err
		}
	}
	err = os.Symlink("../../var/lib/ca-certificates/ca-bundle.pem", f)
	if err != nil {
		l.Log(l.LvlError(), fmt.Sprintf("Cannot create symlink: %v", err))
		return err
	}

	return nil
}

func (p PluginUtils) StatInfo(f string, c configuration.Configuration, l logger.Logger) time.Time {
	var t time.Time
	s, err := os.Stat(f)
	if err == nil {
		t = s.ModTime()
	} else {
		if !errors.Is(err, os.ErrNotExist) {
			l.Log(l.LvlError(), fmt.Sprintf("Cannot stat file: %v", err))
			if !c.Fresh {
				os.Exit(2)
			} else {
				l.Log(l.LvlNotice(), "File does not exist. 'fresh' option selected. Continuing")
			}
		} else if errors.Is(err, os.ErrNotExist) {
			// set our fileTimeStamp to default
			t = time.Time{}
		}
	}

	return t
}

func (p PluginUtils) CheckSymlinkTarget(f, target string, l logger.Logger) bool {
	if p.IsSymLink(f) {
		linkTarget, err := os.Readlink(f)
		if err != nil {
			l.Log(l.LvlError(), fmt.Sprintf("Cannot read symlink: %v", err))
			os.Exit(2)
		}
		if linkTarget == target {
			return true
		}
	}
	return false
}

func (p PluginUtils) FileExists(f string) (bool, error) {
	_, err := os.Stat(f)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (p PluginUtils) FindPlugins(c configuration.Configuration, l logger.Logger) ([]string, error) {
	var plugins []string

	for _, p := range c.HooksDirList {
		dir := filepath.Join(c.DestDir, p)
		_, err := os.Stat(dir)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				l.Log(l.LvlNotice(), fmt.Sprintf("Location %s does not exist. Skipping", dir))
				continue
			}
			l.Log(l.LvlError(), fmt.Sprintf("Cannot stat directory: %v", err))
			continue
		}
		l.Log(l.LvlInfo(), fmt.Sprintf("Checking location %s for plugins", dir))
		matches, err := filepath.Glob(filepath.Join(dir, "*.plugin"))
		if err != nil {
			l.Log(l.LvlError(), fmt.Sprintf("Cannot glob directory: %v", err))
			continue
		}
		if len(matches) > 0 {
			l.Log(l.LvlInfo(), fmt.Sprintf("Found: %v", matches))
			plugins = append(plugins, matches...)
		} else {
			l.Log(l.LvlInfo(), fmt.Sprintf("No plugins found in %s", dir))
		}
	}

	return plugins, nil
}

func (p PluginUtils) RunPlugins(plugins []string, c configuration.Configuration, l logger.Logger) {
	// convert c to json once and pass it to each plugin via stdin
	jsonStr, err := c.ToJson(c)
	if err != nil {
		l.Log(l.LvlError(), fmt.Sprintf("%v", err))
		// swallow the error. If the plugin dies, keep going with the next one.
		// The plugin should log its own errors and we don't want to stop all
		// plugins if one has an error with the configuration
		return
	}

	for _, plugin := range plugins {
		l.Log(l.LvlInfo(), fmt.Sprintf("plugin: %s", plugin))
		cmd := exec.Command(plugin)
		cmd.Stdin = strings.NewReader(jsonStr)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			exitCode := 1
			if cmd.ProcessState != nil {
				exitCode = cmd.ProcessState.ExitCode()
			}
			l.Log(l.LvlError(), fmt.Sprintf("Could not run plugin %s: %v. Exit code: %d", plugin, err, exitCode))
			continue
		}
	}
}

func (p PluginUtils) EnsureVarEndsWithSlash(v string) string {
	if len(v) > 0 && v[len(v)-1] != '/' {
		return v + "/"
	}
	return v
}

func (p PluginUtils) CheckRequiredTools() error {
	for _, tool := range p.RequiredTools {
		_, err := exec.LookPath(tool)
		if err != nil {
			return fmt.Errorf("required tool %s not found in PATH", tool)
		}
	}
	return nil
}
