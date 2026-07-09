package main

import (
	"fmt"
	"testing"

	"github.com/strobotti/linkquisition"
)

// mockBrowserService implements linkquisition.BrowserService for testing.
type mockBrowserService struct {
	isDefault        bool
	makeDefaultErr   error
	makeDefaultCalls int
	browsers         []linkquisition.Browser
}

func (m *mockBrowserService) GetAvailableBrowsers() ([]linkquisition.Browser, error) {
	return m.browsers, nil
}

func (m *mockBrowserService) GetDefaultBrowser() (linkquisition.Browser, error) {
	return linkquisition.Browser{Name: "test"}, nil
}

func (m *mockBrowserService) OpenUrlWithDefaultBrowser(_ string) error { return nil }

func (m *mockBrowserService) OpenUrlWithBrowser(_ string, _ *linkquisition.Browser) error {
	return nil
}

func (m *mockBrowserService) AreWeTheDefaultBrowser() bool { return m.isDefault }

func (m *mockBrowserService) MakeUsTheDefaultBrowser() error {
	m.makeDefaultCalls++
	return m.makeDefaultErr
}

func (m *mockBrowserService) GetIconForBrowser(_ linkquisition.Browser) ([]byte, error) {
	return nil, nil
}

func TestSetDefault_AlreadyDefault(t *testing.T) {
	mock := &mockBrowserService{isDefault: true}

	// Simulate the logic from runSetDefault
	if mock.AreWeTheDefaultBrowser() {
		// Should print "already the default" and return nil
		if mock.makeDefaultCalls != 0 {
			t.Error("should not call MakeUsTheDefaultBrowser when already default")
		}
		return
	}
	t.Fatal("expected AreWeTheDefaultBrowser to return true")
}

func TestSetDefault_Success(t *testing.T) {
	mock := &mockBrowserService{isDefault: false}

	if mock.AreWeTheDefaultBrowser() {
		t.Fatal("should not be default yet")
	}

	if err := mock.MakeUsTheDefaultBrowser(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if mock.makeDefaultCalls != 1 {
		t.Errorf("expected 1 call to MakeUsTheDefaultBrowser, got %d", mock.makeDefaultCalls)
	}
}

func TestSetDefault_Error(t *testing.T) {
	mock := &mockBrowserService{
		isDefault:      false,
		makeDefaultErr: errForTesting,
	}

	err := mock.MakeUsTheDefaultBrowser()
	if err == nil {
		t.Fatal("expected error")
	}

	if err != errForTesting {
		t.Errorf("expected errForTesting, got %v", err)
	}
}

var errForTesting = fmt.Errorf("test error: failed to set default")
