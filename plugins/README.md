# Plugins

Plugins extend Linkquisition with URL processing capabilities beyond simple rule-based matching.
Each plugin can inspect, modify, block, or warn about URLs before they reach the browser.

## [Unwrap](./unwrap/unwrap.go) -plugin

This plugin "unwraps" URLs that are wrapped inside redirect/tracking URLs, extracting the actual
destination. Common use cases include:

- **Microsoft Teams SafeLinks** — links wrapped via `statics.teams.cdn.office.net/evergreen-assets/safelinks`
- **Outlook SafeLinks** — links wrapped via `*.safelinks.protection.outlook.com`

The plugin matches URLs against configurable regex rules and extracts the target from a specified
query parameter. To leverage the safety features of SafeLinks for unknown URLs, the plugin can be
configured to only unwrap URLs where the destination matches a browser rule (`requireBrowserMatchToUnwrap`).

### Configuration

To unwrap both Teams and Outlook SafeLinks:

```json
{
  "browsers": [
    {
      "name": "Firefox",
      "command": "firefox %u",
      "hidden": false,
      "source": "auto",
      "matches": [
        {
          "type": "regex",
          "value": "^https?://github\\.com/Strobotti/"
        }
      ]
    }
  ],
  "plugins": [
    {
      "path": "unwrap.so",
      "settings": {
        "requireBrowserMatchToUnwrap": false,
        "rules": [
          {
            "match": "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
            "parameter": "url"
          },
          {
            "match": "^https://[a-z0-9]+\\.safelinks\\.protection\\.outlook\\.com/",
            "parameter": "url"
          }
        ]
      }
    }
  ]
}
```

### Settings

| Setting                       | Type                     | Default | Description                                                         |
|-------------------------------|--------------------------|---------|---------------------------------------------------------------------|
| `rules`                       | []{"match", "parameter"} | `[]`    | List of regex/parameter pairs defining which URLs to unwrap         |
| `requireBrowserMatchToUnwrap` | bool                     | `false` | Only unwrap if a browser rule matches the unwrapped destination URL |

Each rule has two fields:

- `match` — a regex that the incoming URL is tested against
- `parameter` — the query parameter name containing the actual destination URL

If `requireBrowserMatchToUnwrap` is `true`, the plugin will only unwrap URLs when the extracted
destination matches one of your configured browser rules. This preserves the SafeLinks protection
for unknown/external URLs while unwrapping known internal ones.

## [Terminus](./terminus/terminus.go) -plugin

This plugin can be used to resolve redirects before processing actual rules or showing the browser picker dialog to the
user.

The plugin is configurable in terms of how many redirect-"jumps" to follow and the time-limit how long the requests are
allowed to take before giving up. Here's an example:

```json
{
  "browsers": [
    ...
  ],
  "plugins": [
    {
      "path": "terminus.so",
      "isDisabled": false,
      "settings": {
        "maxRedirects": 10,
        "requestTimeout": "2s"
      }
    }
  ]
}

```

## [Sanitize](./sanitize/sanitize.go) -plugin

This plugin strips tracking and marketing query parameters (UTM tags, click IDs, etc.) from URLs before they are
matched against browser rules or opened in a browser. This keeps your browser history and bookmarks clean from
clutter like `?utm_source=newsletter&utm_medium=email&fbclid=abc123`.

By default, the plugin removes a comprehensive list of well-known tracking parameters from all major platforms
(Google Analytics, Meta/Facebook, Microsoft, HubSpot, Mailchimp, Yandex, and more). You can also add your own
parameters or regex patterns, and optionally limit sanitization to specific URLs.

### Configuration

```json
{
  "browsers": [
    ...
  ],
  "plugins": [
    {
      "path": "sanitize.so",
      "settings": {
        "stripDefaults": true,
        "extraParams": [
          "ref",
          "igshid"
        ],
        "extraPatterns": [
          "^_ga"
        ],
        "onlyMatchingUrls": ""
      }
    }
  ]
}
```

### Settings

| Setting            | Type     | Default | Description                                                                               |
|--------------------|----------|---------|-------------------------------------------------------------------------------------------|
| `stripDefaults`    | bool     | `true`  | Whether to strip the built-in list of known tracking parameters                           |
| `extraParams`      | []string | `[]`    | Additional exact parameter names to strip                                                 |
| `extraPatterns`    | []string | `[]`    | Regex patterns to match parameter names against (e.g. `^_ga` matches `_ga`, `_gac`, etc.) |
| `onlyMatchingUrls` | string   | `""`    | Regex pattern; if set, only URLs matching this pattern are sanitized                      |

### Default parameters stripped

The built-in list includes parameters from: Google Analytics/Ads (`utm_*`, `gclid`, `gclsrc`, `dclid`, `gad_source`),
Meta/Facebook (`fbclid`, `fb_action_ids`, `fb_action_types`, `fb_source`, `fb_ref`),
Microsoft (`msclkid`), Twitter/X (`twclickid`, `twsrc`, `tweetid`),
HubSpot (`_hsenc`, `_hsmi`, `__hssc`, `__hstc`, `__hsfp`, `hsCtaTracking`),
Mailchimp (`mc_cid`, `mc_eid`), Yandex (`yclid`, `ymclid`), Vero (`vero_id`, `vero_conv`),
Marketo (`mkt_tok`), Adobe (`s_cid`), and common social/affiliate trackers (`igshid`, `si`, `ref_src`, `ref_url`).

## [Defang](./defang/defang.go) -plugin

This plugin checks URLs against known-malicious domain blocklists before they reach the browser. It downloads and
caches hosts-format blocklists locally, checking them on every URL open without any network request in the hot path.

By default, it uses two well-known, trusted sources:

- [URLhaus](https://urlhaus.abuse.ch/) (abuse.ch) — malware distribution URLs
- [Steven Black's hosts](https://github.com/StevenBlack/hosts) — aggregated malware/adware/phishing domains

The blocklists are cached in the config directory (e.g. `~/.config/linkquisition/defang/` on Linux) and refreshed
in the background when older than the configured update interval. The app never blocks on network I/O — it uses
whatever is cached and updates lazily.

### Configuration

```json
{
  "browsers": [
    ...
  ],
  "plugins": [
    {
      "path": "defang.so",
      "settings": {
        "sources": [
          "https://urlhaus.abuse.ch/downloads/hostfile/",
          "https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts"
        ],
        "updateInterval": "168h",
        "action": "block"
      }
    }
  ]
}
```

### Settings

| Setting          | Type     | Default     | Description                                                                      |
|------------------|----------|-------------|----------------------------------------------------------------------------------|
| `sources`        | []string | (see above) | URLs to download hosts-format blocklists from                                    |
| `updateInterval` | string   | `"168h"`    | How often to refresh the cached blocklists (Go duration format, default: 7 days) |
| `action`         | string   | `"block"`   | What to do when a blocked domain is detected: `block`, `warn`, or `log`          |

### Actions

- `block` — shows a native dialog informing the user the domain is blocked, without opening anything
- `warn` — shows a native dialog with an "Open anyway" option, letting the user choose to proceed
- `log` — logs the blocked domain but still opens the URL normally (useful for monitoring without disruption)

## [Shenanigans](./shenanigans/shenanigans.go) -plugin

This plugin adds completely useless but entertaining visual effects to the browser picker window.
It demonstrates the `PluginUIHook` interface — an optional extension that allows plugins to interact
with the picker GUI.

The plugin does not modify URLs in any way. It simply overlays animated effects on the picker window
when it appears.

### Available effects

- `random` — Randomly picks one of the effects below (default)
- `asteroids` — Wireframe asteroids with a ship hunting them down
- `aurora` — Northern lights with flowing green/purple/blue bands
- `boids` — Boids flocking simulation with separation, alignment, and cohesion
- `breakout` — Classic breakout with colorful bricks and paddle AI
- `dino` — Chrome offline T-rex jumping over cacti
- `fire` — Realistic flames rising from the bottom of the window
- `fireflies` — Glowing fireflies drifting with pulsing bioluminescence
- `fireworks` — Rockets launching and exploding into colorful particle bursts
- `flappy` — Flappy Bird navigating through pipe gaps
- `football` — Top-down football/soccer pitch with animated spotlight sweep
- `frogger` — Frogger dodging traffic and hopping on logs
- `glitch` — Cyberpunk-style RGB channel splitting with periodic static bursts
- `invaders` — Space invaders with marching formation and explosions
- `lava` — Lava lamp with soft blobs rising and falling
- `life` — Conway's Game of Life cellular automata
- `matrix` — Green Matrix-style falling characters
- `minesweeper` — Minesweeper being auto-solved, revealing the grid progressively
- `pacman` — Pac-man navigating a maze, eating dots, avoiding ghosts
- `pipes` — Windows 3D Pipes screensaver with colorful growing pipes and joints
- `plasma` — Classic demoscene swirling color blobs
- `pong` — Auto-playing ping-pong
- `pride` — Animated rainbow pride flag waving in the wind
- `rain` — Fast vertical rain streaks with bottom splashes
- `raycast` — Wolfenstein 3D-style first-person maze explorer
- `sinescroll` — Retro demoscene sine-wave text scroller with color cycling
- `snake` — Snake chasing food with AI pathfinding
- `snow` — Gentle snowfall with varying sizes and wobble
- `starfield` — 3D warp-speed stars flying toward the viewer
- `tetris` — Tetris with falling tetrominoes stacking and clearing lines

### Configuration

```json
{
  "browsers": [
    ...
  ],
  "plugins": [
    {
      "path": "shenanigans.so",
      "settings": {
        "effect": "random"
      }
    }
  ]
}
```

### Settings

| Setting  | Type   | Default    | Description                                      |
|----------|--------|------------|--------------------------------------------------|
| `effect` | choice | `"random"` | Which visual effect to show on the picker window |

## Developing plugins

The plugin is a shared object file (`.so`) that is loaded by the main application. The plugin must implement the
[linkquisition.Plugin](../plugin.go) interface which has four methods:

- `Metadata() PluginMetadata` — returns static information about the plugin (name, description, author, version,
  and a list of `PluginSettingDescriptor` values describing the configurable settings). This is used by the CLI
  and will be used by the GUI in a future version.
- `Setup(serviceProvider, config) error` — called once when the plugin is loaded. The `serviceProvider` gives access
  to the logger, settings, and the config folder path. Returns an error if the plugin cannot initialize.
- `ProcessURL(ctx context.Context, url string) PluginResult` — called for each URL before browser matching.
  Returns a `PluginResult` with the (possibly modified) URL and an action:
    - `ActionContinue` — pass the URL to the next plugin or browser matching
    - `ActionBlock` — stop processing and show a blocking message to the user
    - `ActionWarn` — show a warning dialog with option to proceed or cancel
    - `ActionOpenDirect` — bypass browser matching and open in the first available browser
- `Shutdown(ctx context.Context)` — called when the application is about to exit. Plugins with background work (e.g.
  downloads) should use this to finish gracefully before the context deadline expires.

### Optional: PluginUIHook interface

Plugins can optionally implement the [`PluginUIHook`](../plugin_ui_hook.go) interface to receive a callback when the
browser picker window is shown:

- `OnPickerShown(window fyne.Window)` — called after the picker window content is set and before it is displayed.
  The plugin can add canvas overlays, start animations, or modify the window appearance.

The host application detects this interface via a type assertion — plugins that don't implement it are simply skipped.
See the [Shenanigans](./shenanigans/shenanigans.go) plugin for an example.
