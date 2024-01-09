package testhelper

import (
	"context"
	"fmt"
	"os"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	"github.com/jiaozifs/jiaozifs/config"
	"github.com/jiaozifs/jiaozifs/models"
	"github.com/jiaozifs/jiaozifs/models/migrations"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
	"go.uber.org/fx/fxtest"
)

var TestConnTmpl = "postgres://postgres:postgres@localhost:%d/jiaozifs?sslmode=disable"

type CloseFunc func()

func SetupDatabase(ctx context.Context, t *testing.T) (CloseFunc, string, *bun.DB) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	tmpDir, err := os.MkdirTemp(os.TempDir(), "*")
	require.NoError(t, err)

	cfg := embeddedpostgres.DefaultConfig().
		RuntimePath(tmpDir).
		Port(uint32(port)).
		Database("jiaozifs")

	postgres := embeddedpostgres.NewDatabase(cfg)
	err = postgres.Start()
	require.NoError(t, err)

	connStr := fmt.Sprintf(TestConnTmpl, port)
	db, err := models.SetupDatabase(ctx, fxtest.NewLifecycle(t), &config.DatabaseConfig{Debug: true, Connection: connStr})
	require.NoError(t, err)

	err = migrations.MigrateDatabase(ctx, db)
	require.NoError(t, err)
	return func() {
		require.NoError(t, postgres.Stop())
		require.NoError(t, os.RemoveAll(tmpDir))
	}, connStr, db
}
