package auth

import (
	"context"
	"fmt"

	"github.com/GitDataAI/jiaozifs/models"
)

var ErrUserNotFound = fmt.Errorf("UserNotFound")

type contextKey string

const (
	userContextKey contextKey = "user"
)

func GetOperator(ctx context.Context) (*models.User, error) {
	user, ok := ctx.Value(userContextKey).(*models.User)
	if !ok {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func WithOperator(ctx context.Context, user *models.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}
