package versionmgr

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/utils/hash"

	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
)

// TestWrapCommitNode
//
//	     A--->B
//	   /        \
//	 root	     E-->F
//	   \	    /
//		C---->D
func TestWrapCommitNode(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	repo := models.NewRepo(db)
	rootCommit, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "root")
	require.NoError(t, err)
	commitA, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit a", rootCommit.Hash)
	require.NoError(t, err)

	commitB, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commt b", commitA.Hash)
	require.NoError(t, err)

	commitC, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit c", rootCommit.Hash)
	require.NoError(t, err)
	commitD, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit d", commitC.Hash)
	require.NoError(t, err)

	commitE, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit e", commitB.Hash, commitD.Hash)
	require.NoError(t, err)

	commitF, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit f", commitE.Hash)
	require.NoError(t, err)

	commitFNode := NewWrapCommitNode(repo.CommitRepo(repoID), commitF)

	require.Equal(t, commitF.Hash.Hex(), commitFNode.Commit().Hash.Hex())
	require.Equal(t, repoID, commitFNode.RepoID())
	require.Equal(t, hash.Empty, commitFNode.TreeHash())
	require.Equal(t, commitF.Hash.Hex(), commitFNode.Hash().Hex())

	t.Run("parent", func(t *testing.T) {
		commitENode := NewWrapCommitNode(repo.CommitRepo(repoID), commitE)
		node, err := commitENode.Parents(ctx)
		require.NoError(t, err)
		require.Equal(t, commitB.Hash.Hex(), node[0].Hash().Hex())
		require.Equal(t, commitD.Hash.Hex(), node[1].Hash().Hex())
	})
	t.Run("get commit", func(t *testing.T) {
		commit, err := commitFNode.GetCommit(ctx, commitA.Hash)
		require.NoError(t, err)
		require.Equal(t, commitA.Hash.Hex(), commit.Hash().Hex())
	})

	t.Run("get commits", func(t *testing.T) {
		commits, err := commitFNode.GetCommits(ctx, []hash.Hash{commitA.Hash, commitB.Hash})
		require.NoError(t, err)
		require.Equal(t, commitA.Hash.Hex(), commits[0].Hash().Hex())
		require.Equal(t, commitB.Hash.Hex(), commits[1].Hash().Hex())
	})

	t.Run("get commits", func(t *testing.T) {
		commits, err := commitFNode.GetCommits(ctx, []hash.Hash{commitA.Hash, commitB.Hash})
		require.NoError(t, err)
		require.Equal(t, commitA.Hash.Hex(), commits[0].Hash().Hex())
		require.Equal(t, commitB.Hash.Hex(), commits[1].Hash().Hex())
	})
	commits, err := commitFNode.GetCommits(ctx, []hash.Hash{commitA.Hash, commitB.Hash})
	require.NoError(t, err)
	require.Equal(t, commitA.Hash.Hex(), commits[0].Hash().Hex())
	require.Equal(t, commitB.Hash.Hex(), commits[1].Hash().Hex())

	t.Run("iter foreach", func(t *testing.T) {
		iter := newArrayCommitIter(commits)
		var exactCommits []*WrapCommitNode
		err := iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 2)
		require.Equal(t, commitA.Hash.Hex(), exactCommits[0].Hash().Hex())
		require.Equal(t, commitB.Hash.Hex(), exactCommits[1].Hash().Hex())
	})

	t.Run("hash next", func(t *testing.T) {
		iter := newArrayCommitIter(commits)
		var exactCommits []*WrapCommitNode
		for iter.Has() {
			node, err := iter.Next()
			require.NoError(t, err)
			exactCommits = append(exactCommits, node)
		}
		require.Len(t, exactCommits, 2)
		require.Equal(t, commitA.Hash.Hex(), exactCommits[0].Hash().Hex())
		require.Equal(t, commitB.Hash.Hex(), exactCommits[1].Hash().Hex())
		assertHash(t, exactCommits, commitA.Hash, commitB.Hash)
	})
}

func assertHash(t *testing.T, exactCommits []*WrapCommitNode, seq ...hash.Hash) {
	require.Equal(t, len(exactCommits), len(seq))
	for index, c := range exactCommits {
		require.Equal(t, c.Hash().Hex(), seq[index].Hex())
	}
}
