# Agent Instructions

This file contains instructions for AI coding agents working on this project.
It documents conventions, gotchas, and the checklist to follow when making changes.

## Project overview

Linkquisition is a browser-picker for Linux and macOS written in Go. It supports
a plugin system (`.so` shared objects) that can modify URLs before they're opened.

## Documentation surfaces to keep in sync

When making changes, ensure ALL relevant documentation is updated:

1. **`README.md`** — project overview, features list, TODO section
2. **`plugins/README.md`** — plugin documentation, configuration examples, "Developing plugins" section
3. **`doc/linkquisition.1.scd`** — man page for the command (scdoc format)
4. **`doc/linkquisition.5.scd`** — man page for the config file format (scdoc format)

### When adding a new plugin

- [ ] Add a section in `plugins/README.md` with description, config example, and settings table
- [ ] Update `Taskfile.build.yml` — add the plugin name to the `PLUGINS` variable in `build-plugins`
- [ ] Update `README.md` if it references plugin capabilities or the TODO list
- [ ] Ensure the plugin's behavior is accurately described (don't leave "for now" placeholders in released docs)

### When changing plugin behavior

- [ ] Update `plugins/README.md` to match the actual implementation
- [ ] If settings or actions changed, update the settings table and examples
- [ ] If the plugin interface changed, update the "Developing plugins" section

### When changing the config file format

- [ ] Update `doc/linkquisition.5.scd` man page
- [ ] Update `README.md` example config if affected
- [ ] Update `plugins/README.md` example configs if affected

### When adding CLI flags or changing app behavior

- [ ] Update `doc/linkquisition.1.scd` man page
- [ ] Update the "Command-line interface" section in `README.md`
- [ ] Add new subcommands via `rootCmd.AddCommand()` in `cmd/root.go` `initRootCmd()`

## Plugin system conventions

- Plugins export `var Plugin <type>` as a package-level variable (value type, not pointer)
- All methods use pointer receivers
- The plugin interface has four methods: `Metadata`, `Setup`, `ProcessURL`, `Shutdown`
- `Metadata` returns `PluginMetadata` (name, description, author, version, settings descriptors)
- `Setup` returns an `error` — invalid config should fail loudly
- `ProcessURL` receives a `context.Context` and returns `PluginResult` (URL, Action, Message, ContinueChain)
- `Shutdown` receives a `context.Context` — respect the deadline
- Use `NewPluginServiceProvider` to access logger, settings, and config folder path
- Plugin `.so` files MUST be built from the same source tree as the main binary (interface mismatch = crash)

### Testing plugins

- Use `NewForTesting()` if the plugin struct contains a `sync.Mutex` (avoid copying the global `Plugin` var)
- Pass `""` for `configFolderPath` in tests unless testing file I/O
- Use `t.TempDir()` for any file system operations in tests
- Use `httptest.NewServer` for testing HTTP-dependent plugins

## Build system

- **Taskfile** (not Make) — see `Taskfile.yml`, `Taskfile.build.yml`, `Taskfile.package.yml`
- `task build` — build binary
- `task build:plugins` / `task build:plugins-darwin` — build all plugin `.so` files
- `task package:deb` — Linux `.deb` package
- `task package:app` — macOS `.app` bundle (includes plugins)
- `task package:app:install` — build + install to `/Applications`
- The `PLUGINS` var in `Taskfile.build.yml` is the single source of truth for which plugins are built

## Linting

The project uses `golangci-lint`. The full rule configuration is in `.golangci.yml` — refer to
it for the authoritative list of enabled linters and their settings. The linter runs in CI only
(the CI version may differ from what's installed locally). Key rules to watch for:

- **goconst** — extract repeated strings (3+ occurrences) into constants
- **gosec** — security checks (e.g. `G306`: file permissions must be 0600 or less)
- **mnd** — no magic numbers in arguments/conditions; use named constants
- **lll** — max 140 characters per line
- **noctx** — use `http.NewRequestWithContext` instead of `http.Get` / `client.Get`
- **gocritic/paramTypeCombine** — combine consecutive same-type params: `func(a, b string)` not
  `func(a string, b string)`
- **gocritic/exitAfterDefer** — don't call `os.Exit` in a function with defers; use a helper
- **unparam** — if a function always returns nil for error, either fix it or add `//nolint:unparam`

## Platform-specific code

- `cmd/application_linux.go` — Linux-specific setup (build tag `//go:build linux`)
- `cmd/application_darwin.go` — macOS-specific setup (build tag `//go:build darwin`)
- Both must stay in sync for shared patterns (e.g. `NewPluginServiceProvider` call)

## The `cmd` package

The `cmd/` directory is `package main` — it cannot be imported by other packages.
Tests for code in `cmd/` must live in `cmd/*_test.go` files (same package). Use
the `linkquisition.FileSettingsService` with a test `PathProvider` for integration-style
tests (see `cmd/log_rotation_test.go` for an example).

### CLI architecture (cobra)

The CLI uses `github.com/spf13/cobra`. Key files:

- `cmd/root.go` — root command, `initRootCmd()` registers all subcommands
- `cmd/cmd_config.go` — `config` subcommand + `newSettingsServiceForCLI()` / `newBrowserServiceForCLI()` helpers
- `cmd/cmd_plugin.go` — `plugin` subcommand (list/enable/disable/add + plugin discovery)
- `cmd/cmd_browsers.go` — `browsers` subcommand (list/scan)
- `cmd/cmd_rule.go` — `rule` subcommand (list/add/remove + `findBrowserByName()`)
- `cmd/cmd_set_default.go` — `set-default` subcommand

Design principles:

- CLI commands only create `SettingsService` / `BrowserService` — no fyne/GUI initialization
- GUI is only initialized in `runRoot()` when no subcommand is matched
- `Application.RunGUI(ctx, url)` is the entry point for GUI modes (configurator or picker)
- Add new subcommands via `rootCmd.AddCommand()` in `initRootCmd()`

### GUI modes

The app has two UI modes, both in `cmd/`:

- **Configurator** (`cmd/configurator.go`) — settings screen, shown when launched with no args.
  The General tab is composed of `build*Section()` methods. Add new sections there.
- **BrowserPicker** (`cmd/browser_picker.go`) — the URL picker, shown when launched with a URL.

## Settings and config model

- `settings.go` — `Settings` struct, constants (`LogLevel*`, `BrowserMatchType*`, `Source*`),
  `SettingsService` interface, `MapSettingsLogLevelToSlog`
- `settings_service.go` — `FileSettingsService` (shared, platform-independent implementation)
- Use named constants for repeated string values (log levels, match types, sources)

## Paths on each platform

| What                  | Linux                             | macOS                                                 |
|-----------------------|-----------------------------------|-------------------------------------------------------|
| Config                | `~/.config/linkquisition/`        | `~/Library/Application Support/linkquisition/`        |
| Logs                  | `~/.local/state/linkquisition/`   | `~/Library/Logs/linkquisition/`                       |
| Plugins (system)      | `/usr/lib/linkquisition/plugins/` | `Linkquisition.app/Contents/Resources/plugins/`       |
| Plugin cache (defang) | `~/.config/linkquisition/defang/` | `~/Library/Application Support/linkquisition/defang/` |

## Localization

- Translation files live in `internal/i18n/translations/` (JSON, keyed by locale code)
- Supported: `en`, `fi`, `es`, `sv`
- The `locale` config key overrides system detection
- Plugins do NOT currently have access to the i18n system
