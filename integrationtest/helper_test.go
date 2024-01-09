package integrationtest

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jiaozifs/jiaozifs/utils"

	"github.com/jiaozifs/jiaozifs/api"
	"github.com/smartystreets/goconvey/convey"

	"github.com/jiaozifs/jiaozifs/testhelper"

	"github.com/stretchr/testify/require"

	"github.com/jiaozifs/jiaozifs/cmd"
	"github.com/phayes/freeport"
)

func InitCmd(ctx context.Context, jzHome string, listen string, db string) error {
	buf := new(bytes.Buffer)
	cmd.RootCmd().SetOut(buf)
	cmd.RootCmd().SetErr(buf)
	cmd.RootCmd().SetArgs([]string{"init", "--listen", listen, "--db_debug", "true", "--db", db,
		"--config", fmt.Sprintf("%s/config.toml", jzHome), "--bs_path", fmt.Sprintf("%s/blockstore", jzHome)})

	return cmd.RootCmd().ExecuteContext(ctx)
}

func Daemon(ctx context.Context, writer io.Writer, jzHome string, listen string) error {
	cmd.RootCmd().SetOut(writer)
	cmd.RootCmd().SetErr(writer)
	cmd.RootCmd().SetArgs([]string{"daemon", "--listen", listen, "--config", fmt.Sprintf("%s/config.toml", jzHome)})
	return cmd.RootCmd().ExecuteContext(ctx)
}

func TestDoubleInit(t *testing.T) { //nolint
	url := "http://127.0.0.1:1234"
	ctx := context.Background()
	tmpDir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)
	require.NoError(t, InitCmd(ctx, tmpDir, url, ""))
	err = InitCmd(ctx, tmpDir, url, "")
	require.Error(t, err)
	require.Contains(t, err.Error(), "config already exit")
}

type Closer func()

func SetupDaemon(t *testing.T, ctx context.Context) (string, Closer) { //nolint
	closeDB, connectString, _ := testhelper.SetupDatabase(ctx, t)

	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	ctx, cancel := context.WithCancel(ctx)
	tmpDir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)
	require.NoError(t, InitCmd(ctx, tmpDir, url, connectString))
	buf := new(bytes.Buffer)

	closer := func() {
		cancel()
		require.NoError(t, os.RemoveAll(tmpDir))
		closeDB()
	}
	go func() {
		err := Daemon(ctx, buf, tmpDir, url)
		if err != nil && !errors.Is(err, context.Canceled) {
			require.NoError(t, err)
		}
	}()
	fmt.Println(connectString)

	//wai for api ready
	ticker := time.NewTicker(time.Second)
	tryCount := 0
	for {
		select {
		case <-ticker.C:
			readAll, err := io.ReadAll(buf)
			require.NoError(t, err)
			if strings.Contains(string(readAll), "") {
				return url, closer
			}
			tryCount++
			if tryCount > 5 {
				require.NoError(t, errors.New("timeout to wait api not ready"))
				return "", nil
			}
		case <-ctx.Done():
			closer()
			require.NoError(t, errors.New("context canceled"))
			return "", nil
		}
	}
}

func createUser(ctx context.Context, c convey.C, client *api.Client, userName string) {
	c.Convey("register "+userName, func() {
		resp, err := client.Register(ctx, api.RegisterJSONRequestBody{
			Name:     userName,
			Password: "12345678",
			Email:    "mock@gmail.com",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
	})
}

func loginAndSwitch(ctx context.Context, c convey.C, client *api.Client, title, userName string, useCookie bool) {
	c.Convey("login "+title, func() {
		resp, err := client.Login(ctx, api.LoginJSONRequestBody{
			Name:     userName,
			Password: "12345678",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
		loginResult, err := api.ParseLoginResponse(resp)
		convey.So(err, convey.ShouldBeNil)

		client.RequestEditors = nil
		client.RequestEditors = append(client.RequestEditors, func(ctx context.Context, req *http.Request) error {
			if useCookie {
				for _, cookie := range resp.Cookies() {
					req.AddCookie(cookie)
				}
			} else {
				req.Header.Add("Authorization", "Bearer "+loginResult.JSON200.Token)
			}
			return nil
		})
	})
}

func createBranch(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, source, refName string) {
	c.Convey("create branch "+title, func() {
		resp, err := client.CreateBranch(ctx, user, repoName, api.CreateBranchJSONRequestBody{
			Source: source,
			Name:   refName,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}

func createRepo(ctx context.Context, c convey.C, client *api.Client, repoName string) {
	c.Convey("create repo "+repoName, func() {
		resp, err := client.CreateRepository(ctx, api.CreateRepositoryJSONRequestBody{
			Name: repoName,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
	})
}

func uploadObject(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, refName string, path string) { //nolint
	c.Convey("upload object "+title, func(c convey.C) {
		resp, err := client.UploadObjectWithBody(ctx, user, repoName, &api.UploadObjectParams{
			RefName: refName,
			Path:    path,
		}, "application/octet-stream", io.LimitReader(rand.Reader, 50))
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}

func deleteObject(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, refName string, path string) { //nolint
	c.Convey("upload object  "+title, func(c convey.C) {
		c.Convey("success upload object", func() {
			resp, err := client.UploadObjectWithBody(ctx, user, repoName, &api.UploadObjectParams{
				RefName: refName,
				Path:    path,
			}, "application/octet-stream", io.LimitReader(rand.Reader, 50))
			convey.So(err, convey.ShouldBeNil)
			convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
		})
	})
}

func createWip(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, refName string) {
	c.Convey("create wip "+title, func() {
		resp, err := client.GetWip(ctx, user, repoName, &api.GetWipParams{
			RefName: refName,
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}

func commitWip(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, refName string, msg string) {
	c.Convey("commit wip "+title, func() {
		resp, err := client.CommitWip(ctx, user, repoName, &api.CommitWipParams{
			RefName: refName,
			Msg:     msg,
		})

		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}

func createMergeRequest(ctx context.Context, c convey.C, client *api.Client, title string, user string, repoName string, sourceBranch string, targetBranch string) {
	c.Convey("create mr "+title, func() {
		resp, err := client.CreateMergeRequest(ctx, user, repoName, api.CreateMergeRequestJSONRequestBody{
			Description:      utils.String("create merge request test"),
			SourceBranchName: sourceBranch,
			TargetBranchName: targetBranch,
			Title:            "Merge: test",
		})
		convey.So(err, convey.ShouldBeNil)
		convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
	})
}
