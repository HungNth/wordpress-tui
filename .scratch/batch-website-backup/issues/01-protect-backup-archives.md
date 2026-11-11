# 01: Protect Backup Archives and Surface Retained Artifacts

**What to build:** Make Website Backup archive finalization safe and observable before it is reused by batch orchestration. A Backup Archive must never replace an existing destination. On collision or finalization failure, preserve the recoverable source artifact and tell the user which archive or database dump remains and what cleanup completed.

**Blocked by:** None (can start immediately).

**Status:** done

**Parent specification:** Batch Website Actions and Backup Specification

- [x] Full ZIP and AI1WM Backup Archive relocation share one cross-platform no-overwrite contract; platform-dependent replacement behavior is removed.
- [x] When a destination already exists, its contents and metadata remain untouched and the failure identifies the conflicting destination.
- [x] A collision or relocation failure preserves the source SQL or `.wpress` recovery artifact when available and removes only partial destination files created by the failed attempt.
- [x] Full ZIP removes its exported database dump only after successful archive finalization; every later failure or cancellation reports the retained dump path and cleanup outcome when known.
- [x] AI1WM reports a retained source archive and cleanup outcome when generation succeeded but relocation failed or was cancelled.
- [x] Retained-artifact information is structured at the backup execution boundary rather than recovered by parsing human-readable errors.
- [x] Single-Website Backup displays known retained artifact paths without exposing credentials, API keys, or unrelated command output.
- [x] Successful Full ZIP and AI1WM backups retain their current archive names, `backup_path` placement, source cleanup semantics, sizes, and completion reports.
- [x] Isolated filesystem tests prove that a pre-existing destination remains byte-for-byte unchanged, the recoverable source remains available, and owned partial files are cleaned.
- [x] Cross-volume finalization is exercised where the verification environment permits it; any unavailable cross-volume surface is recorded explicitly.
