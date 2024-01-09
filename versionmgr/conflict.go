package versionmgr

import (
	"bytes"
	"fmt"

	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/jiaozifs/jiaozifs/versionmgr/merkletrie"
)

// ConflictResolver resolve conflict between two change
type ConflictResolver func(base IChange, merged IChange) (IChange, error)

// LeastHashResolve use least hash change for test
func LeastHashResolve(base IChange, merged IChange) (IChange, error) {
	baseAction, err := base.Action()
	if err != nil {
		return nil, err
	}

	mergeAction, err := merged.Action()
	if err != nil {
		return nil, err
	}

	if baseAction == merkletrie.Delete {
		return merged, nil
	}
	if mergeAction == merkletrie.Delete {
		return base, nil
	}

	if bytes.Compare(base.To().Hash(), merged.To().Hash()) < 0 {
		return base, nil
	}
	return merged, nil
}

func ForbidResolver(_ IChange, _ IChange) (IChange, error) {
	return nil, fmt.Errorf("not allow conflict in this mode")
}

type BaseChange struct {
	Action merkletrie.Action
	Hash   hash.Hash
}

type Conflict struct {
	Path  string
	Base  BaseChange
	Merge BaseChange
}

func OneSideResolver(isBase bool) ConflictResolver {
	return func(base IChange, merge IChange) (IChange, error) {
		if isBase {
			return base, nil
		}
		return merge, nil
	}
}

func ResolveFromSelector(resolveMsg map[string]string) ConflictResolver {
	return func(base IChange, merge IChange) (IChange, error) {
		if resolveMsg[base.Path()] == "base" {
			return base, nil
		}
		return merge, nil
	}
}

func ConflictCollector() ConflictResolver {
	changes := make([]Conflict, 0)
	return func(base IChange, merge IChange) (IChange, error) {
		baseAct, err := base.Action()
		if err != nil {
			return nil, err
		}
		baseHash := hash.Empty
		if baseAct != merkletrie.Delete {
			baseHash = base.To().Hash()
		}
		mergeAct, err := merge.Action()
		if err != nil {
			return nil, err
		}
		mergeHash := hash.Empty
		if mergeAct != merkletrie.Delete {
			mergeHash = merge.To().Hash()
		}
		changes = append(changes, Conflict{
			Path: base.Path(),
			Base: BaseChange{
				Action: baseAct,
				Hash:   baseHash,
			},
			Merge: BaseChange{
				Action: mergeAct,
				Hash:   mergeHash,
			},
		})
		return base, nil
	}
}
