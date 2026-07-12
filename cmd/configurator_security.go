package main

import (
	"context"
	"net/url"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition"
	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/safety"
)

const (
	securityTestTimeout = 10 * time.Second

	googleSafeBrowsingURL = "https://console.cloud.google.com/apis/credentials"
	virusTotalURL         = "https://www.virustotal.com/gui/my-apikey"
)

func (c *Configurator) getSecurityTab() fyne.CanvasObject {
	settings := c.settingsService.GetSettings()

	enabledCheck := widget.NewCheck(i18n.T("config.security_enabled"), nil)
	enabledCheck.Checked = settings.Security.Enabled

	providerSelect := widget.NewSelect(
		[]string{
			i18n.T("config.security_provider_google"),
			i18n.T("config.security_provider_virustotal"),
		},
		nil,
	)

	if settings.Security.GetProvider() == linkquisition.SecurityProviderVirusTotal {
		providerSelect.SetSelectedIndex(1)
	} else {
		providerSelect.SetSelectedIndex(0)
	}

	apiKeyEntry := widget.NewPasswordEntry()
	apiKeyEntry.SetPlaceHolder(i18n.T("config.security_api_key_placeholder"))
	apiKeyEntry.SetText(settings.Security.APIKey)

	providerLink := widget.NewHyperlink(
		i18n.T("config.security_get_key"),
		parseURL(googleSafeBrowsingURL),
	)

	testStatus := widget.NewLabel("")

	updateProviderLink := func() {
		if providerSelect.SelectedIndex() == 1 {
			providerLink.SetURL(parseURL(virusTotalURL))
		} else {
			providerLink.SetURL(parseURL(googleSafeBrowsingURL))
		}
	}

	providerSelect.OnChanged = func(_ string) {
		updateProviderLink()
		c.saveSecuritySettings(enabledCheck, providerSelect, apiKeyEntry)
	}

	enabledCheck.OnChanged = func(_ bool) {
		c.saveSecuritySettings(enabledCheck, providerSelect, apiKeyEntry)
	}

	apiKeyEntry.OnChanged = func(_ string) {
		c.saveSecuritySettings(enabledCheck, providerSelect, apiKeyEntry)
	}

	testButton := widget.NewButton(i18n.T("config.security_test"), func() {
		testStatus.SetText(i18n.T("config.security_testing"))

		go func() {
			provider := c.getSelectedProvider(providerSelect)
			key := apiKeyEntry.Text

			checker, err := safety.NewChecker(provider, key)
			if err != nil {
				fyne.Do(func() {
					testStatus.SetText("✗ " + err.Error())
				})
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), securityTestTimeout)
			defer cancel()

			err = checker.TestCredentials(ctx)

			fyne.Do(func() {
				if err != nil {
					testStatus.SetText("✗ " + err.Error())
				} else {
					testStatus.SetText(i18n.T("config.security_test_success"))
				}
			})
		}()
	})

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel(i18n.T("config.security_provider_label")), providerSelect,
		widget.NewLabel(i18n.T("config.security_api_key_label")), apiKeyEntry,
		widget.NewLabel(""), providerLink,
		widget.NewLabel(""), container.NewHBox(testButton, testStatus),
	)

	return container.NewVBox(
		enabledCheck,
		form,
	)
}

func (c *Configurator) saveSecuritySettings(
	enabled *widget.Check, provider *widget.Select, apiKey *widget.Entry,
) {
	settings := c.settingsService.GetSettings()
	settings.Security.Enabled = enabled.Checked
	settings.Security.Provider = c.getSelectedProvider(provider)
	settings.Security.APIKey = apiKey.Text

	if err := c.settingsService.WriteSettings(settings); err != nil {
		c.logger.Error("Failed to save security settings", "error", err)
	}
}

func (c *Configurator) getSelectedProvider(sel *widget.Select) string {
	if sel.SelectedIndex() == 1 {
		return linkquisition.SecurityProviderVirusTotal
	}

	return linkquisition.SecurityProviderGoogleSafeBrowsing
}

func parseURL(rawURL string) *url.URL {
	u, _ := url.Parse(rawURL)
	return u
}
