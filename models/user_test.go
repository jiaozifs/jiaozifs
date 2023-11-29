package models_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/google/uuid"

	"github.com/uptrace/bun"

	"github.com/jiaozifs/jiaozifs/models/migrations"

	"github.com/jiaozifs/jiaozifs/models"

	"github.com/jiaozifs/jiaozifs/config"
	"github.com/phayes/freeport"
	"go.uber.org/fx/fxtest"

	"github.com/stretchr/testify/require"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"

	_ "github.com/jiaozifs/jiaozifs/models/migrations"
)

var dbTimeCmpOpt = cmp.Comparer(func(x, y time.Time) bool {
	return x.Unix() == y.Unix()
})

func setup(ctx context.Context, t *testing.T) (*embeddedpostgres.EmbeddedPostgres, *bun.DB) {
	port, err := freeport.GetFreePort()
	require.NoError(t, err)
	postgres := embeddedpostgres.NewDatabase(embeddedpostgres.DefaultConfig().Port(uint32(port)).Database("jiaozifs"))
	err = postgres.Start()
	require.NoError(t, err)

	db, err := models.SetupDatabase(ctx, fxtest.NewLifecycle(t), &config.DatabaseConfig{Debug: true, Connection: fmt.Sprintf(testConnTmpl, port)})
	require.NoError(t, err)

	migrations.MigrateDatabase(ctx, db)
	return postgres, db
}
func TestNewUserRepo(t *testing.T) {
	ctx := context.Background()
	postgres, db := setup(ctx, t)
	defer postgres.Stop() //nolint

	repo := models.NewUserRepo(db)
	userModel := &models.User{
		Name:              "aaa",
		Email:             "xx@gmail.com",
		EncryptedPassword: "aaaaa",
		CurrentSignInAt:   time.Now(),
		LastSignInAt:      time.Now(),
		CurrentSignInIP:   "127.0.0.1",
		LastSignInIP:      "127.0.0.1",
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	newUser, err := repo.Insert(ctx, userModel)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, newUser.ID)

	user, err := repo.GetUser(ctx, newUser.ID)
	require.NoError(t, err)

	require.True(t, cmp.Equal(userModel.UpdatedAt, user.UpdatedAt, dbTimeCmpOpt))
}