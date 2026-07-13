linkquisition(1)

# NAME

linkquisition - a fast, configurable browser-picker for Linux and macOS

# SYNOPSIS

*linkquisition* [_URL_]++
*linkquisition* --version++
*linkquisition* config [get|set|path]++
*linkquisition* plugin [list|enable|disable|add]++
*linkquisition* browsers [list|scan]++
*linkquisition* rule [list|add|remove]++
*linkquisition* set-default++
*linkquisition* test-url _URL_

# DESCRIPTION

Linkquisition is a browser-picker that lets you choose which browser opens a
given URL. When configured as the default browser, it intercepts all link
activations and either opens the URL in a matching browser automatically (based
on configured rules) or presents a picker window.

When launched without arguments, Linkquisition opens the configuration screen.

# OPTIONS

*--help*, *-h*
	Print usage information and exit.

*--version*, *-v*
	Print the version and exit.

*--plugin-opt* _plugin.key=value_
	Override a plugin setting at runtime without modifying the config file.
	The format is _pluginname.settingkey=value_ (e.g.
	_shenanigans.effect=matrix_). May be specified multiple times. Overrides
	for unrecognized plugins or disabled plugins are silently ignored.

*--no-plugins*
	Disable all plugin loading for this run. Useful for debugging startup
	issues caused by plugins.

*--log-level* _level_
	Override the log level for this run without modifying the config file.
	Accepted values: _debug_, _info_, _warn_, _error_. When not specified,
	the _logLevel_ setting from the config file is used.

# COMMANDS

## config

View or modify the configuration.

*config*
	Show the full configuration as JSON.

*config get* _KEY_
	Get the value of a configuration key.

*config set* _KEY_ _VALUE_
	Set a configuration key to a value.

*config path*
	Print the path to the configuration file.

Available keys: _locale_, _logLevel_, _ui.theme_.

## plugin

Manage plugins.

*plugin*, *plugin list*
	List configured plugins and their status. Also shows available plugins
	(_.so_ files in the plugin folder) that are not yet configured.

*plugin enable* _NAME_
	Enable a plugin by name.

*plugin disable* _NAME_
	Disable a plugin by name.

*plugin add* _NAME_
	Add an available plugin to the configuration with default settings.

Plugin names are matched case-insensitively against the base name of the
plugin path without the _.so_ extension.

## browsers

List and manage browsers.

*browsers*, *browsers list*
	List configured browsers with their commands, rule counts, and status.

*browsers scan*
	Scan the system for installed browsers and update the configuration.
	Manually added browsers and custom match rules are preserved.

## rule

Manage URL match rules.

*rule*, *rule list* [_BROWSER_]
	List all match rules, or rules for a specific browser.

*rule add* _BROWSER_ _TYPE_ _VALUE_
	Add a match rule to a browser. _TYPE_ must be one of: *site*, *domain*,
	*regex*. Regex patterns are validated before saving.

*rule remove* _BROWSER_ _INDEX_
	Remove a match rule by its 1-based index (as shown in *rule list*).

Browser names are matched case-insensitively with partial matching. Exact
matches take priority. Ambiguous partial matches produce an error.

## set-default

Set Linkquisition as the default browser. On Linux, this sets the
default-web-browser via _xdg-settings_. On macOS, this registers
Linkquisition as the handler for http/https URLs.

## test-url

*test-url* _URL_
	Simulate opening a URL without actually opening it. Traces each plugin's
	effect on the URL (modified, blocked, warned) and shows which browser
	would be selected by the matching rules. Useful for debugging plugin
	configurations and match rules.

# KEYBOARD SHORTCUTS

*Enter*
	Open the URL in the default (first) browser.

*Ctrl+C*
	Copy the URL to clipboard and close.

*Escape*
	Close the picker without opening a browser.

*1-9*
	Select a browser by its position in the list.

# CONFIGURATION

The configuration file is located at
_~/.config/linkquisition/config.json_ (Linux) or
_~/Library/Application Support/linkquisition/config.json_ (macOS).
See *linkquisition*(5) for the file format.

Launching *linkquisition* without arguments opens the configuration screen,
which allows you to set it as the default browser and scan for installed
browsers.

# PLUGINS

Plugins are shared-object files (_.so_) loaded at startup. They can modify URLs
before they are opened. See the project documentation for details.

Plugin locations:
- Linux: _/usr/lib/linkquisition/plugins/_
- macOS: _Linkquisition.app/Contents/Resources/plugins/_

# URL SAFETY CHECKING

When enabled in the configuration (see *linkquisition*(5)), the picker shows a
color-coded indicator next to the URL:

- *Green* — URL is considered safe
- *Yellow* — check failed or URL is suspicious
- *Red* — URL is flagged as malicious

Clicking the indicator opens a popup with the provider name, threat level, and
a link to view the full report on the provider's website.

Supported providers:
- *Google Safe Browsing* — checks against Google's threat lists
- *VirusTotal* — aggregates results from multiple antivirus engines

The check runs asynchronously and does not block the picker from opening.

# CONTEXT MENU

The browser picker has a context menu (⋯ button) next to the URL with these
actions:

*Copy URL*
	Copy the (possibly plugin-modified) URL to clipboard.

*QR Code*
	Generate a scannable QR code for the URL, useful for quickly opening it
	on a mobile device.

*WHOIS*
	Perform a live WHOIS lookup and display domain registration details:
	registrar, creation/expiry dates, domain age, DNSSEC status, name servers,
	and EPP status codes.

# FAVICON

When _ui.showFavicon_ is enabled in the configuration, the picker displays the
website's favicon next to the URL. The icon is fetched lazily and does not delay
picker startup.

Available strategies (set via _ui.faviconStrategy_):
- *direct* (default) — fetches _/favicon.ico_ from the host
- *parsed* — downloads the page HTML and parses the _<link rel="icon">_ tag
- *google* — uses Google's favicon service (URL is sent to Google)

Favicons are cached locally for 7 days.

# UPDATE CHECK

The About tab in the configuration screen has a "Check for updates" button that
queries the GitHub releases API. If a newer version is available, a link to the
release page is shown. No automatic update checking is performed.

# SEE ALSO

*linkquisition*(5), *xdg-settings*(1)

# AUTHORS

Juha Jantunen <juha@strobotti.com>
