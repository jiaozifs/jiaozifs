package integrationtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/GitDataAI/jiaozifs/utils"

	"github.com/GitDataAI/jiaozifs/api"
	apiimpl "github.com/GitDataAI/jiaozifs/api/api_impl"
	"github.com/GitDataAI/jiaozifs/utils/hash"
	"github.com/smartystreets/goconvey/convey"
)

func ObjectSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	return func(c convey.C) {
		userName := "molly"
		repoName := "dataspace"
		branchName := "feat/obj_test"

		c.Convey("init", func(_ convey.C) {
			_ = createUser(ctx, client, userName)
			loginAndSwitch(ctx, client, userName, false)
			_ = createRepo(ctx, client, repoName, false)
			_ = createBranch(ctx, client, userName, repoName, "main", branchName)
			_ = createWip(ctx, client, userName, repoName, branchName)
		})

		c.Convey("upload object", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to create branch in non exit user", func() {
				resp, err := client.UploadObjectWithBody(ctx, "mockuser", "main", &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to upload in non exit repo", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, "fakerepo", &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to upload object in non exit branch", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: "mockref",
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to upload object in no wip ", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: "main",
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden upload object in others", func() {
				resp, err := client.UploadObjectWithBody(ctx, "jimmy", "happygo", &api.UploadObjectParams{
					RefName: "main",
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("empty path", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("no sufix file", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "aaa",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 9}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("success upload object", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("success upload object on subpath", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a/b.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 1, 1, 1}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("success to upload the same resource", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a/b.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 1, 1, 1}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})

			c.Convey("fail to upload difference content to exit path without tell serve to replace", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
					Path:    "a/b.bin",
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 1, 1, 2}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusConflict)
			})
			c.Convey("success to upload difference content to exit path when tell serve to replace", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName:   branchName,
					Path:      "a/b.bin",
					IsReplace: utils.Bool(true),
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 1, 1, 1}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
			})
		})

		//commit object to branch
		c.Convey("commit object to branch", func(_ convey.C) {
			_ = commitWip(ctx, client, userName, repoName, branchName, "test commit msg")
		})

		c.Convey("head object", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.HeadObject(ctx, userName, repoName, &api.HeadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to head object in non exit user", func() {
				resp, err := client.HeadObject(ctx, "mock user", repoName, &api.HeadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to head object in non exit repo", func() {
				resp, err := client.HeadObject(ctx, userName, "fakerepo", &api.HeadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to head object in non exit branch", func() {
				resp, err := client.HeadObject(ctx, userName, repoName, &api.HeadObjectParams{
					RefName: "mockref",
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden head object in others", func() {
				resp, err := client.HeadObject(ctx, "jimmy", "happygo", &api.HeadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("empty path", func() {
				resp, err := client.HeadObject(ctx, userName, repoName, &api.HeadObjectParams{
					RefName: branchName,
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("not exit path", func() {
				resp, err := client.HeadObject(ctx, userName, repoName, &api.HeadObjectParams{
					RefName: branchName,
					Path:    "c/d.txt",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("success to head object", func() {
				resp, err := client.HeadObject(ctx, userName, repoName, &api.HeadObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
				etag := resp.Header.Get("ETag")
				convey.So(etag, convey.ShouldEqual, `"0ee0646c1c77d8131cc8f4ee65c7673b"`)
			})
		})

		c.Convey("get object", func(c convey.C) {
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetObject(ctx, userName, repoName, &api.GetObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get object in non exit user", func() {
				resp, err := client.GetObject(ctx, "mock user", repoName, &api.GetObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get object in non exit repo", func() {
				resp, err := client.GetObject(ctx, userName, "fakerepo", &api.GetObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get object in non exit branch", func() {
				resp, err := client.GetObject(ctx, userName, repoName, &api.GetObjectParams{
					RefName: "mockref",
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden get object in others", func() {
				resp, err := client.GetObject(ctx, "jimmy", "happygo", &api.GetObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("empty path", func() {
				resp, err := client.GetObject(ctx, userName, repoName, &api.GetObjectParams{
					RefName: branchName,
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("not exit path", func() {
				resp, err := client.GetObject(ctx, userName, repoName, &api.GetObjectParams{
					RefName: branchName,
					Path:    "c/d.txt",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
			})

			c.Convey("success to get object", func() {
				resp, err := client.GetObject(ctx, userName, repoName, &api.GetObjectParams{
					RefName: branchName,
					Path:    "a.bin",
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				reader := hash.NewHashingReader(resp.Body, hash.Md5)
				_, err = io.ReadAll(reader)
				convey.So(err, convey.ShouldBeNil)
				etag := resp.Header.Get("ETag")

				exectEtag := fmt.Sprintf(`"%s"`, hex.EncodeToString(reader.Md5.Sum(nil)))
				convey.So(etag, convey.ShouldEqual, exectEtag)
			})
		})

		c.Convey("get files", func(c convey.C) {
			repoName := "testGetFiles"
			branchName := "ggct"
			c.Convey("init", func() {
				_ = createRepo(ctx, client, repoName, false)
				_ = createBranch(ctx, client, userName, repoName, "main", branchName)
			})
			c.Convey("no auth", func() {
				re := client.RequestEditors
				client.RequestEditors = nil
				resp, err := client.GetFiles(ctx, userName, repoName, &api.GetFilesParams{
					RefName: "main",
					Pattern: utils.String("*"),
					Type:    api.RefTypeBranch,
				})
				client.RequestEditors = re
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("fail to get object in non exit user", func() {
				resp, err := client.GetFiles(ctx, "fakeuser", repoName, &api.GetFilesParams{
					RefName: "main",
					Pattern: utils.String("*"),
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get object in non exit repo", func() {
				resp, err := client.GetFiles(ctx, userName, "fakerepo", &api.GetFilesParams{
					RefName: "main",
					Pattern: utils.String("*"),
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("fail to get object in non exit branch", func() {
				resp, err := client.GetFiles(ctx, userName, repoName, &api.GetFilesParams{
					RefName: "main_bak",
					Pattern: utils.String("*"),
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusNotFound)
			})

			c.Convey("forbidden get object in others", func() {
				resp, err := client.GetFiles(ctx, "jimmy", "happygo", &api.GetFilesParams{
					RefName: "main",
					Pattern: utils.String("*"),
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusUnauthorized)
			})

			c.Convey("list success", func() {
				_ = createWip(ctx, client, userName, repoName, branchName)
				_ = uploadObject(ctx, client, userName, repoName, branchName, "a/b.txt", true)
				_ = uploadObject(ctx, client, userName, repoName, branchName, "a/e.txt", true)
				_ = uploadObject(ctx, client, userName, repoName, branchName, "a/g.txt", true)
				_ = commitWip(ctx, client, userName, repoName, branchName, "wip")

				resp, err := client.GetFiles(ctx, userName, repoName, &api.GetFilesParams{
					RefName: branchName,
					Pattern: utils.String("a/*"),
					Type:    api.RefTypeBranch,
				})
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)

				result, err := api.ParseGetFilesResponse(resp)
				convey.So(err, convey.ShouldBeNil)
				convey.ShouldHaveLength(3, *result.JSON200)
				convey.ShouldEqual("a/b.txt", (*result.JSON200)[0])
				convey.ShouldEqual("a/e.txt", (*result.JSON200)[1])
				convey.ShouldEqual("a/g.txt", (*result.JSON200)[2])
			})

		})
	}
}
