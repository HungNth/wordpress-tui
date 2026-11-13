# 02: Display alternating workflow screenshots in the landing page features section

**Parent specification:** [Landing Page TUI Screenshots](../spec.md)

**What to build:** Upgrade the landing page's features section to pair each of the four canonical WPTUI workflows—Provisioning, Websites Hub, Website Configuration with In-Pane Progress Monitor, and Website Backup and Restoration—with its generated TrueColor terminal screenshot in an alternating desktop layout and text-first mobile stack, while keeping the walkthrough video, installation tabs, and copy workflow fast and accessible.

**Blocked by:** 01: Generate TrueColor TUI screenshots for the four core workflows.

**Status:** resolved

- [x] The existing interactive walkthrough video section remains intact above the installation and features showcase.
- [x] The features section presents four dedicated workflow rows covering Provisioning, Websites Hub, In-Pane Progress Monitor (highlighting Website Configuration and Website Debug Mode), and Website Backup and Restoration using canonical glossary terms.
- [x] On desktop viewports, rows alternate horizontally between text-left/image-right and image-left/text-right; on mobile viewports, every row collapses into a single column with explanatory text placed above its screenshot.
- [x] Each screenshot renders inline inside a clean terminal frame using `<picture>` with the WebP source first and the PNG fallback second, without modal lightboxes or zoom popups.
- [x] Every screenshot image specifies intrinsic 1080x600 dimensions, asynchronous decoding, lazy loading for below-the-fold performance, and descriptive `alt` text matching the displayed screen state.
- [x] Feature copy avoids em dashes, decorative emojis, and generic AI buzzwords, and asset paths remain relative for GitHub Pages subpath deployment without introducing external network requests.
- [x] Pages artifact packaging includes all WebP and PNG assets without altering the GitHub Actions deployment workflow or adding web build dependencies.
- [x] Demo: package the GitHub Pages artifact from the current manifest, serve it locally under the project subpath, and verify in a real browser at desktop and mobile widths that all four screenshots decode at 1080x600, the alternating desktop and text-first mobile layouts render without horizontal overflow, console and network errors are zero, and installation tabs and copy buttons continue working.
