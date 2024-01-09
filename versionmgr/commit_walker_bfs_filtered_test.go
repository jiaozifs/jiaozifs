package versionmgr

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

func TestNewFilterCommitIter(t *testing.T) {
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

	t.Run("NewFilterCommitIter", func(t *testing.T) {
		iter := NewFilterCommitIter(ctx, commitFNode, nil, nil)
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 7)
		assertHash(t, exactCommits, commitF.Hash, commitE.Hash, commitB.Hash, commitD.Hash, commitA.Hash, commitC.Hash, rootCommit.Hash)
	})

	t.Run("NewFilterCommitIter valid", func(t *testing.T) {
		valid := func(node *WrapCommitNode) bool {
			return !bytes.Equal(node.Hash(), commitB.Hash)
		}
		iter := NewFilterCommitIter(ctx, commitFNode, (*CommitFilter)(&valid), nil)
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 6)
		assertHash(t, exactCommits, commitF.Hash, commitE.Hash, commitD.Hash, commitA.Hash, commitC.Hash, rootCommit.Hash)
	})
	t.Run("NewFilterCommitIter valid", func(t *testing.T) {
		isLimit := func(node *WrapCommitNode) bool {
			return bytes.Equal(node.Hash(), commitB.Hash)
		}
		iter := NewFilterCommitIter(ctx, commitFNode, nil, (*CommitFilter)(&isLimit))
		var exactCommits []*WrapCommitNode
		err = iter.ForEach(func(node *WrapCommitNode) error {
			exactCommits = append(exactCommits, node)
			return nil
		})
		require.NoError(t, err)
		require.Len(t, exactCommits, 6)
		assertHash(t, exactCommits, commitF.Hash, commitE.Hash, commitB.Hash, commitD.Hash, commitC.Hash, rootCommit.Hash)
	})

	t.Run("NewFilterCommitIter ErrStop", func(t *testing.T) {
		iter := NewFilterCommitIter(ctx, commitFNode, nil, nil)
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

	t.Run("NewFilterCommitIter err", func(t *testing.T) {
		iter := NewFilterCommitIter(ctx, commitFNode, nil, nil)
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
