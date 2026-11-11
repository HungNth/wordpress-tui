# Local development environment command research

## Capability matrix

| Environment | Register Website | HTTPS | Unregister Website | Open Website |
| --- | --- | --- | --- | --- |
| Laravel Herd | WPTUI currently relies on a parked `websites_path`; no registration command is invoked | `herd secure`, `herd unsecure`, and `herd secured` are used by the current integration | No unlink command is used by WPTUI | WPTUI uses the operating-system URL launcher |
| Flock | `flock link [name]` serves the current directory | `flock secure [name]`; `flock unsecure [name]` returns to HTTP | `flock unlink [name]` | The official top-level CLI list has no site-open command; only `flock worktree open` is documented |
| Laragon | Projects under the configured Document Root become automatic virtual hosts after reload | Auto SSL is controlled by Laragon and automatically detects projects when enabled | Removing the directory and reloading refreshes generated virtual hosts | The documented CLI example uses the operating system's URL launcher |

## Sources

- Flock CLI reference: https://flockphp.com/docs/cli
- Flock HTTPS documentation: https://flockphp.com/docs/https
- Laragon CLI documentation: https://laragon.org/docs/cli
- Laragon Pretty URLs and automatic virtual hosts: https://laragon.org/docs/pretty-urls
- Laragon Auto SSL: https://laragon.org/docs/auto-ssl
- Laragon directory structure and default Document Root: https://laragon.org/docs/directory-structure
- Current WPTUI integration: `internal/wpcli/wpcli.go`, `internal/create/create.go`, `internal/restore/restore.go`, and `internal/deprovision/deprovision.go`

## Registration ownership and status

- `flock sites` prints a `SITE PHP URL ROOT PATH` table, using the daemon status when available and configured sites when the daemon is down. It does not expose a JSON flag in the current implementation: https://github.com/flock-php/flock/blob/main/internal/cli/status.go
- `flock link <name>` upserts the selected name and can replace an existing directory mapping. WPTUI must inspect the existing mapping before linking and must not unregister a pre-existing mapping during rollback: https://github.com/flock-php/flock/blob/main/internal/cli/sites.go
- Flock saves configuration before committing runtime updates, so a command error can follow a persistent registration change; rollback ownership must be recorded before propagating that error: https://github.com/flock-php/flock/blob/main/internal/cli/sites.go
