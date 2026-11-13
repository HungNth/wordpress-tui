# Specification: GitHub Pages Landing Page

Status: ready-for-agent

## Problem Statement

Prospective WPTUI users currently land on the repository README to understand the product and find installation commands. The README is intentionally comprehensive, so users who already have a supported local WordPress environment must scan product documentation before they can select their operating system, copy the correct command, and install WPTUI.

The project also has no dedicated public product page. There is no concise install-first explanation of WPTUI, no browser-friendly macOS/Windows command switcher, and no page that presents the current stable Application Release directly from the canonical manifest. The intended GitHub Pages URL currently returns 404.

Manually embedding a version in a published page would create a second version source and could advertise a prerelease or a version whose binaries do not exist. Publishing all repository content as the Pages artifact would also expose unrelated project files unnecessarily.

## Solution

Provide a focused English landing page at the repository's default GitHub Pages project URL. The page introduces WPTUI, highlights its principal local WordPress workflows, summarizes supported platforms and prerequisites, and puts copy-ready macOS and Windows installation commands ahead of detailed documentation.

Build the page from dependency-free HTML, CSS, and JavaScript stored as dedicated site source. During deployment, a separate GitHub Actions workflow reads the canonical version from the manifest, verifies that it matches an existing stable Application Release, injects that version into a prepared static artifact, and deploys only that artifact to GitHub Pages. Stable published Releases deploy automatically. A manually dispatched run supports initial publication and recovery while enforcing the same stable-release invariant.

Keep the repository README as the detailed documentation source and add a visible link from it to the landing page.

## User Stories

1. As a prospective WPTUI user, I want to understand the product's purpose immediately, so that I can decide whether it fits my local WordPress workflow.
2. As a prospective user, I want the page to lead with installation rather than contributor documentation, so that I can start without reading the full README.
3. As a WordPress developer, I want to know that WPTUI manages local Websites through a terminal interface, so that I understand the product boundary.
4. As a WordPress developer, I want to see that WPTUI supports Provisioning, Website Configuration, Website Backup and Website Restoration, so that I can judge its practical coverage.
5. As a Herd, Flock, or Laragon user, I want the supported Local Development Environments stated clearly, so that I know whether WPTUI integrates with my setup.
6. As a macOS user, I want to know that both Intel and Apple Silicon are supported, so that I can install with confidence.
7. As a Windows user, I want to know that 64-bit AMD64 Windows is supported, so that I can install with confidence.
8. As a Linux or Windows ARM64 user, I want unsupported platforms represented accurately, so that I do not run an incompatible installer.
9. As a prospective user, I want a compact prerequisites summary, so that I know PHP, WP-CLI, MySQL, and a supported Local Development Environment must already be available.
10. As a user, I want the current stable Application Release shown on the page, so that I can relate the page to the version available for installation.
11. As a user, I want the displayed release linked to its GitHub Release, so that I can inspect release details and assets.
12. As a macOS user, I want the official default installation command ready to copy, so that I can install WPTUI without reconstructing a command.
13. As a Windows user, I want the official PowerShell installation sequence ready to copy, so that I can install WPTUI without reconstructing a command.
14. As a user with a custom filesystem layout, I want the custom installation-directory command available without dominating the page, so that I can opt into it when needed.
15. As a user, I want a clear macOS/Windows switcher, so that only the commands relevant to my platform occupy the primary installation surface.
16. As a keyboard user, I want to operate the platform switcher and copy controls without a mouse, so that installation remains accessible.
17. As a screen-reader user, I want platform state and copy confirmation announced meaningfully, so that I know which command is visible and whether it was copied.
18. As a user, I want visible copy confirmation, so that I do not repeatedly copy or accidentally paste stale clipboard content.
19. As a user whose browser blocks clipboard access, I want the command to remain selectable, so that copy-button failure does not block installation.
20. As a user with JavaScript disabled, I want both platforms' commands and links to remain readable, so that the core installation information is not hidden.
21. As a security-conscious user, I want a warning to review remote installation scripts, so that executing downloaded code is an informed action.
22. As a security-conscious user, I want direct links to both installer sources, so that I can inspect them before execution.
23. As a security-conscious user, I want the checksum guarantee and unsigned/notarized binary limitation disclosed, so that I understand the available integrity and publisher-identity guarantees.
24. As a mobile visitor, I want the page and code blocks to fit a narrow viewport without losing commands or controls, so that I can read and copy them reliably.
25. As a desktop visitor, I want a focused terminal-inspired presentation, so that the page reflects WPTUI without pretending to show a product screenshot.
26. As a visitor, I want links to the repository, README, and Releases, so that I can move to detailed documentation or source when needed.
27. As a privacy-conscious visitor, I want the page to avoid analytics, cookies, and third-party runtime services, so that viewing installation instructions does not create tracking traffic.
28. As a repository visitor, I want a visible landing-page link near the README introduction or installation guidance, so that I can reach the faster install path.
29. As a maintainer, I want the page version read from the canonical manifest during deployment, so that there is no manually maintained version constant in site source.
30. As a maintainer, I want deployment to reject a Release tag that differs from the manifest version, so that the page never labels a different build as current.
31. As a maintainer, I want prereleases ignored, so that they do not replace the current stable Application Release on the public page.
32. As a maintainer, I want a stable published Release to deploy the page automatically, so that publication follows the existing release lifecycle.
33. As a maintainer, I want a manually dispatched deployment to verify that its manifest version already has a stable published Release, so that initial publication and recovery cannot advertise an unreleased version.
34. As a maintainer, I want failed validation or deployment to leave the previously published site intact, so that a bad release cannot replace a working landing page.
35. As a maintainer, I want only prepared site files uploaded to Pages, so that repository internals and scratch planning files are not exposed in the artifact.
36. As a maintainer, I want no generated site output committed to the repository, so that source and deployment artifacts stay separate.
37. As a maintainer, I want the landing page to use no Node, package manager, framework, or static-site-generator dependency, so that maintenance remains proportional to a single static page.
38. As a maintainer, I want installation commands to remain exact copies of the documented commands, so that the landing page does not introduce a second installation method.
39. As a maintainer, I want relative site asset references, so that the page works under the GitHub Pages project subpath.
40. As a maintainer, I want the deployment workflow to use least-privilege permissions and the protected Pages environment, so that publication authority is narrowly scoped.

## Implementation Decisions

- **Audience and conversion goal**:
  - Optimize for WordPress developers who already use a supported Local Development Environment and want to install WPTUI quickly.
  - Use English to remain consistent with the application and README.
  - Use “Manage local WordPress without leaving your terminal.” as the primary message.
  - Make “Install WPTUI” the primary call to action and “View on GitHub” secondary.
  - Keep contributor and exhaustive operational documentation in the README rather than duplicating it.

- **Information architecture**:
  - Use one focused page containing a hero, stable release indicator, authentic install-and-launch terminal transcript, principal feature summary, supported-platform and prerequisite summary, installation section, security disclosure, and repository/documentation links.
  - Highlight Provisioning, Website Configuration, Website Backup and Website Restoration, and supported Local Development Environment integration.
  - Link the displayed version to the matching GitHub Release.
  - Do not claim that the project is open source while the repository has no license file; use “Source on GitHub.”

- **Visual presentation**:
  - Use a dark terminal-inspired design with a WordPress-blue accent and sufficient text/control contrast.
  - Use text and CSS rather than a logo, screenshot, GIF, fabricated TUI mockup, or new illustration.
  - Support desktop and mobile layouts without horizontal page overflow; command blocks may scroll internally when unavoidable.
  - Respect reduced-motion preferences and do not require animation for comprehension.
  - Include basic page title and description metadata, but no social-preview image that the repository does not own.

- **Installation interaction**:
  - Present macOS and Windows as an accessible tab interface with macOS initially selected; do not infer the visitor's operating system from browser data.
  - Follow the standard keyboard and semantic behavior expected for a tab interface.
  - Show the default installation command in each platform panel and place custom-directory variants in subordinate expandable content.
  - Copy the current README commands verbatim. The README remains the detailed documentation source; no Markdown parser or shared-content generator is introduced.
  - Provide a copy control for every command block. Announce success without relying on color alone, return the control to its idle label, and leave the underlying command selectable when clipboard access fails.
  - Use progressive enhancement: without JavaScript, both platform sections remain visible and every command and link remains usable.

- **Security and trust communication**:
  - Preserve the recommendation to review remote scripts before execution and link directly to both installer sources.
  - State that published archives are checked with SHA-256 during installation.
  - State that executables are not currently code-signed or notarized for Apple Gatekeeper or Windows SmartScreen.
  - Do not weaken, hide, or replace these warnings with marketing language.

- **Static-site architecture**:
  - Keep dedicated site source separate from product source and project documentation.
  - Use dependency-free HTML, CSS, and JavaScript; do not add a framework, package manifest, bundler, static-site generator, or runtime API dependency.
  - Keep a version placeholder in site source. A deployment build copies only site source to a clean artifact directory and replaces the placeholder there.
  - Do not commit the generated artifact.
  - Use relative URLs for local assets so the project page works beneath the repository subpath.
  - Do not fetch the manifest or GitHub API from visitor browsers.

- **Application Release contract**:
  - The canonical version remains the manifest's version value.
  - A deployable stable Release must be published, non-draft, non-prerelease, and tagged exactly as the letter `v` followed by the manifest version.
  - A Release-triggered deployment checks out the published tag and verifies that tag against the checked-out manifest.
  - A manually dispatched deployment uses the default branch state, then verifies through GitHub that the matching stable Release already exists.
  - A mismatch or missing stable Release is a hard failure before artifact upload or deployment.
  - The rendered page shows the validated manifest version and links to the corresponding Release.

- **GitHub Pages workflow**:
  - Add a dedicated Pages workflow rather than coupling page deployment to the binary release workflow.
  - Trigger it for published Releases and manual dispatch. A published prerelease must not deploy.
  - Use official GitHub Pages actions pinned to explicit supported major versions.
  - Grant repository contents read access and only the Pages and identity-token write permissions required for deployment.
  - Deploy through the `github-pages` environment and expose the deployment URL through that environment.
  - Serialize Pages deployments so simultaneous events cannot race; do not cancel an in-progress production deployment.
  - Upload only the clean prepared artifact directory, never the repository root.
  - Keep the existing binary release workflow and release asset contract unchanged.

- **Initial publication and repository configuration**:
  - The current stable Application Release is `v0.2.3`, which matches the current manifest and has the expected supported-platform assets and checksum manifest.
  - The repository owner must select GitHub Actions as the Pages source in repository settings once.
  - After the workflow reaches the default branch, manually dispatch it to publish the existing stable Application Release; future stable published Releases deploy automatically.

- **README and domain documentation**:
  - Add a visible landing-page link near the README introduction or installation section without replacing detailed README content.
  - Use the canonical domain terms Website, Provisioning, Website Configuration, Website Backup, Website Restoration, Local Development Environment, and Application Release.
  - The domain glossary defines Application Release as stable and installable; prereleases do not replace it.
  - No ADR is required because the static implementation and Pages deployment are localized, visible, and inexpensive to replace.

## Testing Decisions

- **Highest testing seam**:
  - Treat the prepared GitHub Pages artifact as the primary test subject.
  - Build the artifact with the same manifest-version injection and validation behavior used by deployment, serve it over local HTTP, and exercise the rendered page in a real browser.
  - Do not test product Go internals because the landing page does not change application runtime behavior.

- **Good test characteristics**:
  - Assert consumer-visible rendering, navigation, interaction, accessibility state, copied content, links, and release version rather than HTML source shape or CSS implementation details.
  - Use the exact supported installation commands and current Application Release as evidence; do not create prose snapshots or tests that merely search source text.
  - Avoid a permanent browser-test framework and Node toolchain for this single dependency-free page. Browser smoke proof is the appropriate seam unless the web surface grows materially.

- **Artifact and browser smoke proof**:
  - Build an artifact from the current manifest and confirm the rendered release is `0.2.3` with a link to Release `v0.2.3`.
  - Serve the artifact locally and inspect the actual page at representative desktop and mobile widths.
  - Confirm the primary call to action, secondary GitHub action, feature summaries, supported-platform information, prerequisites, security disclosure, and documentation links are visible and accurate.
  - Confirm macOS is initially selected when JavaScript runs; switching platforms updates the visible panel and semantic selected state.
  - Confirm the platform interface and copy controls work using only the keyboard and show a visible focus indicator.
  - Confirm each copy control places the complete corresponding multiline command into the clipboard and announces success.
  - Confirm command text remains manually selectable if clipboard access is unavailable.
  - Disable JavaScript and confirm both platform sections, commands, warnings, and links remain readable.
  - Confirm narrow viewports do not clip controls or cause page-level horizontal overflow.
  - Confirm no analytics, cookie, font, image, or other third-party runtime request is made.

- **Release validation proof**:
  - Run the deployment validation against stable Release `v0.2.3` and the matching manifest; it must succeed and produce the expected rendered version.
  - Run the same validation with a deliberately mismatched tag; it must fail before artifact upload.
  - Confirm the workflow excludes prerelease publication and that manual dispatch requires an existing stable Release.
  - After repository Pages is configured, manually dispatch the real workflow and verify the reported deployment URL loads the published artifact.

- **Prior art**:
  - Follow GitHub's official custom Pages workflow shape: configure Pages, upload a prepared Pages artifact, then deploy it through the `github-pages` environment.
  - Follow established CLI/TUI landing-page structure: immediate product value, install command near the top, concise feature proof, and deeper documentation links.
  - Reuse the repository's existing README commands, manifest version, release tags, release assets, and terminology rather than introducing parallel conventions.

## Out of Scope

- A custom domain, DNS records, or a `CNAME` file.
- Vietnamese content, a language selector, or any other localization framework.
- Linux or Windows ARM64 support claims or installation commands.
- Browser operating-system detection or persistent platform preferences.
- A logo, favicon, screenshot, GIF, video, fabricated TUI preview, or social-preview image.
- A light theme, system-theme switcher, or user-selectable color themes.
- Analytics, telemetry, cookies, advertising, tracking pixels, or third-party runtime services.
- Jekyll, React, Vue, Svelte, Node, npm, a package manager, a bundler, or another site generator.
- Parsing README Markdown during deployment or generating README and site content from shared data.
- Runtime fetching of the manifest, Releases API, or any other remote data in visitor browsers.
- Publishing prerelease versions to the public landing page.
- Modifying the binary release workflow, release target matrix, installers, archive names, checksums, or application code.
- Adding or selecting a software license, or describing the repository as open source without one.
- Replacing the README or duplicating its complete feature, configuration, and development documentation.
- Automated browser-test infrastructure unless the site grows beyond this single static surface.

## Further Notes

- The canonical repository is `HungNth/wordpress-tui`.
- The target project page is `https://hungnth.github.io/wordpress-tui/`; it currently returns 404.
- Stable Release `v0.2.3` exists and matches the current manifest, so manual dispatch can publish immediately after the Pages source is enabled and the workflow is available on the default branch.
- GitHub requires the repository's Pages source to be set to GitHub Actions before a custom workflow can publish successfully.
- Installation commands are deliberately duplicated between README and landing-page source to avoid a generator. A maintainer changing an installer command must update both surfaces in the same change.
- GitHub Pages deployment proves hosting but does not prove installer execution on every visitor platform; installer behavior remains covered by its existing release and installer contracts.
