# Title: TUI forms, site selection, and navigation loop
Status: ready-for-agent
Labels: ready-for-agent

## Parent spec
.scratch/site-configuration/spec.md

## Required behavior
Implement interactive terminal forms and wizard views in `internal/tui`:
- `SelectWebsiteForConfig(websites []Candidate) (*Candidate, error)`: displays candidate sites with `< Back` option (or Esc to return).
- `SelectConfigAction() (ConfigAction, error)`: displays 4 actions: Apply tweaks, Change admin, Install plugins, Install themes, plus `< Back`.
- `PromptAdminInputs(current AdminUser, defaults DefaultAdmin) (*AdminInput, error)`: form displaying current values with fallback to defaults.
- Confirmation prompts: "Continue configuring this website?" and "Configure another website?".

## Acceptance criteria
- [ ] Esc or `< Back` gracefully returns to previous step without errors.
- [ ] Visual styling follows cyan focus indicator and green selected indicators.
- [ ] Unit tests for form generation and input validation in `internal/tui`.
