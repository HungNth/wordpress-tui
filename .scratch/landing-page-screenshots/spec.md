# Specification: Landing Page TUI Screenshots

Status: ready-for-agent

## Problem Statement

Prospective WPTUI users visiting the landing page currently see only a hero quick-start snippet, an overview walkthrough video, and text descriptions of the product's workflows. When scanning the features section, visitors must either watch and pause the looping video or imagine how each major workflow looks inside the terminal.

Text cards alone do not show how the Master-Detail Layout partitions Sidebar Navigation and the Content Pane across real tasks, how forms validate Website inputs during Provisioning, how the In-Pane Progress Monitor tracks execution steps alongside live logs, or how Website Backup and Website Restoration present archive choices.

Capturing screenshots manually is also brittle: manual screenshots drift out of date across releases, vary in window chrome and resolution, risk leaking personal filesystem paths, and degrade into blurry captures when recorded without matching WPTUI's TrueColor palette.

## Solution

Expand the landing page's features section into an alternating feature-and-screenshot showcase paired with the existing walkthrough video above. Each of the four core workflows—Provisioning, Websites Hub management, In-Pane Progress Monitor execution, and Website Backup and Restoration—is accompanied by a crisp, TrueColor 1080x600 terminal screenshot generated directly from the real WPTUI binary.

Extend the existing automated media-generation pipeline to drive WPTUI inside an isolated staging environment with mock local Websites and an isolated WP-CLI stub where required, capture all four canonical screens via headless terminal recording, and convert them into optimized WebP images with PNG fallbacks. Store the generated static image pairs with the website source so GitHub Pages deployment packages them with no extra CI tool dependencies.

## User Stories

1. As a prospective WPTUI user, I want to see real screenshots alongside each major workflow description, so that I can immediately understand how WPTUI operates without watching the entire walkthrough video.
2. As a developer evaluating WPTUI, I want to see the Provisioning wizard form, so that I understand what inputs and defaults are collected when creating a new Website.
3. As a developer evaluating WPTUI, I want to see the Websites Hub screen, so that I can see how local Websites, `.test` URLs, and multi-select checkboxes are presented.
4. As a developer evaluating WPTUI, I want to see the In-Pane Progress Monitor during execution, so that I can see the dual Task Stepper and scrolling Log Viewport before installing the tool.
5. As a developer evaluating WPTUI, I want to see the Website Backup and Restoration screen, so that I can confirm support for both consolidated ZIP and All-in-One WP Migration archive formats.
6. As a desktop visitor, I want feature descriptions and screenshots arranged in an alternating zig-zag layout, so that long visual sections remain easy to scan.
7. As a mobile visitor, I want each feature block to stack its explanation before its screenshot without horizontal overflow, so that I can read context before inspecting the image on a narrow screen.
8. As a visitor on a slow connection, I want below-the-fold screenshots served as lightweight WebP images with lazy loading, so that the initial install section and hero remain fast to render.
9. As a visitor using an older browser or saving images directly, I want PNG fallback images provided alongside WebP, so that screenshots render reliably everywhere.
10. As a visitor, I want screenshots to match WPTUI's actual TrueColor cyan and green palette without blurry scaling, so that the website represents the real terminal experience accurately.
11. As a visitor, I want the existing walkthrough video preserved above the installation and feature showcase, so that I can choose between watching interactive navigation and inspecting individual screens.
12. As a keyboard or screen-reader user, I want every screenshot to carry descriptive alternative text explaining the screen state shown, so that visual examples remain accessible.
13. As a visitor browsing without JavaScript, I want all screenshots, captions, and workflows to remain visible, so that static exploration works identically without scripts.
14. As a visitor, I want zero layout shift while images load, so that buttons and installation commands do not jump under my pointer.
15. As a visitor, I want to inspect screenshots directly in place without modal popups or lightbox overlays blocking scrolling, so that reading stays uninterrupted.
16. As a maintainer, I want all four screenshots generated automatically from the compiled WPTUI binary in a single script run, so that I never have to capture or crop screenshots by hand.
17. As a maintainer, I want screenshot generation to run in an isolated temporary home and Website directory using safe example names, so that personal Websites, credentials, or local configuration never leak into public assets.
18. As a maintainer, I want the In-Pane Progress Monitor screenshot produced using an isolated command stub rather than touching real databases or local services, so that media generation stays deterministic and side-effect free.
19. As a maintainer, I want generated WebP and PNG screenshots committed alongside the website source, so that the GitHub Pages deployment workflow remains fast and needs no browser, terminal recorder, or video encoder on CI runners.
20. As a maintainer, I want the landing page copy and screenshots to follow the project's domain glossary and anti-slop visual rules, so that terminology and presentation remain consistent across releases.

## Implementation Decisions

- **Feature Showcase Architecture**:
  - Replace the two-column text-only feature groups with a four-row alternating showcase covering the four canonical workflows: Provisioning, Websites Hub, In-Pane Progress Monitor (highlighting Website Configuration and Website Debug Mode), and Website Backup and Restoration.
  - On desktop viewports, alternate text and screenshot placement across rows (text left with image right on odd rows; image left with text right on even rows).
  - On mobile and narrow viewports, collapse every row into a single column ordered consistently with explanatory text above its screenshot so reading flow stays logical.
  - Keep the existing interactive walkthrough video section intact above the installation and features sections; screenshots complement rather than replace the video.
  - Present screenshots directly inline inside clean terminal viewport frames without interactive lightbox modals or zoom dialogs.

- **Canonical Screenshot Scenarios**:
  - **Provisioning**: Capture the Create Wizard form showing populated Website Name, generated Website Slug, default administrative fields, tweaks option, and active cyan focus styling.
  - **Websites Hub**: Capture the Websites inventory showing sample local Websites, `.test` URLs, multi-select `[x]` indicators, and active Content Pane framing.
  - **In-Pane Progress Monitor**: Capture the execution monitor after running Toggle Debug Mode inside Website Configuration, displaying completed `[✓]` stepper items, live log lines, and completion summary.
  - **Website Backup and Restoration**: Capture the Backup Strategy or Restoration format selection interface showing both full ZIP and All-in-One WP Migration (`.wpress`) options.

- **Automated TrueColor Capture Pipeline**:
  - Extend the existing isolated media-generation script so one invocation builds the current WPTUI binary, prepares an isolated staging environment, records the walkthrough video, and captures all four PNG screenshots at 1080x600 using the same TrueColor terminal palette (`NO_COLOR` unset, truecolor enabled, matching `DESIGN.md` colors and crisp monospace typography).
  - Use representative non-sensitive Website fixtures (such as client-store, developer-blog, and marketing-site) inside temporary staging and delete all temporary staging state after asset generation.
  - Provide an isolated WP-CLI executable stub inside the temporary staging `PATH` solely to satisfy Toggle Debug Mode queries (`wp config get WP_DEBUG --format=json` and `wp config set` calls) so the real WPTUI binary drives the genuine In-Pane Progress Monitor without modifying production files, Go source code, or real databases.
  - Convert each captured PNG screenshot into a high-efficiency WebP image while retaining the PNG as a fallback source, keeping both formats directly inside the website asset directory.

- **Web Image Delivery and Accessibility**:
  - Render each screenshot using a semantic `<picture>` element providing the WebP source first and the PNG `<img>` fallback second.
  - Declare explicit intrinsic `width` and `height` attributes matching the 1080x600 capture dimensions to prevent cumulative layout shift during page load.
  - Mark below-the-fold showcase images with lazy loading and asynchronous decoding so above-the-fold hero and installation controls load immediately.
  - Write specific, accurate `alt` descriptions for every image reflecting the actual screen, Websites, and state depicted, using canonical glossary terms (`Website`, `Websites Hub`, `Provisioning`, `Website Configuration`, `Website Debug Mode`, `In-Pane Progress Monitor`, `Website Backup`, `Website Restoration`, `Backup Archive`).
  - Maintain zero external image/CDN/font requests and ensure asset references remain relative so deployment under the GitHub Pages project subpath works without modification.
  - Do not introduce em dashes or decorative AI copywriting patterns when expanding feature descriptions.

- **Deployment Boundary**:
  - Commit generated WebP and PNG screenshots alongside website source so the existing static Pages packaging and GitHub Actions workflow require no additional runtime dependencies or build steps.

## Testing Decisions

- **Highest Testing Seam**:
  - Verify the feature end to end at two coupled boundaries: running the isolated media-generation script to produce valid assets, and serving the resulting GitHub Pages build artifact to a real headless browser across desktop and mobile viewports.
  - Do not modify or add Go application unit tests because no Go runtime logic changes.

- **Good Test Characteristics**:
  - Assert externally observable asset properties (presence of all four WebP and PNG pairs, exact 1080x600 dimensions, TrueColor rendering presence rather than monochrome fallback) and real browser behavior (images decoded with expected `naturalWidth`, alternating desktop row order, stacked mobile order without horizontal overflow, lazy-loading and dimension attributes, zero broken assets or external network requests, zero console errors, preserved video and copy-command functionality).
  - Avoid brittle pixel-by-pixel snapshot comparisons that break on minor font rendering differences across machines.

- **Verification Scenarios**:
  - Execute the media-generation script in a clean isolated run and verify all four screenshot pairs and walkthrough media files are produced without leaving temporary staging behind.
  - Build the GitHub Pages artifact with manifest validation and verify all image assets are included cleanly in the output.
  - Serve the artifact under the GitHub Pages subpath in a real browser at desktop (1280px+) and mobile (375px) viewports; confirm every screenshot completes loading with `naturalWidth === 1080` and `naturalHeight === 600`, no horizontal overflow occurs, no third-party requests fire, and copy/tab controls still pass.
  - Inspect rendered desktop and mobile full-page screenshots visually to confirm contrast, spacing, alignment, and TrueColor clarity.

- **Prior Art**:
  - Follows the same browser-rendered Pages artifact verification seam established for the landing page and the existing VHS TrueColor walkthrough recording pipeline.

## Out of Scope

- Automated generation or re-recording of screenshots inside GitHub Actions CI runners during deployment.
- Interactive image zoom, modal lightbox overlays, or screenshot carousels.
- Capturing additional screens beyond the four chosen workflows (such as De-provisioning confirmations or Application Settings) unless expanded in a future change.
- Modifying WPTUI's Go source code or adding a screenshot-capture mode to the compiled application.
- Multi-language captions or localized screenshot variants.

## Further Notes

- Because Toggle Debug Mode execution is driven by an isolated WP-CLI stub during recording, the stub contract must remain aligned with `internal/siteconfig`'s `WP_DEBUG` read/write command invocations if those flags change in future releases.
- Whenever WPTUI's terminal UI layout or color palette changes substantially, maintainers should re-run the media-generation script to refresh both the walkthrough video and the four feature screenshots before publishing a new Application Release.
