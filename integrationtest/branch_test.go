package integrationtest

import (
	"context"
	"net/http"
	"strings"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/jiaozifs/jiaozifs/controller"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/smartystreets/goconvey/convey"
)

func BranchSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	return func(c convey.C) {
		userName := "mike"
		repoName := "mlops"
		branchName := "feat/test"
		amount := 1
		newAmount := 0
		prefix := "feat/"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "mike login", userName, false)
		createRepo(ctx, c, client, repoName)

		c.Convey("create branch", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.CreateBranch(ctx, userName, repoName, api.CreateBranchJSONRequestBody{
					Name:   "feat/no_auth",
					Source: "main",
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success create branch", func() {
				resp, err := client.CreateBranch(ctx, userName, repoName, api.CreateBranchJSONRequestBody{
					Name:   branchName,
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("fail to create branch in non exit repo", func() {
				resp, err := client.CreateBranch(ctx, userName, "fakerepo", api.CreateBranchJSONRequestBody{
					Name:   "feat/test",
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("too long name", func() {
				resp, err := client.CreateBranch(ctx, userName, repoName, api.CreateBranchJSONRequestBody{
					Name:   "feat/" + strings.Repeat("a", controller.MaxBranchNameLength),
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("invalid format", func() {
				resp, err := client.CreateBranch(ctx, userName, repoName, api.CreateBranchJSONRequestBody{
					Name:   "feat/aaaa/aaa",
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("invalid char", func() {
				resp, err := client.CreateBranch(ctx, userName, repoName, api.CreateBranchJSONRequestBody{
					Name:   "feat&*",
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("forbidden create branch in others", func() {
				resp, err := client.CreateBranch(ctx, "jimmy", "happygo", api.CreateBranchJSONRequestBody{
					Name:   "feat/test",
					Source: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("get branch", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{
					RefName: branchName,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success get branch", func() {
				resp, err := client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{
					RefName: branchName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseGetBranchResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200.Name, convey.ShouldEqual, branchName)
			})

			c.Convey("fail to get non exit ref", func() {
				resp, err := client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{
					RefName: "mock_ref",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get ref from non exit user", func() {
				resp, err := client.GetBranch(ctx, "mock_owner", repoName, &api.GetBranchParams{
					RefName: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get non exit branch", func() {
				resp, err := client.GetBranch(ctx, userName, "mock_repo", &api.GetBranchParams{
					RefName: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to others branch", func() {
				resp, err := client.GetBranch(ctx, "jimmy", "happygo", &api.GetBranchParams{
					RefName: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		createBranch(ctx, c, client, "create sec branch", userName, repoName, "main", "feat/sec_branch")

		c.Convey("list branch", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.ListBranches(ctx, userName, repoName, &api.ListBranchesParams{})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success list branch", func() {
				resp, err := client.ListBranches(ctx, userName, repoName, &api.ListBranchesParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseListBranchesResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200.Results, convey.ShouldHaveLength, 3)
			})

			c.Convey("success list branch by prefix", func() {
				resp, err := client.ListBranches(ctx, userName, repoName, &api.ListBranchesParams{
					Prefix: &prefix,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseListBranchesResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200.Results, convey.ShouldHaveLength, 2)
			})

			c.Convey("success list branch and next page exists", func() {
				resp, err := client.ListBranches(ctx, userName, repoName, &api.ListBranchesParams{
					After:  utils.String(branchName),
					Amount: utils.Int(amount),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseListBranchesResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200.Pagination.HasMore, convey.ShouldBeTrue)
				convey.So(respResult.JSON200.Results, convey.ShouldHaveLength, 1)
			})

			c.Convey("success list branch, set amount 0", func() {
				resp, err := client.ListBranches(ctx, userName, repoName, &api.ListBranchesParams{
					Amount: utils.Int(newAmount),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseListBranchesResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200.Results, convey.ShouldHaveLength, 3)
			})

			c.Convey("fail to list branchs from non exit user", func() {
				resp, err := client.ListBranches(ctx, "mock_owner", repoName, &api.ListBranchesParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list branchs in non exit repo", func() {
				resp, err := client.ListBranches(ctx, userName, "mockrepo", &api.ListBranchesParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list branches in others repo", func() {
				resp, err := client.ListBranches(ctx, "jimmy", "happygo", &api.ListBranchesParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("delete branch", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.DeleteBranch(ctx, userName, repoName, &api.DeleteBranchParams{RefName: "feat/sec_branch"})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})
			c.Convey("delete branch in not exit repo", func() {
				resp, err := client.DeleteBranch(ctx, userName, repoName, &api.DeleteBranchParams{RefName: "feat/third_branch"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete branch in non exit user", func() {
				resp, err := client.DeleteBranch(ctx, "telo", repoName, &api.DeleteBranchParams{RefName: "feat/sec_branch"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete branch in other's repo", func() {
				resp, err := client.DeleteBranch(ctx, "jimmy", "happygo", &api.DeleteBranchParams{RefName: "main"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("delete branch successful", func() {
				resp, err := client.DeleteBranch(ctx, userName, repoName, &api.DeleteBranchParams{RefName: "feat/sec_branch"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResp, err := client.GetBranch(ctx, userName, repoName, &api.GetBranchParams{RefName: "feat/sec_branch"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})
		})

	}
}
