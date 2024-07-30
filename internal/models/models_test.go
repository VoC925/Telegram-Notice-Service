package models

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathToResponseFile = "response.txt"
)

func TestUnmarshalJSON(t *testing.T) {
	file, err := os.OpenFile(pathToResponseFile, os.O_RDONLY, 0644)
	require.NoError(t, err)
	defer file.Close()
	sliceData := UpdateInfoSlice{}
	err = json.NewDecoder(file).Decode(&sliceData)
	require.NoError(t, err)
	assert.NotEqual(t, len(sliceData), 0)
}
