# Title: Real-time progress feedback and version-aware package installation
Status: resolved
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
1. **Real-time Progress Indicator (`internal/tui`)**:
   - Add progress reporter / status line callbacks for:
     - Tweaks execution (`[1/N] Applying <type> <key>...`).
     - Admin updates (`Connecting to database...`, `Updating MySQL user_login...`, `Updating user_pass...`, `Updating user_email and admin_email...`).
     - Package installation (`Checking version of <slug>...`, `Installing <slug>...`).
2. **Catalog Fetching in Config (`internal/app/config_flow.go`)**:
   - In `RunDefaultConfigFlow`, eagerly call `packages.FetchCatalog(ctx, nil, cfg.PackagesAPIURL, cfg.PackagesAPIKey)` (identical to Create flow) so that the catalog is populated and `Type to search plugin catalog` is displayed at index 0.
3. **Version Comparison & Conditional Force (`internal/siteconfig/siteconfig.go`)**:
   - Implement `CompareVersions(v1, v2 string) int` supporting standard WordPress versioning (e.g. `6.8.10` vs `6.8.9`).
   - Implement `GetInstalledPackageVersion(ctx context.Context, siteDir string, pkgType PackageType, slug string, client WPClient) (string, bool, error)` calling `wp <type> get <slug> --field=version`.
   - In `InstallPackages`:
     - Check if package is installed:
       - Not installed: install normally via `wp <type> install <pathOrSlug> [--activate]`.
       - Already installed:
         - Compare versions:
           - If target > installed: install with `--force` (`wp <type> install <pathOrSlug> --force [--activate]`).
           - If target == installed: skip install, report `Already up to date (vX.Y.Z)`.
           - If target < installed: skip install, report `Current version is newer (vX.Y.Z > vA.B.C)`.
           - If unparseable: install with `--force`.

## Acceptance criteria
- [ ] Real-time progress is visible during tweaks, admin updates, and package installation.
- [ ] `Type to search plugin catalog` and `Type to search theme catalog` appear at the top of the package picker in Config flow when catalog is available.
- [ ] When plugin/theme is already installed and target version is higher, `--force` is used and upgrade succeeds.
- [ ] When plugin/theme is already at target version, installation is skipped cleanly with informative status.
- [ ] Unit tests and composed e2e tests cover progress reporter, version comparison, and conditional `--force` logic.
