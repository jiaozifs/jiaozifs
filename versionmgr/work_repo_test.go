package versionmgr

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/block/mem"
	"github.com/jiaozifs/jiaozifs/config"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/models/filemode"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeWriteBlob(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	adapter := mem.New(ctx)
	repo := models.NewRepo(db)

	userModel := &models.User{}
	require.NoError(t, gofakeit.Struct(userModel))
	userModel, err := repo.UserRepo().Insert(ctx, userModel)
	require.NoError(t, err)

	repoModel := &models.Repository{}
	require.NoError(t, gofakeit.Struct(repoModel))
	repoModel.CreatorID = userModel.ID
	repoModel.StorageNamespace = utils.String("mem://data")
	repoModel, err = repo.RepositoryRepo().Insert(ctx, repoModel)
	require.NoError(t, err)

	workRepo := NewWorkRepositoryFromAdapter(ctx, userModel, repoModel, repo, adapter)

	binary := []byte("Build simple, secure, scalable systems with Go")
	bLen := int64(len(binary))
	r := bytes.NewReader(binary)
	blob, err := workRepo.WriteBlob(ctx, r, bLen, models.DefaultLeafProperty())
	require.NoError(t, err)
	assert.Equal(t, bLen, blob.Size)
	assert.Equal(t, "99b91d4c517d0cded9506be9298b8d02", blob.Hash.Hex())
	assert.Equal(t, "f3b39786b86a96372589aa1166966643", blob.CheckSum.Hex())

	reader, err := workRepo.ReadBlob(ctx, blob, nil)
	require.NoError(t, err)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.Equal(t, binary, content)
}

func TestNewWorkRepositoryFromConfig(t *testing.T) {
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)

	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepo(db)
	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	t.Run("use public", func(t *testing.T) {
		pubCfg := &config.BlockStoreConfig{
			Type: "local",
			Local: (*struct {
				Path                    string   `mapstructure:"path" json:"path"`
				ImportEnabled           bool     `mapstructure:"import_enabled" json:"import_enabled"`
				ImportHidden            bool     `mapstructure:"import_hidden" json:"import_hidden"`
				AllowedExternalPrefixes []string `mapstructure:"allowed_external_prefixes" json:"allowed_external_prefixes"`
			})(&struct {
				Path                    string
				ImportEnabled           bool
				ImportHidden            bool
				AllowedExternalPrefixes []string
			}{Path: path.Join(tmpDir, "d1"), ImportEnabled: false, ImportHidden: false, AllowedExternalPrefixes: nil}),
		}

		project, err := repo.RepositoryRepo().Insert(ctx, &models.Repository{
			Name:             "testproject",
			Description:      utils.String("test"),
			UsePublicStorage: true,
			HEAD:             "main",
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			CreatorID:        user.ID,
			StorageNamespace: utils.String("mem://data"),
		})
		require.NoError(t, err)
		newRepo, err := NewWorkRepositoryFromConfig(ctx, user, project, repo, pubCfg)
		require.NoError(t, err)
		require.Equal(t, "local", newRepo.adapter.BlockstoreType())
	})

	t.Run("use private", func(t *testing.T) {
		pubCfg := &config.BlockStoreConfig{
			Type: "local",
			Local: (*struct {
				Path                    string   `mapstructure:"path" json:"path"`
				ImportEnabled           bool     `mapstructure:"import_enabled" json:"import_enabled"`
				ImportHidden            bool     `mapstructure:"import_hidden" json:"import_hidden"`
				AllowedExternalPrefixes []string `mapstructure:"allowed_external_prefixes" json:"allowed_external_prefixes"`
			})(&struct {
				Path                    string
				ImportEnabled           bool
				ImportHidden            bool
				AllowedExternalPrefixes []string
			}{Path: path.Join(tmpDir, "d1"), ImportEnabled: false, ImportHidden: false, AllowedExternalPrefixes: nil}),
		}
		storageCfg := fmt.Sprintf(`{"Type":"local","Local":{"Path":"%s"}}`, path.Join(tmpDir, "d2"))
		project, err := repo.RepositoryRepo().Insert(ctx, &models.Repository{
			Name:                 "testproject2",
			Description:          utils.String("test"),
			UsePublicStorage:     false,
			HEAD:                 "main",
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			CreatorID:            user.ID,
			StorageAdapterParams: &storageCfg,
		})
		require.NoError(t, err)
		newRepo, err := NewWorkRepositoryFromConfig(ctx, user, project, repo, pubCfg)
		require.NoError(t, err)
		require.Equal(t, "local", newRepo.adapter.BlockstoreType())
	})

	t.Run("false storage format", func(t *testing.T) {
		pubCfg := &config.BlockStoreConfig{
			Type: "local",
			Local: (*struct {
				Path                    string   `mapstructure:"path" json:"path"`
				ImportEnabled           bool     `mapstructure:"import_enabled" json:"import_enabled"`
				ImportHidden            bool     `mapstructure:"import_hidden" json:"import_hidden"`
				AllowedExternalPrefixes []string `mapstructure:"allowed_external_prefixes" json:"allowed_external_prefixes"`
			})(&struct {
				Path                    string
				ImportEnabled           bool
				ImportHidden            bool
				AllowedExternalPrefixes []string
			}{Path: path.Join(tmpDir, "d1"), ImportEnabled: false, ImportHidden: false, AllowedExternalPrefixes: nil}),
		}
		storageCfg := fmt.Sprintf(`{"Type":"local",Local":{"Path":"%s"}}`, path.Join(tmpDir, "d2"))
		project, err := repo.RepositoryRepo().Insert(ctx, &models.Repository{
			Name:                 "testproject3",
			Description:          utils.String("test"),
			UsePublicStorage:     false,
			HEAD:                 "main",
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			CreatorID:            user.ID,
			StorageAdapterParams: &storageCfg,
		})
		require.NoError(t, err)
		_, err = NewWorkRepositoryFromConfig(ctx, user, project, repo, pubCfg)
		require.Error(t, err)
	})
}

func TestWorkRepositoryRootTree(t *testing.T) {
	ctx := context.Background()

	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepo(db)
	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	project, err := makeRepository(ctx, repo, user, "testproject")
	require.NoError(t, err)

	workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, mem.New(ctx))
	_, err = workRepo.RootTree(ctx)
	require.NoError(t, err)

	testData := `
1|a.txt	|h1
1|b/c.txt	|h2
`

	commit, err := addChangesToWip(ctx, workRepo, "main", "init commit", testData)
	require.NoError(t, err)
	require.Error(t, workRepo.CheckOut(ctx, WorkRepoState("mock"), "main"))

	workTree, err := workRepo.RootTree(ctx)
	require.NoError(t, err)
	require.Equal(t, commit.TreeHash.Hex(), hex.EncodeToString(workTree.Root().Hash()))

	workRepo.Reset()
	workTree, err = workRepo.RootTree(ctx)
	require.NoError(t, err)
	require.Equal(t, commit.TreeHash.Hex(), hex.EncodeToString(workTree.Root().Hash()))
	require.Equal(t, "main", workRepo.CurBranch().Name)
}

func TestWorkRepositoryRevert(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepo(db)
	adapter := mem.New(ctx)

	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	var checkFn = func(workRepo *WorkRepository, compareTree hash.Hash, path string, num int) {
		beforeTree, err := workRepo.RootTree(ctx)
		require.NoError(t, err)
		beforeChanges, err := beforeTree.Diff(ctx, compareTree, path)
		require.NoError(t, err)
		require.Equal(t, num, beforeChanges.Num())

		//revert
		err = workRepo.Revert(ctx, path)
		require.NoError(t, err)

		//after tree
		afterTree, err := workRepo.RootTree(ctx)
		require.NoError(t, err)
		afterChanges, err := afterTree.Diff(ctx, compareTree, path)
		require.NoError(t, err)
		require.Equal(t, 0, afterChanges.Num())
	}

	t.Run("revert", func(t *testing.T) {
		project, err := makeRepository(ctx, repo, user, "testRevert")
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
1|b/c.txt	|c
1|b/e.txt |e1
`

		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		initCommit, err := addChangesToWip(ctx, workRepo, "main", "base commit", testData1)
		require.NoError(t, err)
		testData2 := `
3|a.txt	|a1
2|b/c.txt	|d
3|b/e.txt |e2
1|b/g.txt |g1
`
		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.Error(t, err) //state not correct

		err = workRepo.CheckOut(ctx, InWip, "main")
		require.NoError(t, err)

		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.NoError(t, err)

		checkFn(workRepo, initCommit.TreeHash, "a.txt", 1)
		checkFn(workRepo, initCommit.TreeHash, "b/c.txt", 1)
		checkFn(workRepo, initCommit.TreeHash, "b/e.txt", 1)
		checkFn(workRepo, initCommit.TreeHash, "b/g.txt", 1)
	})

	t.Run("revert dir", func(t *testing.T) {
		project, err := makeRepository(ctx, repo, user, "testRevertDir")
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
1|b/c.txt	|c
1|b/e.txt |e1
`

		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		initCommit, err := addChangesToWip(ctx, workRepo, "main", "base commit", testData1)
		require.NoError(t, err)
		testData2 := `
3|a.txt	|a1
2|b/c.txt	|d
3|b/e.txt |e2
1|b/g.txt |g1
`
		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.Error(t, err) //state not correct

		err = workRepo.CheckOut(ctx, InWip, "main")
		require.NoError(t, err)

		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.NoError(t, err)

		checkFn(workRepo, initCommit.TreeHash, "b", 3)
	})

	t.Run("revert all", func(t *testing.T) {
		project, err := makeRepository(ctx, repo, user, "testRevertAll")
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
1|b/c.txt	|c
1|b/e.txt |e1
`

		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		initCommit, err := addChangesToWip(ctx, workRepo, "main", "base commit", testData1)
		require.NoError(t, err)
		testData2 := `
3|a.txt	|a1
2|b/c.txt	|d
3|b/e.txt |e2
1|b/g.txt |g1
`
		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.Error(t, err) //state not correct

		err = workRepo.CheckOut(ctx, InWip, "main")
		require.NoError(t, err)

		err = workRepo.ChangeInWip(ctx, func(workTree *WorkTree) error {
			return appendChangeToWorkTree(ctx, workTree, testData2)
		})
		require.NoError(t, err)

		checkFn(workRepo, initCommit.TreeHash, "", 4)
	})
}

func TestWorkRepositoryMergeState(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()
	repo := models.NewRepo(db)

	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	t.Run("not on branch", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
`
		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		//base branch
		err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/base")
		require.NoError(t, err)
		baseCommit, err := addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData1)
		require.NoError(t, err)

		_, err = workRepo.GetMergeState(ctx, baseCommit.Hash)
		require.Error(t, err)
	})

	t.Run("base is nil", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
`
		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		//base branch
		err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/base")
		require.NoError(t, err)
		baseCommit, err := addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData1)
		require.NoError(t, err)

		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		changes, err := workRepo.GetMergeState(ctx, baseCommit.Hash)
		require.NoError(t, err)
		require.Len(t, changes, 1)
	})
	t.Run("toMerge is nil", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)
		testData1 := `
1|a.txt	|a
`
		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		//base branch
		err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/base")
		require.NoError(t, err)
		_, err = addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData1)
		require.NoError(t, err)

		err = workRepo.CheckOut(ctx, InBranch, "feat/base")
		require.NoError(t, err)
		changes, err := workRepo.GetMergeState(ctx, hash.Empty)
		require.NoError(t, err)
		require.Len(t, changes, 1)
	})

	t.Run("both is nil", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)

		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		_, err = workRepo.GetMergeState(ctx, hash.Empty)
		require.Error(t, err)
	})

	t.Run("no common root", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)
		//commit1  a.txt b/c.txt  b/e.txt
		//commit2  a.txt b/d.txt  b/e.txt
		testData1 := `
1|a.txt	|a
`
		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

		//base branch
		err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/base")
		require.NoError(t, err)
		_, err = addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData1)
		require.NoError(t, err)

		testData2 := `
1|b/g.txt |g1
`

		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/diff")
		require.NoError(t, err)

		secondCommit, err := addChangesToWip(ctx, workRepo, "feat/diff", "merge commit", testData2)
		require.NoError(t, err)

		err = workRepo.CheckOut(ctx, InBranch, "feat/base")
		require.NoError(t, err)
		_, err = workRepo.GetMergeState(ctx, secondCommit.Hash)
		require.Error(t, err)
	})

	t.Run("get merge state", func(t *testing.T) {
		adapter := mem.New(ctx)

		project, err := makeRepository(ctx, repo, user, t.Name())
		require.NoError(t, err)
		//commit1  a.txt b/c.txt  b/e.txt
		//commit2  a.txt b/d.txt  b/e.txt

		workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)
		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/init")
		require.NoError(t, err)
		testData := `
1|readme.md	|a
`
		_, err = addChangesToWip(ctx, workRepo, "main", "init commit", testData)
		require.NoError(t, err)

		testData1 := `
1|a.txt	|a
`
		//base branch
		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/base")
		require.NoError(t, err)
		_, err = addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData1)
		require.NoError(t, err)

		testData2 := `
1|b/g.txt |g1
`

		err = workRepo.CheckOut(ctx, InBranch, "main")
		require.NoError(t, err)
		_, err = workRepo.CreateBranch(ctx, "feat/diff")
		require.NoError(t, err)
		secondCommit, err := addChangesToWip(ctx, workRepo, "feat/diff", "merge commit", testData2)
		require.NoError(t, err)

		err = workRepo.CheckOut(ctx, InBranch, "feat/base")
		require.NoError(t, err)
		changes, err := workRepo.GetMergeState(ctx, secondCommit.Hash)
		require.NoError(t, err)
		require.Len(t, changes, 2)
	})
}

func makeUser(ctx context.Context, userRepo models.IUserRepo, name string) (*models.User, error) {
	user := &models.User{
		Name:              name,
		Email:             "xxx@gg.com",
		EncryptedPassword: "123",
		CurrentSignInAt:   time.Time{},
		LastSignInAt:      time.Time{},
		CurrentSignInIP:   "",
		LastSignInIP:      "",
		CreatedAt:         time.Time{},
		UpdatedAt:         time.Time{},
	}
	return userRepo.Insert(ctx, user)
}

func makeRepository(ctx context.Context, repo models.IRepo, user *models.User, name string) (*models.Repository, error) {
	repoModel, err := repo.RepositoryRepo().Insert(ctx, &models.Repository{
		Name:             name,
		Description:      utils.String("test"),
		HEAD:             "main",
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
		CreatorID:        user.ID,
		StorageNamespace: utils.String("mem://data"),
	})
	if err != nil {
		return nil, err
	}
	_, err = repo.BranchRepo().Insert(ctx, &models.Branch{
		RepositoryID: repoModel.ID,
		CommitHash:   hash.Empty,
		Name:         "main",
		Description:  nil,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		CreatorID:    user.ID,
	})
	if err != nil {
		return nil, err
	}
	return repoModel, nil
}

// nolint
func makeCommit(ctx context.Context, commitRepo models.ICommitRepo, treeHash hash.Hash, msg string, parentsHash ...hash.Hash) (*models.Commit, error) {
	commit := &models.Commit{
		RepositoryID: commitRepo.RepositoryID(),
		Author: models.Signature{
			Name:  "admin",
			Email: "xxx@gg.com",
			When:  time.Now(),
		},
		Committer: models.Signature{
			Name:  "admin",
			Email: "xxx@gg.com",
			When:  time.Now(),
		},
		TreeHash:     treeHash,
		ParentHashes: parentsHash,
		Message:      msg,
	}
	hash, err := commit.GetHash()
	if err != nil {
		return nil, err
	}
	commit.Hash = hash
	obj, err := commitRepo.Insert(ctx, commit)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func makeBranch(ctx context.Context, branchRepo models.IBranchRepo, user *models.User, name string, repoID uuid.UUID, commitHash hash.Hash) (*models.Branch, error) {
	branch := &models.Branch{
		RepositoryID: repoID,
		CommitHash:   commitHash,
		Name:         name,
		Description:  utils.String("test"),
		CreatorID:    user.ID,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	return branchRepo.Insert(ctx, branch)
}

func addChangesToWip(ctx context.Context, workRepo *WorkRepository, branchName string, msg string, testData string) (*models.Commit, error) {
	err := workRepo.CheckOut(ctx, InBranch, branchName)
	if err != nil {
		return nil, err
	}
	_, _, err = workRepo.GetOrCreateWip(ctx)
	if err != nil {
		return nil, err
	}

	err = workRepo.CheckOut(ctx, InWip, branchName)
	if err != nil {
		return nil, err
	}

	return workRepo.ChangeAndCommit(ctx, msg, func(workTree *WorkTree) error {
		return appendChangeToWorkTree(ctx, workTree, testData)
	})
}

func appendChangeToWorkTree(ctx context.Context, workTree *WorkTree, testData string) error {
	lines := strings.Split(testData, "\n")
	var err error
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		commitData := strings.Split(strings.TrimSpace(line), "|")
		fullPath := strings.TrimSpace(commitData[1])
		fileHash := strings.TrimSpace(commitData[2])
		blob := &models.Blob{
			Hash:         hash.Hash(fileHash),
			RepositoryID: workTree.RepositoryID(),
			Type:         models.BlobObject,
			Size:         10,
			Properties:   models.Property{Mode: filemode.Regular},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		if commitData[0] == "1" {
			err = workTree.AddLeaf(ctx, fullPath, blob)
			if err != nil {
				return err
			}
		} else if commitData[0] == "3" {
			err = workTree.ReplaceLeaf(ctx, fullPath, blob)
			if err != nil {
				return err
			}
		} else {
			//2
			err = workTree.RemoveEntry(ctx, fullPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
