# WordPress Debug and Tweak Defaults Specification

Status: ready-for-agent

## Problem Statement

WPTUI's default WordPress tweaks currently enable `WP_DEBUG`, `WP_DEBUG_LOG`, and `WP_DEBUG_DISPLAY` together. A local developer who accepts the default Apply Tweaks choice therefore gets useful debug logging but may also have PHP and WordPress debug output rendered in the Website frontend or administrative interface. Debug output should be retained in the log without being displayed to visitors or interfering with page rendering.

WPTUI must also keep applying the established local-development defaults for WordPress memory, autosave frequency, post revision retention, and trash retention. Those four defaults already exist in the configured tweak list and must remain config-driven rather than becoming a second hard-coded policy.

The change must preserve the existing Apply Tweaks workflows for both Provisioning and Website Configuration. It must not introduce a separate Debug Mode action, migrate existing configuration files, or override user-authored `wp_tweaks` values.

## Solution

Change the newly generated default configuration so Apply Tweaks sets `WP_DEBUG` and `WP_DEBUG_LOG` to `true` while setting `WP_DEBUG_DISPLAY` to `false`.

Retain the existing default tweaks that set the WordPress memory limit to `256M`, autosave interval to `600` seconds, retained post revisions to `5`, and trash retention to `21` days. Preserve their existing raw-versus-string value modes so WP-CLI writes valid PHP values.

Continue using the existing Apply Tweaks entry points. Provisioning keeps Apply Tweaks selected by default, and Website Configuration keeps its existing Apply Tweaks action. Both workflows consume `config.json/wp_tweaks` and delegate each configuration mutation to WP-CLI. WP-CLI adds a missing constant or updates an existing one according to its normal `config set` behavior.

Existing `config.json` files remain unchanged and authoritative. A user whose existing configuration contains `WP_DEBUG_DISPLAY=true` must change that value manually to receive the new behavior.

## User Stories

1. As a local WordPress developer, I want WordPress debugging enabled by default when I apply the configured tweaks, so that development errors are captured.
2. As a local WordPress developer, I want debug logging enabled with WordPress debugging, so that I can inspect failures after they occur.
3. As a Website user, I want debug messages hidden from rendered pages, so that diagnostics do not disrupt the frontend or administrative interface.
4. As a developer testing responses and layouts, I want debug output written to the log instead of injected into HTTP output, so that diagnostic text does not corrupt page content.
5. As a user provisioning a Website, I want the existing Apply Tweaks choice to configure debug logging safely, so that I do not need an additional setup step.
6. As a user provisioning a Website, I want Apply Tweaks to remain selected by default, so that the established local-development setup remains unchanged apart from hidden debug display.
7. As a user configuring an existing Website, I want the existing Apply Tweaks action to apply the same debug defaults, so that new and existing Websites use one configuration contract.
8. As a WPTUI user, I want debug behavior to remain part of Apply Tweaks, so that the Website Configuration menu does not gain a redundant Debug Mode action.
9. As a local WordPress developer, I want the default WordPress memory limit to remain `256M`, so that development workloads have the expected memory allowance.
10. As a content editor, I want the default autosave interval to remain `600` seconds, so that Apply Tweaks preserves the accepted autosave policy.
11. As a content editor, I want WordPress to retain no more than `5` post revisions by default, so that revision storage remains bounded.
12. As a Website administrator, I want trash to be retained for `21` days by default, so that deleted content follows the accepted recovery window.
13. As a WPTUI user, I want all seven constants represented in `config.json/wp_tweaks`, so that one configuration source controls their values.
14. As a user with custom local requirements, I want to edit or remove configured tweaks, so that WPTUI does not override my explicit configuration choices.
15. As a user with an existing `config.json`, I want WPTUI to leave it unchanged during upgrades, so that an application update does not silently rewrite my configuration.
16. As a user adopting the new default with an existing configuration, I want the manual configuration change documented, so that I understand why my current Website still displays debug output.
17. As a user applying tweaks to a Website that lacks one of these constants, I want WPTUI to add it through WP-CLI, so that the requested value takes effect.
18. As a user applying tweaks to a Website with a different existing value, I want WPTUI to update it through WP-CLI, so that the configured value becomes effective.
19. As a user applying several tweaks, I want progress and a result for each tweak, so that I can identify any value that was not applied.
20. As a user applying several tweaks, I want an individual failure to leave the remaining eligible tweaks running, so that one failed mutation does not discard unrelated configuration work.
21. As a user reviewing an Apply Tweaks result, I want partial failures reported rather than presented as unconditional success, so that the Website's actual state is clear.
22. As a maintainer, I want Provisioning and Website Configuration to consume the same configured tweak list, so that their default WordPress behavior cannot drift.
23. As a maintainer, I want the change to reuse the established WP-CLI mutation boundary, so that WPTUI does not gain a second `wp-config.php` editor.
24. As a maintainer, I want the default values and value modes covered through one application-level behavior test, so that a regression is caught without duplicating lower-level tests.

## Implementation Decisions

- Keep WordPress debug settings inside the configured Apply Tweaks list. Do not add a standalone Debug Mode action, toggle, or command.
- Change only the newly generated default for `WP_DEBUG_DISPLAY`, from the raw boolean value `true` to the raw boolean value `false`.
- Retain `WP_DEBUG=true` and `WP_DEBUG_LOG=true` as raw boolean defaults.
- Retain the four existing non-debug defaults: `WP_MEMORY_LIMIT=256M`, `AUTOSAVE_INTERVAL=600`, `WP_POST_REVISIONS=5`, and `EMPTY_TRASH_DAYS=21`.
- Preserve `WP_MEMORY_LIMIT` as a string value and preserve the other six constants as raw PHP values.
- Keep `config.json/wp_tweaks` as the sole source for these values. Do not enforce hidden built-in values outside that list.
- Preserve the current generic tweak schema and validation. No schema, public interface, or configuration version change is required.
- Preserve the current Apply Tweaks selection behavior during Provisioning, including its default-on state.
- Preserve the current Website Configuration action and site-first navigation established by the accepted Site Configuration ADR.
- Continue delegating configuration mutations to WP-CLI `config set`. Do not parse, format, or rewrite `wp-config.php` directly.
- Rely on WP-CLI to add missing constants and update existing constants. Do not add repository-owned duplicate detection or deduplication.
- Preserve the current per-tweak progress, configured ordering, best-effort execution, and partial-failure reporting behavior.
- Do not mutate or backfill an existing `config.json`. The updated defaults apply only when WPTUI generates a new default configuration.
- Update the provisioning documentation so the documented default for `WP_DEBUG_DISPLAY` matches the new `false` value and the existing raw/string modes remain accurate.
- No new domain glossary term or ADR is required. This is a reversible default-value correction within the existing Apply Tweaks model.

## Testing Decisions

- Use one existing application-level Website Configuration seam. Construct the flow with the generated default configuration, select a discovered Website, choose Apply Tweaks through scripted dependencies, and observe calls made to the existing mock WP client.
- Assert the externally relevant mutation contract: the mock receives configuration-set requests for all seven constants with the expected values and raw/string modes, especially `WP_DEBUG_DISPLAY=false` and string-valued `WP_MEMORY_LIMIT=256M`.
- Keep the assertion focused on the WP client boundary rather than source text, list indexes, JSON formatting, or private helper calls. Reordering unrelated configured tweaks must not break the test.
- Reuse the prior-art application test that already scripts Website selection and the Apply Tweaks action. Extend its observation detail rather than introducing a new test harness or lower-level seam.
- Do not add a second default-configuration unit test. The selected application seam covers default generation, action dispatch, and the mutation requests in one deterministic test.
- Do not add new WP-CLI command-construction tests because that implementation and its `--raw` behavior are unchanged by this feature.
- Do not add new Create wizard tests because the default-on Apply Tweaks choice is unchanged and already covered.
- A good regression test must fail if debug display returns to `true`, any retained constant disappears or changes value, or a raw/string mode changes. It must not assert incidental progress wording or implementation structure.

## Out of Scope

- A standalone Enable Debug Mode or Disable Debug Mode action.
- A separate Debug Mode choice in the Create wizard.
- Individual tweak selection in the terminal interface.
- Changing the default-on state of Apply Tweaks during Provisioning.
- Automatically migrating, backfilling, or rewriting existing `config.json` files.
- Forcing `WP_DEBUG_DISPLAY=false` when a user explicitly configures another value.
- Moving the seven constants out of `config.json/wp_tweaks` or adding a second configuration source.
- Direct parsing, formatting, or deduplication of `wp-config.php`.
- Guaranteeing the exact whitespace or quote style shown in illustrative PHP snippets; WP-CLI owns file formatting.
- Adding retries, rollback, preflight checks, or new failure-handling behavior to Apply Tweaks.
- Changing Restore behavior or applying these defaults automatically during Website Restoration.
- Changing any unrelated WordPress option, permalink, language, Package, or administrative credential behavior.

## Further Notes

- The four requested non-debug constants and their accepted values already exist in the default configuration. Implementation must retain them rather than add duplicate entries or a second execution path.
- The current documented provisioning default says `WP_DEBUG_DISPLAY=true`; it must be corrected to `false` with the implementation.
- Existing user configurations remain authoritative. Release or change documentation must call out that users with `WP_DEBUG_DISPLAY=true` must edit their existing `config.json` manually before running Apply Tweaks if they want hidden debug display.
- The chosen test seam was explicitly confirmed: one application-level Website Configuration flow test, with no duplicate lower-level default-config test.
