package service

import (
	"context"
	"fmt"

	"mcp-server/app/internal/clients/orderly"
	"mcp-server/app/internal/clients/perptools"
	"mcp-server/app/internal/service/auth"
)

type Config struct {
	PerptoolsBaseURL string
	OrderlyBaseURL   string
	BrokerID         string
}

type Service struct {
	cfg       Config
	auth      *auth.Service
	orderly   orderly.Client
	perptools perptools.Client
}

func NewService(cfg Config) *Service {
	orderlyClient := orderly.NewClient(orderly.Config{BaseURL: cfg.OrderlyBaseURL})
	authSvc := auth.NewService(auth.Config{BrokerID: cfg.BrokerID}, orderlyClient)

	s := &Service{
		cfg:     cfg,
		auth:    authSvc,
		orderly: orderlyClient,
	}
	s.perptools = perptools.NewClient(cfg.PerptoolsBaseURL)
	return s
}

func (s *Service) rebuildPerptoolsClient() {
	creds := s.auth.GetCredentials()
	if creds != nil && creds.OrderlyPrivateKey != nil {
		s.perptools = perptools.NewClientWithAuth(
			s.cfg.PerptoolsBaseURL,
			creds.AccountID,
			creds.OrderlyPublicKey,
			creds.OrderlyPrivateKey,
		)
	}
}

// --- Auth ---

func (s *Service) IsAuthenticated() bool {
	return s.auth.IsAuthenticated()
}

func (s *Service) PrepareRegistration(ctx context.Context, walletAddress string) ([]byte, error) {
	return s.auth.PrepareRegistration(ctx, walletAddress)
}

func (s *Service) CompleteRegistration(ctx context.Context, walletAddress, signature string) (string, error) {
	if err := s.auth.CompleteRegistration(ctx, walletAddress, signature); err != nil {
		return "", err
	}
	return s.auth.GetCredentials().AccountID, nil
}

func (s *Service) PrepareOrderlyKey(ctx context.Context, walletAddress string) ([]byte, error) {
	return s.auth.PrepareOrderlyKey(ctx, walletAddress)
}

func (s *Service) CompleteOrderlyKey(ctx context.Context, walletAddress, signature string) error {
	if err := s.auth.CompleteOrderlyKey(ctx, walletAddress, signature); err != nil {
		return err
	}
	s.rebuildPerptoolsClient()
	return nil
}

// --- Perptools (public) ---

func (s *Service) Health(ctx context.Context) (*perptools.HealthResponse, error) {
	return s.perptools.Health(ctx)
}

func (s *Service) GetMarkets(ctx context.Context, limit, offset int32) (*perptools.MarketsResponse, error) {
	return s.perptools.GetMarkets(ctx, limit, offset)
}

func (s *Service) GetLendingVaults(ctx context.Context) ([]perptools.Vault, error) {
	return s.perptools.GetLendingVaults(ctx)
}

// --- Perptools (authenticated) ---

func (s *Service) GetUserPoints(ctx context.Context, publicKey string) (*perptools.UserPoints, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	return s.perptools.GetUserPoints(ctx, publicKey)
}

func (s *Service) GetLeaderboard(ctx context.Context, publicKey string, limit, offset int32) ([]perptools.LeaderboardEntry, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	return s.perptools.GetLeaderboard(ctx, publicKey, limit, offset)
}

func (s *Service) LendingDeposit(ctx context.Context, req perptools.LendingTxRequest) (*perptools.Transaction, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	return s.perptools.LendingDeposit(ctx, req)
}

func (s *Service) LendingWithdraw(ctx context.Context, req perptools.LendingTxRequest) (*perptools.Transaction, error) {
	if err := s.requireAuth(); err != nil {
		return nil, err
	}
	return s.perptools.LendingWithdraw(ctx, req)
}

func (s *Service) requireAuth() error {
	if !s.auth.IsAuthenticated() {
		return fmt.Errorf("not authenticated — run prepare_registration, complete_registration, prepare_orderly_key, complete_orderly_key first")
	}
	return nil
}
