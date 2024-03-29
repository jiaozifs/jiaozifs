package local_test

import (
	"path"
	"regexp"
	"testing"

	"github.com/GitDataAI/jiaozifs/block"
	"github.com/GitDataAI/jiaozifs/block/blocktest"
	"github.com/GitDataAI/jiaozifs/block/local"
	"github.com/stretchr/testify/require"
)

const testStorageNamespace = "local://test"

func TestLocalAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := path.Join(tmpDir, "jiaozfs")
	externalPath := block.BlockstoreTypeLocal + "://" + path.Join(tmpDir, "jiaozfs", "external")
	adapter, err := local.NewAdapter(localPath, local.WithRemoveEmptyDir(false))
	if err != nil {
		t.Fatal("Failed to create new adapter", err)
	}
	blocktest.AdapterTest(t, adapter, testStorageNamespace, externalPath)
}

func TestAdapterNamespace(t *testing.T) {
	tmpDir := t.TempDir()
	localPath := path.Join(tmpDir, "jiaozfs")
	adapter, err := local.NewAdapter(localPath, local.WithRemoveEmptyDir(false))
	require.NoError(t, err, "create new adapter")
	expr, err := regexp.Compile(adapter.GetStorageNamespaceInfo().ValidityRegex)
	require.NoError(t, err)

	tests := []struct {
		Name      string
		Namespace string
		Success   bool
	}{
		{
			Name:      "valid_path",
			Namespace: "local://test/path/to/repo1",
			Success:   true,
		},
		{
			Name:      "invalid_path",
			Namespace: "~/test/path/to/repo1",
			Success:   false,
		},
		{
			Name:      "s3",
			Namespace: "s3://test/adls/core/windows/net",
			Success:   false,
		},
		{
			Name:      "invalid_string",
			Namespace: "this is a bad string",
			Success:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			require.Equal(t, tt.Success, expr.MatchString(tt.Namespace))
		})
	}
}
