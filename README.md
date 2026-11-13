# WordPress TUI (WPTUI)

WPTUI creates, configures, removes, backs up, and restores local WordPress sites from the terminal.

[Install guide and overview](https://hungnth.github.io/wordpress-tui/)

## Supported platforms

- macOS: Intel (`amd64`) and Apple Silicon (`arm64`), using Laravel Herd or Flock
- Windows: 64-bit (`amd64`), using Laravel Herd, Flock, or Laragon

> [!NOTE]
> Linux and Windows ARM64 are not supported. Laragon is Windows-only.

## Features

- Create sites with a database, `.test` domain, optional tweaks, and plugins or themes from your Package Service
- Update tweaks, admin credentials, plugins, or themes
- Preview the database and verify the target before deletion; WPTUI removes the site directory and cleans up the local environment
- Back up to `backup_path` as `.zip` or `.wpress`, then restore either format into a fresh site from `backup_path` or any filesystem path; name and path collisions are rejected
- Open `config.json` in VS Code (`code --wait`) or the local cache directory in your file manager

## Prerequisites

WPTUI calls these local tools:

- PHP (`php`)
- WP-CLI (`wp`)
- MySQL client and server (`mysql`)
- One local environment configured through `app_name` and `app_command`: Laravel Herd (`herd`), Flock (`flock`), or Laragon (`laragon.exe`, Windows only). `app_command` may be on `PATH` or given as a full path.
- Visual Studio Code (`code`), optional and only used to edit config from Settings

## Installation

### macOS

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | sh
```

By default, the installer writes `~/.local/bin/wptui` and updates `~/.zprofile` or `~/.profile` when needed. To use another directory:

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | WPTUI_INSTALL_DIR="$HOME/bin" sh
```

### Windows

Run this in PowerShell:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1
Remove-Item install.ps1
```

By default, the installer writes `%LOCALAPPDATA%\Programs\wptui\wptui.exe` and updates your user `PATH`. To use another directory:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1 -InstallDir 'C:\Tools\wptui'
Remove-Item install.ps1
```

> [!WARNING]
> Read the installer scripts before running them:
>
> - [`scripts/install/install.sh`](scripts/install/install.sh)
> - [`scripts/install/install.ps1`](scripts/install/install.ps1)

> [!NOTE]
> The installer verifies release archives with SHA-256 checksums. The binaries are not code-signed or notarized for Apple Gatekeeper or Windows SmartScreen.

## Getting started

Start the TUI:

```sh
wptui
```

### Command-line flags

Pass one CLI flag to exit before the TUI starts or `config.json` loads:

- `-v`, `--version`: print the installed WPTUI version
- `-u`, `--update`: check the `main` branch manifest and print the GitHub Release link when a newer version exists, without downloading or installing it
- `-c`, `--config`: print the expected `config.json` path without creating directories or files
- `-h`, `--help`: print usage

### First-run setup

On first launch, the setup wizard asks for local paths, database credentials, and preferences. It saves the result to:

- macOS: `~/.config/wptui/config.json`
- Windows: `%USERPROFILE%\.config\wptui\config.json`

Later, edit `config.json`, run the wizard again, or open **Settings** from the main menu.

### Local environments

Setup requires one local development environment:

- `herd` (Laravel Herd) uses your parked sites directory (default `$HOME/Herd`) and manages `.test` domains. Site creation calls `herd secure <slug>` for TLS; deletion calls `herd unsecure <slug>` on a best-effort basis.
- `flock` (Flock) defaults to `$HOME/Sites`. Before calling `flock link <slug>` and `flock secure <slug>`, WPTUI checks `flock sites` and leaves existing registrations alone. On deletion, it calls `flock unsecure <slug>` and `flock unlink <slug>` only for sites WPTUI created. Site URLs open through the OS default URL handler.
- `laragon` (Laragon, Windows only) expects sites under its Document Root (default `C:\laragon\www`). WPTUI runs `app_command` (`laragon.exe reload`) after creating, restoring, or deleting a site and uses `https://<slug>.test` URLs. For HTTPS, enable Auto SSL and set Laragon's Document Root hostname pattern to `{name}.test`; WPTUI does not create vhost templates or configure Auto SSL.

### Packages and themes

All plugins and themes must come from `package_service_url`. If a package cannot be resolved, WPTUI reports the error instead of treating it as a WordPress.org slug.

Package controls are hidden when `package_service_url` is unset, unavailable, or returns an empty catalog. A plugins-only or themes-only catalog hides the irrelevant picker. Search selections remain selected when the query changes. Choosing an active theme is optional; WordPress admin remains available even when the frontend has no active theme.

### Backup and restore

WPTUI exports immutable backups to `backup_path` as a Full ZIP (`.zip`, SQL dump and files) or All-in-One WP Migration (`.wpress`) archive.

The restore screen lists `[Enter custom path...]` first. You can paste any `.zip` or `.wpress` path even when `backup_path` is empty, missing, or unreadable. On confirmation, WPTUI trims outer whitespace and one balanced pair of double quotes, including Windows Explorer "Copy as path" values such as `"D:\Downloads\client-site.zip"`. External archives are validated and read in place, never copied, moved, renamed, or modified.

### Config schema

`config.json` identifies the local environment with `app_name` and its executable with `app_command`:

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
> Validation rejects missing or invalid environment names and legacy keys (`used_herd`, `themes`, `plugins`, `default_theme_slug`). There is no automatic migration, alias, or compatibility fallback. After upgrading from an older version, run first-run setup again or edit `config.json` directly.

## Development

Development requires Go 1.27.0 or later:

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
