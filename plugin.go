package linkquisition

import (
	"context"
	"log/slog"
)

// PluginAction determines what the host application should do after a plugin processes a URL.
type PluginAction int

const (
	// ActionContinue passes the (possibly modified) URL to the next plugin or browser matching.
	ActionContinue PluginAction = iota

	// ActionBlock stops processing and shows the Message to the user without opening anything.
	ActionBlock

	// ActionWarn shows the Message with an option for the user to proceed or cancel.
	ActionWarn

	// ActionOpenDirect bypasses browser matching and opens the URL in the first available browser.
	ActionOpenDirect
)

// PluginResult represents what a plugin wants to happen with the URL.
type PluginResult struct {
	// URL is the (possibly modified) URL. Ignored when Action is ActionBlock.
	URL string

	// Action tells the host app what to do next.
	Action PluginAction

	// Message is shown to the user when Action is ActionBlock or ActionWarn.
	Message string

	// ContinueChain indicates whether subsequent plugins should still run.
	// Default (false) means stop the chain when Action is not ActionContinue.
	// When Action is ActionContinue, the chain always continues regardless of this field.
	ContinueChain bool
}

// PluginSettingType describes the data type of a plugin configuration setting.
type PluginSettingType int

const (
	SettingTypeString     PluginSettingType = iota // free-form string
	SettingTypeBool                                // boolean toggle
	SettingTypeInt                                 // integer number
	SettingTypeDuration                            // Go duration string (e.g. "5s", "168h")
	SettingTypeStringList                          // list of strings
	SettingTypeChoice                              // one of the values in Options
)

// PluginSettingDescriptor tells the host application (and eventually the GUI)
// how to render and validate a configuration field for a plugin.
type PluginSettingDescriptor struct {
	// Key is the JSON key in the plugin's settings map
	Key string

	// Label is a human-readable label for the setting
	Label string

	// Description provides additional context (tooltip / help text)
	Description string

	// Type indicates what kind of value this setting holds
	Type PluginSettingType

	// Default is the default value if not configured
	Default interface{}

	// Required indicates whether the setting must be provided
	Required bool

	// Options lists the valid values when Type is SettingTypeChoice
	Options []string
}

// PluginMetadata describes a plugin to the host application and GUI.
type PluginMetadata struct {
	// Name is the human-readable plugin name (e.g. "Defang", "Sanitize")
	Name string

	// Description is a short description of what the plugin does
	Description string

	// Author is the plugin author's name or handle
	Author string

	// Version is the plugin's version string
	Version string

	// URL is a link to the plugin's documentation or source
	URL string

	// Settings describes all configuration options the plugin accepts
	Settings []PluginSettingDescriptor
}

// Plugin is the interface that all plugins must implement.
type Plugin interface {
	// Metadata returns static information about the plugin, including
	// its name, description, and configurable settings.
	Metadata() PluginMetadata

	// Setup is called once when the plugin is loaded. Returns an error
	// if the plugin cannot initialize (e.g. invalid configuration).
	Setup(serviceProvider PluginServiceProvider, config map[string]interface{}) error

	// ProcessURL is called for each URL before browser matching.
	// The context carries a deadline — plugins must respect it.
	// Returns a PluginResult indicating what should happen next.
	ProcessURL(ctx context.Context, url string) PluginResult

	// Shutdown is called when the application is about to exit.
	// Plugins should use this to finish any background work (e.g. writing files).
	// The context carries a deadline — plugins must return before it expires.
	Shutdown(ctx context.Context)
}

// PluginServiceProvider is an interface that provides the logger and settings to the plugin.
// This is passed to the plugin as a dependency during Setup.
type PluginServiceProvider interface {
	GetLogger() *slog.Logger
	GetSettings() *Settings
	GetConfigFolderPath() string
}

// pluginServiceProvider is a struct that implements the PluginServiceProvider interface
type pluginServiceProvider struct {
	logger           *slog.Logger
	Settings         *Settings
	configFolderPath string
}

func (p *pluginServiceProvider) GetLogger() *slog.Logger {
	return p.logger
}

func (p *pluginServiceProvider) GetSettings() *Settings {
	return p.Settings
}

func (p *pluginServiceProvider) GetConfigFolderPath() string {
	return p.configFolderPath
}

func NewPluginServiceProvider(logger *slog.Logger, settings *Settings, configFolderPath string) PluginServiceProvider {
	return &pluginServiceProvider{logger: logger, Settings: settings, configFolderPath: configFolderPath}
}
