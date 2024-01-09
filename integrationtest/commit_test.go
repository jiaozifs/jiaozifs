package integrationtest

import (
	"context"
	"net/http"

	"github.com/jiaozifs/jiaozifs/utils/hash"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/smartystreets/goconvey/convey"
)

func GetEntriesInRefSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	baseHead := utils.String("")
	return func(c convey.C) {
		userName := "kitty"
		repoName := "black"
		branchName := "feat/get_entries_test"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "kitty login", userName, false)
		createRepo(ctx, c, client, repoName)
		createBranch(ctx, c, client, "create branch", userName, repoName, "main", branchName)
		createWip(ctx, c, client, "feat get entries test0", userName, repoName, branchName)
		uploadObject(ctx, c, client, "update f1 to test branch", userName, repoName, branchName, "m.dat")
		uploadObject(ctx, c, client, "update f2 to test branch", userName, repoName, branchName, "g/x.dat")
		uploadObject(ctx, c, client, "update f3 to test branch", userName, repoName, branchName, "g/m.dat")

		c.Convey("get wip entries", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get entries in non exit user", func() {
				resp, err := client.GetEntriesInRef(ctx, "mock user", repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get entries in non exit repo", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, "fakerepo", &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get entries in non exit branch", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String("feat/fake_repo"),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden get entries in others", func() {
				resp, err := client.GetEntriesInRef(ctx, "jimmy", "happygo", &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String("main"),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("not exit path", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("a/b/c/d"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})
			c.Convey("success to get entries", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 2)
				convey.So((*result.JSON200)[0].Name, convey.ShouldEqual, "m.dat")
				convey.So((*result.JSON200)[1].Name, convey.ShouldEqual, "x.dat")
			})

			c.Convey("success to get entries in root", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeWip,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 2)
				convey.So((*result.JSON200)[0].Name, convey.ShouldEqual, "g")
				convey.So((*result.JSON200)[1].Name, convey.ShouldEqual, "m.dat")
			})
		})

		commitWip(ctx, c, client, "commit kitty first changes", userName, repoName, branchName, "test")

		c.Convey("get branch entries", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get entries in non exit user", func() {
				resp, err := client.GetEntriesInRef(ctx, "mock user", repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get entries in non exit repo", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, "fakerepo", &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get entries in non exit branch", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String("feat/fake_repo"),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden get entries in others", func() {
				resp, err := client.GetEntriesInRef(ctx, "jimmy", "happygo", &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String("main"),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("not exit path", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("a/b/c/d"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("success to get entries", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("g"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 2)
				convey.So((*result.JSON200)[0].Name, convey.ShouldEqual, "m.dat")
				convey.So((*result.JSON200)[1].Name, convey.ShouldEqual, "x.dat")
			})

			c.Convey("success to get entries in root", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String(branchName),
					Type: api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 2)
				convey.So((*result.JSON200)[0].Name, convey.ShouldEqual, "g")
				convey.So((*result.JSON200)[1].Name, convey.ShouldEqual, "m.dat")
			})
		})

		createWip(ctx, c, client, "main wip", userName, repoName, "main")
		uploadObject(ctx, c, client, "update f1 to main branch", userName, repoName, "main", "a.dat")   //delete\
		uploadObject(ctx, c, client, "update f2 to main branch", userName, repoName, "main", "g/m.dat") //modify
		commitWip(ctx, c, client, "commit branch change", userName, repoName, "main", "test")

		c.Convey("get commit entries", func(c convey.C) {
			c.Convey("fail to get entries in uncorrected hash", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String("123"),
					Type: api.RefTypeCommit,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("fail to get entries in not found", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String("46780d412b4b3c71ba6cdfcb52105c7b"),
					Type: api.RefTypeCommit,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("success to get entries in empty hash", func() {
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String(hash.Empty.Hex()),
					Type: api.RefTypeCommit,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 0)
			})

			c.Convey("success to get entries in commit", func() {
				getCommitsResp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(branchName),
				})
				convey.So(err, convey.ShouldBeNil)
				getCommitsResult, err := api.ParseGetCommitsInRefResponse(getCommitsResp)
				convey.So(err, convey.ShouldBeNil)

				commit := (*getCommitsResult.JSON200)[0]
				resp, err := client.GetEntriesInRef(ctx, userName, repoName, &api.GetEntriesInRefParams{
					Path: utils.String("/"),
					Ref:  utils.String(commit.Hash),
					Type: api.RefTypeCommit,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetEntriesInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 2)
				convey.So((*result.JSON200)[0].Name, convey.ShouldEqual, "g")
				convey.So((*result.JSON200)[1].Name, convey.ShouldEqual, "m.dat")
			})
		})

		c.Convey("compare commit", func(c convey.C) {
			c.Convey("get base and head", func() {
				resp, err := client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{RefName: "main"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
				baseBranch, err := api.ParseGetBranchResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				resp, err = client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{RefName: branchName})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
				headBranch, err := api.ParseGetBranchResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				baseHead = utils.String(baseBranch.JSON200.CommitHash + "..." + headBranch.JSON200.CommitHash)
			})
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.CompareCommit(ctx, userName, repoName, utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/"),
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to diff in non exit user", func() {
				resp, err := client.CompareCommit(ctx, "mockuser", repoName, utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to diff in non exit repo", func() {
				resp, err := client.CompareCommit(ctx, userName, "fakerepo", utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden diff in others", func() {
				resp, err := client.CompareCommit(ctx, "jimmy", "happygo", utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("not exit path", func() {
				resp, err := client.CompareCommit(ctx, userName, repoName, utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/a/b/c/d"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
			})

			c.Convey("success to  diff", func() {
				resp, err := client.CompareCommit(ctx, userName, repoName, utils.StringValue(baseHead), &api.CompareCommitParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseCompareCommitResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 4)
				convey.So((*result.JSON200)[0].Path, convey.ShouldEqual, "a.dat")
				convey.So((*result.JSON200)[0].Action, convey.ShouldEqual, 2)
				convey.So((*result.JSON200)[1].Path, convey.ShouldEqual, "g/m.dat")
				convey.So((*result.JSON200)[1].Action, convey.ShouldEqual, 3)
				convey.So((*result.JSON200)[2].Path, convey.ShouldEqual, "g/x.dat")
				convey.So((*result.JSON200)[2].Action, convey.ShouldEqual, 1)
				convey.So((*result.JSON200)[3].Path, convey.ShouldEqual, "m.dat")
				convey.So((*result.JSON200)[3].Action, convey.ShouldEqual, 1)
			})
		})
	}
}

func GetCommitChangesSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	var commits []api.Commit
	return func(c convey.C) {
		userName := "kelly"
		repoName := "gcc"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "getCommitChanges login", userName, false)
		createRepo(ctx, c, client, repoName)
		createWip(ctx, c, client, "feat get entries test0", userName, repoName, "main")
		uploadObject(ctx, c, client, "update f1 to test branch", userName, repoName, "main", "m.dat")
		commitWip(ctx, c, client, "commit kelly first changes", userName, repoName, "main", "test")

		uploadObject(ctx, c, client, "update f2 to test branch", userName, repoName, "main", "g/x.dat")
		commitWip(ctx, c, client, "commit kelly second changes", userName, repoName, "main", "test")

		uploadObject(ctx, c, client, "update f3 to test branch", userName, repoName, "main", "g/m.dat")
		commitWip(ctx, c, client, "commit kelly third changes", userName, repoName, "main", "test")

		c.Convey("get commit change", func(c convey.C) {
			c.Convey("list commit history", func() {
				resp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{RefName: utils.String("main")})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
				result, err := api.ParseGetCommitsInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				commits = *result.JSON200
			})

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetCommitChanges(ctx, userName, repoName, commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get commit changes in non exit user", func() {
				resp, err := client.GetCommitChanges(ctx, "mockuser", repoName, commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to  get commit changes in non exit repo", func() {
				resp, err := client.GetCommitChanges(ctx, userName, "fakerepo", commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden  get commit changes in others", func() {
				resp, err := client.GetCommitChanges(ctx, "jimmy", "happygo", commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("not exit path", func() {
				resp, err := client.GetCommitChanges(ctx, userName, repoName, commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/a/b/c/d"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
			})

			c.Convey("success to get first changes", func() {
				resp, err := client.GetCommitChanges(ctx, userName, repoName, commits[0].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseCompareCommitResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 1)
			})

			c.Convey("success to get first commit changes", func() {
				resp, err := client.GetCommitChanges(ctx, userName, repoName, commits[2].Hash, &api.GetCommitChangesParams{
					Path: utils.String("/"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseCompareCommitResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 1)
			})
		})
	}
}
