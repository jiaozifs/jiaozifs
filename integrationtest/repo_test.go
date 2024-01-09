package integrationtest

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/jiaozifs/jiaozifs/controller"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/smartystreets/goconvey/convey"
)

func RepoSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	return func(c convey.C) {
		userName := "jimmy"
		repoName := "happyrun"
		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "jimmy login", userName, false)

		c.Convey("create repo", func(c convey.C) {
			c.Convey("forbidden create repo name", func() {
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        "repo",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("config error", func() {
				cfg := `{"Type":"local",DefaultNamespacePrefix":null,"Local":{"Path":"~/.jiaozifs/blockstore","ImportEnabled":false,"ImportHidden":false,"AllowedExternalPrefixes":null},"S3":null,"Azure":null,"GS":null}`
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description:      utils.String("test resp"),
					Name:             "happygo",
					BlockstoreConfig: utils.String(cfg),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("local not support", func() {
				cfg := `{"Type":"local","DefaultNamespacePrefix":null,"Local":{"Path":"~/.jiaozifs/blockstore","ImportEnabled":false,"ImportHidden":false,"AllowedExternalPrefixes":null},"S3":null,"Azure":null,"GS":null}`
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description:      utils.String("test resp"),
					Name:             "happygo",
					BlockstoreConfig: utils.String(cfg),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("success create repo name", func() {
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        repoName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				grp, err := api.ParseGetRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(grp.JSON200.Head, convey.ShouldEqual, controller.DefaultBranchName)
				fmt.Println(grp.JSON200.Id)
				//check default branch created
				branchResp, err := client.GetBranch(ctx, userName, grp.JSON200.Name, &api.GetBranchParams{RefName: controller.DefaultBranchName})
				convey.So(err, convey.ShouldBeNil)
				convey.So(branchResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				brp, err := api.ParseGetBranchResponse(branchResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(brp.JSON200.Name, convey.ShouldEqual, controller.DefaultBranchName)
			})

			c.Convey("add second repo ", func() {
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        "happygo",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
			})

			c.Convey("duplicate repo name", func() {
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        repoName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusInternalServerError)
			})

			c.Convey("invalid repo name", func() {
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        "happyrun1@#%",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.CreateRepository(ctx, api.CreateRepository{
					Description: utils.String("test resp"),
					Name:        "happyrun2",
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})
		})

		c.Convey("list repository", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.ListRepositoryOfAuthenticatedUser(ctx, &api.ListRepositoryOfAuthenticatedUserParams{})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("list repository in authenticated user", func() {
				resp, err := client.ListRepositoryOfAuthenticatedUser(ctx, &api.ListRepositoryOfAuthenticatedUserParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)
			})

			c.Convey("success list repository of authenticatedUser and next page exists", func() {
				resp, err := client.ListRepositoryOfAuthenticatedUser(ctx, &api.ListRepositoryOfAuthenticatedUserParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)

				newResp, err := client.ListRepositoryOfAuthenticatedUser(ctx, &api.ListRepositoryOfAuthenticatedUserParams{
					After:  utils.Time(listRepos.JSON200.Results[0].UpdatedAt),
					Amount: utils.Int(1),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(newResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				newListRepos, err := api.ParseListRepositoryResponse(newResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(newListRepos.JSON200.Pagination.HasMore, convey.ShouldBeTrue)
				convey.So(len(newListRepos.JSON200.Results), convey.ShouldEqual, 1)
			})

			c.Convey("success list repository of authenticatedUser, set page amount 0", func() {
				resp, err := client.ListRepositoryOfAuthenticatedUser(ctx, &api.ListRepositoryOfAuthenticatedUserParams{
					Amount: utils.Int(0),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)
			})

			c.Convey("list repository", func() {
				resp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)
			})

			c.Convey("list repository by prefix", func() {
				resp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{Prefix: utils.String("happy")})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)
			})

			c.Convey("success list repository and next page exists", func() {
				resp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)

				newResp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{
					After:  utils.Time(listRepos.JSON200.Results[0].UpdatedAt),
					Amount: utils.Int(1),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(newResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				newListRepos, err := api.ParseListRepositoryResponse(newResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(newListRepos.JSON200.Pagination.HasMore, convey.ShouldBeTrue)
				convey.So(len(newListRepos.JSON200.Results), convey.ShouldEqual, 1)
			})

			c.Convey("success list repository, set page amount 0", func() {
				resp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{
					Amount: utils.Int(0),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 2)
			})

			c.Convey("list repository by prefix but found nothing", func() {
				resp, err := client.ListRepository(ctx, userName, &api.ListRepositoryParams{Prefix: utils.String("bad")})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				listRepos, err := api.ParseListRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)

				convey.So(len(listRepos.JSON200.Results), convey.ShouldEqual, 0)
			})

			c.Convey("list others repository", func() {
				resp, err := client.ListRepository(ctx, "admin", &api.ListRepositoryParams{Prefix: utils.String("bad")})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("get repository", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetRepository(ctx, userName, repoName)
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("get repository", func() {
				resp, err := client.GetRepository(ctx, userName, repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResult, err := api.ParseGetRepositoryResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResult.JSON200.Name, convey.ShouldEqual, repoName)
			})

			c.Convey("get not exit repo", func() {
				resp, err := client.GetRepository(ctx, userName, "happyrun_mock")
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("get from non exit user", func() {
				resp, err := client.GetRepository(ctx, "telo", repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("get other's repo", func() {
				resp, err := client.GetRepository(ctx, "admin", repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("update repository", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.UpdateRepository(ctx, userName, repoName, api.UpdateRepositoryJSONRequestBody{
					Description: utils.String(""),
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success update repository", func() {
				description := "mock description"
				resp, err := client.UpdateRepository(ctx, userName, repoName, api.UpdateRepositoryJSONRequestBody{
					Description: utils.String(description),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResp, err := client.GetRepository(ctx, userName, repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResult, err := api.ParseGetRepositoryResponse(getResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*getResult.JSON200.Description, convey.ShouldEqual, description)
			})

			c.Convey("update repository in not exit repo", func() {
				description := ""
				resp, err := client.UpdateRepository(ctx, userName, "happyrunfake", api.UpdateRepositoryJSONRequestBody{
					Description: utils.String(description),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("update repository in non exit user", func() {
				description := ""
				resp, err := client.UpdateRepository(ctx, "telo", repoName, api.UpdateRepositoryJSONRequestBody{
					Description: utils.String(description),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("update repository in other's repo", func() {
				description := ""
				resp, err := client.UpdateRepository(ctx, "admin", repoName, api.UpdateRepositoryJSONRequestBody{
					Description: utils.String(description),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("update head to not exit", func() {
				resp, err := client.UpdateRepository(ctx, userName, repoName, api.UpdateRepositoryJSONRequestBody{
					Head: utils.String("xxx"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			createBranch(ctx, c, client, "create branch", userName, repoName, "main", "feat/ano_branch")
			c.Convey("update default head success", func() {
				resp, err := client.UpdateRepository(ctx, userName, repoName, api.UpdateRepositoryJSONRequestBody{
					Head: utils.String("feat/ano_branch"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
			})
		})

		c.Convey("get commits in repository", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})
			c.Convey("update repository in not exit repo", func() {
				resp, err := client.GetCommitsInRef(ctx, userName, "happyrunfake", &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("update repository in non exit user", func() {
				resp, err := client.GetCommitsInRef(ctx, "telo", repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("update repository in other's repo", func() {
				resp, err := client.GetCommitsInRef(ctx, "admin", repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)

			})

			createWip(ctx, c, client, "add commit to random branch", userName, repoName, controller.DefaultBranchName)
			uploadObject(ctx, c, client, "add rand object", userName, repoName, controller.DefaultBranchName, "a.txt")
			commitWip(ctx, c, client, "commit object", userName, repoName, controller.DefaultBranchName, "first commit")
			c.Convey("success get commits", func() {
				resp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetCommitsInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 1)
				convey.So((*result.JSON200)[0].Message, convey.ShouldEqual, "first commit")
			})

			uploadObject(ctx, c, client, "add sec object", userName, repoName, controller.DefaultBranchName, "b.txt")
			commitWip(ctx, c, client, "commit sec object", userName, repoName, controller.DefaultBranchName, "second commit")
			uploadObject(ctx, c, client, "add third object", userName, repoName, controller.DefaultBranchName, "c.txt")
			commitWip(ctx, c, client, "commit third object", userName, repoName, controller.DefaultBranchName, "third commit")
			c.Convey("success get commits by params", func() {
				resp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetCommitsInRefResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*result.JSON200, convey.ShouldHaveLength, 3)
				convey.So((*result.JSON200)[0].Message, convey.ShouldEqual, "third commit")

				newResp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					After:   utils.String((*result.JSON200)[0].Committer.When.Format(time.RFC3339Nano)),
					Amount:  utils.Int(1),
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				newResult, err := api.ParseGetCommitsInRefResponse(newResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(*newResult.JSON200, convey.ShouldHaveLength, 1)
				convey.So((*newResult.JSON200)[0].Message, convey.ShouldEqual, "second commit")
			})

			c.Convey("failed get commits by wrong params", func() {
				resp, err := client.GetCommitsInRef(ctx, userName, repoName, &api.GetCommitsInRefParams{
					After:   utils.String("123"),
					RefName: utils.String(controller.DefaultBranchName),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusInternalServerError)
			})
		})

		c.Convey("delete repository", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.DeleteRepository(ctx, userName, repoName)
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})
			c.Convey("delete repository in not exit repo", func() {
				resp, err := client.DeleteRepository(ctx, userName, "happyrunfake")
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete repository in non exit user", func() {
				resp, err := client.DeleteRepository(ctx, "telo", repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete repository in other's repo", func() {
				resp, err := client.DeleteRepository(ctx, "admin", repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("delete repository successful", func() {
				resp, err := client.DeleteRepository(ctx, userName, repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResp, err := client.GetRepository(ctx, userName, repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})
		})
	}
}
