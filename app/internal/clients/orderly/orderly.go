package orderly

import (
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type Client interface {
	// GetNonce fetches a registration nonce required for account registration.
	GetNonce(ctx context.Context) (*GetNonceResponse, error)

	// RegisterAccount registers a new account to Orderly (unique per wallet + builder).
	// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/public/register-account
	RegisterAccount(ctx context.Context, req RegisterAccountRequest) (*RegisterAccountResponse, error)

	// AddOrderlyKey registers an Orderly access key for an account.
	// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/public/add-orderly-key
	AddOrderlyKey(ctx context.Context, req AddOrderlyKeyRequest) (*AddOrderlyKeyResponse, error)
}

type Config struct {
	BaseURL string // https://api.orderly.org (mainnet) or https://testnet-api.orderly.org
	Timeout time.Duration
}

type client struct {
	http *resty.Client
}

func NewClient(cfg Config) Client {
	c := resty.New().
		SetBaseURL(cfg.BaseURL).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	if cfg.Timeout > 0 {
		c.SetTimeout(cfg.Timeout)
	}

	return &client{http: c}
}

func (c *client) GetNonce(ctx context.Context) (*GetNonceResponse, error) {
	var out GetNonceResponse
	r, err := c.http.R().SetContext(ctx).
		SetResult(&out).
		Get("/v1/registration_nonce")
	if err != nil {
		return nil, fmt.Errorf("get nonce: %w", err)
	}
	if r.IsError() {
		return nil, fmt.Errorf("get nonce: %s %s", r.Status(), r.String())
	}
	if !out.Success {
		return nil, fmt.Errorf("get nonce: %s", out.Message)
	}
	return &out, nil
}

func (c *client) RegisterAccount(ctx context.Context, req RegisterAccountRequest) (*RegisterAccountResponse, error) {
	var out RegisterAccountResponse
	r, err := c.http.R().SetContext(ctx).
		SetBody(req).
		SetResult(&out).
		Post("/v1/register_account")
	if err != nil {
		return nil, fmt.Errorf("register account: %w", err)
	}
	if r.IsError() {
		return nil, fmt.Errorf("register account: %s %s", r.Status(), r.String())
	}
	if !out.Success {
		return nil, fmt.Errorf("register account: %s", out.Message)
	}
	return &out, nil
}

func (c *client) AddOrderlyKey(ctx context.Context, req AddOrderlyKeyRequest) (*AddOrderlyKeyResponse, error) {
	var out AddOrderlyKeyResponse
	r, err := c.http.R().SetContext(ctx).
		SetBody(req).
		SetResult(&out).
		Post("/v1/orderly_key")
	if err != nil {
		return nil, fmt.Errorf("add orderly key: %w", err)
	}
	if r.IsError() {
		return nil, fmt.Errorf("add orderly key: %s %s", r.Status(), r.String())
	}
	if !out.Success {
		return nil, fmt.Errorf("add orderly key: %s", out.Message)
	}
	return &out, nil
}
