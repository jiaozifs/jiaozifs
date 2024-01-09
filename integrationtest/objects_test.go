package integrationtest

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"

	"github.com/jiaozifs/jiaozifs/api"
	apiimpl "github.com/jiaozifs/jiaozifs/api/api_impl"
	"github.com/jiaozifs/jiaozifs/utils/hash"
	"github.com/smartystreets/goconvey/convey"
)

func ObjectSpec(ctx context.Context, urlStr string) func(c convey.C) {
	client, _ := api.NewClient(urlStr + apiimpl.APIV1Prefix)
	return func(c convey.C) {
		userName := "molly"
		repoName := "dataspace"
		branchName := "feat/obj_test"

		createUser(ctx, c, client, userName)
		loginAndSwitch(ctx, c, client, "molly login", userName, false)
		createRepo(ctx, c, client, repoName)
		createBranch(ctx, c, client, "create branch", userName, repoName, "main", branchName)
		createWip(ctx, c, client, "feat get obj test", userName, repoName, branchName)

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
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
			})

			c.Convey("empty path", func() {
				resp, err := client.UploadObjectWithBody(ctx, userName, repoName, &api.UploadObjectParams{
					RefName: branchName,
				}, "application/octet-stream", bytes.NewReader([]byte{1, 2, 3, 4, 5, 6, 7, 8}))
				convey.So(err, convey.ShouldBeNil)
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusBadRequest)
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
		})

		//commit object to branch
		commitWip(ctx, c, client, "commit wip", userName, repoName, branchName, "test commit msg")

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
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
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
				convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusForbidden)
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
				data, err := io.ReadAll(reader)
				fmt.Println(data)
				convey.So(err, convey.ShouldBeNil)
				etag := resp.Header.Get("ETag")

				exectEtag := fmt.Sprintf(`"%s"`, hex.EncodeToString(reader.Md5.Sum(nil)))
				convey.So(etag, convey.ShouldEqual, exectEtag)
			})
		})
	}
}
