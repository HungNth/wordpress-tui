# Herd path trust and terminal TLS

When Laravel Herd mode is enabled, WPTUI trusts the configured `websites_path` without preflight inspection via `herd paths` or `herd parked`, avoiding CLI formatting discrepancies across Windows and Herd setups. Website subdirectories are automatically recognized as `.test` sites by Herd; after WordPress core, database, configuration, tweaks, and packages are fully provisioned, WPTUI executes `herd secure` on the site slug as the final step before completion, rolling back TLS if provisioning fails.
