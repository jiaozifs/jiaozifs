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
		refName := "feat/wip_test"
		refNameForDelete := "feat/wip_test2"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, userName)
		createRepo(ctx, c, client, repoName)
		createBranch(ctx, c, client, userName, repoName, "main", refName)
		createBranch(ctx, c, client, userName, repoName, "main", refNameForDelete)
		c.Convey("list non exit wip", func(c convey.C) {
			resp, err := client.ListWip(ctx, userName, repoName)
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

			respResult, err := api.ParseListWipResponse(resp)
			convey.So(err, convey.ShouldBeNil)
			convey.So(respResult.JSON200, convey.ShouldHaveLength, 0)
		})

		c.Convey("create wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.CreateWip(ctx, userName, repoName, &api.CreateWipParams{
					RefName: refName,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to create branch in non exit repo", func() {
				resp, err := client.CreateWip(ctx, userName, "fakerepo", &api.CreateWipParams{
					RefName: refName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create branch in non exit user", func() {
				resp, err := client.CreateWip(ctx, "mock_user", "main", &api.CreateWipParams{
					RefName: refName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create branch in non exit ref", func() {
				resp, err := client.CreateWip(ctx, userName, repoName, &api.CreateWipParams{
					RefName: "mock ref",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden create branch in others", func() {
				resp, err := client.CreateWip(ctx, "jimmy", "happygo", &api.CreateWipParams{
					RefName: "main",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("success create branch", func() {
				resp, err := client.CreateWip(ctx, userName, repoName, &api.CreateWipParams{
					RefName: refName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("user only have one wip for one userName", func() {
				resp, err := client.CreateWip(ctx, userName, repoName, &api.CreateWipParams{
					RefName: refName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})
		})

		createWip(ctx, c, client, userName, repoName, "main")

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

		c.Convey("get wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: refName,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("success get branch", func() {
				resp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{
					RefName: refName,
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
					RefName: refName,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get non exit branch", func() {
				resp, err := client.GetWip(ctx, userName, "mock_repo", &api.GetWipParams{
					RefName: refName,
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

		c.Convey("delete wip", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: refNameForDelete})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("delete non exit wip", func() {
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: refNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in not exit repo", func() {
				resp, err := client.DeleteWip(ctx, userName, "mock_repo", &api.DeleteWipParams{RefName: refNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in non exit user", func() {
				resp, err := client.DeleteWip(ctx, "telo", repoName, &api.DeleteWipParams{RefName: refNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("delete wip in other's repo", func() {
				resp, err := client.DeleteWip(ctx, "jimmy", "happygo", &api.DeleteWipParams{RefName: "main"})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			createWip(ctx, c, client, userName, repoName, refNameForDelete)
			c.Convey("delete branch successful", func() {
				//delete
				resp, err := client.DeleteWip(ctx, userName, repoName, &api.DeleteWipParams{RefName: refNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				//ensure delete work
				getResp, err := client.GetWip(ctx, userName, repoName, &api.GetWipParams{RefName: refNameForDelete})
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})
		})
	}
}

func createWip(ctx context.Context, c convey.C, client *api.Client, user string, repoName string, refName string) {
	c.Convey("create wip "+refName, func() {
		resp, err := client.CreateWip(ctx, user, repoName, &api.CreateWipParams{
			RefName: refName,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}