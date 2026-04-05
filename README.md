# ca-certificates - Utilities for system wide CA certificate installation

The update-ca-certificates command and its plugins are intended to keep the certificate stores of various components in sync with the system CA certificates.

The canonical source of CA certificates is normally managed by `p11-kit`. By default `p11-kit` looks in /usr/share/pki/trust/ and /etc/pki/trust/ for root trust certificate anchors, however there could be other plugins that serve as source for certificates as well.

## Supported Certificate Stores

The update-ca-certificates command supports a number of legacy certificate stores for applications that don't integrate with `p11-kit` directly yet. It does so by generating the certificate stores in /var/lib/ca-certificates and generating filesystem symbolic links in the locations where applications expect those files to exist.

- `/etc/ssl/certs`: Hashed directory readable by OpenSSL. Only for legacy applications. Only contains CA certificates for server identification purposes. Avoid using this within new applications.
- `/etc/ssl/ca-bundle.pem`: Concatenated bundle of CA certificates for server identification purposes. Avoid using this in new applications.
- java-cacerts: Key store fore Java. Only filled with CA certificates with purpose server identification.
- openssl: hashed directory with CA certificates of all purposes. Your system openSSL knows how to read that, don't hardcode the path! Call `SSL_CTX_set_default_verify_paths()` instead within your application.

## Plugins

This version of the update-ca-certificates command uses a plugin-based architecture instead of shell "hooks". Plugins are standalone executables with a `.plugin` extension that receive configuration data via `stdin` as JSON.

### Plugin Locations

Plugins are discovered in the following directories:

- `/usr/lib/ca-certificates/update.d/*.plugin` - System-provided plugins
- `/etc/ca-certificates/update.d/*.plugin` - Administrator-provided override plugins

### Plugin Architecture

Each plugin:

1. **Receives configuration via stdin** as a JSON-encoded Configuration object
2. **Processes certificates** based on the configuration settings
3. **Logs to syslog** using the DAEMON facility
4. **Uses shared modules**:
   - `github.com/greeneg/ca-certificates/configuration` - Configuration data structures
   - `github.com/greeneg/ca-certificates/pluginUtils` - Common plugin utilities

### Configuration JSON Structure

The JSON structure passed to plugins contains the following fields:

```json
{
  "stateDir": "var/lib/ca-certificates",
  "pluginDirectories": [
    "etc/ca-certificates/update.d",
    "usr/lib/ca-certificates/update.d"
  ],
  "verbose": false,
  "rootDir": "/",
  "clean": false,
  "useSyslog": true,
  "logFile": "/var/log/update-ca.log",
  "syslogFacility": "DAEMON",
  "defaultSyslogLevel": "INFO"
}
```

### Available Plugins

The following plugins are included:

- **certbundle** - Generates PEM bundle files for applications
- **etcssl** - Manages /etc/ssl symlinks for legacy OpenSSL applications
- **java** - Manages Java cacerts keystore
- **openssl** - Manages OpenSSL hashed certificate directories

### Writing Custom Plugins

To create a custom plugin:

1. Create an executable that reads JSON from stdin (handle both newline-terminated and non-terminated input)
2. Parse the JSON into a Configuration struct
3. Use the pluginUtils module for common operations (file checking, trust extraction, etc.)
4. Name your plugin with a `.plugin` extension
5. Place it in `/etc/ca-certificates/update.d/` for local customization

### Transactional Update Support

When the `TRANSACTIONAL_UPDATE` environment variable is set to "true", "yes", or "1", the update-ca-certificates tool will skip plugin execution and create a lock file (`/etc/pki/trust/.updated`). This prevents conflicts during package management operations.

## Differences between MidgardOS and openSUSE

- Rewritten in Golang for better performance and maintainability.
- Uses a modular plugin architecture with shared Go modules (`configuration` and `pluginUtils`).
- Plugins are standalone executables (`.plugin` files) that receive JSON configuration via stdin.
- Plugins use syslog for logging by default (DAEMON facility).
- Command requires root privileges to execute (checks at startup).
- Supports transactional updates via the `TRANSACTIONAL_UPDATE` environment variable.
- Packages are expected to install their CA certificates in `/usr/share/pki/trust/anchors` or `/usr/share/pki/trust` (no extra subdir) instead of the deprecated `/usr/share/ca-certificates/<vendor>` now. The anchors subdirectory is for regular pem files, the directory one above for pem files in openssl's 'trusted' format.
- The older configuration file from Debian, `/etc/ca-certificates.conf` is no longer supported. To block the use of certificates you don't want to use, create a filesystem symbolic link to the certificates you don't want in `/etc/pki/trust/blocklist`.

## Differences to Debian

- The `/etc/ca-certificates.conf` configuration file is not supported.
- Plugins are standalone Go executables (`.plugin` files) rather than shell scripts.
- Plugins receive full configuration data as JSON via stdin, not a list of changed certificates.
- All command line options are encoded in the JSON configuration passed to plugins.
- All certificate stores are created via the plugin system.
- Plugins use shared Go modules for common functionality.
- Root privileges are required to run the update-ca-certificates command.
- Syslog integration is built-in and enabled by default.
