package versionmgr

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

// TestNewCommitIterCTime
//
//	     A--->B
//	   /        \
//	 root	     E-->F
//	   \	    /
//		C---->D
//
// use time.sleep to control the order, expect f-e-d-b-c-a-root
func TestNewCommitIterCTime(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repoID := uuid.New()
	repo := models.NewRepo(db)
	rootCommit, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "root")
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitA, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit a", rootCommit.Hash)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitC, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit c", rootCommit.Hash)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitB, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commt b", commitA.Hash)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitD, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit d", commitC.Hash)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitE, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit e", commitB.Hash, commitD.Hash)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 500)

	commitF, err := makeCommit(ctx, repo.CommitRepo(repoID), hash.Empty, "commit f", commitE.Hash)
	require.NoError(t, err)

	commitFNode := NewWrapCommitNode(repo.CommitRepo(repoID), commitF)

	t.Run("NewCommitIterCTime", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, nil, nil)
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 7)
		assertHash(t, exactCommits, commitF.Hash, commitE.Hash, commitD.Hash, commitB.Hash, commitC.Hash, commitA.Hash, rootCommit.Hash)
	})

	t.Run("NewCommitIterCTime ignore", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, nil, []hash.Hash{commitE.Hash})
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 1)
		assertHash(t, exactCommits, commitF.Hash)
	})

	t.Run("NewCommitIterCTime ignore", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, map[string]bool{commitE.Hash.Hex(): true}, nil)
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 1)
		assertHash(t, exactCommits, commitF.Hash)
	})

	t.Run("NewCommitIterCTime ignore", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, nil, []hash.Hash{commitD.Hash})
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 5)
		assertHash(t, exactCommits, commitF.Hash, commitE.Hash, commitB.Hash, commitA.Hash, rootCommit.Hash)
	})

	t.Run("NewCommitIterCTime ErrStop", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, nil, nil)
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			if bytes.Equal(node.Hash(), commitF.Hash) {
				return ErrStop
			}
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 1)
		assertHash(t, exactCommits, commitF.Hash)
	})

	t.Run("NewCommitIterCTime err", func(t *testing.T) {
		iter := NewCommitIterCTime(ctx, commitFNode, nil, nil)
		var exactCommits []*WrapCommitNode
		mockErr := errors.New("mock")
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			if bytes.Equal(node.Hash(), commitF.Hash) {
				return mockErr
			}
			return nil
		})
		require.ErrorIs(t, err, mockErr)
		require.Len(t, exactCommits, 1)
		assertHash(t, exactCommits, commitF.Hash)
	})
}
