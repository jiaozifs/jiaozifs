package integrationtest

import (
	"context"
	"net/http"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/smartystreets/goconvey/convey"
)

func WipSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	return func(c convey.C) {
		userName := "july"
		repoName := "mlops"
		branchName := "feat/wip_test"
		branchNameForDelete := "feat/wip_test2"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "july login", userName, false)
		createRepo(ctx, c, client, repoName)
		createBranch(ctx, c, client, "create branch", userName, repoName, "main", branchName)
		createBranch(ctx, c, client, "create branch for delete", userName, repoName, "main", branchNameForDelete)
		c.Convey("list non exit wip", func(c convey.C) {
			resp, err := client.ListWip(ctx, userName, repoName)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

			respResult, err := api.ParseListWipResponse(resp)
			convey.So(err, convey.ShouldBeNil)
			convey.So(respResult.JSON200, convey.ShouldHaveLength, 0)
		})

		createWip(ctx, c, client, "create main wip", userName, repoName, "main")

		c.Convey("get wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: branchName,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("auto create a wip", func() {
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: branchName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)

				_, err = api.ParseGetWipResponse(resp)
				convey.So(err, convey.ShouldBeNil)
			})

			c.Convey("success get wip", func() {
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: branchName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				_, err = api.ParseGetWipResponse(resp)
				convey.So(err, convey.ShouldBeNil)
			})

			c.Convey("fail to get wip in non exit ref", func() {
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: "mock_ref",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get wip from non exit user", func() {
				resp, err := client.GetWip(ctx, "mock_owner", repoName, &api.GetWipParams{
					RefName: branchName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get non exit branch", func() {
				resp, err := client.GetWip(ctx, userName, "mock_repo", &api.GetWipParams{
					RefName: branchName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to others repo's wips", func() {
				resp, err := client.GetWip(ctx, "jimmy", "happygo", &api.GetWipParams{
					RefName: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("list wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.ListWip(ctx, userName, repoName)
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success list wips", func() {
				resp, err := client.ListWip(ctx, userName, repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				respResult, err := api.ParseListWipResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(respResult.JSON200, convey.ShouldHaveLength, 2)
			})

			c.Convey("fail to list wip from non exit user", func() {
				resp, err := client.ListWip(ctx, "mock_owner", repoName)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list wips in non exit branch", func() {
				resp, err := client.ListWip(ctx, userName, "mockrepo")
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list wip in others's repo", func() {
				resp, err := client.ListWip(ctx, "jimmy", "happygo")
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
		})

		c.Convey("delete wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: branchNameForDelete})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("delete non exit wip", func() {
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: branchNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in not exit repo", func() {
				resp, err := client.DeleteWip(ctx, userName, "mock_repo", &api.DeleteWipParams{RefName: branchNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in non exit user", func() {
				resp, err := client.DeleteWip(ctx, "telo", repoName, &api.DeleteWipParams{RefName: branchNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in other's repo", func() {
				resp, err := client.DeleteWip(ctx, "jimmy", "happygo", &api.DeleteWipParams{RefName: "main"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			createWip(ctx, c, client, "creat wip for test delete", userName, repoName, branchNameForDelete)
			c.Convey("delete branch successful", func() {
				//delete
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: branchNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				//ensure delete work
				getResp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{RefName: branchNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})
		})
	}
}
