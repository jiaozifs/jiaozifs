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
	"sync/atomic"
	"testing"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/jiaozifs/jiaozifs/api"
	"github.com/jiaozifs/jiaozifs/cmd"
	"github.com/jiaozifs/jiaozifs/testhelper"
	"github.com/jiaozifs/jiaozifs/utils"
	"github.com/phayes/freeport"
	"github.com/smartystreets/goconvey/convey"
	"github.com/stretchr/testify/require"
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
	ctx := context.Background()
	closeDB, connectString, _ := testhelper.SetupDatabase(ctx, t)
	defer closeDB()

	url := "http://127.0.0.1:1234"
	tmpDir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)
	require.NoError(t, InitCmd(ctx, tmpDir, url, connectString))
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

var count atomic.Int32

func createUser(ctx context.Context, client *api.Client, userName string) {
	resp, err := client.Register(ctx, api.RegisterJSONRequestBody{
		Name:     userName,
		Password: "12345678",
		Email:    openapi_types.Email(fmt.Sprintf("mock%d@gmail.com", count.Add(1))),
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func loginAndSwitch(ctx context.Context, client *api.Client, userName string, useCookie bool) {
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
}

func createBranch(ctx context.Context, client *api.Client, user string, repoName string, source, refName string) {
	resp, err := client.CreateBranch(ctx, user, repoName, api.CreateBranchJSONRequestBody{
		Source: source,
		Name:   refName,
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func createRepo(ctx context.Context, client *api.Client, repoName string) {
	resp, err := client.CreateRepository(ctx, api.CreateRepositoryJSONRequestBody{
		Name: repoName,
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func uploadObject(ctx context.Context, client *api.Client, user string, repoName string, refName string, path string) { //nolint
	resp, err := client.UploadObjectWithBody(ctx, user, repoName, &api.UploadObjectParams{
		RefName: refName,
		Path:    path,
	}, "application/octet-stream", io.LimitReader(rand.Reader, 50))
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func deleteObject(ctx context.Context, client *api.Client, user string, repoName string, refName string, path string) { //nolint
	resp, err := client.DeleteObject(ctx, user, repoName, &api.DeleteObjectParams{
		RefName: refName,
		Path:    path,
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusOK)
}

func createWip(ctx context.Context, client *api.Client, user string, repoName string, refName string) {
	resp, err := client.GetWip(ctx, user, repoName, &api.GetWipParams{
		RefName: refName,
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func commitWip(ctx context.Context, client *api.Client, user string, repoName string, refName string, msg string) {
	resp, err := client.CommitWip(ctx, user, repoName, &api.CommitWipParams{
		RefName: refName,
		Msg:     msg,
	})

	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func createMergeRequest(ctx context.Context, client *api.Client, user string, repoName string, sourceBranch string, targetBranch string) {
	resp, err := client.CreateMergeRequest(ctx, user, repoName, api.CreateMergeRequestJSONRequestBody{
		Description:      utils.String("create merge request test"),
		SourceBranchName: sourceBranch,
		TargetBranchName: targetBranch,
		Title:            "Merge: test",
	})
	convey.So(err, convey.ShouldBeNil)
	convey.So(resp.StatusCode, convey.ShouldEqual, http.StatusCreated)
}

func createAksk(ctx context.Context, client *api.Client) (*api.Aksk, error) {
	resp, err := client.CreateAksk(ctx, &api.CreateAkskParams{Description: utils.String("create ak sk")})
	if err != nil {
		return nil, err
	}

	akskResult, err := api.ParseCreateAkskResponse(resp)
	if err != nil {
		return nil, err
	}
	return akskResult.JSON201, nil
}
