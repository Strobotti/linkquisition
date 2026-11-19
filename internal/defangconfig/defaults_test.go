package defangconfig

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaultSourcesUseNetAPIAndStevenBlack(t *testing.T) {
	sources := DefaultSources()
	require.Equal(t, []string{
		"https://netapi.com/api2/?method=compromised&dataset_type=url&format=hosts",
		"https://raw.githubusercontent.com/StevenBlack/hosts/master/hosts",
	}, sources)
	sources[0] = "changed"
	require.NotEqual(t, "changed", DefaultSources()[0])
}
