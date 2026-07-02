linkquisition(1)

# NAME

linkquisition - a fast, configurable browser-picker for Linux

# SYNOPSIS

*linkquisition* [_URL_]++
*linkquisition* --version

# DESCRIPTION

Linkquisition is a browser-picker that lets you choose which browser opens a
given URL. When configured as the default browser, it intercepts all link
activations and either opens the URL in a matching browser automatically (based
on configured rules) or presents a picker window.

When launched without arguments, Linkquisition opens the configuration screen.

# OPTIONS

*--version*, *-v*
	Print the version and exit.

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
_~/.config/linkquisition/config.json_. See *linkquisition*(5) for the file
format.

Launching *linkquisition* without arguments opens the configuration screen,
which allows you to set it as the default browser and scan for installed
browsers.

# PLUGINS

Plugins are shared-object files (_.so_) loaded at startup. They can modify URLs
before they are opened. See the project documentation for details.

# SEE ALSO

*linkquisition*(5), *xdg-settings*(1)

# AUTHORS

Juha Jantunen <juha@strobotti.com>
