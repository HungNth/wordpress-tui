# WordPress TUI (WPTUI)

WordPress TUI manages local WordPress **Websites** through an interactive terminal interface. It provides guided workflows to provision fresh Websites, configure existing installations, manage backups, restore archives, and maintain local development settings.

## Supported Platforms

- **macOS:** Intel (`amd64`) and Apple Silicon (`arm64`)
- **Windows:** 64-bit (`amd64`)

> [!NOTE]
> Linux and Windows ARM64 are not currently supported.

## Features

- **Provisioning:** Create new local WordPress Websites with database creation, WordPress installation, local `.test` domain configuration, and optional WordPress tweaks and Package installations.
- **Website Configuration:** Apply tweaks to existing Websites, update administrator credentials, and install or upgrade plugins and themes.
- **De-provisioning:** Safely remove Websites with database preview and verification, directory removal, and optional Laravel Herd TLS cleanup.
- **Website Backup:** Create full consolidated ZIP backups (database SQL dump and files) or export All-in-One WP Migration (`.wpress`) archives.
- **Website Restoration:** Reconstitute Websites from full ZIP backups or All-in-One WP Migration archives into fresh, non-colliding local Websites.
- **Application Settings:** Open the application configuration in VS Code (`code --wait`) or view the local user cache directory in the native file manager.

## Prerequisites

WPTUI orchestrates native tools on your machine. Ensure the following runtime dependencies are installed and available in your `PATH`:

- **PHP** (`php`)
- **WP-CLI** (`wp`)
- **MySQL Client / Server** (`mysql`)
- **Laravel Herd** (`herd`) _(optional, required only when Herd integration is enabled)_
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

### First-Run Configuration

When launched for the first time, WPTUI automatically starts an interactive configuration wizard to establish your local development paths, database credentials, and preferences. Configuration is saved locally to:

- **macOS:** `~/.config/wptui/config.json`
- **Windows:** `%USERPROFILE%\.config\wptui\config.json`

You can modify these settings anytime directly, through the first-run wizard, or via **Settings** in the main menu.

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
