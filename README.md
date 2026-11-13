# WordPress TUI (WPTUI)

Interactive terminal interface for managing local WordPress sites: create sites, tweak settings, install plugins and themes, back up, and restore.

Quick install and overview: [hungnth.github.io/wordpress-tui](https://hungnth.github.io/wordpress-tui/)

## Supported platforms

- macOS: Intel (`amd64`) and Apple Silicon (`arm64`), using Laravel Herd or Flock
- Windows: 64-bit (`amd64`), using Laravel Herd, Flock, or Laragon

> [!NOTE]
> Linux and Windows ARM64 are not supported. Laragon is Windows-only.

## Features

- Create local WordPress sites with database setup, `.test` domain config, optional tweaks, and plugin/theme installs from your Package Service
- Update existing site tweaks, admin credentials, and plugins or themes
- Remove sites safely with a database preview and verification step, directory deletion, and environment cleanup
- Export backups to `backup_path` as full consolidated ZIP archives (`.zip`, SQL dump and files) or All-in-One WP Migration (`.wpress`) files
- Restore `.zip` or `.wpress` archives into fresh, non-colliding local sites from `backup_path` or a custom filesystem path
- Open `config.json` in VS Code (`code --wait`) or open the local cache directory in your file manager

## Prerequisites

WPTUI calls your local tools directly, so make sure these are installed and available on your `PATH`:

- PHP (`php`)
- WP-CLI (`wp`)
- MySQL client/server (`mysql`)
- One local environment (`herd`, `flock`, or `laragon.exe` on Windows), configured via `app_name` and `app_command`, either in `PATH` or given as a full path
- Visual Studio Code (`code`), optional, only needed to edit config from Settings

## Installation

### macOS

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | sh
```

Installs to `~/.local/bin/wptui` and updates `~/.zprofile` or `~/.profile` if needed. To change the install location:

```sh
curl -fsSL https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.sh | WPTUI_INSTALL_DIR="$HOME/bin" sh
```

### Windows

Run in PowerShell:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1
Remove-Item install.ps1
```

Installs to `%LOCALAPPDATA%\Programs\wptui\wptui.exe` and updates your User `PATH`. To change the install location:

```powershell
irm https://raw.githubusercontent.com/HungNth/wordpress-tui/main/scripts/install/install.ps1 -OutFile install.ps1
powershell -NoProfile -ExecutionPolicy RemoteSigned -File .\install.ps1 -InstallDir 'C:\Tools\wptui'
Remove-Item install.ps1
```

> [!WARNING]
> Review remote scripts before running them in your shell:
>
> - [`scripts/install/install.sh`](scripts/install/install.sh)
> - [`scripts/install/install.ps1`](scripts/install/install.ps1)

> [!NOTE]
> Release archives are verified with SHA-256 checksums during install, though the binaries are not code-signed or notarized for Apple Gatekeeper or Windows SmartScreen.

## Getting started

Launch the TUI:

```sh
wptui
```

### Command-line flags

Pass one flag to run a CLI command without launching the TUI or loading `config.json`:

- `-v`, `--version`: print the installed WPTUI release version
- `-u`, `--update`: check the `main` branch manifest and print the GitHub Release link when a newer version exists (read-only; never downloads or installs)
- `-c`, `--config`: print the expected `config.json` path without creating directories or files
- `-h`, `--help`: print usage

### First-run setup

First launch opens a setup wizard for your local paths, database credentials, and preferences, saving to:

- macOS: `~/.config/wptui/config.json`
- Windows: `%USERPROFILE%\.config\wptui\config.json`

You can change these later by editing `config.json`, re-running the wizard, or opening **Settings** in the main menu.

### Local environments

You must choose one local development environment during setup:

- `herd` (Laravel Herd) uses your parked sites directory (default `$HOME/Herd`), manages `.test` domains, runs `herd secure <slug>` for TLS certificates on create, and runs `herd unsecure <slug>` (best-effort) on delete.
- `flock` (Flock) defaults to `$HOME/Sites`, checks `flock sites` before running `flock link <slug>` and `flock secure <slug>` (TLS) so it won't overwrite existing registrations, and only runs `flock unsecure <slug>` and `flock unlink <slug>` on sites WPTUI created. Browser launching uses your OS default URL opener.
- `laragon` (Laragon, Windows only) expects sites inside Laragon's Document Root (default `C:\laragon\www`), reloads virtual hosts via `app_command` (`laragon.exe reload`) after create, restore, and delete, and uses `https://<slug>.test` URLs. For HTTPS, enable Auto SSL in Laragon and set its Document Root hostname pattern to `{name}.test`; WPTUI does not generate vhost templates or configure Auto SSL.

### Packages and themes

Plugins and themes come strictly from `package_service_url`. If a package fails to resolve, WPTUI reports an error and never falls back to arbitrary WordPress.org slugs.

If `package_service_url` is unset, fails to fetch, or returns an empty catalog, WPTUI hides package steps during site creation and package actions in site configuration. When a catalog only contains plugins, theme selection is omitted (and vice versa). Searchable pickers keep your selections across query changes, and picking an active theme is optional: the admin interface stays accessible even when the frontend has no active theme assigned.

### Backup and restore

Backups export to `backup_path` as immutable Full ZIP (`.zip`, SQL dump and files) or All-in-One WP Migration (`.wpress`) archives.

During restore, `[Enter custom path...]` appears first so you can enter or paste any `.zip` or `.wpress` path on disk, even when `backup_path` is empty, missing, or unreadable. Outer whitespace and one balanced pair of double quotes (such as Windows Explorer "Copy as path" values like `"D:\Downloads\client-site.zip"`) are trimmed on confirmation. External archives are validated and read in place without copying, moving, renaming, or modifying them.

### Config schema

`config.json` stores your environment under `app_name` and its executable under `app_command`:

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
> Config validation rejects missing/invalid environment names and legacy keys (`used_herd`, `themes`, `plugins`, `default_theme_slug`). WPTUI has no automatic migration, aliases, or backward-compatibility fallbacks; if upgrading from an older version, re-run first-run setup or update `config.json` directly.

## Development

Requires **Go 1.27.0** or later:

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
