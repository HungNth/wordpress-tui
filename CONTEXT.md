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

