# Restore Archives from Custom Paths Specification

Status: ready-for-agent

## Problem Statement

A local WordPress developer can currently select a Restore Archive only when the integrated Restore Wizard discovers it in the configured `backup_path`. Restore Archives downloaded from a browser, copied from another machine, or stored elsewhere on disk cannot be selected through the production Master-Detail interface, even though the restoration engines can consume an archive from an arbitrary filesystem location.

This is especially inconvenient on Windows, where Explorer's **Copy as path** action places a balanced pair of double quotes around an absolute path. The current path validation contract treats those quotes as part of the filename. The integrated wizard also does not validate a selected archive before it advances or revalidate it before Website Restoration starts, so a missing, unreadable, non-regular, or format-mismatched file may not fail until restoration progress has begun.

The configured Backup Archive list must remain convenient, but it must not be the only source of Restore Archives or become a blocker when `backup_path` is empty, missing, or unreadable.

## Solution

Extend the integrated Bubble Tea Restore Wizard so the first archive-selection option is always `[Enter custom path...]`. Selecting it opens an in-pane text input for an absolute Restore Archive path while preserving the existing list of compatible archives discovered in `backup_path`.

The input accepts paths pasted with surrounding whitespace and one balanced pair of outer double quotes. WPTUI removes that wrapping, normalizes the path using host operating-system semantics, and validates it when the user presses Enter. A valid Restore Archive must exist, be a readable regular file, and have the case-insensitive extension required by the already-selected Restore Engine: `.zip` for Full ZIP or `.wpress` for AI1WM.

The same validation applies to archives selected from the scanned list and runs again when the user starts Website Restoration. Validation failures are shown inline before progress begins, preserve all entered target parameters, and return focus to the archive selection that needs correction. A Restore Archive outside `backup_path` is used directly without warning, copying, moving, or changing the source file.

## User Stories

1. As a local WordPress developer, I want to restore a downloaded archive from outside `backup_path`, so that I do not need to move it before Website Restoration.
2. As a Restore Wizard user, I want `[Enter custom path...]` to be the first archive option, so that an external Restore Archive is immediately discoverable.
3. As a developer with existing Backup Archives, I want scanned archives to remain available after custom-path support is added, so that the common selection flow stays fast.
4. As a developer whose configured backup directory is empty, I want custom-path entry to remain available, so that the absence of Backup Archives does not block restoration.
5. As a developer whose configured backup directory does not exist, I want a concise non-blocking scan error and working custom-path entry, so that a configuration problem does not prevent restoration from another location.
6. As a developer whose configured backup directory is unreadable, I want the scan failure shown without disabling custom-path entry, so that I understand the problem and can continue safely.
7. As a keyboard user, I want Enter on the custom-path option to open an editable path field in the Content Pane, so that the workflow remains fully keyboard accessible.
8. As a keyboard user, I want Esc from custom-path entry to return to the archive list without discarding my other Restore Wizard values, so that navigation is reversible.
9. As a Windows user, I want to paste an Explorer **Copy as path** value such as `"D:\backups\website.zip"`, so that I do not need to remove its quotes manually.
10. As a user pasting a path, I want surrounding whitespace removed before validation, so that harmless clipboard whitespace does not cause a false missing-file error.
11. As a user pasting a quoted path, I want exactly one balanced pair of outer double quotes removed, so that quote handling is predictable and does not alter the actual path.
12. As a user entering an unmatched outer quote, I want an inline validation error, so that malformed input is not silently rewritten.
13. As a user entering a relative path, I want an inline absolute-path error, so that resolution never depends on WPTUI's process working directory.
14. As a Full ZIP user, I want only `.zip` Restore Archives accepted, so that the selected Restore Engine matches the source format.
15. As an AI1WM user, I want only `.wpress` Restore Archives accepted, so that the selected Restore Engine matches the source format.
16. As a user with an uppercase or mixed-case extension, I want extension matching to be case-insensitive, so that valid platform filenames are not rejected.
17. As a user with a compatible third-party archive, I want arbitrary filenames accepted, so that Restore Archives do not need WPTUI's Backup Archive naming convention.
18. As a user entering a nonexistent path, I want an immediate inline error on Enter, so that I can correct the path before entering target parameters.
19. As a user entering a directory path, I want it rejected as not being a regular file, so that restoration never starts against a directory.
20. As a user entering an unreadable file, I want an explicit inline error, so that access problems are identified before restoration progress starts.
21. As a user entering an extension that does not match the selected Restore Engine, I want the expected extension named in the error, so that I know whether to correct the path or return to format selection.
22. As a user typing a path, I want filesystem validation to wait until I press Enter, so that incomplete input does not display errors after every keystroke.
23. As a user selecting a scanned archive, I want it validated when selected, so that a file removed or changed after the scan cannot advance the wizard.
24. As a user who selected a valid Restore Archive, I want it revalidated when I press `Start Website Restoration`, so that changes made while I entered target parameters are caught before progress begins.
25. As a user whose Restore Archive disappears before final submission, I want to return to archive selection with a clear error, so that the failure is associated with the field I must fix.
26. As a user whose final archive validation fails, I want my Website Name, Website Slug, and administrator fields preserved, so that correcting the archive does not require re-entering target data.
27. As a user reviewing target parameters, I want the full normalized Restore Archive path displayed, so that I can distinguish files with the same name in different directories.
28. As a user viewing a long path, I want it rendered within the Content Pane without hiding the identifying path, so that the 80x24 terminal contract remains usable.
29. As a user choosing a Restore Archive outside `backup_path`, I want it used in place without an extra warning, so that the primary custom-path use case has no redundant confirmation.
30. As a user choosing a Restore Archive outside `backup_path`, I want WPTUI not to copy or move it, so that storage use and file location remain under my control.
31. As a user restoring a Website, I want the source Restore Archive preserved on success, failure, and rollback, so that restoration never destroys my recovery artifact.
32. As a user with a structurally invalid archive, I want the selected Restore Engine to report its existing structural error during execution, so that the picker stays fast and does not duplicate engine logic.
33. As a terminal user, I want archive validation errors to use English text, error styling, and non-color symbols consistent with the terminal design contract, so that feedback remains understandable with or without ANSI color.
34. As a maintainer, I want custom and scanned Restore Archives to share one normalization and validation contract, so that the two selection paths cannot drift.
35. As a maintainer, I want invalid archives blocked before the restore handler and In-Pane Progress Monitor start, so that UI state accurately represents whether Website Restoration has begun.

## Implementation Decisions

1. **Authoritative workflow**
   - The integrated Bubble Tea Restore Wizard in the Master-Detail Content Pane is the production workflow for this feature.
   - Keep the established sequence: choose Restore Engine, choose Restore Archive, enter target parameters, then start Website Restoration.
   - Do not introduce another standalone picker or a parallel custom-path workflow. Existing callers that retain archive validation use the same contract.

2. **Archive picker behavior**
   - Always prepend `[Enter custom path...]` to the archive list for both Restore Engines.
   - Keep scanned entries filtered by the selected Restore Engine and preserve the existing non-recursive scan of `backup_path`.
   - An empty existing directory shows no matching scanned archives but leaves custom entry available.
   - A missing or unreadable configured directory produces a concise non-blocking scan error and leaves custom entry available.
   - Scan errors are no longer discarded silently by the integrated wizard.

3. **Custom-path interaction**
   - Enter on `[Enter custom path...]` switches the archive step into text-entry mode; it does not add a fourth wizard step.
   - Enter validates and, on success, stores the normalized path and advances to target parameters.
   - A validation failure remains inline, keeps focus in the path field, and does not advance.
   - Esc returns from custom-path entry to the archive list. Back navigation preserves the entered archive path and target parameters until the form is explicitly discarded.
   - The target-parameter view displays the full normalized path and wraps it within the available Content Pane width instead of reducing it to only the basename.

4. **Path normalization contract**
   - Remove surrounding whitespace first.
   - Accept and remove exactly one balanced pair of outer double quotes.
   - Reject unmatched outer double quotes rather than trimming them independently.
   - Normalize the resulting value using host operating-system path semantics.
   - Require the normalized path to be absolute.
   - Store and pass the normalized path, never the raw quoted input.

5. **Restore Archive validation contract**
   - The path must identify an existing, readable regular file.
   - Full ZIP requires `.zip`; AI1WM requires `.wpress`. Extension comparison is case-insensitive.
   - Compatible third-party filenames are accepted; WPTUI Backup Archive naming patterns are not required.
   - Apply the same contract to custom paths and scanned list entries.
   - Run validation when the archive is selected and again on final submission before the restore handler or In-Pane Progress Monitor starts.
   - Do not validate after every keystroke.
   - Picker validation remains filesystem-and-extension validation only. Archive integrity, WordPress Root discovery, SQL dump discovery, and AI1WM content compatibility remain Restore Engine responsibilities.

6. **Final revalidation behavior**
   - If final validation fails, do not invoke Website Restoration or transition to progress.
   - Return the user to the archive interaction associated with the selected source and show the validation error there.
   - Preserve all target parameters and the selected Restore Engine so the user only corrects the invalid source.

7. **External-source policy**
   - A Restore Archive outside `backup_path` is expected and requires no warning or confirmation.
   - Use the source path directly. Do not copy, move, import, register, or rename the source in `backup_path`.
   - Preserve the existing restoration invariant that the original source archive is never altered or deleted.

8. **Architecture and interfaces**
   - Reuse or deepen the existing archive scanning and path validation boundary rather than creating a second validator in the wizard model.
   - The normalized path is the value passed from the Restore Wizard through the application restore handler into the existing restoration request.
   - No configuration schema, Restore Engine request schema, database schema, command interface, or external dependency changes are required.
   - Keep all filesystem work synchronous with explicit Enter/submission actions; no background watcher or path polling is introduced.

9. **Presentation and navigation**
   - All labels, help, warnings, and validation messages are English.
   - Follow the existing focus, error styling, keyboard navigation, cancellation, minimum 80x24 geometry, and colorless-readability rules.
   - The custom option participates in normal list scrolling and cursor movement, including when many scanned archives are present.
   - Validation errors identify the failed condition and expected correction without exposing credentials or starting automatic retries.

10. **Documentation cutover**
   - Use the canonical domain term **Restore Archive** for any `.zip` or `.wpress` selected as Website Restoration input from any location.
   - Reserve **Backup Archive** for the immutable artifact produced by WPTUI and relocated to `backup_path`.
   - Keep user-facing documentation, the existing Website Restoration decision record, tests, and UI wording aligned with this distinction.

## Testing Decisions

1. **Primary behavioral seam**
   - Use the composed Master-Detail application model's Restore route as the primary seam. Drive it with Bubble Tea key messages, temporary directories/files, a temporary configuration, and an injected restore handler.
   - This is the highest existing seam that can prove archive-list rendering, custom text entry, inline errors, normalized path propagation, target-value preservation, final revalidation, and whether the restore handler starts.
   - No new production seam is required.

2. **Primary behavioral scenarios**
   - Verify `[Enter custom path...]` is first with matching scanned archives, an empty directory, and a failed scan.
   - Verify a quoted absolute path advances, displays the full normalized path, and reaches the restore handler without quotes.
   - Verify Full ZIP and AI1WM accept only their corresponding extensions, including mixed-case extensions and arbitrary filenames.
   - Verify missing files, directories, unreadable files where the platform permits deterministic permission control, relative paths, unmatched quotes, and mismatched extensions show inline errors and do not advance or invoke the handler.
   - Verify a scanned file removed before selection is rejected.
   - Verify a file removed after archive selection but before final submission returns to archive selection, preserves target parameters, and never starts progress or invokes the handler.
   - Verify Esc/Back transitions preserve entered data and follow the established Restore Wizard navigation.

3. **Narrow contract coverage**
   - Use one table-driven normalization/validation test only for path syntax and filesystem cases that are clearer below the composed UI seam, especially balanced quotes, unmatched quotes, absolute-path enforcement, case-insensitive extensions, regular-file enforcement, and normalized return values.
   - Do not duplicate the same user journey at both seams or assert private state, helper names, source text, copied values, or incidental cursor indexes.

4. **Prior art**
   - Extend the existing Master-Detail Restore flow test that drives Sidebar navigation into the Restore Wizard.
   - Follow the existing Restore Wizard model tests for multi-step transitions and archive scrolling.
   - Follow the existing archive scanning and validation tests that use isolated temporary directories and files.
   - Preserve existing Restore Engine tests for source-archive preservation, staging cleanup, structural validation, and rollback; do not duplicate those engine behaviors in picker tests.

5. **Terminal-surface verification**
   - Launch the actual TUI and exercise list selection, custom-path paste with Windows-style outer quotes, inline invalid-path feedback, Back/Cancel, final revalidation failure, and successful transition to progress with an isolated fixture.
   - Verify the custom input, full path, and errors at the 80x24 minimum; confirm readability without relying on color.
   - Record any platform-specific unreadable-file scenario that cannot be exercised deterministically rather than replacing it with a tautological test.

6. **Good-test standard**
   - Tests assert consumer-visible state transitions, rendered feedback, normalized handler input, source preservation, and absence of restore dispatch for invalid input.
   - Tests remain deterministic, isolated, and independent of live Websites, databases, WP-CLI, external services, or the user's actual backup directory.

## Out of Scope

- Downloading Restore Archives from URLs, cloud storage, or a remote Package Service.
- Adding a native graphical file browser, filesystem autocomplete, drag-and-drop handling, recent-path history, bookmarks, or persisted custom paths.
- Copying, moving, renaming, registering, or importing an external Restore Archive into `backup_path`.
- Accepting relative paths, directories, symbolic links, devices, or other non-regular filesystem entries.
- Supporting archive formats other than `.zip` and `.wpress`, converting formats, or inferring/switching the Restore Engine automatically from a custom path.
- Recursively scanning subdirectories under `backup_path`.
- Performing archive integrity checks, ZIP extraction, WordPress Root discovery, SQL dump selection, or AI1WM content validation in the picker.
- Changing Website Restoration execution, rollback, collision handling, credential reconciliation, URL migration, environment registration, TLS behavior, or progress-step definitions.
- Redesigning the obsolete standalone Huh workflow or adding a second custom-path UX.
- Changing configuration keys, defaults, setup behavior, or Backup Archive naming conventions.
- Adding automatic retries, telemetry, filesystem watchers, or validation on every keystroke.

## Further Notes

- The design follows the terminal design contract and the existing Website Restoration and Backup Archive decisions.
- The domain glossary now distinguishes a **Restore Archive** selected from any location from a WPTUI-produced **Backup Archive** stored in `backup_path`.
- `website.zip|wpress` is interpreted as shorthand for either `website.zip` or `website.wpress`; `|` is not a supported delimiter or filename character on Windows.
- The source Restore Archive remains immutable from the user's perspective and is preserved on success, failure, and rollback.
- No new dependency is justified; the standard filesystem/path facilities and existing Bubble Tea model are sufficient.
