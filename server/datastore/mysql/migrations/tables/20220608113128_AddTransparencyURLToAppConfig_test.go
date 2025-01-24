package tables

import (
	"encoding/json"
	"testing"

	"github.com/it-laborato/MDM_Lab/server/mdmlab"
	"github.com/stretchr/testify/require"
)

func TestUp_20220608113128(t *testing.T) {
	db := applyUpToPrev(t)

	var prevRaw []byte
	var prevConfig mdmlab.AppConfig
	err := db.Get(&prevRaw, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)

	err = json.Unmarshal(prevRaw, &prevConfig)
	require.NoError(t, err)
	require.Empty(t, prevConfig.MDMlabDesktop.TransparencyURL)

	applyNext(t, db)

	var newRaw []byte
	var newConfig mdmlab.AppConfig
	err = db.Get(&newRaw, `SELECT json_value FROM app_config_json`)
	require.NoError(t, err)

	err = json.Unmarshal(newRaw, &newConfig)
	require.NoError(t, err)
	require.Equal(t, "", newConfig.MDMlabDesktop.TransparencyURL)
}
