package main

import "time"

const (
	pluginShutdownTimeout = 10 * time.Second
	pluginProcessTimeout  = 30 * time.Second
	testURLTimeout        = 30 * time.Second
	faviconFetchTimeout   = 5 * time.Second
	safetyCheckTimeout    = 10 * time.Second
	whoisTimeout          = 10 * time.Second
	securityTestTimeout   = 10 * time.Second
)
