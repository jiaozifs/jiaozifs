package hash

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHashJSON(t *testing.T) {
	type A struct {
		H Hash
	}

	t.Run("success", func(t *testing.T) {
		data, err := json.Marshal(A{H: Hash("aaaa")})
		require.NoError(t, err)
		require.Equal(t, "{\"H\":\"61616161\"}", string(data))

		a := A{}
		err = json.Unmarshal(data, &a)
		require.NoError(t, err)
		require.Equal(t, "aaaa", string(a.H))
	})
	t.Run("null", func(t *testing.T) {
		data, err := json.Marshal(A{})
		require.NoError(t, err)
		require.Equal(t, "{\"H\":\"\"}", string(data))

		a := A{}
		err = json.Unmarshal(data, &a)
		require.NoError(t, err)
		require.Equal(t, "", string(a.H))
	})
}

func TestHexArrayOfHashes(t *testing.T) {
	hashes := []Hash{Hash("aaaaaa"), Hash("bbbbbb"), Hash("cccccc"), Hash("dddddd")}
	hexArray := HexArrayOfHashes(hashes...)
	require.Equal(t, "616161616161", hexArray[0])
	require.Equal(t, "636363636363", hexArray[2])

	hashes2, err := HashesOfHexArray(hexArray...)
	require.NoError(t, err)
	require.Equal(t, hashes, hashes2)
}

func TestHashFromHex(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		hash, err := FromHex("")
		require.NoError(t, err)
		require.Equal(t, Empty, hash)
	})

	t.Run("data", func(t *testing.T) {
		hash, err := FromHex("616161616161")
		require.NoError(t, err)
		require.Equal(t, Hash("aaaaaa"), hash)
	})
}
