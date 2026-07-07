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
  ]
}
```

# SEE ALSO

*linkquisition*(1)

# AUTHORS

Juha Jantunen <juha@strobotti.com>
