# 03: Laragon Website lifecycle

**Parent specification:** `../spec.md`

**What to build:** Add Laragon as a complete Windows Local Development Environment choice. A Laragon user can place Websites under the configured Document Root, reload generated virtual hosts after lifecycle changes, and use the agreed HTTPS URL policy without WPTUI manipulating Laragon Auto SSL.

**Blocked by:** 01: Selected environment schema and Herd baseline.

**Status:** completed

## Acceptance criteria

- [x] Windows setup offers Laragon with editable `C:\laragon\www` and `C:\laragon\laragon.exe` suggestions; macOS setup does not offer Laragon.
- [x] Configuration and setup make the Document Root, `{name}.test` hostname pattern, and enabled Auto SSL explicit Laragon prerequisites.
- [x] The configured Laragon executable is checked before mutation and invoked directly with fixed reload arguments.
- [x] Provisioning reloads Laragon after creating the Website; reload failure is critical and triggers ownership-aware rollback.
- [x] Website Restoration reloads Laragon before reporting success; reload failure rolls back the fresh restoration.
- [x] De-provisioning removes the verified resources and then attempts Laragon reload on a best-effort basis, preserving accurate partial-success output.
- [x] Laragon Website URLs use `https://<slug>.test` without probing, enabling, disabling, or modifying Auto SSL.
- [x] WPTUI does not create or edit Laragon virtual-host files, hostname templates, trust stores, or certificates.
- [x] `Open in Browser` uses the operating-system launcher and never invokes a Laragon browser command.
- [x] Progress and result output identify reload outcomes without Herd/Flock TLS wording.

## Testing seam

Use composed application tests on Windows-neutral fixtures with an injected Laragon runner. Cover successful create/restore/delete, critical reload failures, best-effort post-delete reload failure, setup platform filtering, and HTTPS results without requiring a real Laragon installation.

## Demo path

On Windows, select Laragon during first-run setup, retain or edit its Document Root and executable path, Provision a Website, observe reload and the HTTPS result, open it through the system browser launcher, and De-provision it with a final reload outcome.

## Verification

- `go test ./...` passed, including composed Laragon create/delete and failure rollback, correct surviving working directory for post-delete reload, and setup platform filtering.
- Laragon was not available in PATH, so real Laragon reload and Auto SSL were not exercised. Auto SSL remains an explicit external prerequisite.
