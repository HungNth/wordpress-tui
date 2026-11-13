# 02: Deploy validated Application Releases to GitHub Pages

**Parent specification:** [GitHub Pages Landing Page](../spec.md)

**What to build:** Automate publication of the approved landing page through a dedicated GitHub Pages workflow. A stable published Application Release must deploy automatically, while manual dispatch must support initial publication and recovery. Both paths must validate the manifest against a real stable Release before preparing and uploading only the public site artifact. Add a visible README link to the landing page so repository visitors can reach the install-first experience.

**Blocked by:** 01: Create the install-first landing page.

**Status:** resolved

- [x] A dedicated Pages workflow runs for published Releases and manual dispatch without changing the binary release workflow.
- [x] Published prereleases and drafts cannot replace the current public landing page.
- [x] A Release-triggered run checks out the published tag and fails unless it equals `v` plus the checked-out manifest version.
- [x] A manually dispatched run uses the default branch and fails unless the manifest version has a matching published, non-draft, non-prerelease GitHub Release.
- [x] Validation failure occurs before artifact upload or Pages deployment, preserving the previously published site.
- [x] The build prepares a clean, uncommitted artifact, injects the validated manifest version, and uploads only public site files rather than the repository root.
- [x] Local asset references work beneath the repository's GitHub Pages project subpath.
- [x] Official Pages actions are pinned to explicit supported major versions and the workflow grants only repository-read, Pages-write, and identity-token-write permissions required by its jobs.
- [x] Deployment uses the `github-pages` environment, reports its URL, serializes Pages deployments, and does not cancel an in-progress production deployment.
- [x] The README visibly links to the default GitHub Pages project URL without replacing its detailed installation documentation.
- [x] Positive validation against stable Application Release v0.2.3 succeeds and renders version 0.2.3; a deliberately mismatched tag fails before upload.
- [x] Demo: run the same artifact preparation and validation used by the workflow locally, serve the result, and confirm it is identical in behavior to ticket 01's accepted surface.
