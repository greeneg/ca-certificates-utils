package pluginUtils

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/greeneg/ca-certificates-utils/configuration"
	"github.com/greeneg/ca-certificates-utils/logger"
	"github.com/jlaffaye/ftp"

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

func (p PluginUtils) RunCommand(name string, args []string, l logger.Logger) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
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

func (p PluginUtils) downloadHTTP(url, dest string, useTLSValidation bool) error {
	// use net/http to download the file and write it to dest
	// use a temporary file and then move it to the destination to avoid partial
	// files if the download fails
	tr := &http.Transport{}
	if !useTLSValidation {
		tr = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			// set reasonable timeouts
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   30 * time.Second,
			ExpectContinueTimeout: 30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		}
	} else {
		tr = &http.Transport{
			// set reasonable timeouts
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   30 * time.Second,
			ExpectContinueTimeout: 30 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		}
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   30 * time.Second,
	}

	// make the request
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("error making HTTP request: %v", err)
	}
	defer resp.Body.Close()

	// check for non-200 status code
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("received non-200 response code: %d", resp.StatusCode)
	}

	// Create the output file
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// progress writer
	progress := NewProgressWriter(out, resp.ContentLength)

	// create the destination file
	_, err = io.Copy(progress, resp.Body)
	if err != nil {
		return fmt.Errorf("error copying response body to destination: %v", err)
	}

	progress.Finish()

	return nil
}

func (p PluginUtils) downloadFTP(parsedURL *url.URL, dest string) error {
	// use net/ftp to download the file and write it to dest
	// use a temporary file and then move it to the destination to avoid partial
	// files if the download fails
	// Default FTP port
	host := parsedURL.Host
	if !strings.Contains(host, ":") {
		host = host + ":21"
	}

	// Connect to FTP server
	conn, err := ftp.Dial(host, ftp.DialWithTimeout(30*time.Second))
	if err != nil {
		return fmt.Errorf("failed to connect to FTP server: %w", err)
	}
	defer conn.Quit()

	// Login (use anonymous if no credentials provided)
	username := "anonymous"
	password := "anonymous"
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pass, hasPass := parsedURL.User.Password(); hasPass {
			password = pass
		}
	}

	if err := conn.Login(username, password); err != nil {
		return fmt.Errorf("failed to login to FTP server: %w", err)
	}

	// Retrieve the file
	resp, err := conn.Retr(parsedURL.Path)
	if err != nil {
		return fmt.Errorf("failed to retrieve file from FTP server: %w", err)
	}
	defer resp.Close()

	// Get file size if available
	var fileSize int64
	if entry, err := conn.GetEntry(parsedURL.Path); err == nil {
		fileSize = int64(entry.Size)
	}

	// Create the output file
	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer out.Close()

	// Create progress writer
	progress := NewProgressWriter(out, fileSize)

	// Write the response to file with progress tracking
	_, err = io.Copy(progress, resp)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Finish progress display
	progress.Finish()

	return nil
}

func (p PluginUtils) DownloadFile(u, dest string, l logger.Logger) error {
	useTLSValidation := true
	if strings.HasPrefix(u, "http://") {
		useTLSValidation = false
	}
	// download file using net.http and write to destination.
	// use a temporary file and then move it to the destination to avoid partial
	// files if the download fails
	parsedURL, err := url.Parse(u)
	if err != nil {
		return fmt.Errorf("invalid URL: %v", err)
	}
	scheme := strings.ToLower(parsedURL.Scheme)
	switch scheme {
	case "http", "https":
		return p.downloadHTTP(u, dest, useTLSValidation)
	case "ftp":
		return p.downloadFTP(parsedURL, dest)
	default:
		return fmt.Errorf("unsupported URL scheme: %s", scheme)
	}
}

func (p PluginUtils) CleanupTempFiles(tempDir string, l logger.Logger) error {
	err := os.RemoveAll(tempDir)
	if err != nil {
		return fmt.Errorf("failed to clean up temporary files: %w", err)
	}
	return nil
}

func (p PluginUtils) GetCertificateKey(f string) (string, error) {
	cmd := exec.Command("openssl", "x509", "-in", f, "-noout", "-pubkey")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get certificate key: %w", err)
	}
	return string(output), nil
}

func (p PluginUtils) GetCertificateData(f string) (string, error) {
	cmd := exec.Command("openssl", "x509", "-in", f)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get certificate data: %w", err)
	}
	return string(output), nil
}

func (p PluginUtils) GetCertificateText(f string) (string, error) {
	cmd := exec.Command("openssl", "x509", "-in", f, "-text", "-noout")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get certificate text: %w", err)
	}
	return string(output), nil
}

func (p PluginUtils) GetCertificateKeyHash(f string) (string, error) {
	cmd := exec.Command("openssl", "x509", "-in", f, "-noout", "-pubkey")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get certificate public key: %w", err)
	}
	return string(output), nil
}

func (p PluginUtils) ConvertTrustValue(trustType, f string) string {
	return ""
}

func (p PluginUtils) GetTrustValues(f string, keyHash string) (string, string, string, string, string, error) {
	// read in the file and extract the trust values for SSL Client, SSL Server, and Code Signing
	// along with the distrust values for SSL Client and SSL Server

	saTrust := p.ConvertTrustValue("CKA_TRUST_SERVER_AUTH", f)
	smTrust := p.ConvertTrustValue("CKA_TRUST_EMAIL_PROTECTION", f)
	csTrust := p.ConvertTrustValue("CKA_TRUST_CODE_SIGNING", f)

	saDistrust := ""
	smDistrust := ""

	return saTrust, smTrust, csTrust, saDistrust, smDistrust, nil
}

func (p PluginUtils) GetP11Label(f string) string {
	return ""
}

func (p PluginUtils) GetP11Trust(saTrust, smTrust, csTrust string) (string, string, string) {
	return "", "", ""
}

func (p PluginUtils) WriteAnchor(fileName, label, oid, val, certKey, trust string, mozTrust bool,
	saDistrust, smDistrust, certCer, certTxt, saTrust, smTrust, csTrust string, l logger.Logger) error {

	return nil
}

func (p PluginUtils) ProcessCertdataEntryToPEM(fileName string, l logger.Logger) (string, error) {
	tempDir := os.TempDir() + "/certdata_processing"
	l.Log(l.LvlDebug(), fmt.Sprintf("FileName: %s", fileName))
	if fileName != "/tmp/certdata_processing/cert_74.tmp" {
		return "", nil
	} // for debugging. Only process the first cert entry for now. We can expand this later once we have the processing working correctly
	// read in the file, extract the certificate data, convert it to PEM format, and write it back out to the same file
	content, err := os.ReadFile(fileName)
	if err != nil {
		return "", fmt.Errorf("failed to read certdata entry file: %w", err)
	}

	tempPemFile := fileName + ".pem"

	// extract the certificate data from the certdata entry file. The certificate data is between the lines
	// "CKA_VALUE MULTILINE_OCTAL" and "END"
	lines := strings.Split(string(content), "\n")
	var certDataLines []string
	inCertData := false
	for _, line := range lines {
		if strings.HasPrefix(line, "CKA_VALUE MULTILINE_OCTAL") {
			inCertData = true
			continue
		}
		if inCertData {
			if strings.HasPrefix(line, "END") {
				inCertData = false
				// strip new line character from the last line if it exists
				if len(certDataLines) > 0 {
					lastLine := certDataLines[len(certDataLines)-1]
					certDataLines[len(certDataLines)-1] = strings.TrimSuffix(lastLine, "\n")
				}
				break
			}
			certDataLines = append(certDataLines, line)
		}
	}

	// join without separator so the octal stream is contiguous — joining with
	// "\n" inserts single-byte newlines that break the 4-byte stride below
	certData := strings.Join(certDataLines, "")

	// convert the cert data from octal to binary and then to PEM format
	octalBytes := []byte(certData)
	binaryData := make([]byte, 0, len(octalBytes)/4) // each octal byte is 4 characters long (e.g. \123)
	for i := 0; i < len(octalBytes); i += 4 {
		if i+4 > len(octalBytes) {
			break
		}
		octalByte := string(octalBytes[i : i+4])
		var b byte
		fmt.Sscanf(octalByte, "\\%o", &b)
		binaryData = append(binaryData, b)
	}

	// pipe the binary data through openssl to convert it to text format
	cmd := exec.Command("openssl", "x509", "-in", "/dev/stdin", "-text", "-inform", "DER", "-fingerprint")
	cmd.Stdin = strings.NewReader(string(binaryData))
	pemData, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to convert certificate to PEM format: %w", err)
	}

	err = os.WriteFile(tempPemFile, pemData, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write PEM file: %w", err)
	}

	// get values from for certificate
	certKey, err := p.GetCertificateKey(tempPemFile)
	if err != nil {
		return "", fmt.Errorf("failed to get certificate key: %w", err)
	}
	certCer, err := p.GetCertificateData(tempPemFile)
	if err != nil {
		return "", fmt.Errorf("failed to get certificate data: %w", err)
	}
	certTxt, err := p.GetCertificateText(tempPemFile)
	if err != nil {
		return "", fmt.Errorf("failed to get certificate text: %w", err)
	}
	keyHash, err := p.GetCertificateKeyHash(tempPemFile)
	if err != nil {
		return "", fmt.Errorf("failed to get certificate key hash: %w", err)
	}

	l.Log(l.LvlInfo(), fmt.Sprintf("Certificate Key: %s", certKey))
	l.Log(l.LvlInfo(), fmt.Sprintf("Certificate Subject: %s", certCer))
	l.Log(l.LvlInfo(), fmt.Sprintf("Certificate Text: %s", certTxt))
	l.Log(l.LvlInfo(), fmt.Sprintf("Certificate Key Hash: %s", keyHash))

	// get the trust values for the certificate
	saTrust, smTrust, csTrust, saDistrust, smDistrust, err := p.GetTrustValues(fileName, keyHash)
	if err != nil {
		return "", fmt.Errorf("failed to get trust values: %w", err)
	}
	l.Log(l.LvlInfo(), fmt.Sprintf("Trust Values - SSL Client: %s, SSL Server: %s, Code Signing: %s, SSL Client Distrust: %s, SSL Server Distrust: %s", saTrust, smTrust, csTrust, saDistrust, smDistrust))

	// Get the p11 label
	p11Label := p.GetP11Label(fileName)

	// get p11 trust, oid, and value based on the trust values we extracted above
	p11Trust, p11Oid, p11Val := p.GetP11Trust(saTrust, smTrust, csTrust)

	l.Log(l.LvlInfo(), fmt.Sprintf("PKCS#11 Label: %s, Certificate Key Hash: %s", p11Label, keyHash))

	// put the certificate into the trust anchors directory
	anchorFile := fmt.Sprintf("/%s/pki/anchors/%s.p11-kit", tempDir, keyHash)
	mozTrust := true
	err = p.WriteAnchor(anchorFile, p11Label, p11Oid, p11Val, certKey, p11Trust, mozTrust, saDistrust, smDistrust, certCer, certTxt, saTrust, smTrust, csTrust, l)
	if err != nil {
		return "", fmt.Errorf("failed to write anchor: %w", err)
	}

	// import the certificate into the NSS database using certutil

	return tempPemFile, nil
}

func (p PluginUtils) ProcessCertdataAndImportToNSSDB(certDataFile, nssdbDir string, l logger.Logger) error {
	// read certdata.txt, extract the certificates, and import them into the NSS database using certutil
	// certdata.txt has a specialized format.
	// We need to extract the certificates that are marked as "Trust: C,," which means they are trusted for SSL.
	// We also need to convert the certificate from the format in certdata.txt to PEM format before importing it
	// into the NSS database.

	// read certdata.txt
	l.Log(l.LvlInfo(), fmt.Sprintf("Reading in file %s to process", certDataFile))
	content, err := os.ReadFile(certDataFile)
	if err != nil {
		return fmt.Errorf("failed to read certdata.txt: %w", err)
	}

	// split content into lines and process each line
	l.Log(l.LvlInfo(), "Getting the list of certificates")
	lines := strings.Split(string(content), "\n")

	// first, get the line number of each "certificate" entry in certdata.txt
	var certEntries []int
	for i, line := range lines {
		if strings.HasPrefix(line, "# Certificate") {
			certEntries = append(certEntries, i)
		}
	}

	// create a temp working directory for processing the certificates
	tempWorkDir := os.TempDir() + "/certdata_processing"
	err = os.MkdirAll(tempWorkDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary work directory: %w", err)
	}
	// now dump the individual certificates to temporary files
	for _, entry := range certEntries {
		l.Log(l.LvlInfo(), fmt.Sprintf("Entry: %d", entry))
		// create our temporary file name
		fileName := fmt.Sprintf("%s/cert_%d.tmp", tempWorkDir, entry)
		l.Log(l.LvlInfo(), fmt.Sprintf("Processing certificate entry starting at line %d", entry))
		l.Log(l.LvlInfo(), fmt.Sprintf("Outputting to temporary file %s", fileName))

		// using a bufioReader read until we hit a line with the regex ^CKA_TRUST_STEP_UP_APPROVED,
		// which indicates the end of the certificate entry
		var certLines []string
		for i := entry; i < len(lines); i++ {
			certLines = append(certLines, lines[i])
			if strings.HasPrefix(lines[i], "CKA_TRUST_STEP_UP_APPROVED") {
				break
			}
		}

		// now dump these into the temporary file we create in the tempWorkDir
		certContent := strings.Join(certLines, "\n")
		certContent += "\n" // ensure the file ends with a newline for better readability
		err = os.WriteFile(fileName, []byte(certContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write certificate to temporary file: %w", err)
		}

		// process file into PEM format
		pemfile, err := p.ProcessCertdataEntryToPEM(fileName, l)
		if err != nil {
			return fmt.Errorf("failed to process certdata entry to PEM: %w", err)
		}

		// for debugging, print the PEM file content to the logs
		pemContent, err := os.ReadFile(pemfile)
		if err != nil {
			return fmt.Errorf("failed to read PEM file: %w", err)
		}
		l.Log(l.LvlInfo(), fmt.Sprintf("PEM content for certificate entry starting at line %d:\n%s", entry, string(pemContent)))
	}

	return nil
}
