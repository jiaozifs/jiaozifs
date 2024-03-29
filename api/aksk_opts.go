package api

import (
	"context"
	"net/http"

	"github.com/GitDataAI/jiaozifs/auth/aksk"
)

func AkSkOption(ak, sk string) ClientOption {
	return func(client *Client) error {
		client.RequestEditors = append(client.RequestEditors, func(_ context.Context, req *http.Request) error {
			signer := aksk.NewV0Signer(ak, sk)
			return signer.Sign(req)
		})
		return nil
	}
}

func UPOption(user, password string) ClientOption {
	return func(client *Client) error {
		client.RequestEditors = append(client.RequestEditors, func(_ context.Context, req *http.Request) error {
			req.SetBasicAuth(user, password)
			return nil
		})
		return nil
	}
}
