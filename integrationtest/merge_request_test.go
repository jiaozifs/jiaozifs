package integrationtest

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/jiaozifs/jiaozifs/models"

	"github.com/jiaozifs/jiaozifs/utils"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/smartystreets/goconvey/convey"
)

func MergeRequestSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	var firstMrID *uint64
	return func(c convey.C) {
		userName := "navi"
		repoName := "mr_test"
		branchName := "feat/obj_test"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "molly login", userName, false)
		createRepo(ctx, c, client, repoName)
		createBranch(ctx, c, client, "create branch", userName, repoName, "main", branchName)
		createWip(ctx, c, client, "feat get obj test", userName, repoName, branchName)
		uploadObject(ctx, c, client, "update f1 to test branch", userName, repoName, branchName, "a.bin")
		commitWip(ctx, c, client, "commit object", userName, repoName, branchName, "test")

		c.Convey("create merge request", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to create mergerequest in non exit repo", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, "fakerepo", api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create mergerequest from non exit user", func() {
				resp, err := client.CreateMergeRequest(ctx, "mock_user", repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create mergerequest from non exit repo", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, "fakerepo", api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create mergerequest from others user", func() {
				resp, err := client.CreateMergeRequest(ctx, "jimmy", "happygo", api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("fail to create mergerequest from non exit branch", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: "fakeb1",
					TargetBranchName: "fakeb2",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create mergerequest from non exit branch 2", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "fakeb2",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to create mergerequest from same source and target branch", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: branchName,
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("success to create mergerequest", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)

				createResp, err := api.ParseCreateMergeRequestResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				firstMrID = &createResp.JSON201.Sequence
			})
			c.Convey("fail to create dup mergerequest", func() {
				resp, err := client.CreateMergeRequest(ctx, userName, repoName, api.CreateMergeRequestJSONRequestBody{
					Description:      utils.String("create merge request test"),
					SourceBranchName: branchName,
					TargetBranchName: "main",
					Title:            "Merge: test",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})
		})

		c.Convey("get merge request", func(c convey.C) {

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetMergeRequest(ctx, userName, repoName, *firstMrID)
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get merge request in non exit repo", func() {
				resp, err := client.GetMergeRequest(ctx, userName, "fakerepo", *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get merge request from non exit user", func() {
				resp, err := client.GetMergeRequest(ctx, "mock_user", repoName, *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get merge request from others user", func() {
				resp, err := client.GetMergeRequest(ctx, "jimmy", "happygo", *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("fail to get merge request with non exit mergerequest", func() {
				resp, err := client.GetMergeRequest(ctx, userName, repoName, 1000)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("success to get merge request", func() {
				resp, err := client.GetMergeRequest(ctx, userName, repoName, *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
			})
		})

		c.Convey("update merge request", func(c convey.C) {

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.UpdateMergeRequest(ctx, userName, repoName, *firstMrID, api.UpdateMergeRequestJSONRequestBody{
					Description: utils.String("update merge request test"),
					Title:       utils.String("Merge: test title"),
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to update merge request in non exit repo", func() {
				resp, err := client.UpdateMergeRequest(ctx, userName, "fakerepo", *firstMrID, api.UpdateMergeRequestJSONRequestBody{
					Description: utils.String("update merge request test"),
					Title:       utils.String("Merge: test title"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to update merge request from non exit user", func() {
				resp, err := client.UpdateMergeRequest(ctx, "mockuser", repoName, *firstMrID, api.UpdateMergeRequestJSONRequestBody{
					Description: utils.String("update merge request test"),
					Title:       utils.String("Merge: test title"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to update merge request from others user", func() {
				resp, err := client.UpdateMergeRequest(ctx, "jimmy", "happygo", *firstMrID, api.UpdateMergeRequestJSONRequestBody{
					Description: utils.String("update merge request test"),
					Title:       utils.String("Merge: test title"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("success to update merge request", func() {
				resp, err := client.UpdateMergeRequest(ctx, userName, repoName, *firstMrID, api.UpdateMergeRequestJSONRequestBody{
					Description: utils.String("update merge request test"),
					Title:       utils.String("Merge: test title"),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResp, err := client.GetMergeRequest(ctx, userName, repoName, *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				updatedResult, err := api.ParseGetMergeRequestResponse(getResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So("Merge: test title", convey.ShouldEqual, (*updatedResult.JSON200).Title)
				convey.So("update merge request test", convey.ShouldEqual, *(*updatedResult.JSON200).Description)
			})
		})

		for i := 0; i < 10; i++ {
			branchName := fmt.Sprintf("feat/list_merge_test_%d", i)
			createBranch(ctx, c, client, fmt.Sprintf("create branch %d", i), userName, repoName, "main", branchName)
			createWip(ctx, c, client, "feat list merge test "+strconv.Itoa(i), userName, repoName, branchName)
			uploadObject(ctx, c, client, "update f1 to test branch "+strconv.Itoa(i), userName, repoName, branchName, fmt.Sprintf("%d.txt", i))
			commitWip(ctx, c, client, "commit object "+strconv.Itoa(i), userName, repoName, branchName, "test")
			createMergeRequest(ctx, c, client, "create  merge request "+strconv.Itoa(i), userName, repoName, branchName, "main")
		}

		c.Convey("list merge request", func(c convey.C) {

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.ListMergeRequests(ctx, userName, repoName, &api.ListMergeRequestsParams{})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to list merge request in non exit repo", func() {
				resp, err := client.ListMergeRequests(ctx, userName, "fakerepo", &api.ListMergeRequestsParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list merge request from non exit user", func() {
				resp, err := client.ListMergeRequests(ctx, "mockuser", repoName, &api.ListMergeRequestsParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to list merge request from others user", func() {
				resp, err := client.ListMergeRequests(ctx, "jimmy", "happygo", &api.ListMergeRequestsParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("success to list merge request", func() {
				resp, err := client.ListMergeRequests(ctx, userName, repoName, &api.ListMergeRequestsParams{})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseListMergeRequestsResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.ShouldHaveLength(*result.JSON200, 11)
			})

			c.Convey("success to list merge request over max page", func() {
				resp, err := client.ListMergeRequests(ctx, userName, repoName, &api.ListMergeRequestsParams{
					Amount: utils.Int(-1),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseListMergeRequestsResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.ShouldHaveLength(*result.JSON200, 11)
			})

			c.Convey("success to list by state merge request", func() {
				resp, err := client.ListMergeRequests(ctx, userName, repoName, &api.ListMergeRequestsParams{
					State: utils.Int(int(models.MergeStateClosed)),
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseListMergeRequestsResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.ShouldHaveLength(*result.JSON200, 0)
			})

			c.Convey("success to list page merge quest", func() {
				var after *time.Time
				for i := 0; i < 6; i++ {
					resp, err := client.ListMergeRequests(ctx, userName, repoName, &api.ListMergeRequestsParams{
						After:  after,
						Amount: utils.Int(2),
					})
					convey.So(err, convey.ShouldBeNil)
					convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

					result, err := api.ParseListMergeRequestsResponse(resp)
					convey.So(err, convey.ShouldBeNil)
					convey.ShouldHaveLength(*result.JSON200, 2)
					if i >= 5 {
						convey.ShouldBeFalse((*result.JSON200).Pagination.HasMore)
					} else {
						convey.ShouldBeTrue((*result.JSON200).Pagination.HasMore)
						next, err := time.Parse(`2006-01-02 15:04:05.999999999 -0700 MST`, (*result.JSON200).Pagination.NextOffset)
						convey.So(err, convey.ShouldBeNil)
						after = &next
					}
				}
			})
		})

		c.Convey(" merge request", func(c convey.C) {

			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.Merge(ctx, userName, repoName, *firstMrID, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to update merge request in non exit repo", func() {
				resp, err := client.Merge(ctx, userName, "fakerepo", *firstMrID, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to update merge request from non exit user", func() {
				resp, err := client.Merge(ctx, "mockuser", repoName, *firstMrID, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to update merge request from others user", func() {
				resp, err := client.Merge(ctx, "jimmy", "happygo", *firstMrID, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})
			c.Convey("fail to update merge request from non exit mr", func() {
				resp, err := client.Merge(ctx, userName, repoName, 100, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("success to update merge request", func() {
				resp, err := client.Merge(ctx, userName, repoName, *firstMrID, api.MergeJSONRequestBody{
					Msg: "test merge",
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				getResp, err := client.GetMergeRequest(ctx, userName, repoName, *firstMrID)
				convey.So(err, convey.ShouldBeNil)
				convey.So(getResp.StatusCode, convey.ShouldEqual, http.StatusOK)

				updatedResult, err := api.ParseGetMergeRequestResponse(getResp)
				convey.So(err, convey.ShouldBeNil)
				convey.So(int(models.MergeStateMerged), convey.ShouldEqual, (*updatedResult.JSON200).MergeStatus)
			})
		})
	}
}
