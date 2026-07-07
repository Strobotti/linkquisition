package linkquisition

type Browser struct {
	Name    string
	Command string
}

type BrowserService interface {
	// GetAvailableBrowsers returns a list of available browsers in the system
	GetAvailableBrowsers() ([]Browser, error)

	// GetDefaultBrowser returns the default browser in the system
	GetDefaultBrowser() (Browser, error)

	// OpenUrlWithDefaultBrowser launches the given url with the default system browser
	OpenUrlWithDefaultBrowser(url string) error

	// OpenUrlWithBrowser launches the given url with the given browser
	OpenUrlWithBrowser(url string, browser *Browser) error

	// AreWeTheDefaultBrowser returns true if Linkquisition is the default browser
	AreWeTheDefaultBrowser() bool

	// MakeUsTheDefaultBrowser sets Linkquisition as the default browser
	MakeUsTheDefaultBrowser() error

	// GetIconForBrowser returns the icon for the given browser
	GetIconForBrowser(browser Browser) ([]byte, error)
}
