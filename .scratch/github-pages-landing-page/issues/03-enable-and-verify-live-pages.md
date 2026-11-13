# 03: Enable and verify the live Pages deployment

**Parent specification:** [GitHub Pages Landing Page](../spec.md)

**What to build:** Complete the one-time repository cutover that source changes cannot perform. Configure GitHub Pages to accept GitHub Actions deployments, publish the existing stable Application Release through manual dispatch, and verify the public project URL serves the approved install-first landing page. This establishes the live environment used automatically by subsequent stable Releases.

**Blocked by:** 02: Deploy validated Application Releases to GitHub Pages.

**Status:** ready-for-human

- [ ] The repository's Pages build source is set to GitHub Actions.
- [ ] The Pages workflow is manually dispatched from the default branch after tickets 01 and 02 are present there.
- [ ] The run validates manifest version 0.2.3 against stable Release v0.2.3 and completes the `github-pages` deployment successfully.
- [ ] The deployment environment reports `https://hungnth.github.io/wordpress-tui/` and the URL no longer returns 404.
- [ ] The live page shows Application Release 0.2.3, links to Release v0.2.3, and serves its local assets correctly from the repository project subpath.
- [ ] The live macOS/Windows switcher and at least one copy control are exercised in a browser over HTTPS.
- [ ] Browser network inspection confirms the live page makes no analytics or other third-party runtime requests.
- [ ] Demo: open the public project URL in a clean browser session and complete the primary install-path interaction through selecting a platform and copying its default command.
