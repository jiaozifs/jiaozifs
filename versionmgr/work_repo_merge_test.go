package versionmgr

import (
	"context"
	"testing"

	"github.com/jiaozifs/jiaozifs/block/mem"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/stretchr/testify/require"
)

//  TestCommitOpMerge
//example
//       A -----C
//       |      | \
//       |      |  \
//		/ \    F    \
//    /    \  /      \
//  root    AB       CG
//    \    /  \     /
//     \ /     \   /
//      B-D-E--- G

func TestCommitOpMerge(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	adapter := mem.New(ctx)
	repo := models.NewRepo(db)

	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	project, err := makeRepository(ctx, repo, user, "testproject")
	require.NoError(t, err)

	testData := `
1|a.txt	|h1
1|b/c.txt	|h2
`
	workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

	err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/base")
	require.NoError(t, err)

	oriCommit, err := addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData)
	require.NoError(t, err)

	//modify a.txt
	//---------------CommitA
	testData = `
3|a.txt	|h5 
3|b/c.txt	|h2
`
	err = workRepo.CheckOut(ctx, InCommit, oriCommit.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchA")
	require.NoError(t, err)

	commitA, err := addChangesToWip(ctx, workRepo, "feat/branchA", "commit a", testData)
	require.NoError(t, err)

	//modify a.txt
	//---------------CommitB
	testData = `
3|a.txt	|h4
3|b/c.txt	|h2
`
	err = workRepo.CheckOut(ctx, InCommit, oriCommit.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchB")
	require.NoError(t, err)

	commitB, err := addChangesToWip(ctx, workRepo, "feat/branchB", "commit b", testData)
	require.NoError(t, err)

	//--------------CommitAB
	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchA"))
	commitAB, err := workRepo.Merge(ctx, commitB.Hash, "commit ab", LeastHashResolve)
	require.NoError(t, err)

	//--------------CommitF
	testData = `
1|x.txt	|h4
`
	err = workRepo.CheckOut(ctx, InCommit, commitAB.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchF")
	require.NoError(t, err)

	_, err = addChangesToWip(ctx, workRepo, "feat/branchF", "commit f", testData)
	require.NoError(t, err)

	//commitC
	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchF"))
	commitC, err := workRepo.Merge(ctx, commitA.Hash, "commit c", LeastHashResolve)
	require.NoError(t, err)

	//commitD
	testData = `
3|a.txt	|h5
3|b/c.txt	|h6
1|g/c.txt	|h7
`
	err = workRepo.CheckOut(ctx, InCommit, commitB.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchD_E")
	require.NoError(t, err)

	_, err = addChangesToWip(ctx, workRepo, "feat/branchD_E", "commit d", testData)
	require.NoError(t, err)

	//commitE
	testData = `
2|a.txt	|h4
`
	commitE, err := addChangesToWip(ctx, workRepo, "feat/branchD_E", "commit e", testData)
	require.NoError(t, err)

	//test fast-ward
	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchB"))
	commitBE, err := workRepo.Merge(ctx, commitE.Hash, "commit ab", LeastHashResolve)
	require.NoError(t, err)
	require.Equal(t, commitE.Hash.Hex(), commitBE.Hash.Hex())

	//commitG
	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchD_E"))
	commitG, err := workRepo.Merge(ctx, commitAB.Hash, "commit g", LeastHashResolve)
	require.NoError(t, err)

	_, err = makeBranch(ctx, repo.BranchRepo(), user, "feat/branchG", project.ID, commitG.Hash)
	require.NoError(t, err)

	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchG"))
	_, err = workRepo.Merge(ctx, commitC.Hash, "commit cg", LeastHashResolve)
	require.NoError(t, err)
}

//	TestCrissCrossMerge
//
// example
//
//	          C--------D
//	        /   \    /  \
//		   /     \ /     \
//		 root     *AB     CG
//		   \    /  \    /
//		    \ /     \ /
//		     B-------G
func TestCrissCrossMerge(t *testing.T) {
	ctx := context.Background()
	closeDB, _, db := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	repo := models.NewRepo(db)
	adapter := mem.New(ctx)

	user, err := makeUser(ctx, repo.UserRepo(), "admin")
	require.NoError(t, err)

	project, err := makeRepository(ctx, repo, user, "testproject")
	require.NoError(t, err)
	workRepo := NewWorkRepositoryFromAdapter(ctx, user, project, repo, adapter)

	testData := `
1|a.txt	|h1
1|b.txt	|h2
`

	err = workRepo.CheckOut(ctx, InCommit, hash.Empty.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/base")
	require.NoError(t, err)

	oriCommit, err := addChangesToWip(ctx, workRepo, "feat/base", "base commit", testData)
	require.NoError(t, err)

	//------------------CommitC
	testData = `
3|a.txt	|h1 
3|b.txt	|h3
`
	err = workRepo.CheckOut(ctx, InCommit, oriCommit.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchC")
	require.NoError(t, err)

	commitC, err := addChangesToWip(ctx, workRepo, "feat/branchC", "base commit", testData)
	require.NoError(t, err)

	//modify a.txt
	//-----------------CommitB
	testData = `
3|a.txt	|h4 
3|b.txt	|h2
`
	err = workRepo.CheckOut(ctx, InCommit, oriCommit.Hash.Hex())
	require.NoError(t, err)
	_, err = workRepo.CreateBranch(ctx, "feat/branchB")
	require.NoError(t, err)

	commitB, err := addChangesToWip(ctx, workRepo, "feat/branchB", "base commit", testData)
	require.NoError(t, err)

	//-----------------CommitAB
	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchB"))
	commiyBC, err := workRepo.Merge(ctx, commitC.Hash, "commit bc", LeastHashResolve)
	require.NoError(t, err)

	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchC"))
	commitCB, err := workRepo.Merge(ctx, commitB.Hash, "commit cb", LeastHashResolve)
	require.NoError(t, err)

	_, err = makeBranch(ctx, repo.BranchRepo(), user, "feat/branchBC", project.ID, commiyBC.Hash)
	require.NoError(t, err)

	require.NoError(t, workRepo.CheckOut(ctx, InBranch, "feat/branchBC"))
	_, err = workRepo.Merge(ctx, commitCB.Hash, "cross commit", LeastHashResolve)
	require.NoError(t, err)
}
