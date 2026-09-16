# Use Huh for the terminal interface

WPTUI uses `charm.land/huh/v2` in standalone mode because its forms, validation, multi-select, accessible mode, and spinner cover the v1 workflow without a custom Bubble Tea application. Package Catalog search is implemented as a multi-step Huh form loop—query, filtered multi-select, then return to the selection menu—rather than an inline searchable widget.
