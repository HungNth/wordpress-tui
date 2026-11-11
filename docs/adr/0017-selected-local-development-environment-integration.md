# 0017: Selected Local Development Environment Integration and API Package Catalog

## Status
Accepted (supersedes [ADR-0005](0005-herd-path-trust-and-terminal-tls.md) general environment policy; updates [ADR-0002](0002-atomic-provisioning.md), [ADR-0008](0008-configured-database-identity-during-deprovisioning.md), [ADR-0009](0009-head-search-option-and-live-package-selection.md), [ADR-0012](0012-version-aware-package-reconciliation-and-progress-feedback.md), and [ADR-0015](0015-dual-engine-website-restoration-and-staged-reconstitution.md))

## Context
WPTUI previously coupled local WordPress serving to Laravel Herd via a `used_herd` boolean, assuming all sites lived in a parked directory and executing Herd CLI commands directly. This prevented WPTUI from supporting alternative local environments such as Flock or Laragon and scattered Herd-specific assumptions across provisioning, restoration, de-provisioning, dependency checks, and reporting.

In addition, `config.json` embedded static plugin and theme arrays along with a named default theme (`default_theme_slug`). These static names remained visible even when no Package Service was configured or catalog fetching failed, and failed package resolutions fell back to arbitrary WordPress.org slugs.

Developers require:
1. Support for their actual installed Local Development Environment: Laravel Herd, Flock, or Laragon.
2. Platform-aware environment availability: Windows supports Herd, Flock, and Laragon; macOS supports Herd and Flock (Laragon is Windows-only).
3. Explicit environment selection during first-run setup without silent defaults, paired with editable executable paths and website directory suggestions.
4. Strict configuration validation that cleanly rejects obsolete schema without silent migrations or compatibility shims.
5. API-only Package provenance from the configured Package Service, with hidden package controls when no service is available, optional Active Theme selection, and no default theme.
6. Safe lifecycle contracts for registration, TLS, rollback ownership, and de-provisioning across all supported environments.

## Decision

1. **Configuration Schema and Validation**:
   - Replace `used_herd`, `themes`, `plugins`, and `default_theme_slug` with:
     - `app_name` (string, required): exact lower-case enum matching `"herd"`, `"flock"`, or `"laragon"`.
     - `app_command` (string, required): executable name or full path (`"herd"`, `"flock"`, or `"C:\\laragon\\laragon.exe"` by default). Custom paths are fully supported. The command is invoked directly with an argument vector; it is never parsed or executed through a shell.
   - Schema rejection: Configuration loading and validation strictly reject missing or invalid `app_name`/`app_command` and reject any configuration containing removed legacy fields (`used_herd`, `themes`, `plugins`, `default_theme_slug`). No automatic migration, fallback compatibility, or field aliasing is provided.

2. **First-Run Setup**:
   - Replace the Herd confirmation with an explicit, required single-choice Local Development Environment selection. No environment is silently defaulted.
   - Filter choices by host operating system: Windows presents Laravel Herd, Flock, and Laragon; macOS presents Laravel Herd and Flock.
   - Provide editable environment-specific default suggestions upon selection:
     - **Herd**: Websites directory `$HOME/Herd`, command `herd`.
     - **Flock**: Websites directory `$HOME/Sites`, command `flock`.
     - **Laragon**: Websites directory `C:\laragon\www`, command `C:\laragon\laragon.exe`.
   - The Package Service URL and key remain optional in setup.

3. **Environment Command Contracts and Integration**:
   - **Laravel Herd**:
     - Registration: Satisfied by the parked Websites directory (per [ADR-0005](0005-herd-path-trust-and-terminal-tls.md)).
     - TLS: Executed via `herd secure <slug>` during Provisioning; best-effort `herd unsecure <slug>` during De-provisioning.
   - **Flock**:
     - Registration: Linked via `flock link <slug>` executed in the Website directory.
     - Pre-registration safety: Source integration inspects `flock sites` status prior to linking to verify existing registrations, preventing WPTUI from overwriting or replacing pre-existing site registrations.
     - TLS: Secured via `flock secure <slug>` executed in the Website directory.
     - Rollback ownership: On failure, rollback unsecures and unlinks only resources created and owned by the active run, protecting pre-existing registrations.
     - De-provisioning: Best-effort `flock unsecure <slug>` followed by `flock unlink <slug>`.
     - Browser launch: Uses the native operating system URL launcher; WPTUI never invokes a nonexistent `flock open` command.
   - **Laragon**:
     - Document Root: Websites must reside within Laragon's Document Root (`C:\laragon\www`).
     - Lifecycle updates: Reloads virtual hosts via `app_command` (`laragon.exe reload`) from the Website directory, or from its parent directory after directory deletion during De-provisioning.
     - TLS and URLs: Emits canonical `https://<slug>.test` URLs.
     - **Prerequisite:** Laragon HTTPS strictly requires externally enabled Auto SSL in Laragon's settings and the Document Root configured with the `{name}.test` hostname pattern. WPTUI documents this prerequisite and does not generate virtual-host templates, manipulate certificates, or verify Auto SSL status.

4. **Lifecycle Failure and Rollback Policy**:
   - **Provisioning**: Registration, reload, and TLS setup are critical steps. Failure rolls back newly created resources (directory, database, and certificates/links), while preserving shared Package Cache artifacts and leaving pre-existing environment resources untouched.
   - **Website Restoration**: Registration and reload failures are critical and trigger full rollback. TLS setup failure under Herd or Flock is non-fatal: it logs a warning, reconciles the database site URL to HTTP (`http://<slug>.test`), and retains the restored Website.
   - **De-provisioning**: Environment cleanup is best-effort. Command failure never blocks verified database dropping or mandatory directory removal.

5. **URL and Browser Policy**:
   - Canonical URLs default to `https://<slug>.test` across successfully secured Herd, Flock, and Laragon installations.
   - Restorations with TLS warnings use `http://<slug>.test`.
   - `Open in Browser` always launches the canonical URL through the host operating system launcher, providing consistent behavior across all environments.

6. **API-Only Package Catalog and Theme Selection**:
   - Static theme/plugin arrays and `default_theme_slug` are removed from configuration.
   - All user-selectable packages originate exclusively from the configured Package Service API.
   - When no Package Service URL is configured, or catalog fetching fails/returns empty, package selection steps in Create and package installation actions in Website Configuration are completely hidden. Provisioning proceeds cleanly without packages or themes.
   - Plugin and theme selections are presented in separate searchable multiselects with accumulated selections preserved across search queries.
   - Optional Active Theme: If one or more themes are selected, the user may choose at most one as the Active Theme (or choose none). Provisioning without an active theme is fully valid; the summary confirms that WordPress administration is accessible while the frontend has no active renderer.
   - Package resolution failure is treated as an error; fallback installation by WordPress.org slug is eliminated.

## Consequences
- Clean separation of environment lifecycle behind a unified boundary without Herd boolean flags.
- Transparent multi-environment support across Windows and macOS with platform-appropriate validation.
- Rollback and cleanup operations respect resource ownership and prevent data loss.
- Package provenance is deterministic, eliminating phantom static choices and unverified fallback sources.
