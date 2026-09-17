# 0010: High-Contrast Dynamic UI Palette and Summary Visual System

## Context

WPTUI previously rendered interactive Select / MultiSelect form options with muted monochrome text, causing significant navigation ambiguity when moving with Up/Down arrows (the active cursor option did not highlight). Additionally, checked items (`[x]`) did not stand out visually from unselected items, and post-execution terminal summaries (provisioning and de-provisioning results) lacked color coding, making warnings, URLs, and errors difficult to scan quickly.

## Decision

1. **Active Option Focus Highlight**:
   - In `internal/tui/theme.go`, configure `theme.Focused.SelectSelector`, `theme.Focused.MultiSelectSelector`, and `theme.Focused.Option`:
     - Active cursor `> ` is colored bright cyan (`#00FFFF`) and bolded.
     - The option currently under the cursor renders in bright cyan and bold text, contrasting sharply with blurred/inactive options.
2. **Selected Multi-Select Item Highlight**:
   - Configure `theme.Focused.SelectedOption` and `theme.Focused.SelectedPrefix`:
     - Checked indicator `[x]` renders in vibrant green (`#04B575` / `#00FF00`) and bold text.
     - Checked option labels remain clearly highlighted even when the cursor moves away.
   - Mirror this green `[x]` indicator inside the Bubbletea `LiveSearchModel` for complete visual cohesion.
3. **Structured Summary Report Palette**:
   - Success headers, completed steps, and cached/healthy packages use bold green.
   - Resource locations, web URLs, and site paths use bright cyan (`#00FFFF`).
   - Stale package fallbacks, warnings, and skipped items use bold yellow (`#FFA500` / `#FFFF00`).
   - Operation errors and failed steps use bold red (`#FF4444`).
