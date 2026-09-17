# 0012: Version-Aware Package Reconciliation and Progress Feedback

## Status
Accepted

## Context
When running Website Configuration on an existing Website, users previously received no real-time progress feedback while lengthy commands (such as database updates, tweak iterations, or package downloads) were running, resulting in a frozen-looking terminal.

Additionally, installing packages on an existing site using standard `wp plugin install` or `wp theme install` fails if the plugin or theme directory already exists. Blindly passing `--force` every time triggers unnecessary archive extractions and downloads, whereas ignoring existing packages prevents legitimate upgrades.

## Decision
1. **Real-time Progress Indicator**: Wrap long-running operations in interactive step progress feedback (`internal/tui.NewProgressSpinner` or status reporter) informing the user of the active step before rendering the final summary report.
2. **Catalog Fetching in Config**: Eagerly fetch the package catalog in `RunDefaultConfigFlow` (matching the Create flow) so that the live search option (`Type to search plugin catalog`) is consistently available at the top of the package picker.
3. **Version Comparison & Conditional Force**: Before installing a package on an existing website:
   - Run `wp plugin get <slug> --field=version` (or `theme`).
   - If the package is not yet installed: execute standard install `wp plugin install <archiveOrSlug> [--activate]`.
   - If already installed: compare installed version against target package version:
     - `target > installed`: run `wp plugin install <archiveOrSlug> --force [--activate]` to perform an in-place upgrade.
     - `target == installed`: skip installation, logging `Already up to date (v...)`.
     - `target < installed`: skip installation, warning that the installed version is newer.
     - Unparseable versions: default to `--force` to ensure user's requested package is applied.

## Consequences
- Prevents errors caused by directory collisions on existing plugins/themes.
- Saves time and network resources by avoiding redundant reinstalls of identical package versions.
- Significantly improves terminal UX with real-time feedback during administrative and tweak operations.
