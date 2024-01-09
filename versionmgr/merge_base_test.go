package versionmgr

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

func TestCommitNodeMergeBase(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	commitRepo := models.NewCommitRepo(db, repoID)
	//mock data
	//     | -> c -------
	//     |             |
	//a ------> b ------d--f1-f2---f--merge?
	//          |                       |
	//          | ----------------->e----

	testData := `
a|
b|a
c|a
d|b,c
f1|d
f2|f1
f|f2
e|b
`
	commitMap, err := loadCommitTestData(ctx, commitRepo, testData)
	require.NoError(t, err)

	t.Run("simple", func(t *testing.T) {
		//simple
		baseCommit := commitMap["b"]
		mergeCommit := commitMap["c"]
		ancestorNode, err := baseCommit.MergeBase(ctx, mergeCommit)
		require.NoError(t, err)
		require.Len(t, ancestorNode, 1)
		require.Equal(t, "a", string(ancestorNode[0].Commit().Hash))
	})

	t.Run("fast forward", func(t *testing.T) {
		//simple
		baseCommit := commitMap["f"]
		mergeCommit := commitMap["f1"]
		ancestorNode, err := baseCommit.MergeBase(ctx, mergeCommit)
		require.NoError(t, err)
		require.Len(t, ancestorNode, 1)
		require.Equal(t, "f1", string(ancestorNode[0].Commit().Hash))
	})

	t.Run("multiple merge", func(t *testing.T) {
		baseCommit := commitMap["f"]
		mergeCommit := commitMap["e"]
		ancestorNode, err := baseCommit.MergeBase(ctx, mergeCommit)
		require.NoError(t, err)
		require.Len(t, ancestorNode, 1)
		require.Equal(t, "b", string(ancestorNode[0].Commit().Hash))
	})
}

func loadCommitTestData(ctx context.Context, commitRepo models.ICommitRepo, testData string) (map[string]*WrapCommitNode, error) {
	lines := strings.Split(testData, "\n")
	commitMap := make(map[string]*WrapCommitNode)
	for _, line := range lines {
		if len(strings.TrimSpace(line)) == 0 {
			continue
		}
		commitData := strings.Split(strings.TrimSpace(line), "|")
		hashName := strings.TrimSpace(commitData[0])
		commit := newCommit(commitRepo.RepositoryID(), hashName, strings.Split(commitData[1], ","))
		commitMap[hashName] = NewWrapCommitNode(commitRepo, commit)
		_, err := commitRepo.Insert(ctx, commit)
		if err != nil {
			return nil, err
		}
	}
	return commitMap, nil
}

func newCommit(repoID uuid.UUID, hashStr string, parentHash []string) *models.Commit {
	var p []hash.Hash
	for _, pHashStr := range parentHash {
		pHashStr = strings.TrimSpace(pHashStr)
		if len(pHashStr) == 0 {
			continue
		}
		p = append(p, hash.Hash(pHashStr))
	}
	return &models.Commit{
		Hash:         hash.Hash(hashStr),
		Author:       models.Signature{},
		RepositoryID: repoID,
		Committer: models.Signature{
			When: time.Now(),
		},
		MergeTag:     "",
		Message:      hashStr,
		TreeHash:     hash.Hash{},
		ParentHashes: p,
		CreatedAt:    time.Time{},
		UpdatedAt:    time.Time{},
	}
}
