package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadfromFile(t *testing.T) {
	_, err := Load("./config.yml")
	require.NoError(t, err, "error must be nil.")
}
