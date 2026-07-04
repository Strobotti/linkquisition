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
*linkquisition* set-default

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

Available keys: _locale_, _logLevel_.

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

# SEE ALSO

*linkquisition*(5), *xdg-settings*(1)

# AUTHORS

Juha Jantunen <juha@strobotti.com>
