# WordPress TUI

WordPress TUI manages local WordPress websites through a terminal interface.

## Language

**Website**:
A local WordPress installation managed by WordPress TUI, with its own directory, database, and `.test` hostname.
_Avoid_: Project, instance, site

**Website Name**:
The human-readable name used as the WordPress title of a Website.
_Avoid_: Project name, site name

**Website Slug**:
The user-confirmed 1–63 character ASCII param-case identifier used by the Website directory and `.test` hostname. A newly provisioned Website initially uses it as the database name, but an existing Website may reference a differently named database.
_Avoid_: Project slug, folder name

**Website Database**:
The MySQL database configured for and referenced by a Website; its identity is independent of the Website Slug.
_Avoid_: Slug database, assumed database

**Package**:
An installable WordPress plugin or theme offered by the configured package service.
_Avoid_: Extension, asset

**Package Catalog**:
The complete collection of Packages returned by the configured package service for local filtering and selection.
_Avoid_: Package list, repository

**Package Selection Accumulator**:
The preserved collection of user-chosen Packages maintained across multiple search queries during interactive selection.
_Avoid_: Temporary cart, search buffer

**Package Cache**:
The local collection of verified Package archives retained to avoid repeated downloads and to permit a stale fallback during network failures. It keeps only the latest cached version for each Package identity.
_Avoid_: Download folder, temporary files

**Cache Entry**:
The manifest record that associates a Package type and slug with its cached version, verified archive, size, checksum, and download time.
_Avoid_: Download record, package metadata

**Core Archive**:
The verified, content-stripped WordPress core zip archive retained in the Core Cache for Website Provisioning.
_Avoid_: Core zip, WordPress bundle, WordPress installer

**Core Cache**:
The local storage containing the verified Core Archive and its manifest record to provision Websites without repeated network downloads.
_Avoid_: Core folder, WordPress cache, temporary core

**Provisioning**:
The operation that creates a Website atomically through its critical core, database, Herd, and Package steps, then applies WordPress tweaks on a best-effort basis. Individual tweak failures are reported without removing the Website.
_Avoid_: Setup, scaffolding

**De-provisioning**:
The operation that destroys one or more existing Websites, safely unsecuring Herd TLS, dropping the accurately extracted database, and removing the Website directory.
_Avoid_: Cleanup, wipe, uninstall

**Visual Theme Palette**:
The unified terminal color system that styles focus navigation, selected items, and status indicators across interactive forms and completion summaries.
_Avoid_: Skin, custom CSS

**Website Configuration**:
The post-provisioning management workflow that modifies an existing Website by applying system tweaks, updating administrative credentials, or installing packages.
_Avoid_: Site edit, site modification, site settings

**Administrative Credential Reconciler**:
The mechanism that synchronizes administrative username, password, and email for an existing Website, updating the user login directly via MySQL prepared statements and updating password and email via WP-CLI.
_Avoid_: User updater, account changer

**Configuration Progress Reporter**:
The visual progress indicator that displays real-time step descriptions and spinners during Website Configuration actions before presenting the final color-coded summary.
_Avoid_: Loading bar, task monitor

**Version-Aware Package Upgrader**:
The mechanism that queries currently installed WordPress plugin or theme versions, compares them against target package versions, and conditionally performs clean installs or force upgrades while skipping redundant reinstalls.
_Avoid_: Package overwriter, package refresher

**Application Settings**:
The menu workflow that facilitates system-level maintenance tasks, including launching an external editor for the configuration file and opening the system cache directory.
_Avoid_: Preferences, options, app config

**External Launcher**:
The platform-aware subsystem responsible for safely launching external tools such as code editors and file managers.
_Avoid_: Shell runner, process spawner

**Website Backup**:
The operation that captures the complete state of an existing Website—either by creating a consolidated zip archive containing database and source code, or by orchestrating an All-in-One WP Migration package export.
_Avoid_: Snapshot, site dump, archive bundle

**Backup Archive**:
The final immutable backup artifact (`.zip` or `.wpress`) relocated to `backup_path` under standardized naming conventions.
_Avoid_: Dump file, backup folder, zip target

**Website Restoration**:
The operation that reconstitutes a Website from a Backup Archive into a fresh, non-colliding Website, reconciles database configuration, synchronizes administrative credentials, and configures local development hostnames.
_Avoid_: Site unarchive, backup unpack, website overwrite

**Full ZIP Restore Engine**:
The restoration mechanism that reconstitutes a Website from a consolidated zip archive containing WordPress source files and a database dump.
_Avoid_: Zip unpacker, raw site extractor

**AI1WM Restore Engine**:
The restoration mechanism that reconstitutes a Website from an All-in-One WP Migration package archive.
_Avoid_: Wpress runner, migration importer

**WordPress Root**:
The specific directory inside an archive that represents the root of the WordPress core installation.
_Avoid_: App root, site root, extracted folder

