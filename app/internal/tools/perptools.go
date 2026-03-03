package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-server/app/internal/clients/perptools"
	"mcp-server/app/internal/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

type ToolDef struct {
	Tool    mcp.Tool
	Handler server.ToolHandlerFunc
}

func RegisterPerptoolsTools(svc *service.Service) []ToolDef {
	return []ToolDef{
		{
			Tool: mcp.NewTool("health",
				mcp.WithDescription("Check Perptools API health status."),
			),
			Handler: healthTool(svc),
		},
		{
			Tool: mcp.NewTool("get_markets",
				mcp.WithDescription("Get paginated list of available trading markets."),
				mcp.WithNumber("limit", mcp.Description("Max results (default 10)")),
				mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
			),
			Handler: getMarkets(svc),
		},
		{
			Tool: mcp.NewTool("get_lending_vaults",
				mcp.WithDescription("Get available lending vaults with TVL and APY info."),
			),
			Handler: getLendingVaults(svc),
		},
		{
			Tool: mcp.NewTool("get_user_points",
				mcp.WithDescription("Get user points and distribution breakdown. Requires authentication."),
				mcp.WithString("public_key", mcp.Required(), mcp.Description("User's Solana public key (base58)")),
			),
			Handler: getUserPoints(svc),
		},
		{
			Tool: mcp.NewTool("get_leaderboard",
				mcp.WithDescription("Get points leaderboard. Requires authentication."),
				mcp.WithString("public_key", mcp.Required(), mcp.Description("User's Solana public key (base58)")),
				mcp.WithNumber("limit", mcp.Description("Max results (default 10)")),
				mcp.WithNumber("offset", mcp.Description("Offset for pagination (default 0)")),
			),
			Handler: getLeaderboard(svc),
		},
		{
			Tool: mcp.NewTool("lending_deposit",
				mcp.WithDescription("Create a lending deposit transaction. Returns base64-encoded transaction to sign. Requires authentication."),
				mcp.WithString("public_key", mcp.Required(), mcp.Description("User's Solana public key (base58)")),
				mcp.WithString("token_mint", mcp.Required(), mcp.Description("Token mint address")),
				mcp.WithNumber("amount", mcp.Required(), mcp.Description("Amount in smallest units")),
			),
			Handler: lendingDeposit(svc),
		},
		{
			Tool: mcp.NewTool("lending_withdraw",
				mcp.WithDescription("Create a lending withdraw transaction. Returns base64-encoded transaction to sign. Requires authentication."),
				mcp.WithString("public_key", mcp.Required(), mcp.Description("User's Solana public key (base58)")),
				mcp.WithString("token_mint", mcp.Required(), mcp.Description("Token mint address")),
				mcp.WithNumber("amount", mcp.Required(), mcp.Description("Amount in smallest units")),
			),
			Handler: lendingWithdraw(svc),
		},
	}
}

func jsonResult(v any) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("marshal error: %v", err)), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func healthTool(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := svc.Health(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("health check failed: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func getMarkets(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		limit := int32(optNumber(req, "limit", 10))
		offset := int32(optNumber(req, "offset", 0))

		resp, err := svc.GetMarkets(ctx, limit, offset)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get markets failed: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func getLendingVaults(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		resp, err := svc.GetLendingVaults(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get lending vaults failed: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func getUserPoints(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pk, err := req.RequireString("public_key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		resp, err := svc.GetUserPoints(ctx, pk)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get user points failed: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func getLeaderboard(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pk, err := req.RequireString("public_key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		limit := int32(optNumber(req, "limit", 10))
		offset := int32(optNumber(req, "offset", 0))

		resp, err := svc.GetLeaderboard(ctx, pk, limit, offset)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get leaderboard failed: %v", err)), nil
		}
		return jsonResult(resp)
	}
}

func lendingDeposit(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pk, err := req.RequireString("public_key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tokenMint, err := req.RequireString("token_mint")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount := uint64(optNumber(req, "amount", 0))
		if amount == 0 {
			return mcp.NewToolResultError("amount is required and must be > 0"), nil
		}

		resp, err := svc.LendingDeposit(ctx, perptools.LendingTxRequest{
			PublicKey: pk,
			TokenMint: tokenMint,
			Amount:    amount,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("lending deposit failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Sign this transaction with your wallet:\n%s", resp.TxbBase64)), nil
	}
}

func lendingWithdraw(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		pk, err := req.RequireString("public_key")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		tokenMint, err := req.RequireString("token_mint")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount := uint64(optNumber(req, "amount", 0))
		if amount == 0 {
			return mcp.NewToolResultError("amount is required and must be > 0"), nil
		}

		resp, err := svc.LendingWithdraw(ctx, perptools.LendingTxRequest{
			PublicKey: pk,
			TokenMint: tokenMint,
			Amount:    amount,
		})
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("lending withdraw failed: %v", err)), nil
		}
		return mcp.NewToolResultText(fmt.Sprintf("Sign this transaction with your wallet:\n%s", resp.TxbBase64)), nil
	}
}

func optNumber(req mcp.CallToolRequest, key string, def float64) float64 {
	args := req.GetArguments()
	if v, ok := args[key]; ok {
		if n, ok := v.(float64); ok {
			return n
		}
	}
	return def
}
