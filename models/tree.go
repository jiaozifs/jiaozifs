package models

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jiaozifs/jiaozifs/models/filemode"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/uptrace/bun"
)

var ErrRepoIDMisMatch = errors.New("repo id mismatch")

// ObjectType internal object type
// Integer values from 0 to 7 map to those exposed by git.
// AnyObject is used to represent any from 0 to 7.
type ObjectType int8

const (
	InvalidObject ObjectType = 0
	CommitObject  ObjectType = 1
	TreeObject    ObjectType = 2
	BlobObject    ObjectType = 3
	TagObject     ObjectType = 4
)

type TreeEntry struct {
	Name  string    `bun:"name" json:"name"`
	IsDir bool      `bun:"is_dir" json:"is_dir"`
	Hash  hash.Hash `bun:"hash" json:"hash"`
}

func SortSubObjects(subObjects []TreeEntry) []TreeEntry {
	sort.Slice(subObjects, func(i, j int) bool {
		return subObjects[i].Name < subObjects[j].Name
	})
	return subObjects
}

func NewRootTreeEntry(hash hash.Hash) TreeEntry {
	return TreeEntry{
		Name:  "",
		Hash:  hash,
		IsDir: true,
	}
}
func (treeEntry TreeEntry) Equal(other TreeEntry) bool {
	return bytes.Equal(treeEntry.Hash, other.Hash) && treeEntry.Name == other.Name
}

type Property struct {
	Mode filemode.FileMode `json:"mode"`
}

func DefaultDirProperty() Property {
	return Property{
		Mode: filemode.Dir,
	}
}

func DefaultLeafProperty() Property {
	return Property{
		Mode: filemode.Regular,
	}
}

func (props Property) ToMap() map[string]string {
	return map[string]string{
		"mode": props.Mode.String(),
	}
}

type Blob struct {
	bun.BaseModel `bun:"table:trees"`
	Hash          hash.Hash  `bun:"hash,pk,type:bytea"`
	RepositoryID  uuid.UUID  `bun:"repository_id,pk,type:uuid,notnull"`
	CheckSum      hash.Hash  `bun:"check_sum,type:bytea"`
	Type          ObjectType `bun:"type,notnull"`
	Size          int64      `bun:"size"`
	Properties    Property   `bun:"properties,type:jsonb,notnull"`

	CreatedAt time.Time `bun:"created_at,notnull"`
	UpdatedAt time.Time `bun:"updated_at,notnull"`
}

func NewBlob(props Property, repoID uuid.UUID, checkSum hash.Hash, size int64) (*Blob, error) {
	blob := &Blob{
		CheckSum:     checkSum,
		RepositoryID: repoID,
		Type:         BlobObject,
		Size:         size,
		Properties:   props,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	hash, err := blob.calculateHash()
	if err != nil {
		return nil, err
	}
	blob.Hash = hash
	return blob, err
}

func (blob *Blob) calculateHash() (hash.Hash, error) {
	hasher := hash.NewHasher(hash.Md5)
	err := hasher.WriteInt8(int8(blob.Type))
	if err != nil {
		return nil, err
	}

	_, err = hasher.Write(blob.CheckSum)
	if err != nil {
		return nil, err
	}

	//write mode property  todo change reflect
	for k, v := range blob.Properties.ToMap() {
		err = hasher.WriteString(k)
		if err != nil {
			return nil, err
		}

		err = hasher.WriteString(v)
		if err != nil {
			return nil, err
		}
	}
	return hasher.Md5.Sum(nil), nil
}

func (blob *Blob) FileTree() *FileTree {
	return &FileTree{
		Hash:         blob.Hash,
		RepositoryID: blob.RepositoryID,
		Type:         blob.Type,
		Size:         blob.Size,
		CheckSum:     blob.CheckSum,
		Properties:   blob.Properties,
		CreatedAt:    blob.CreatedAt,
		UpdatedAt:    blob.UpdatedAt,
	}
}

type TreeNode struct {
	bun.BaseModel `bun:"table:trees"`
	Hash          hash.Hash `bun:"hash,pk,type:bytea" json:"hash"`
	RepositoryID  uuid.UUID `bun:"repository_id,pk,type:uuid,notnull" json:"repository_id"`

	Type       ObjectType  `bun:"type,notnull" json:"type"`
	SubObjects []TreeEntry `bun:"sub_objects,type:jsonb" json:"sub_objects"`
	Properties Property    `bun:"properties,type:jsonb,notnull" json:"properties"`

	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

func NewTreeNode(props Property, repoID uuid.UUID, subObjects ...TreeEntry) (*TreeNode, error) {
	if subObjects == nil {
		subObjects = make([]TreeEntry, 0) //to ensure tree entry not null
	}
	newTree := &TreeNode{
		Type:         TreeObject,
		RepositoryID: repoID,
		SubObjects:   SortSubObjects(subObjects),
		Properties:   props,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	hash, err := newTree.calculateHash()
	if err != nil {
		return nil, err
	}
	newTree.Hash = hash
	return newTree, nil
}

func (tn *TreeNode) FileTree() *FileTree {
	return &FileTree{
		Hash:         tn.Hash,
		RepositoryID: tn.RepositoryID,
		Type:         tn.Type,
		SubObjects:   tn.SubObjects,
		Properties:   tn.Properties,
		CreatedAt:    tn.CreatedAt,
		UpdatedAt:    tn.UpdatedAt,
	}
}

func (tn *TreeNode) calculateHash() (hash.Hash, error) {
	hasher := hash.NewHasher(hash.Md5)
	err := hasher.WriteInt8(int8(tn.Type))
	if err != nil {
		return nil, err
	}
	for _, obj := range tn.SubObjects {
		_, err = hasher.Write(obj.Hash)
		if err != nil {
			return nil, err
		}
		err = hasher.WriteString(obj.Name)
		if err != nil {
			return nil, err
		}
	}

	for name, value := range tn.Properties.ToMap() {
		err = hasher.WriteString(name)
		if err != nil {
			return nil, err
		}
		err = hasher.WriteString(value)
		if err != nil {
			return nil, err
		}
	}

	return hasher.Md5.Sum(nil), nil
}

type FileTree struct {
	bun.BaseModel `bun:"table:trees"`
	Hash          hash.Hash  `bun:"hash,pk,type:bytea"`
	RepositoryID  uuid.UUID  `bun:"repository_id,pk,type:uuid,notnull"`
	CheckSum      hash.Hash  `bun:"check_sum,type:bytea"`
	Type          ObjectType `bun:"type,notnull"`
	Size          int64      `bun:"size"`
	Properties    Property   `bun:"properties,type:jsonb,notnull"`
	//tree
	SubObjects []TreeEntry `bun:"sub_objects,type:jsonb,notnull" json:"sub_objects"`

	CreatedAt time.Time `bun:"created_at,notnull" json:"created_at"`
	UpdatedAt time.Time `bun:"updated_at,notnull" json:"updated_at"`
}

func (fileTree *FileTree) Blob() *Blob {
	return &Blob{
		Hash:         fileTree.Hash,
		Type:         fileTree.Type,
		RepositoryID: fileTree.RepositoryID,
		Size:         fileTree.Size,
		Properties:   fileTree.Properties,
		CheckSum:     fileTree.CheckSum,
		CreatedAt:    fileTree.CreatedAt,
		UpdatedAt:    fileTree.UpdatedAt,
	}
}

func (fileTree *FileTree) TreeNode() *TreeNode {
	return &TreeNode{
		Hash:         fileTree.Hash,
		Type:         fileTree.Type,
		Properties:   fileTree.Properties,
		RepositoryID: fileTree.RepositoryID,
		SubObjects:   fileTree.SubObjects,
		CreatedAt:    fileTree.CreatedAt,
		UpdatedAt:    fileTree.UpdatedAt,
	}
}

type GetObjParams struct {
	hash hash.Hash
}

func NewGetObjParams() *GetObjParams {
	return &GetObjParams{}
}

func (gop *GetObjParams) SetHash(hash hash.Hash) *GetObjParams {
	gop.hash = hash
	return gop
}

type DeleteTreeParams struct {
	hash hash.Hash
}

func NewDeleteTreeParams() *DeleteTreeParams {
	return &DeleteTreeParams{}
}

func (dtp *DeleteTreeParams) SetHash(hash hash.Hash) *DeleteTreeParams {
	dtp.hash = hash
	return dtp
}

type IFileTreeRepo interface {
	RepositoryID() uuid.UUID
	Insert(ctx context.Context, repo *FileTree) (*FileTree, error)
	Get(ctx context.Context, params *GetObjParams) (*FileTree, error)
	Count(ctx context.Context) (int, error)
	List(ctx context.Context) ([]FileTree, error)
	Blob(ctx context.Context, hash hash.Hash) (*Blob, error)
	TreeNode(ctx context.Context, hash hash.Hash) (*TreeNode, error)
	Delete(ctx context.Context, params *DeleteTreeParams) (int64, error)
}

var _ IFileTreeRepo = (*FileTreeRepo)(nil)

type FileTreeRepo struct {
	db           bun.IDB
	repositoryID uuid.UUID
}

func NewFileTree(db bun.IDB, repositoryID uuid.UUID) IFileTreeRepo {
	return &FileTreeRepo{
		db:           db,
		repositoryID: repositoryID,
	}
}

func (o FileTreeRepo) RepositoryID() uuid.UUID {
	return o.repositoryID
}

func (o FileTreeRepo) Insert(ctx context.Context, obj *FileTree) (*FileTree, error) {
	if obj.RepositoryID != o.repositoryID {
		return nil, ErrRepoIDMisMatch
	}
	_, err := o.db.NewInsert().Model(obj).Ignore().Exec(ctx)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (o FileTreeRepo) Get(ctx context.Context, params *GetObjParams) (*FileTree, error) {
	repo := &FileTree{}
	query := o.db.NewSelect().Model(repo).Where("repository_id = ?", o.repositoryID)

	if params.hash != nil {
		query = query.Where("hash = ?", params.hash)
	}

	err := query.Limit(1).Scan(ctx, repo)
	if err != nil {
		return nil, err
	}
	return repo, nil
}

func (o FileTreeRepo) Blob(ctx context.Context, hash hash.Hash) (*Blob, error) {
	blob := &Blob{}
	err := o.db.NewSelect().
		Model(blob).Limit(1).
		Where("repository_id = ?", o.repositoryID).
		Where("hash = ?", hash).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return blob, nil
}

func (o FileTreeRepo) TreeNode(ctx context.Context, hash hash.Hash) (*TreeNode, error) {
	tree := &TreeNode{}
	err := o.db.NewSelect().
		Model(tree).Limit(1).
		Where("repository_id = ?", o.repositoryID).
		Where("hash = ?", hash).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return tree, nil
}

func (o FileTreeRepo) Count(ctx context.Context) (int, error) {
	return o.db.NewSelect().
		Model((*FileTree)(nil)).
		Where("repository_id = ?", o.repositoryID).
		Count(ctx)
}

func (o FileTreeRepo) List(ctx context.Context) ([]FileTree, error) {
	var obj []FileTree
	err := o.db.NewSelect().Model(&obj).
		Where("repository_id = ?", o.repositoryID).
		Scan(ctx)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func (o FileTreeRepo) Delete(ctx context.Context, params *DeleteTreeParams) (int64, error) {
	query := o.db.NewDelete().Model((*TreeNode)(nil)).Where("repository_id = ?", o.repositoryID)
	if params.hash != nil {
		query = query.Where("hash = ?", params.hash)
	}

	sqlResult, err := query.Exec(ctx)
	if err != nil {
		return 0, err
	}
	affectedRows, err := sqlResult.RowsAffected()
	if err != nil {
		return 0, err
	}
	return affectedRows, err
}
