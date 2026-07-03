# Plugins (experimental)

Plugins are a way to add more complex functionality to Linkquisition, beyond just simple rule-based matching.

## [Unwrap](./unwrap/unwrap.go) -plugin

This is the first plugin that I have written for Linkquisition. I use it to "unwrap" the internal links (in my company)
that are "wrapped" inside Microsoft Defender (Evergreen) URL when they are passed via Microsoft Teams or Outlook.

Now, the actual Defender URL is a good feature, but it is not very useful when the links refer to internal resources,
such as GitLab, Jira, Confluence, whatnot. To leverage the actually safeguarding part of the Defender URL, the plugin
can (and should) be configured to only unwrap the known sources, and leave the rest to be opened via the Defender URL.

To enable the plugin and configure it to unwrap the Defender links from Microsoft Teams you can use the following
configuration:

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
        "requireBrowserMatchToUnwrap": true,
        "rules": [
          {
            "match": "^https://statics\\.teams\\.cdn\\.office\\.net/evergreen-assets/safelinks",
            "parameter": "url"
          }
        ]
      }
    }
  ]
}
```

In the above example, the `requireBrowserMatchToUnwrap` -setting is set to `true`, which means that the plugin will only
unwrap the links if there is a matching browser-rule for that URL and all the rest of the URLs are opened with full
Evergreen-protected URL.

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

- `block` — replaces the URL with `about:blank`, preventing the browser from opening the malicious site
- `warn` — same as `block` for now (future: show a confirmation dialog)
- `log` — logs the blocked domain but still opens the URL (useful for monitoring without disruption)

## Developing plugins

As stated before, the plugins-feature is experimental, the API is not stable and therefore subject to change. However,
the plugin-interface is quite simple and should be easy to implement.

The plugin is a shared object file (`.so`) that is loaded by the main application. The plugin must implement the
[linkquisition.Plugin](../plugin.go) -interface which has three methods:

- `Setup(serviceProvider, config)` — called once when the plugin is loaded. The `serviceProvider` gives access to the
  logger, settings, and the config folder path (for storing cache files etc.)
- `ModifyUrl(url) string` — called just before the URL is matched against the browser-rules. Should return the modified
  URL, or the original if no modification is needed.
- `Shutdown(ctx context.Context)` — called when the application is about to exit. Plugins with background work (e.g.
  downloads) should use this to finish gracefully before the context deadline expires.
