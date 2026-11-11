# Herd path trust and terminal TLS

> [!NOTE]
> The general Local Development Environment policy recorded here is superseded by [ADR-0017](0017-selected-local-development-environment-integration.md). Herd parked directory behavior described below remains active specifically when `app_name: "herd"` is selected, while WPTUI now centrally supports Laravel Herd, Flock, and Laragon through explicit `app_name` and `app_command` configuration.

When Laravel Herd mode is selected, WPTUI trusts the configured `websites_path` without preflight inspection via `herd paths` or `herd parked`, avoiding CLI formatting discrepancies across Windows and Herd setups. Website subdirectories are automatically recognized as `.test` sites by Herd; after WordPress core, database, configuration, tweaks, and packages are fully provisioned, WPTUI executes `herd secure` on the site slug as the final step before completion, rolling back TLS if provisioning fails.
