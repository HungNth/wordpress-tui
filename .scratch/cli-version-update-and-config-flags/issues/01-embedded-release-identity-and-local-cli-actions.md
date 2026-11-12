# 01: Embedded Release Identity and Local CLI Actions

**Parent specification:** [CLI Version, Update Check, and Configuration Path Specification](../spec.md)

**What to build:** Give an installed WPTUI binary a manifest-backed release identity and a strict non-interactive command surface. Users can print the running version, discover the expected `config.json` path, or request help without initializing the TUI or first-run configuration. Invalid command syntax fails predictably, while invoking WPTUI without arguments preserves the existing interactive experience.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] The root application manifest remains the single source of the WPTUI product name and release version and is embedded into ordinary local, test, Make, and release builds without release-only linker flags or generated fallback values.
- [x] `wptui --version` and `wptui -v` print `wptui v<embedded-version>` to standard output and exit with status zero without network or configuration access.
- [x] Invalid or wrong-product embedded metadata produces a concise operational error rather than a fabricated `dev`, `unknown`, or fallback version.
- [x] `wptui --config` and `wptui -c` print only the absolute expected `config.json` path to standard output and exit with status zero, even when the file and parent directory do not exist.
- [x] Configuration-path discovery does not create a directory or file; failure to determine the user home directory writes an error to standard error and exits with status one.
- [x] `wptui --help` and `wptui -h` print concise English usage to standard output, list all agreed short/long action pairs including the upcoming update action, and exit with status zero.
- [x] Unknown flags, positional arguments, and invocations combining multiple action flags write an error with concise usage guidance to standard error and exit with status two.
- [x] No-argument invocation continues into the existing interactive WPTUI application with unchanged behavior.
- [x] Version, configuration-path, help, and invalid-syntax invocations finish before TUI initialization, first-run configuration, or `config.json` loading.
- [x] One command-level runner immediately below the process entrypoint accepts arguments and output streams and returns the intended process exit code; the process entrypoint remains a thin adapter.
- [x] Permanent tests exercise the complete command outcomes through that runner, covering both aliases, outputs, streams, exit statuses, invalid syntax, absent configuration, home-resolution failure, and isolation from interactive startup without asserting private parser structure or trivial entrypoint forwarding.
- [x] A real command smoke run demonstrates version, configuration-path, help, and representative invalid syntax on the built executable.
- [x] The README documents version, configuration-path, and help actions, including that configuration-path discovery does not create the file.
