# WordPress TUI (WPTUI)

WordPress TUI manages local WordPress **Websites** through an interactive terminal interface. It provides guided workflows to provision fresh Websites, configure existing installations, manage backups, restore archives, and maintain local development settings.

## Supported Platforms and Environments

- **macOS:** Intel (`amd64`) and Apple Silicon (`arm64`)
  - Supported environments: **Laravel Herd**, **Flock**
- **Windows:** 64-bit (`amd64`)
  - Supported environments: **Laravel Herd**, **Flock**, **Laragon**

> [!NOTE]
> Linux and Windows ARM64 are not currently supported. Laragon is available on Windows only.
## Features

- **Provisioning:** Create new local WordPress Websites with database creation, WordPress installation, local `.test` domain configuration (via Laravel Herd, Flock, or Laragon), and optional WordPress tweaks and API-driven Package installations.
- **Website Configuration:** Apply tweaks to existing Websites, update administrator credentials, and install or upgrade plugins and themes from the configured Package Service.
- **De-provisioning:** Safely remove Websites with database preview and verification, directory removal, and environment cleanup (Herd unsecure, Flock unsecure/unlink, or Laragon reload).
- **Website Backup:** Export immutable **Backup Archives** to `backup_path` as full consolidated ZIP archives (database SQL dump and files) or All-in-One WP Migration (`.wpress`) archives.
- **Website Restoration:** Reconstitute Websites into fresh, non-colliding local Websites from any compatible **Restore Archive** (`.zip` or `.wpress`), either discovered in `backup_path` or specified via a custom absolute filesystem path.
- **Application Settings:** Open the application configuration in VS Code (`code --wait`) or view the local user cache directory in the native file manager.

## Prerequisites

WPTUI orchestrates native tools on your machine. Ensure the following runtime dependencies are installed and available in your `PATH`:

- **PHP** (`php`)
- **WP-CLI** (`wp`)
- **MySQL Client / Server** (`mysql`)
- **Local Development Environment executable:**
  - **Laravel Herd** (`herd`)
  - **Flock** (`flock`)
  - **Laragon** (`laragon.exe` on Windows)
  Configured via `app_name` and `app_command`. The executable must be installed and accessible in your `PATH` or specified via an editable path.
- **Visual Studio Code** (`code`) _(optional, required only for editing configuration from Settings)_
## Installation

Install WPTUI for the current user using the official installation scripts:

### macOS

Open your terminal and run:

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | sh
```

By default, WPTUI installs to `~/.local/bin/wptui` and updates your shell profile (`~/.zprofile` or `~/.profile`) if needed. To customize the install location:

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | WPTUI_INSTALL_DIR="$HOME/bin" sh
```

### Windows

Open PowerShell and run:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1
Remove-Item install.ps1
```

By default, WPTUI installs to `%LOCALAPPDATA%\Programs\wptui\wptui.exe` and updates your User `PATH`. To customize the install location, download and run the script with `-InstallDir`:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1 -InstallDir 'C:\Tools\wptui'
Remove-Item install.ps1
```

> [!WARNING]
> Always review remote scripts before executing them in your shell:
>
> - [`scripts/install/install.sh`](scripts/install/install.sh)
> - [`scripts/install/install.ps1`](scripts/install/install.ps1)

> [!NOTE]
> Published release archives are verified with SHA-256 checksums during installation. However, executables are not currently code-signed or notarized for Apple Gatekeeper or Windows SmartScreen.

## Getting Started

Run WPTUI directly from your terminal:

```sh
wptui
```

### Command-Line Flags

Pass at most one action flag to run a non-interactive CLI action without launching the TUI or loading `config.json`:

- `-v`, `--version`: Print the installed WPTUI release version.
- `-u`, `--update`: Perform a read-only **Application Update Check** against the canonical `main` branch manifest and print the GitHub Release link when a newer version exists; it never downloads or installs a release.
- `-c`, `--config`: Print the expected absolute path to `config.json` without creating directories or files.
- `-h`, `--help`: Print command-line usage.

### First-Run Configuration

When launched for the first time, WPTUI automatically starts an interactive configuration wizard to establish your local development paths, database credentials, and preferences. Configuration is saved locally to:

- **macOS:** `~/.config/wptui/config.json`
- **Windows:** `%USERPROFILE%\.config\wptui\config.json`

You can modify these settings anytime directly, through the first-run wizard, or via **Settings** in the main menu.


### Local Development Environments

WPTUI requires an explicit Local Development Environment selection:

- **Laravel Herd (`herd`)**:
  - Trusts the configured Websites directory being parked (default suggestion: `$HOME/Herd`).
  - Automatically manages `.test` domains.
  - Executes `herd secure <slug>` during provisioning to issue TLS certificates, and best-effort `herd unsecure <slug>` during de-provisioning.

- **Flock (`flock`)**:
  - Websites directory defaults to `$HOME/Sites`.
  - Registers sites via `flock link <slug>` and secures TLS via `flock secure <slug>`.
  - Source integration inspects `flock sites` status prior to linking to prevent replacing pre-existing site registrations.
  - Rollback and de-provisioning reverse only resources created and owned by WPTUI (`flock unsecure <slug>` and `flock unlink <slug>`), protecting pre-existing registrations.
  - Browser launching uses the native operating system URL launcher rather than an emulated CLI command.

- **Laragon (`laragon`)** _(Windows only)_:
  - Websites must be created within Laragon's Document Root (default suggestion: `C:\laragon\www`).
  - Reloads virtual hosts via `app_command` (`laragon.exe reload`) following site creation, restoration, and deletion.
  - Emits canonical `https://<slug>.test` URLs.
  - **HTTPS Prerequisite:** Laragon HTTPS strictly requires that **Auto SSL** is externally enabled in Laragon's settings and the Document Root is configured with the `{name}.test` hostname pattern. WPTUI does not generate virtual-host templates or configure Auto SSL.

### Packages & Themes

- **API-Only Source**: All selectable plugins and themes originate strictly from the configured Package Service API (`package_service_url`). Static package lists in `config.json` and the built-in default theme have been removed.
- **Conditional Visibility**: If no Package Service URL is configured, or if catalog fetching fails or returns an empty catalog, package selection steps in Provisioning and package actions in Website Configuration are hidden entirely. Provisioning proceeds cleanly without packages or an active theme.
- **Independent Category Controls**: If the catalog contains only plugins, theme selection is omitted; if only themes, plugin selection is omitted.
- **Search & Selection Accumulation**: Separate searchable pickers for plugins and themes preserve selected items across query modifications.
- **Optional Active Theme**: When one or more themes are selected, users can optionally choose at most one as the Active Theme (or choose none). Provisioning without an active theme is fully supported; the administration interface is accessible while the frontend indicates no active theme is assigned.
- **Deterministic Provenance**: Failed package resolutions are reported as errors; WPTUI never falls back to arbitrary WordPress.org slugs.

### Website Backup & Restoration

- **Backup Archives vs. Restore Archives**: WPTUI exports immutable **Backup Archives** to the configured `backup_path`. During Website Restoration, any compatible `.zip` (Full ZIP) or `.wpress` (All-in-One WP Migration) file can be selected as a **Restore Archive**.
- **Custom Restore Archive Paths**: The Restore Wizard always lists `[Enter custom path...]` first, allowing you to enter or paste an absolute path to a Restore Archive stored anywhere on disk (even when `backup_path` is empty, missing, or unreadable).
- **Quoted Clipboard Paths**: Surrounding whitespace and one balanced pair of outer double quotes (such as Windows Explorer **Copy as path** values like `"D:\Downloads\client-site.zip"`) are normalized automatically on confirmation.
- **Source Preservation**: External Restore Archives are validated before restoration starts and read directly in place without being copied, moved, renamed, or modified.

### Configuration Schema & Strict Validation

`config.json` stores your environment choice under `app_name` and the executable under `app_command`:

```json
{
  "app_name": "flock",
  "app_command": "flock",
  "websites_path": "/Users/developer/Sites",
  "mysql_host": "127.0.0.1",
  "mysql_port": 3306,
  "mysql_user": "root",
  "mysql_password": ""
}
```

> [!IMPORTANT]
> Configuration validation strictly rejects missing/invalid environment names and legacy schema (`used_herd`, `themes`, `plugins`, `default_theme_slug`). WPTUI provides no automatic migration, aliases, or backward-compatibility fallbacks. If upgrading from an older version, run first-run setup or update your configuration file directly.
## Development

To build and test WPTUI from source, ensure you have **Go 1.27.0** or later installed:

```sh
# Clone the repository
git clone https://github.com/HungNth/wordpress-tui.git
cd wordpress-tui

# Build the executable for your current OS
make build

# Run the application
make run

# Run tests
make test

# Run tests with race detection
make test-race

# Format and vet source code
make fmt
make vet

# Clean build artifacts
make clean
```
