package main

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/strobotti/linkquisition/internal/i18n"
	"github.com/strobotti/linkquisition/internal/ui"
	"github.com/strobotti/linkquisition/internal/updater"
	"github.com/strobotti/linkquisition/resources"
)

func (c *Configurator) getAboutTab(w fyne.Window) fyne.CanvasObject {
	const githubURL = "https://github.com/Strobotti/linkquisition"

	openURL := func() {
		if err := c.openExternalURL(githubURL); err != nil {
			c.logger.Error("Error opening URL", "error", err)
		}
	}

	icon := widget.NewButtonWithIcon("", resources.LinkquisitionIcon, openURL)
	title := widget.NewLabel(fmt.Sprintf("Linkquisition %s", version))
	title.TextStyle = fyne.TextStyle{Bold: true}

	description := widget.NewLabel(i18n.T("about.description"))
	description.Wrapping = fyne.TextWrapWord

	githubLink := ui.NewLinkWithCopy("github.com/Strobotti/linkquisition", githubURL, w, c.urlOpener())
	details := container.NewVBox(
		container.NewHBox(widget.NewLabel(i18n.T("about.author_label")), widget.NewLabel("Juha Jantunen")),
		container.NewHBox(widget.NewLabel(i18n.T("about.license_label")), widget.NewLabel("MIT")),
		container.NewHBox(widget.NewLabel(i18n.T("about.github_label")), githubLink),
	)

	updateSection := c.buildUpdateCheckSection(w)
	bugReportSection := c.buildBugReportSection(w)
	topContent := container.NewVBox(
		container.NewHBox(icon, title),
		widget.NewSeparator(),
		description,
		widget.NewSeparator(),
		details,
		widget.NewSeparator(),
		updateSection,
	)

	return container.NewBorder(topContent, bugReportSection, nil, nil)
}

func (c *Configurator) buildUpdateCheckSection(w fyne.Window) fyne.CanvasObject {
	statusLabel := widget.NewLabel("")
	releaseLink := container.NewHBox()
	releaseLink.Hide()

	checkButton := widget.NewButton(i18n.T("about.check_updates"), nil)
	checkButton.OnTapped = func() {
		checkButton.Disable()
		checkButton.SetText(i18n.T("about.checking_updates"))
		statusLabel.SetText("")
		releaseLink.Hide()

		go func() {
			result, err := updater.Check(context.Background(), version)
			fyne.Do(func() {
				checkButton.Enable()
				checkButton.SetText(i18n.T("about.check_updates"))

				if err != nil {
					c.logger.Error("Update check failed", "error", err)
					statusLabel.SetText(i18n.T("about.update_error"))
					return
				}

				if result.IsNewer {
					statusLabel.SetText(i18n.T("about.update_available", map[string]interface{}{
						"Version": result.LatestVersion,
					}))

					releaseLink.RemoveAll()
					releaseLink.Add(ui.NewLinkWithCopy(i18n.T("about.view_release"), result.ReleaseURL, w, c.urlOpener()))
					releaseLink.Show()
				} else {
					statusLabel.SetText(i18n.T("about.up_to_date"))
				}
			})
		}()
	}

	return container.NewHBox(checkButton, statusLabel, releaseLink)
}

func (c *Configurator) buildBugReportSection(w fyne.Window) fyne.CanvasObject {
	const issuesURL = "https://github.com/Strobotti/linkquisition/issues"

	titleLabel := widget.NewLabel(i18n.T("about.bug_report_title"))
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}

	description := widget.NewLabel(i18n.T("about.bug_report_description"))
	description.Wrapping = fyne.TextWrapWord

	openLogButton := widget.NewButtonWithIcon(
		i18n.T("about.bug_report_open_log"),
		theme.FileIcon(),
		func() {
			logPath := c.settingsService.GetLogFilePath()
			c.logger.Debug("Opening log file in editor", "path", logPath)
			if err := openFileInEditor(logPath); err != nil {
				c.logger.Error("Failed to open log file", "path", logPath, "error", err)
			}
		},
	)

	issuesLink := ui.NewLinkWithCopy(i18n.T("about.bug_report_issues_link"), issuesURL, w, c.urlOpener())
	return container.NewVBox(
		titleLabel,
		description,
		container.NewHBox(openLogButton, issuesLink),
	)
}
