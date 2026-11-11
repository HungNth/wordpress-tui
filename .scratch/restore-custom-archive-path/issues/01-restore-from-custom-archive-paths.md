# 01: Restore Websites from custom Restore Archive paths

**Parent specification:** [Restore Archives from Custom Paths](../spec.md)

**What to build:** Extend the production Restore Wizard so a user can select a compatible Restore Archive from any absolute filesystem location, including a Windows **Copy as path** value wrapped in double quotes. The complete flow must retain configured Backup Archive selection, normalize and validate custom or listed sources before advancement, revalidate the source before Website Restoration starts, preserve entered target parameters when correction is required, and use the normalized external source in place without copying or modifying it.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] `[Enter custom path...]` is the first archive option for both Full ZIP and AI1WM, while compatible archives discovered in `backup_path` remain selectable and scroll normally.
- [x] Custom-path entry remains available when `backup_path` is empty, missing, or unreadable; missing or unreadable directories show a concise non-blocking scan error instead of silently disabling or blocking the picker.
- [x] Custom input removes surrounding whitespace and exactly one balanced pair of outer double quotes, normalizes with host path semantics, requires an absolute path, and rejects unmatched outer quotes.
- [x] Enter validates that either a custom path or listed entry identifies an existing, readable regular file and has the case-insensitive extension required by the selected Restore Engine: `.zip` for Full ZIP or `.wpress` for AI1WM.
- [x] Arbitrary compatible third-party filenames are accepted without requiring WPTUI Backup Archive naming conventions; relative paths, directories, unsupported extensions, inaccessible files, and missing files remain on the archive interaction with an English inline error.
- [x] Filesystem validation occurs on Enter rather than after every keystroke, and the picker does not duplicate Restore Engine content or structural validation.
- [x] A valid custom Restore Archive advances to target parameters, displays its full normalized path within the Content Pane, and reaches the restore handler without surrounding quotes.
- [x] Esc and Back follow the established Restore Wizard navigation while preserving the entered Restore Archive path and target parameters until the form is explicitly discarded.
- [x] Pressing `Start Website Restoration` revalidates the selected source before the restore handler or In-Pane Progress Monitor starts.
- [x] If final revalidation fails, Website Restoration does not start; the wizard returns to the relevant archive interaction with a clear error while preserving the selected Restore Engine and all target parameters.
- [x] A Restore Archive outside `backup_path` is used directly without warning, copying, moving, renaming, or modifying the source; existing source-preservation and rollback behavior remains intact.
- [x] Composed Master-Detail behavioral coverage proves the custom happy path, inline failures, normalized handler input, scan-failure resilience, final stale-file rejection, target-value preservation, and absence of restore dispatch for invalid input; narrow path-contract coverage is added only where the composed seam is impractical.
- [x] The actual terminal surface is exercised for quoted-path paste, list/custom navigation, invalid feedback, Back/Cancel, final revalidation, full-path rendering at 80x24, and colorless readability; any platform-specific unreadable-file limitation is recorded rather than replaced with a fake assertion.
- [x] User-facing documentation uses the canonical Restore Archive terminology and describes custom absolute path support without changing configuration, Restore Engine, or Backup Archive naming contracts.
