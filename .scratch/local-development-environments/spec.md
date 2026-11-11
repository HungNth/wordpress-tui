# Local Development Environments and API Package Catalog Specification

Status: ready-for-agent

## Problem Statement

WPTUI models local Website serving as a Herd-specific boolean. The first-run setup defaults to Laravel Herd, configuration stores `used_herd`, lifecycle workflows call Herd commands directly, and Website URLs infer HTTPS from that boolean. This prevents WPTUI from representing Flock or Laragon accurately and spreads environment-specific assumptions across Provisioning, Website Restoration, De-provisioning, progress, reports, dependency checks, and browser actions.

The configuration also embeds named theme and plugin lists plus a named default theme. Those static names remain visible when the Package Service is not configured or its Package Catalog cannot be loaded, so the UI can offer Packages that did not come from the configured service. Some existing installation paths additionally fall back to a WordPress.org slug when Package resolution fails.

A local WordPress developer wants to select the Local Development Environment actually installed on the machine, receive the correct Website Registration and TLS lifecycle for that environment, and see Package choices only when the configured Package Service has supplied a usable Package Catalog.

## Solution

WPTUI replaces the Herd boolean with one required Local Development Environment selection and one executable command setting. First-run setup requires an explicit environment choice, supplies environment-specific path and command suggestions, and validates the resulting configuration without migrating or accepting the old schema.

Environment behavior is selected centrally:

1. Laravel Herd continues to rely on the configured Websites directory being parked and manages TLS with Herd secure/unsecure operations.
2. Flock registers each Website with link, enables TLS with secure, and reverses owned resources with unsecure and unlink.
3. Laragon requires Websites under its Document Root and reloads generated virtual hosts after lifecycle changes. WPTUI treats Laragon Websites as HTTPS and documents Auto SSL as an external prerequisite; WPTUI does not control or verify Laragon TLS.
4. `Open in Browser` remains available for every environment through the operating-system URL launcher. WPTUI never emulates or invokes an environment-specific open command.

The configuration no longer stores static theme/plugin names or a default theme. Package UI is available only after the Package Service successfully returns a non-empty catalog containing the relevant Package type. Missing, empty, or failed catalogs produce a concise warning and remove unavailable Package choices for the current application session. Website Provisioning remains available without Packages or an Active Theme.

When a catalog is available, plugin and theme choices use separate searchable selections. Multiple themes may be installed, and the user may explicitly choose at most one selected theme as the Active Theme. No theme is also valid.

## User Stories

1. As a first-time WPTUI user, I want setup to ask which Local Development Environment I use, so that WPTUI does not silently assume Laravel Herd.
2. As a Windows user, I want setup to offer Laravel Herd, Flock, and Laragon, so that I can select an environment supported on my platform.
3. As a macOS user, I want setup to offer Laravel Herd and Flock but not Laragon, so that I cannot select an unsupported environment.
4. As a setup user, I want no Local Development Environment silently selected for me, so that saving the configuration represents an explicit choice.
5. As a Herd user, I want `$HOME/Herd` suggested as my Websites directory, so that setup follows Herd's conventional layout while remaining editable.
6. As a Flock user, I want `$HOME/Sites` suggested as my Websites directory, so that linked Websites have a clear editable default location.
7. As a Laragon user, I want `C:\laragon\www` suggested as my Websites directory, so that projects are placed under the conventional Document Root.
8. As a user with a custom installation, I want to edit the environment executable command during setup, so that WPTUI can invoke an executable outside the default location.
9. As a configuration editor, I want invalid or unsupported environment names rejected clearly, so that WPTUI never guesses lifecycle behavior.
10. As a user upgrading from an old configuration, I want an explicit configuration error for the removed `used_herd` schema, so that I know to run setup or replace the configuration rather than receiving hidden compatibility behavior.
11. As a Herd user, I want Provisioning to secure a completed Website, so that its canonical `.test` URL uses HTTPS.
12. As a Herd user, I want De-provisioning to unsecure the Website on a best-effort basis, so that stale TLS state is cleaned without blocking database and directory removal.
13. As a Flock user, I want Provisioning and Website Restoration to link the Website directory, so that Flock serves the intended `.test` hostname.
14. As a Flock user, I want WPTUI to secure a linked Website, so that the Website uses a trusted HTTPS URL.
15. As a Flock user, I want rollback and De-provisioning to unsecure and unlink resources created or managed by WPTUI, so that Flock does not retain stale Website registrations.
16. As a Laragon user, I want WPTUI to reload Laragon after a Website is created, restored, or removed, so that generated virtual hosts reflect the filesystem.
17. As a Laragon user, I want WPTUI to use `https://<slug>.test`, so that its canonical URL matches my required Auto SSL setup.
18. As a Laragon user, I want setup and documentation to state that Auto SSL and the `{name}.test` hostname pattern are prerequisites, so that WPTUI does not imply it configured them.
19. As a user provisioning a Website, I want registration, reload, and TLS failures treated as critical, so that WPTUI does not report success for a Website that was not served as intended.
20. As a user restoring a Website, I want registration or reload failures to roll back the fresh restoration, so that an inaccessible partial Website is not retained.
21. As a user restoring through Herd or Flock, I want a TLS failure to retain the restored Website over HTTP with a warning, so that a usable restoration is not destroyed solely because HTTPS failed.
22. As a user deleting a Website, I want environment cleanup to remain best-effort, so that a command failure never prevents the verified database and selected directory from being removed.
23. As a user observing progress, I want environment-neutral step names and results, so that Flock and Laragon operations are not mislabeled as Herd.
24. As a user reviewing a destructive result, I want the report to distinguish environment cleanup, database removal, and directory removal, so that partial success remains visible.
25. As a Website user, I want `Open in Browser` to remain available for every Local Development Environment, so that I can launch the canonical URL even when the environment has no open command.
26. As a Flock user, I want browser opening to use my operating system rather than a nonexistent `flock open`, so that the action works with Flock's documented CLI.
27. As a configuration editor, I want static theme and plugin arrays removed, so that `config.json` does not advertise stale Package names.
28. As a configuration editor, I want the named default theme removed, so that WPTUI does not install a Package that was not selected from the Package Catalog.
29. As a user without a configured Package Service, I want Package selection steps and actions hidden, so that I am not offered unavailable choices.
30. As a user whose Package Catalog request fails, I want a concise warning and no Package choices for the current session, so that failure is visible without presenting an empty or misleading picker.
31. As a user whose Package Catalog has only plugins, I want theme choices hidden while plugin choices remain available, so that each action reflects usable catalog data.
32. As a user whose Package Catalog has only themes, I want plugin choices hidden while theme choices remain available, so that each action reflects usable catalog data.
33. As a user creating a Website with a usable Package Catalog, I want separate searchable plugin and theme selections, so that Package types remain clear.
34. As a user searching Packages, I want selections preserved across searches, so that filtering does not discard prior choices.
35. As a user selecting multiple themes, I want to choose at most one selected theme as the Active Theme, so that activation is explicit rather than order-dependent.
36. As a user who selects no Active Theme, I want Provisioning to complete, so that Package Service availability is not a prerequisite for administering WordPress.
37. As a user who creates a Website without an Active Theme, I want the result to state that the administration interface is available while the frontend has no active renderer, so that the limitation is explicit.
38. As a Package Service user, I want selected Package artifacts resolved through that service and its verified cache, so that installation does not silently switch sources.
39. As a Package Service user, I want a resolution failure reported instead of falling back to a WordPress.org slug, so that provenance remains deterministic.
40. As an existing Package Cache user, I want the current verified stale-cache behavior retained during artifact resolution, so that removing catalog fallbacks does not remove the established cache resilience policy.
41. As a user provisioning selected Packages, I want resolution to finish before Website mutation, so that an unavailable selected Package does not leave a partial Website.
42. As a user installing Packages on an existing Website, I want per-Package failures reported without source substitution, so that I know exactly which selected artifacts were installed.
43. As a maintainer, I want Herd-specific names removed from shared configuration, lifecycle abstractions, progress, and reports, so that adding environment behavior does not require more boolean branches.
44. As a maintainer, I want obsolete compatibility code, static Package option builders, fields, tests, and documentation removed, so that only the new model remains.

## Implementation Decisions

1. **Configuration schema**:
   - Remove `used_herd`, `themes`, `plugins`, and `default_theme_slug` completely.
   - Add required `app_name` with the exact lower-case values `herd`, `flock`, and `laragon`.
   - Add required `app_command`, interpreted only as an executable name or executable path. Arguments are fixed by WPTUI; the value is never parsed or executed through a shell.
   - Suggested commands are `herd`, `flock`, and `C:\laragon\laragon.exe` respectively.
   - Loading and validation reject missing/unknown environment values and the removed schema. No aliases, migration, deprecated fields, or compatibility fallback are retained.

2. **First-run setup**:
   - Replace the Herd confirmation with a required single-choice Local Development Environment field.
   - Filter choices by supported WPTUI platform: Windows exposes all three; macOS excludes Laragon.
   - Apply the agreed editable Websites directory and command suggestions after environment selection.
   - Keep Package Service URL and key optional. A URL is the configuration signal; a key remains optional because the service may not require authentication.

3. **Environment boundary**:
   - Centralize dependency checks, Website Registration, TLS, cleanup, and URL policy behind one environment-selected boundary rather than distributing `app_name` branches through workflows.
   - Continue requiring PHP, WP-CLI, and MySQL. Also require the selected `app_command` executable before mutation.
   - Invoke commands directly with an argument vector and the Website directory as the working directory where required.

4. **Environment command contracts**:
   - Herd: registration is satisfied by the parked Websites directory; secure/unsecure operate on the Website Slug; existing secured-site detection may remain Herd-specific inside the environment boundary.
   - Flock: link and secure operate from the Website directory with the Website Slug; unsecure and unlink reverse those operations. WPTUI does not call a top-level Flock open command.
   - Laragon: reload is invoked through `app_command`; Websites must be under the configured Document Root. WPTUI does not create custom virtual-host files or manipulate Auto SSL.

5. **Lifecycle and ownership policy**:
   - Provisioning treats registration, reload, and TLS as critical. Failure rolls back only resources created by the current run.
   - Website Restoration treats registration/reload as critical. TLS failure is a warning: the restored database URL is reconciled to HTTP and the Website is retained.
   - De-provisioning attempts unsecure/unregister/reload as applicable but always proceeds with its existing database verification and mandatory directory-removal rules.
   - Rollback does not remove pre-existing environment resources that the run did not create.

6. **URL policy**:
   - Successfully secured Herd and Flock Websites use `https://<slug>.test`.
   - A Herd or Flock restoration whose TLS step fails uses `http://<slug>.test` in that operation's result.
   - Laragon Websites use `https://<slug>.test`; Auto SSL is a documented prerequisite and is not probed or changed by WPTUI.
   - Browser opening uses the existing platform launcher and the WPTUI URL policy, never an environment-specific open command.

7. **Package Catalog availability**:
   - Fetch the Package Catalog once while building the application session when a Package Service URL is configured.
   - A request error, invalid/empty response, or absence of Packages of a given type makes that type unavailable for the session and produces one concise, non-secret-bearing warning.
   - The Create workflow omits Package selection entirely when no relevant Packages are available. Existing-Website actions omit plugin or theme installation independently based on catalog contents.
   - No automatic retry is introduced. A later application session may fetch again.

8. **Package selection and activation**:
   - Remove static option builders and all merging of configuration-provided names into the fetched Package Catalog.
   - Present separate searchable plugin and theme selection states using the Package Catalog and the existing Package Selection Accumulator behavior.
   - If one or more themes are selected, offer an optional single choice among those themes as the Active Theme.
   - Install selected themes without relying on ordering. Activate only the explicit Active Theme. Empty selections and no Active Theme are valid.
   - Do not require an active-theme lookup when no Active Theme was selected; summaries omit or explicitly report the absent Active Theme without failing Provisioning.

9. **Package provenance and failure**:
   - All user-selectable Packages originate from the Package Service catalog.
   - Remove fallback installation by WordPress.org slug when service resolution fails.
   - Preserve verified Package Cache and stale-cache resolution behavior after a Package has been selected from a successfully loaded catalog.
   - Preserve pre-mutation resolution for Provisioning and per-Package outcome reporting for existing Website configuration.

10. **UI and reporting**:
    - Use environment-neutral labels for progress and results while naming the selected environment where useful.
    - Hide unavailable choices rather than rendering disabled or empty pickers.
    - Preserve Master-Detail navigation, cancellation, progress, warning, secret-redaction, minimum-size, and colorless-readability rules from the terminal design contract.
    - Report the accepted no-Active-Theme consequence without presenting Provisioning as failed.

11. **Clean cutover**:
    - Migrate every caller, test fixture, setup path, workflow, report, document, and example to the new schema and terminology in one change set.
    - Delete Herd-only shared APIs, old boolean branches, static Package config types, obsolete builders, aliases, migrations, and deprecated documentation.
    - Architectural rationale is recorded in ADR-0017 and domain terms in the project glossary.

## Testing Decisions

1. Use composed application workflow tests as the primary behavioral seam. Inject command runners, Package Catalog data, Package resolvers, temporary configuration, and isolated filesystem/database stand-ins to exercise user-visible behavior without installed Herd, Flock, Laragon, or a live Package Service.
2. Cover one successful end-to-end lifecycle per environment at this seam:
   - Herd Provisioning/TLS and best-effort De-provisioning cleanup.
   - Flock link/secure plus unsecure/unlink rollback and deletion.
   - Laragon reload around creation/restoration/deletion with HTTPS URL output.
3. Cover failure boundaries at the composed seam: pre-mutation dependency failure, registration/reload rollback, Provisioning TLS rollback, Restoration TLS warning with HTTP reconciliation, and environment cleanup that does not block verified destructive work.
4. Use narrow configuration tests only for JSON shape, exact enum validation, executable-command validation, removed-schema rejection, file permissions, and first-run conversion because those contracts are clearer below the composed seam.
5. Use narrow TUI model tests only for OS-filtered setup choices, no silent default, conditional Package steps/actions, separate plugin/theme selection, preserved selections, optional Active Theme, cancellation, and empty-catalog behavior.
6. Extend existing runner-spy tests to assert executable names, working directories, argument order, and absence of shell execution or nonexistent open commands. Do not assert private adapter types or source-code structure.
7. Use deterministic HTTP fixtures for successful, empty, malformed, unauthorized, and failed Package Catalog responses. Never call external services in permanent tests.
8. Retain existing Package Cache tests for verified stale-cache resolution. Add no duplicate cache coverage merely for the new UI gating.
9. Exercise the actual terminal surface once for first-run environment selection, Package visibility with and without a catalog, Active Theme choice, Back/Cancel, progress labels, warning output, 80x24 behavior, and colorless readability, as required by the terminal design contract. Record any unavailable real executable integration as unexercised rather than simulating it manually.
10. Good tests assert user-visible state transitions, command contracts, rollback ownership, final URLs, available actions, and destructive safety. They do not pin helper names, copied values, source text, incidental defaults, or mock forwarding.

## Out of Scope

- Migrating or accepting configuration files containing `used_herd` or static Package arrays.
- Automatically detecting which Local Development Environment is installed.
- Automatically switching or migrating existing Websites between environments after `app_name` changes.
- Installing, updating, configuring, or launching Herd, Flock, or Laragon.
- Supporting Laragon on macOS, Linux support for WPTUI, or Windows ARM64 support.
- Changing Laragon's Document Root, hostname pattern, generated virtual-host templates, certificate trust, or Auto SSL setting.
- Adding a Flock site-open command or invoking Herd/Laragon browser commands.
- Persisting per-Website TLS state beyond existing Website data and operation results.
- Retrying Package Catalog requests automatically during a session.
- Reintroducing built-in Package names, a named default theme, WordPress.org fallback installation, or another Package source.
- Bundling or downloading an implicit fallback theme when no Active Theme is selected.
- Redesigning Package Service endpoints, authentication, archive verification, or Package Cache policy.
- Redesigning unrelated WordPress tweaks, backup engines, database safety checks, or terminal navigation.

## Further Notes

- The official Flock CLI documents link, unlink, secure, and unsecure but no top-level site-open command.
- Laragon documents reload commands, automatic virtual hosts under its Document Root, and Auto SSL as application-managed behavior rather than a per-site CLI contract.
- The accepted Laragon risk is that WPTUI emits HTTPS URLs while Auto SSL remains an external prerequisite that WPTUI does not verify.
- The accepted no-theme risk is that WordPress administration remains available while the Website frontend cannot render until a theme is installed and activated.
- Primary-source research is recorded in `.scratch/research/local-development-environments.md`.

## Comments

- The user confirmed the specification and ticket breakdown, then chose to stop after planning documentation. No implementation was started.
- The user subsequently explicitly approved implementation of all five tickets. Implementation is in progress; repository changes remain uncommitted.
