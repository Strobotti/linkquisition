# Agent Instructions

This file contains instructions for AI coding agents working on this project.
It documents conventions, gotchas, and the checklist to follow when making changes.

## Project overview

Linkquisition is a browser-picker for Linux and macOS written in Go. It supports
a plugin system (`.so` shared objects) that can modify URLs before they're opened.

## Commit conventions

This project uses **Angular conventional commits**. Every commit message must have
a type prefix:

- `feat:` — new feature (minor version bump)
- `fix:` — bug fix (patch version bump)
- `feat!:` or `fix!:` with `BREAKING CHANGE:` in body — major version bump
- `refactor:`, `docs:`, `style:`, `test:`, `chore:`, `ci:`, `perf:` — no release

Prefer **small, self-contained commits** over large monolithic changes. Each commit
should ideally do one thing and be independently reviewable. If a feature requires
multiple steps (e.g. interface change + migration + tests + docs), split them into
separate commits on the same branch rather than squashing everything together.

## Branching and pull requests

Always work on a **feature branch** — never commit directly to `main`. Branch naming:

- `feat/<short-description>` — new features
- `fix/<short-description>` — bug fixes
- `chore/<short-description>` — maintenance, CI, docs-only
- `refactor/<short-description>` — code restructuring

Push the branch and open a **pull request** for review. PRs are merged via GitHub
(squash or merge commit, depending on the number of meaningful commits). The `main`
branch is protected and triggers the release-please flow on every push.

## Release flow (CI)

The project uses [release-please](https://github.com/googleapis/release-please-action)
for automated releases:

1. Push to `main` → `release-please.yml` creates/updates a **Release PR** with a
   changelog and version bump based on conventional commit types
2. The same workflow updates the Flatpak metainfo XML on the Release PR branch
3. Merge the Release PR → release-please creates a GitHub release + pushes a tag
4. Tag push (`v*`) triggers `publish.yml` → test → build → upload assets (`.deb`,
   `.rpm`, AppImage, macOS `.zip`) → update Homebrew tap → update coverage badge

Key files:

- `.github/workflows/release-please.yml` — Release PR + metainfo update
- `.github/workflows/publish.yml` — build/package/upload on tag push
- `release-please-config.json` + `.release-please-manifest.json` — release-please config

## Documentation surfaces to keep in sync

When making changes, ensure ALL relevant documentation is updated:

1. **`README.md`** — project overview, features list, TODO section
2. **`plugins/README.md`** — plugin documentation, configuration examples, "Developing plugins" section
3. **`doc/linkquisition.1.scd`** — man page for the command (scdoc format)
4. **`doc/linkquisition.5.scd`** — man page for the config file format (scdoc format)

### When adding a new plugin

- [ ] Add a section in `plugins/README.md` with description, config example, and settings table
- [ ] Update `Taskfile.build.yml` — add the plugin name to the `PLUGINS` variable in `build-plugins`
- [ ] Update `.goreleaser.yaml` — add a build entry for the plugin and a contents entry in the `nfpms` section
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

### When changing macOS packaging or app identity

- [ ] Update `package/darwin/Info.plist` — bundle metadata, URL schemes, version keys
- [ ] If adding new URL schemes or entitlements, update the plist accordingly
- [ ] If changing the minimum macOS version, update `LSMinimumSystemVersion`
- [ ] Update `homebrew/Casks/linkquisition.rb` if install paths or dependencies change

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

### Wayland protocol generation (Linux only)

GLFW v3.4 compiles with both X11 and Wayland support by default on Linux. The Wayland
backend requires generated protocol headers that are not in the vendored source tree.
Run `./scripts/generate-wayland-protocols.sh` once after cloning (requires `libwayland-dev`
which provides `wayland-scanner`). CI workflows do this automatically.

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

### UI helpers (`internal/ui/`)

- **`ui.NewLinkWithCopy(text, rawURL, window)`** — creates a clickable hyperlink with a
  copy-to-clipboard button beside it. Use this whenever displaying a URL or link in the GUI
  (e.g. reference links, documentation URLs, external resources). Prefer this over a plain
  `widget.NewHyperlink` for consistency across the app.
- **`ui.FormLabel(text)`** / **`ui.FormValue(value)`** — bold label and wrapping value label
  for form-style key-value layouts (grids). `FormValue` substitutes "—" for empty strings.
- **`ui.NewTappableContainer(content, onTapped)`** — wraps any `CanvasObject` to make it
  tappable with a hover highlight (rounded rectangle). Used for grid-style interactive layouts
  like the horizontal browser picker.
- **`ui.WithAltRowBackground(obj)`** — wraps a widget with a subtle alternating-row background
  tint for zebra-striping in lists.
- **Color constants** (`ui.ColorSuccess`, `ui.ColorWarning`, `ui.ColorDanger`, `ui.ColorNeutral`,
  `ui.ColorHoverBg`, `ui.ColorAltRowBg`) — semantic colors for status indicators, validation
  feedback, and background highlights. Always use these instead of hardcoding `color.NRGBA` values.

### Configurator helpers

- **`c.parentWindow()`** — returns the main configurator window. Use this instead of the
  verbose `c.fapp.Driver().AllWindows()` pattern when you need a parent window for dialogs.

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
- Supported: `de`, `en`, `es`, `fi`, `fr`, `hu`, `pt`, `pt-BR`, `sv`, `uk`
- The `locale` config key overrides system detection
- Plugins do NOT currently have access to the i18n system

### When adding or changing i18n keys

- [ ] Add the new key to `en.json` first (this is the fallback/source of truth)
- [ ] Add translated values to ALL other locale files (`de`, `es`, `fi`, `fr`, `hu`, `pt`, `pt-BR`, `sv`, `uk`)
- [ ] Keep the key ordering consistent across all files (append new keys at the end)
- [ ] If a translation is uncertain, use the English text as a placeholder — it's better
  than a missing key (which causes the raw key string to appear in the UI)

## Linux packaging

The project produces `.deb`, `.rpm`, and AppImage packages.

### Key files

- `.goreleaser.yaml` — build entries for the binary + all plugins, nfpm packaging config
- `Taskfile.build.yml` — `PLUGINS` variable (source of truth for which plugins exist)
- `Taskfile.package.yml` — local packaging tasks (for dev testing only)
- `scripts/build-appimage.sh` — AppImage build script (downloads linuxdeploy, bundles deps)
- `templates/DEBIAN/control.tpl` — local deb control template (used by `task package:deb`)
- `templates/linkquisition.desktop` — `.desktop` file installed to `/usr/share/applications/`
- `.github/workflows/publish.yml` — CI release workflow

### Keeping packaging in sync

The `PLUGINS` variable in `Taskfile.build.yml` is the canonical list of plugins.
When adding or removing a plugin, ALL of the following must be updated:

1. `Taskfile.build.yml` — `PLUGINS` variable
2. `.goreleaser.yaml` — add/remove a `builds` entry AND a `contents` entry under `nfpms`
3. `.github/workflows/publish.yml` — if the workflow references plugins explicitly

The `.goreleaser.yaml` plugin build ID pattern is `"<name>-plugin"` and the contents
path pattern is `./dist/<name>-plugin_{{ .Os }}_{{ .Arch }}_v1/plugins/<name>.so`
→ `/usr/lib/linkquisition/plugins/<name>.so`.

### AppImage

The AppImage is built by `scripts/build-appimage.sh` which:

1. Assembles an AppDir with the binary, plugins, desktop file, icon, and man pages
2. Uses `linuxdeploy` to bundle shared library dependencies (libGL, X11, etc.)
3. Outputs `dist/Linkquisition-<VERSION>-x86_64.AppImage`

The script downloads `linuxdeploy` automatically into `dist/tools/` on first run.
Use `--appimage-extract-and-run` mode so it works in CI without FUSE.

To build locally: `task package:appimage` (Linux only).

### Flatpak / Flathub

Flatpak packaging files live in `flatpak/`. See `flatpak/README.md` for the full
submission and build guide. Key points:

- **App ID**: `io.github.strobotti.linkquisition`
- **Manifest**: `flatpak/io.github.strobotti.linkquisition.yml`
- **MetaInfo**: `flatpak/io.github.strobotti.linkquisition.metainfo.xml`
- **Runtime**: `org.freedesktop.Platform//24.08` with `org.freedesktop.Sdk.Extension.golang`
- **Dependencies**: Go modules must be vendored (`go mod vendor`) — no network during build
- **Plugins**: built inside the Flatpak sandbox alongside the main binary

When releasing a new version, update the `<releases>` section in the metainfo XML.
The Flathub manifest (in the separate Flathub repo) must be updated with the new
tag and commit SHA.
