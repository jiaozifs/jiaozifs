package versionmgr

import (
	"context"
	"errors"
	"testing"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

func TestWorkTree(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	objRepo := models.NewFileTree(db, repoID)

	workTree, err := NewWorkTree(ctx, objRepo, EmptyDirEntry)
	require.NoError(t, err)

	blob := &models.Blob{}
	require.NoError(t, gofakeit.Struct(blob))
	blob.Type = models.BlobObject
	blob.RepositoryID = repoID

	err = workTree.AddLeaf(ctx, "a/b/c.txt", blob)
	require.NoError(t, err)

	//add again expect get an error
	err = workTree.AddLeaf(ctx, "a/b/c.txt", blob)
	require.True(t, errors.Is(err, ErrEntryExit))

	//update path
	blob = &models.Blob{}
	require.NoError(t, gofakeit.Struct(blob))
	blob.Type = models.BlobObject
	blob.RepositoryID = repoID
	err = workTree.ReplaceLeaf(ctx, "a/b/c.txt", blob)
	require.NoError(t, err)

	{
		//find blob
		findBlob, name, err := workTree.FindBlob(ctx, "a/b/c.txt")
		require.NoError(t, err)
		require.Equal(t, "c.txt", name)
		require.Equal(t, blob.Hash.Hex(), findBlob.Hash.Hex())
	}
	{
		//add another branch
		err = workTree.AddLeaf(ctx, "a/b/d.txt", blob)
		require.NoError(t, err)

	}

	{
		//check fs structure
		rootDir, err := objRepo.TreeNode(ctx, workTree.Root().Hash())
		require.NoError(t, err)
		require.Len(t, rootDir.SubObjects, 1)
		require.Equal(t, "a", rootDir.SubObjects[0].Name)

		aDir, err := objRepo.TreeNode(ctx, rootDir.SubObjects[0].Hash)
		require.NoError(t, err)
		require.Len(t, aDir.SubObjects, 1)
		require.Equal(t, "b", aDir.SubObjects[0].Name)

		bDir, err := objRepo.TreeNode(ctx, aDir.SubObjects[0].Hash)
		require.NoError(t, err)
		require.Len(t, bDir.SubObjects, 2)
		require.Equal(t, "c.txt", bDir.SubObjects[0].Name)
		require.Equal(t, "d.txt", bDir.SubObjects[1].Name)
	}

	{
		//check ls
		subObjects, err := workTree.Ls(ctx, "a")
		require.NoError(t, err)
		require.Len(t, subObjects, 1)
		require.Equal(t, "b", subObjects[0].Name)

		subObjects, err = workTree.Ls(ctx, "a/b")
		require.NoError(t, err)
		require.Len(t, subObjects, 2)
		require.Equal(t, "c.txt", subObjects[0].Name)
		require.Equal(t, "d.txt", subObjects[1].Name)
	}

	err = workTree.RemoveEntry(ctx, "a/b/c.txt")
	require.NoError(t, err)

	err = workTree.RemoveEntry(ctx, "a/b/d.txt")
	require.NoError(t, err)
	require.Equal(t, "", hash.Hash(workTree.Root().Hash()).Hex())
}

func TestRemoveEntry(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	objRepo := models.NewFileTree(db, repoID)

	workTree, err := NewWorkTree(ctx, objRepo, EmptyDirEntry)
	require.NoError(t, err)

	blob := &models.Blob{}
	require.NoError(t, gofakeit.Struct(blob))
	blob.Type = models.BlobObject
	blob.RepositoryID = repoID
	err = workTree.AddLeaf(ctx, "a/b/c.txt", blob)
	require.NoError(t, err)

	//update path
	blob = &models.Blob{}
	require.NoError(t, gofakeit.Struct(blob))
	blob.Type = models.BlobObject
	blob.RepositoryID = repoID
	//add another branch
	err = workTree.AddLeaf(ctx, "a/b/d.txt", blob)
	require.NoError(t, err)

	err = workTree.RemoveEntry(ctx, "a/b")
	require.NoError(t, err)
	require.Equal(t, "", hash.Hash(workTree.Root().Hash()).Hex())
	entries, err := workTree.Ls(ctx, "")
	require.NoError(t, err)
	require.Len(t, entries, 0)

	entries, err = workTree.Ls(ctx, "/")
	require.NoError(t, err)
	require.Len(t, entries, 0)
}

func TestCleanPath(t *testing.T) {
	require.Equal(t, "", CleanPath(""))
	require.Equal(t, "", CleanPath("/"))

	require.Equal(t, "a/b/c", CleanPath("a/b/c"))
	require.Equal(t, "a/b/c", CleanPath("/a/b/c/"))
	require.Equal(t, "a/b/c", CleanPath("\\a\\b\\c\\"))
}
