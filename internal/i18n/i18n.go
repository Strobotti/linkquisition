// Package i18n provides localization support for the application.
// It uses go-i18n for message management and go-locale for system locale detection.
// English is the default/fallback language. The system locale is auto-detected
// unless overridden via the configuration file.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed translations/*.json
var translationFS embed.FS

var localizer *i18n.Localizer

// localeNames maps locale codes to their display names (in their own language).
var localeNames = map[string]string{
	"en": "English",
	"es": "Español",
	"fi": "Suomi",
	"sv": "Svenska",
}

// AvailableLocales returns the locale codes of all embedded translation files,
// sorted alphabetically.
func AvailableLocales() []string {
	entries, err := translationFS.ReadDir("translations")
	if err != nil {
		return []string{"en"}
	}

	var locales []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			code := strings.TrimSuffix(entry.Name(), ".json")
			locales = append(locales, code)
		}
	}

	return locales
}

// LocaleDisplayName returns a human-readable name for a locale code
// (e.g. "en" → "English", "fi" → "Suomi"). Falls back to the code itself.
func LocaleDisplayName(code string) string {
	if name, ok := localeNames[code]; ok {
		return name
	}

	return code
}

// Init initializes the i18n system. If localeOverride is non-empty, it is used
// as the locale. Otherwise the system locale is detected. English is always
// loaded as the fallback.
func Init(localeOverride string) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all available translation files
	entries, err := translationFS.ReadDir("translations")
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
				_, _ = bundle.LoadMessageFileFS(translationFS, "translations/"+entry.Name())
			}
		}
	}

	lang := detectLocale(localeOverride)
	localizer = i18n.NewLocalizer(bundle, lang, "en")
}

// T returns the translated string for the given message ID.
// Optional templateData is used for template substitution in the message.
func T(messageID string, templateData ...map[string]interface{}) string {
	cfg := &i18n.LocalizeConfig{
		MessageID: messageID,
	}

	if len(templateData) > 0 && templateData[0] != nil {
		cfg.TemplateData = templateData[0]
	}

	msg, err := localizer.Localize(cfg)
	if err != nil {
		// Fallback: return the message ID so it's obvious what's missing
		return fmt.Sprintf("[%s]", messageID)
	}

	return msg
}

// TWithCount returns the translated string for the given message ID with
// pluralization support.
func TWithCount(messageID string, count int, templateData ...map[string]interface{}) string {
	cfg := &i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: count,
	}

	if len(templateData) > 0 && templateData[0] != nil {
		cfg.TemplateData = templateData[0]
	} else {
		cfg.TemplateData = map[string]interface{}{"Count": count}
	}

	msg, err := localizer.Localize(cfg)
	if err != nil {
		return fmt.Sprintf("[%s]", messageID)
	}

	return msg
}

func detectLocale(override string) string {
	if override != "" {
		return override
	}

	if lang, err := locale.GetLocale(); err == nil && lang != "" {
		return lang
	}

	return "en"
}
