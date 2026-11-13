# 01: Create the install-first landing page

**Parent specification:** [GitHub Pages Landing Page](../spec.md)

**What to build:** Deliver a complete locally demonstrable landing-page artifact for prospective WPTUI users. It must introduce the product, present the current stable Application Release, summarize supported workflows and prerequisites, preserve installation security disclosures, and let visitors select and copy the exact macOS or Windows installation commands. The surface must remain useful without JavaScript and must not add a web framework or runtime service.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] The English hero uses the approved message, makes installation the primary action, and links to the GitHub source as the secondary action.
- [x] The page presents an authentic install-and-launch terminal transcript without a fabricated TUI screenshot or mockup.
- [x] Provisioning, Website Configuration, Website Backup and Website Restoration, and Herd/Flock/Laragon integration are represented accurately using glossary terminology.
- [x] Supported macOS and Windows targets, unsupported-platform boundaries, and the PHP/WP-CLI/MySQL/Local Development Environment prerequisites match current documentation.
- [x] A prepared artifact derives its displayed version from the manifest and links to the matching GitHub Release; site source contains no manually maintained release number.
- [x] Accessible macOS and Windows tabs expose the exact default README commands, with custom-directory variants in subordinate expandable content and no browser operating-system detection.
- [x] Every command block has a keyboard-operable copy control with visible and screen-reader confirmation; clipboard failure leaves the command selectable.
- [x] With JavaScript disabled, both platform sections, every command, security warning, and documentation link remain readable.
- [x] The page retains the remote-script review warning, installer-source links, SHA-256 verification statement, and unsigned/notarized executable limitation.
- [x] The dark terminal presentation is responsive, has visible keyboard focus and sufficient contrast, respects reduced motion, and avoids page-level horizontal overflow at desktop and mobile widths.
- [x] The page makes no analytics, cookie, font, image, or other third-party runtime request and does not claim the unlicensed repository is open source.
- [x] Only dependency-free static source is added; generated artifact output is not committed.
- [x] Demo: build the artifact with the current manifest, serve it over local HTTP, and verify the approved desktop/mobile, keyboard, clipboard, no-JavaScript, link, and version scenarios in a real browser.
