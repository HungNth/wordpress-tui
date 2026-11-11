# 03: Complete Batch Website Backup Failure and Cancellation Outcomes

**What to build:** Make Batch Website Backup resilient and truthful when Websites fail or the user cancels. Continue after independent failures, preserve and report recovery artifacts, stop queued work on cancellation, and present every selected Website's final outcome under the correct aggregate result.

**Blocked by:** 02: Run Successful Batch Website Backups from the Websites Hub.

**Status:** done

**Parent specification:** Batch Website Actions and Backup Specification

- [x] A failed Website records its cause and the batch continues sequentially with every remaining queued Website; no failed operation is retried automatically.
- [x] A Backup Archive collision uses the shared no-overwrite failure, reports the conflicting destination, leaves the existing archive untouched, and does not stop later Websites.
- [x] `Batch Backup Completed with Warnings` is used when at least one Website succeeds and at least one Website fails.
- [x] `Batch Backup Failed` is used when execution finishes without any successful Website.
- [x] Failure rows identify the Website and underlying cause and include retained SQL or `.wpress` paths and cleanup outcomes whenever known.
- [x] Pressing `q` or Ctrl+C cancels the active backup through the operation context and waits for its existing cleanup path before navigation is re-enabled.
- [x] No queued Website starts after cancellation is requested; the active Website is marked cancelled unless it had already completed, and every queued Website is marked skipped due to cancellation.
- [x] `Batch Backup Cancelled` takes aggregate precedence even when earlier Websites succeeded or failed, while all earlier per-Website outcomes remain visible.
- [x] Per-Website outcomes remain in original batch order and distinguish success, failure, cancelled active work, and skipped queued work with non-color status markers.
- [x] The common strategy is displayed once; successful rows include Backup Archive path and size, and reports never expose credentials or API keys.
- [x] The persistent log retains output from completed, failed, and cancelled Website attempts and does not replace the concise aggregate report.
- [x] The report remains visible until dismissed after warnings, total failure, or cancellation; dismissal refreshes discovery and clears all selection state because earlier work may have produced archives.
- [x] Deterministic workflow tests cover partial failure followed by success, all failure, collision continuation, cancellation during active work, skipped queue entries, aggregate-heading precedence, and absence of retries.
- [x] Actual terminal verification uses isolated fixtures to exercise a partial failure and cancellation, including focus lock during cleanup, retained-artifact reporting, report dismissal, the 80x24 layout, and colorless readability.
