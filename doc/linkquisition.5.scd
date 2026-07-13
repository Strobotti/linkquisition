linkquisition(5)

# NAME

linkquisition - configuration file format

# SYNOPSIS

_~/.config/linkquisition/config.json_

# DESCRIPTION

The configuration file controls which browsers are available, how URLs are
matched to browsers, and which plugins are loaded.

The file is JSON and is managed either manually, via the configuration screen
(launched by running *linkquisition* with no arguments), or via the command-line
interface (see *linkquisition*(1) for details on *config*, *plugin*, *browsers*,
and *rule* subcommands).

# TOP-LEVEL KEYS

*locale* (string, optional)
	Override the UI language (e.g. _"fi"_, _"es"_, _"sv"_). If empty or omitted,
	the system locale is auto-detected. Falls back to English when no matching
	translation is available.

*logLevel* (string, optional)
	Log verbosity. One of: _debug_, _info_, _warn_, _error_. Default: _warn_.

*browsers* (array)
	List of browser objects. See *BROWSER OBJECT* below.

*plugins* (array, optional)
	List of plugin objects. See *PLUGIN OBJECT* below.

*ui* (object, optional)
	UI settings. See *UI OBJECT* below.

*security* (object, optional)
	URL safety checking settings. See *SECURITY OBJECT* below.

# BROWSER OBJECT

*name* (string)
	Display name of the browser.

*command* (string)
	Command to execute. Use _%U_ or _%u_ as the URL placeholder.

*iconPath* (string, optional)
	Absolute path to the browser icon file (e.g. _.png_). Populated
	automatically during browser scan from the _.desktop_ file's _Icon=_
	field. Can be set manually for custom browsers that are not discovered
	by scan.

*hidden* (boolean)
	If _true_, the browser is not shown in the picker but rules still apply.

*source* (string)
	Either _auto_ (discovered by scan) or _manual_ (user-added). Browsers with
	source _manual_ are never removed by a re-scan.

*matches* (array, optional)
	List of match-rule objects. See *MATCH OBJECT* below.

# MATCH OBJECT

*type* (string)
	One of:
	- *site* — matches the full hostname (e.g. _www.example.com_)
	- *domain* — matches the registered domain (e.g. _example.com_)
	- *regex* — matches the full URL against a regular expression

*value* (string)
	The value to match against (hostname, domain, or regex pattern).

# PLUGIN OBJECT

*path* (string)
	Path to the plugin _.so_ file, absolute or relative to the plugin folder
	(_/usr/lib/linkquisition/plugins/_).

*isDisabled* (boolean)
	If _true_, the plugin is not loaded.

*settings* (object, optional)
	Plugin-specific key-value settings.

# UI OBJECT

*hideKeyboardGuideLabel* (boolean, optional)
	If _true_, hides the keyboard shortcut guide at the bottom of the picker.

*pickerLayout* (string, optional)
	Layout style for the browser picker window. One of: _horizontal_ (default,
	grid of square icon buttons) or _vertical_ (list of buttons).

*maxItemsPerRow* (integer, optional)
	Maximum number of browser buttons per row in _horizontal_ layout.
	Default: _5_. Browsers exceeding this number wrap to additional rows.

*theme* (string, optional)
	UI color theme. One of: _system_ (default, follows the operating system
	preference), _dark_, or _light_. Changes take effect on next launch.

*showFavicon* (boolean, optional)
	If _true_, the browser picker displays the website's favicon next to the
	URL. The icon is fetched lazily (does not block startup) and a placeholder
	is shown until loading completes. Default: _false_.

*faviconStrategy* (string, optional)
	How to retrieve the favicon. Only relevant when _showFavicon_ is _true_.
	One of:
	- *direct* (default) — fetches _/favicon.ico_ from the host. Fast and
	  privacy-friendly.
	- *parsed* — downloads the site's HTML root page and parses the
	  _<link rel="icon">_ tag. More accurate (finds custom icon paths) but
	  requires two HTTP requests and may be slower.
	- *google* — uses Google's favicon service. Reliable and fast, but the
	  URL is sent to Google (privacy concern).

# SECURITY OBJECT

*enabled* (boolean, optional)
	If _true_, URL safety checking is active. The picker shows a color-coded
	dot indicating the safety status of the URL being opened. Default: _false_.

*provider* (string, optional)
	The safety checking provider. One of:
	- *google_safe_browsing* (default) — uses Google Safe Browsing Lookup API
	  v4. Checks against Google's threat lists (malware, social engineering,
	  unwanted software, potentially harmful applications).
	- *virustotal* — uses the VirusTotal URL analysis API. Aggregates results
	  from multiple antivirus engines and URL scanners.

*apiKey* (string, required when enabled)
	API key for the selected provider. Obtain from:
	- Google Safe Browsing: _https://console.cloud.google.com/apis/credentials_
	- VirusTotal: _https://www.virustotal.com/gui/my-apikey_

*cache* (object, optional)
	Settings for caching security check results. When enabled, results are
	stored locally and reused within the TTL window to reduce API calls and
	improve picker responsiveness. Results are cached separately per provider.

	*enabled* (boolean, optional)
		If _true_, security check results are cached. Default: _false_.

	*ttlHours* (integer, optional)
		Cache time-to-live in hours. Entries older than this are discarded and
		re-checked on next use. Default: _24_.

# EXAMPLE

```
{
  "logLevel": "info",
  "browsers": [
    {
      "name": "Firefox",
      "command": "firefox %u",
      "iconPath": "/usr/share/icons/hicolor/128x128/apps/firefox.png",
      "hidden": false,
      "source": "auto",
      "matches": [
        {
          "type": "site",
          "value": "www.mozilla.org"
        }
      ]
    }
  ],
  "plugins": [
    {
      "path": "unwrap",
      "isDisabled": false
    }
  ],
  "ui": {
    "theme": "system",
    "pickerLayout": "horizontal",
    "maxItemsPerRow": 5,
    "showFavicon": true,
    "faviconStrategy": "direct"
  },
  "security": {
    "enabled": true,
    "provider": "google_safe_browsing",
    "apiKey": "YOUR_API_KEY",
    "cache": {
      "enabled": true,
      "ttlHours": 24
    }
  }
}
```

# SEE ALSO

*linkquisition*(1)

# AUTHORS

Juha Jantunen <juha@strobotti.com>
