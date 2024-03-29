package versionmgr

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gobwas/glob"

	"github.com/GitDataAI/jiaozifs/models"
	"github.com/GitDataAI/jiaozifs/models/filemode"
	"github.com/GitDataAI/jiaozifs/utils/hash"
	"github.com/GitDataAI/jiaozifs/versionmgr/merkletrie"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

var EmptyRoot = &models.TreeNode{
	Hash:       hash.Hash([]byte{}),
	Type:       models.TreeObject,
	SubObjects: make([]models.TreeEntry, 0),
}

var EmptyDirEntry = models.TreeEntry{
	Name: "",
	Hash: hash.Hash([]byte{}),
}

type FullTreeEntry struct {
	Name  string    `json:"name"`
	IsDir bool      `json:"is_dir"`
	Hash  hash.Hash `json:"hash"`
	Size  int64     `json:"size"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

var (
	ErrPathNotFound   = fmt.Errorf("path not found")
	ErrEntryExit      = fmt.Errorf("entry exit")
	ErrBlobMustBeLeaf = fmt.Errorf("blob must be leaf")
	ErrNotDirectory   = fmt.Errorf("path must be a directory")
)

type FullObject struct {
	node  *models.FileTree
	entry models.TreeEntry
}

func (objectName FullObject) Entry() models.TreeEntry {
	return objectName.entry
}
func (objectName FullObject) Node() *models.FileTree {
	return objectName.node
}

type WorkTree struct {
	object models.IFileTreeRepo
	root   *TreeNode
}

func NewWorkTree(ctx context.Context, object models.IFileTreeRepo, root models.TreeEntry) (*WorkTree, error) {
	rootNode, err := NewTreeNode(ctx, root, object)
	if err != nil {
		return nil, err
	}
	return &WorkTree{
		object: object,
		root:   rootNode,
	}, nil
}

func (workTree *WorkTree) Root() *TreeNode {
	return workTree.root
}
func (workTree *WorkTree) RepositoryID() uuid.UUID {
	return workTree.object.RepositoryID()
}

func (workTree *WorkTree) AppendDirectEntry(ctx context.Context, treeEntry models.TreeEntry) (*models.TreeNode, error) {
	children, err := workTree.root.Children()
	if err != nil {
		return nil, err
	}
	for _, node := range children {
		if node.Name() == treeEntry.Name {
			return nil, ErrEntryExit
		}
	}

	subObjects := models.SortSubObjects(append(workTree.root.SubObjects(), treeEntry))

	newTree, err := models.NewTreeNode(models.Property{Mode: filemode.Dir}, workTree.RepositoryID(), subObjects...)
	if err != nil {
		return nil, err
	}

	obj, err := workTree.object.Insert(ctx, newTree.FileTree())
	if err != nil {
		return nil, err
	}
	return obj.TreeNode(), nil
}

func (workTree *WorkTree) DeleteDirectEntry(ctx context.Context, name string) (*models.TreeNode, bool, error) {
	subObjects := []models.TreeEntry{} //ensure subobject not nul
	for _, sub := range workTree.root.SubObjects() {
		if sub.Name != name { //filter tree entry by name
			subObjects = append(subObjects, sub)
		}
	}

	if len(subObjects) == 0 {
		//this node has no element return nothing
		return nil, true, nil
	}

	newTree, err := models.NewTreeNode(workTree.root.Properties(), workTree.RepositoryID(), subObjects...)
	if err != nil {
		return nil, false, err
	}

	obj, err := workTree.object.Insert(ctx, newTree.FileTree())
	if err != nil {
		return nil, false, err
	}
	return obj.TreeNode(), false, nil
}

func (workTree *WorkTree) ReplaceSubTreeEntry(ctx context.Context, treeEntry models.TreeEntry) (*models.TreeNode, error) {
	index := -1
	var sub models.TreeEntry
	for index, sub = range workTree.root.SubObjects() {
		if sub.Name == treeEntry.Name {
			break
		}
	}
	if index == -1 {
		return nil, ErrPathNotFound
	}

	subObjects := make([]models.TreeEntry, len(workTree.root.SubObjects()))
	copy(subObjects, workTree.root.SubObjects())
	subObjects[index] = treeEntry

	newTree, err := models.NewTreeNode(workTree.Root().Properties(), workTree.RepositoryID(), subObjects...)
	if err != nil {
		return nil, err
	}

	obj, err := workTree.object.Insert(ctx, newTree.FileTree())
	if err != nil {
		return nil, err
	}
	return obj.TreeNode(), nil
}

func (workTree *WorkTree) findNodeByPath(ctx context.Context, path string) ([]FullObject, []string, error) {
	pathSegs := strings.Split(path, "/") //path must be unix style
	var existNodes []FullObject
	var missingPath []string
	//a/b/c/d/e
	//a/b/c
	//a/b/c/d/e/f/g

	curNode := workTree.root
	for index, seg := range pathSegs {
		entry, err := curNode.SubEntry(ctx, seg)
		if errors.Is(err, ErrPathNotFound) {
			missingPath = pathSegs[index:]
			return existNodes, missingPath, nil
		}

		if entry.IsDir {
			curNode, err = curNode.SubDir(ctx, entry.Name)
			if err != nil {
				return nil, nil, err
			}
			existNodes = append(existNodes, FullObject{
				node:  curNode.TreeNode().FileTree(),
				entry: entry,
			})
		} else {
			//must be file
			blob, err := curNode.SubFile(ctx, entry.Name)
			if err != nil {
				return nil, nil, err
			}
			existNodes = append(existNodes, FullObject{
				node:  blob.FileTree(),
				entry: entry,
			})

			if index != len(pathSegs)-1 {
				//blob must be leaf
				return nil, nil, ErrBlobMustBeLeaf
			}
		}

	}
	return existNodes, nil, nil
}

// AddLeaf insert new leaf in entry, if path not exit, create new
func (workTree *WorkTree) AddLeaf(ctx context.Context, fullPath string, blob *models.Blob) error {
	fullPath = CleanPath(fullPath)
	existNode, missingPath, err := workTree.findNodeByPath(ctx, fullPath)
	if err != nil {
		return err
	}

	if len(missingPath) == 0 {
		return ErrEntryExit
	}

	_, err = workTree.object.Insert(ctx, blob.FileTree())
	if err != nil {
		return err
	}

	slices.Reverse(missingPath)
	var lastEntry models.TreeEntry
	for index, path := range missingPath {
		if len(path) == 0 { //todo add validate name check
			return fmt.Errorf("name is empty")
		}
		if index == 0 {
			_, err = workTree.object.Insert(ctx, blob.FileTree())
			if err != nil {
				return err
			}
			lastEntry = models.TreeEntry{
				Name:  path,
				IsDir: false,
				Hash:  blob.Hash,
			}
			continue
		}

		newTree, err := models.NewTreeNode(models.DefaultDirProperty(), workTree.RepositoryID(), lastEntry)
		if err != nil {
			return err
		}
		_, err = workTree.object.Insert(ctx, newTree.FileTree())
		if err != nil {
			return err
		}
		lastEntry = models.TreeEntry{
			Name:  path,
			IsDir: true,
			Hash:  newTree.Hash,
		}
	}

	slices.Reverse(existNode)
	existNode = append(existNode, FullObject{
		node:  workTree.root.TreeNode().FileTree(),
		entry: models.NewRootTreeEntry(workTree.root.Hash()), //root node have no name
	})

	for index, node := range existNode {
		newWorkTree, err := NewWorkTree(ctx, workTree.object, node.Entry())
		if err != nil {
			return err
		}
		var newNode *models.TreeNode
		if index == 0 { //insert new node
			newNode, err = newWorkTree.AppendDirectEntry(ctx, lastEntry)
		} else { //replace node
			newNode, err = newWorkTree.ReplaceSubTreeEntry(ctx, lastEntry)
		}
		if err != nil {
			return err
		}
		lastEntry = models.TreeEntry{
			Name:  node.Entry().Name, // use old name but replace with new hase
			IsDir: true,
			Hash:  newNode.Hash,
		}
	}
	workTree.root, err = NewTreeNode(ctx, lastEntry, workTree.object)
	return err
}

// ReplaceLeaf replace leaf with a new blob, all parent directory updated
func (workTree *WorkTree) ReplaceLeaf(ctx context.Context, fullPath string, blob *models.Blob) error {
	fullPath = CleanPath(fullPath)
	existNode, missingPath, err := workTree.findNodeByPath(ctx, fullPath)
	if err != nil {
		return err
	}

	if len(missingPath) > 0 {
		return ErrPathNotFound
	}

	_, err = workTree.object.Insert(ctx, blob.FileTree())
	if err != nil {
		return err
	}

	slices.Reverse(existNode)
	existNode = append(existNode, FullObject{
		node:  workTree.root.TreeNode().FileTree(),
		entry: models.NewRootTreeEntry(workTree.root.Hash()), //root node have no name
	})

	var lastEntry models.TreeEntry
	var newNode *models.TreeNode
	for index, node := range existNode {
		if index == 0 {
			lastEntry = models.TreeEntry{
				Name:  node.Entry().Name,
				IsDir: false,
				Hash:  blob.Hash,
			}
			continue
		}

		subWorkTree, err := NewWorkTree(ctx, workTree.object, node.Entry())
		if err != nil {
			return err
		}
		newNode, err = subWorkTree.ReplaceSubTreeEntry(ctx, lastEntry)
		if err != nil {
			return err
		}
		lastEntry = models.TreeEntry{
			Name:  node.Entry().Name,
			IsDir: true,
			Hash:  newNode.Hash,
		}
	}
	workTree.root, err = NewTreeNode(ctx, lastEntry, workTree.object)
	return err
}

// RemoveEntry remove tree entry from specific tree, if directory have only one entry, this directory was remove too
// examples:  a -> b
// a
// └── b
//
//	├── c.txt
//	└── d.txt
//
// RemoveEntry(ctx, root, "a") return empty root
// RemoveEntry(ctx, root, "a/b/c.txt") return new root of(a/b/c.txt)
// RemoveEntry(ctx, root, "a/b") return empty root. a b c.txt d.txt all removed
func (workTree *WorkTree) RemoveEntry(ctx context.Context, fullPath string) error {
	fullPath = CleanPath(fullPath)
	existNode, missingPath, err := workTree.findNodeByPath(ctx, fullPath)
	if err != nil {
		return err
	}

	if len(missingPath) > 0 {
		return ErrPathNotFound
	}

	slices.Reverse(existNode)
	existNode = append(existNode, FullObject{
		node:  workTree.root.TreeNode().FileTree(),
		entry: models.NewRootTreeEntry(workTree.root.Hash()), //root node have no name
	})

	lastEntry := existNode[0].Entry()
	existNode = existNode[1:]

	var newNode *models.TreeNode
	for index, node := range existNode {
		subWorkTree, err := NewWorkTree(ctx, workTree.object, node.Entry())
		if err != nil {
			return err
		}
		if index == 0 || lastEntry.Hash.IsEmpty() {
			var isEmpty bool
			newNode, isEmpty, err = subWorkTree.DeleteDirectEntry(ctx, lastEntry.Name)
			if err != nil {
				return err
			}

			lastEntry = models.TreeEntry{
				Name:  node.Entry().Name,
				IsDir: true, //leaf node has skip before loop,exist node must be directory
			}
			if !isEmpty {
				lastEntry.Hash = newNode.Hash
			}
		} else {
			newNode, err = subWorkTree.ReplaceSubTreeEntry(ctx, lastEntry)
			if err != nil {
				return err
			}
			lastEntry = models.TreeEntry{
				Name:  node.Entry().Name,
				IsDir: true,
				Hash:  newNode.Hash,
			}
		}
	}
	if newNode == nil {
		workTree.root, _ = NewTreeNode(ctx, EmptyDirEntry, workTree.object)
		return nil
	}

	workTree.root, err = NewTreeNode(ctx, lastEntry, workTree.object)
	return err
}

// Ls list tree entry of specific path of specific root
// examples:  a -> b
// a
// └── b
//
//	├── c.txt
//	└── d.txt
//
// Ls(ctx, root, "a") return b
// Ls(ctx, root, "a/b" return c.txt and d.txt
func (workTree *WorkTree) Ls(ctx context.Context, pattern string) ([]FullTreeEntry, error) {
	fullPath := CleanPath(pattern)
	if len(fullPath) == 0 {
		return workTree.getFullEntry(ctx, workTree.root.SubObjects())
	}

	existNode, missingPath, err := workTree.findNodeByPath(ctx, fullPath)
	if err != nil {
		return nil, err
	}

	if len(missingPath) > 0 {
		return nil, ErrPathNotFound
	}

	lastNode := existNode[len(existNode)-1]
	if lastNode.Node().Type != models.TreeObject {
		return nil, ErrNotDirectory
	}
	return workTree.getFullEntry(ctx, lastNode.Node().SubObjects)
}

type TreeManifest struct {
	Size     int64    `json:"size"`
	FileList []string `json:"file_list"`
}

func (workTree *WorkTree) GetTreeManifest(ctx context.Context, pattern string) (TreeManifest, error) {
	//todo match all files, it maybe slow maybe need a new algo like filepath.Glob
	pattern = CleanPath(pattern)
	wk := FileWalk{curNode: workTree.root, object: workTree.object}
	g, err := glob.Compile(pattern)
	if err != nil {
		return TreeManifest{}, err
	}

	files := make([]string, 0)
	var size int64
	err = wk.Walk(ctx, func(entry *models.TreeEntry, blob *models.Blob, path string) error {
		if entry.IsDir {
			return nil
		}
		if g.Match(path) {
			size += blob.Size
			files = append(files, path)
		}
		return nil
	})
	return TreeManifest{
		Size:     size,
		FileList: files,
	}, err
}

func (workTree *WorkTree) getFullEntry(ctx context.Context, treeEntries []models.TreeEntry) ([]FullTreeEntry, error) {
	entries := make([]FullTreeEntry, 0)
	for _, entry := range treeEntries {
		fe := FullTreeEntry{
			Name:  entry.Name,
			IsDir: entry.IsDir,
			Hash:  entry.Hash,
		}
		if entry.IsDir {
			blob, err := workTree.object.TreeNode(ctx, entry.Hash)
			if err != nil {
				return nil, err
			}
			fe.CreatedAt = blob.CreatedAt
			fe.UpdatedAt = blob.UpdatedAt
			entries = append(entries, fe)
		} else {
			blob, err := workTree.object.Blob(ctx, entry.Hash)
			if err != nil {
				return nil, err
			}
			fe.Size = blob.Size
			fe.CreatedAt = blob.CreatedAt
			fe.UpdatedAt = blob.UpdatedAt
			entries = append(entries, fe)
		}
	}
	return entries, nil
}

func (workTree *WorkTree) FindBlob(ctx context.Context, fullPath string) (*models.Blob, string, error) {
	fullPath = CleanPath(fullPath)
	existNode, missingPath, err := workTree.findNodeByPath(ctx, fullPath)
	if err != nil {
		return nil, "", err
	}

	if len(missingPath) > 0 {
		return nil, "", ErrPathNotFound
	}

	lastNode := existNode[len(existNode)-1]
	if lastNode.Node().Type != models.BlobObject {
		return nil, "", ErrPathNotFound
	}

	return lastNode.Node().Blob(), lastNode.Entry().Name, nil
}

func (workTree *WorkTree) ApplyOneChange(ctx context.Context, change IChange) error {
	action, err := change.Action()
	if err != nil {
		return err
	}
	switch action {
	case merkletrie.Insert:
		blob, err := workTree.object.Blob(ctx, change.To().Hash())
		if err != nil {
			return err
		}
		return workTree.AddLeaf(ctx, change.To().String(), blob)
	case merkletrie.Delete:
		return workTree.RemoveEntry(ctx, change.From().String())
	case merkletrie.Modify:
		blob, err := workTree.object.Blob(ctx, change.To().Hash())
		if err != nil {
			return err
		}
		return workTree.ReplaceLeaf(ctx, change.To().String(), blob)
	}
	return fmt.Errorf("unexpect change action: %s", action)
}

func (workTree *WorkTree) Diff(ctx context.Context, rootTreeHash hash.Hash, prefix string) (*Changes, error) {
	toNode, err := NewTreeNode(ctx, models.NewRootTreeEntry(rootTreeHash), workTree.object)
	if err != nil {
		return nil, err
	}

	changes, err := merkletrie.DiffTreeContext(ctx, workTree.Root(), toNode)
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}

	prefix = CleanPath(prefix)
	var filteredChange []merkletrie.Change
	for _, change := range changes {
		path := change.Path()
		if !strings.HasPrefix(path, prefix) {
			continue
		}
		filteredChange = append(filteredChange, change)
	}
	return newChanges(filteredChange), nil
}

// CleanPath clean path
// 1. replace \\ to /
// 2. trim space
// 3. trim first or last /
func CleanPath(fullPath string) string {
	path := strings.ReplaceAll(fullPath, "\\", "/")
	return filepath.ToSlash(strings.Trim(strings.TrimSpace(path), "/\\"))
}
