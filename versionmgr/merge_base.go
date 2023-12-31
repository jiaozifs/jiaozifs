package versionmgr

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
)

// errIsReachable is thrown when first commit is an ancestor of the second
var errIsReachable = fmt.Errorf("first is reachable from second")

// MergeBase mimics the behavior of `git merge-base actual other`, returning the
// best common ancestor between the actual and the passed one.
// The best common ancestors can not be reached from other common ancestors.
func (c *WrapCommitNode) MergeBase(ctx context.Context, other *WrapCommitNode) ([]*WrapCommitNode, error) {
	// use sortedByCommitDateDesc strategy
	sorted := sortByCommitDateDesc(c, other)
	newer := sorted[0]
	older := sorted[1]

	newerHistory, err := ancestorsIndex(ctx, older, newer)
	if errors.Is(err, errIsReachable) {
		return []*WrapCommitNode{older}, nil
	}

	if err != nil {
		return nil, err
	}

	var res []*WrapCommitNode
	inNewerHistory := isInIndexCommitFilter(newerHistory)
	resIter := NewFilterCommitIter(ctx, older, &inNewerHistory, &inNewerHistory)
	_ = resIter.ForEach(func(commit *WrapCommitNode) error {
		res = append(res, commit)
		return nil
	})

	return Independents(ctx, res)
}

// IsAncestor returns true if the actual commit is ancestor of the passed one.
// It returns an error if the history is not transversable
// It mimics the behavior of `git merge --is-ancestor actual other`
func (c *WrapCommitNode) IsAncestor(ctx context.Context, other *WrapCommitNode) (bool, error) {
	found := false
	iter := NewCommitPreorderIter(ctx, other, nil, nil)
	err := iter.ForEach(func(comm *WrapCommitNode) error {
		if !bytes.Equal(comm.Commit().Hash, c.Commit().Hash) {
			return nil
		}

		found = true
		return ErrStop
	})

	return found, err
}

// ancestorsIndex returns a map with the ancestors of the starting commit if the
// excluded one is not one of them. It returns errIsReachable if the excluded commit
// is ancestor of the starting, or another error if the history is not traversable.
func ancestorsIndex(ctx context.Context, excluded, starting *WrapCommitNode) (map[string]struct{}, error) {
	if bytes.Equal(excluded.Commit().Hash, starting.Commit().Hash) {
		return nil, errIsReachable
	}

	startingHistory := map[string]struct{}{}
	startingIter := NewCommitIterBSF(ctx, starting, nil, nil)
	err := startingIter.ForEach(func(commit *WrapCommitNode) error {
		if bytes.Equal(commit.Commit().Hash, excluded.Commit().Hash) {
			return errIsReachable
		}

		startingHistory[commit.Commit().Hash.Hex()] = struct{}{}
		return nil
	})

	if err != nil {
		return nil, err
	}

	return startingHistory, nil
}

// Independents returns a subset of the passed commits, that are not reachable the others
// It mimics the behavior of `git merge-base --independent commit...`.
func Independents(ctx context.Context, commits []*WrapCommitNode) ([]*WrapCommitNode, error) {
	// use sortedByCommitDateDesc strategy
	candidates := sortByCommitDateDesc(commits...)
	candidates = removeDuplicated(candidates)

	seen := map[string]struct{}{}
	var isLimit CommitFilter = func(commit *WrapCommitNode) bool {
		_, ok := seen[commit.Commit().Hash.Hex()]
		return ok
	}

	if len(candidates) < 2 {
		return candidates, nil
	}

	pos := 0
	for {
		from := candidates[pos]
		others := remove(candidates, from)
		fromHistoryIter := NewFilterCommitIter(ctx, from, nil, &isLimit)
		err := fromHistoryIter.ForEach(func(fromAncestor *WrapCommitNode) error {
			for _, other := range others {
				if bytes.Equal(fromAncestor.Commit().Hash, other.Commit().Hash) {
					candidates = remove(candidates, other)
					others = remove(others, other)
				}
			}

			if len(candidates) == 1 {
				return ErrStop
			}

			seen[fromAncestor.Commit().Hash.Hex()] = struct{}{}
			return nil
		})

		if err != nil {
			return nil, err
		}

		nextPos := indexOf(candidates, from) + 1
		if nextPos >= len(candidates) {
			break
		}

		pos = nextPos
	}

	return candidates, nil
}

// sortByCommitDateDesc returns the passed commits, sorted by `committer.When desc`
//
// Following this strategy, it is tried to reduce the time needed when walking
// the history from one commit to reach the others. It is assumed that ancestors
// use to be committed before its descendant;
// That way `Independents(A^, A)` will be processed as being `Independents(A, A^)`;
// so starting by `A` it will be reached `A^` way sooner than walking from `A^`
// to the initial commit, and then from `A` to `A^`.
func sortByCommitDateDesc(commits ...*WrapCommitNode) []*WrapCommitNode {
	sorted := make([]*WrapCommitNode, len(commits))
	copy(sorted, commits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Commit().Committer.When.After(sorted[j].Commit().Committer.When)
	})

	return sorted
}

// indexOf returns the first position where target was found in the passed commits
func indexOf(commits []*WrapCommitNode, target *WrapCommitNode) int {
	for i, commit := range commits {
		if bytes.Equal(target.Commit().Hash, commit.Commit().Hash) {
			return i
		}
	}

	return -1
}

// remove returns the passed commits excluding the commit toDelete
func remove(commits []*WrapCommitNode, toDelete *WrapCommitNode) []*WrapCommitNode {
	res := make([]*WrapCommitNode, len(commits))
	j := 0
	for _, commit := range commits {
		if bytes.Equal(commit.Commit().Hash, toDelete.Commit().Hash) {
			continue
		}

		res[j] = commit
		j++
	}

	return res[:j]
}

// removeDuplicated removes duplicated commits from the passed slice of commits
func removeDuplicated(commits []*WrapCommitNode) []*WrapCommitNode {
	seen := make(map[string]struct{}, len(commits))
	res := make([]*WrapCommitNode, len(commits))
	j := 0
	for _, commit := range commits {
		if _, ok := seen[commit.Commit().Hash.Hex()]; ok {
			continue
		}

		seen[commit.Commit().Hash.Hex()] = struct{}{}
		res[j] = commit
		j++
	}

	return res[:j]
}

// isInIndexCommitFilter returns a commitFilter that returns true
// if the commit is in the passed index.
func isInIndexCommitFilter(index map[string]struct{}) CommitFilter {
	return func(c *WrapCommitNode) bool {
		_, ok := index[c.Commit().Hash.Hex()]
		return ok
	}
}
