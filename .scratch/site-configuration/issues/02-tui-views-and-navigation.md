# Title: TUI views and site-first navigation sub-menu
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Implement user interface components in `internal/tui`:
1. `SelectWebsiteForConfig(websites []Candidate) (*Candidate, error)`:
   - Lists candidate websites discovered in `websites_path`.
   - Includes `< Back` option and supports Esc to cancel.
2. `SelectConfigAction(siteSlug string) (ConfigAction, error)`:
   - Displays 4 actions for the selected site:
     1. Apply wp_tweaks
     2. Change admin info
     3. Install plugins
     4. Install themes
     `< Back` (returns to website list).
3. `PromptAdminInfoForm(current AdminUser, defaults DefaultAdmin) (*AdminInput, error)`:
   - Shows detected administrator information.
   - Form inputs for Username, Password, Email.
   - Blank inputs retain current values.
4. `PromptThemeActivation() (bool, error)`:
   - Confirm whether to activate newly installed themes.
5. `PrintSiteConfigSummary(results []ConfigResult)`:
   - Color-coded summary output using cyan focus, green success, and red error styling.

## Acceptance criteria
- [ ] Keyboard navigation (Up/Down, Enter, Esc) works seamlessly.
- [ ] Styling adheres to high-contrast cyan cursor and green selection theme.
- [ ] Unit tests for form rendering and option builders in `internal/tui/siteconfig_test.go`.

## Testing seam
Unit test suite in `internal/tui/siteconfig_test.go`.
