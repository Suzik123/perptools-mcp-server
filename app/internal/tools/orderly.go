package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-server/app/internal/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterOrderlyTools(svc *service.Service) []ToolDef {
	return []ToolDef{
		{
			Tool: mcp.NewTool("prepare_orderly_deposit",
				mcp.WithDescription("Build an unsigned Solana transaction that deposits tokens into the Orderly vault via LayerZero. Returns base64-encoded transaction for wallet signing."),
				mcp.WithString("wallet_address", mcp.Required(), mcp.Description("Solana wallet public key (base58)")),
				mcp.WithString("symbol", mcp.Required(), mcp.Description("Token symbol: USDC, USDT, or SOL")),
				mcp.WithNumber("amount", mcp.Required(), mcp.Description("Amount in smallest token units (e.g. lamports for SOL, 1e6 units for USDC)")),
				mcp.WithNumber("native_fee", mcp.Required(), mcp.Description("LayerZero native fee in lamports (cross-chain messaging fee)")),
			),
			Handler: prepareOrderlyDeposit(svc),
		},
		{
			Tool: mcp.NewTool("prepare_orderly_withdraw",
				mcp.WithDescription("Build an unsigned Solana memo transaction with the Orderly withdraw message. Returns base64-encoded transaction for wallet signing. After signing, the signature is used to call the Orderly withdrawal API."),
				mcp.WithString("wallet_address", mcp.Required(), mcp.Description("Solana wallet public key (base58)")),
				mcp.WithString("token", mcp.Required(), mcp.Description("Token symbol: USDC, USDT, or SOL")),
				mcp.WithNumber("amount", mcp.Required(), mcp.Description("Amount in smallest token units")),
				mcp.WithNumber("withdraw_nonce", mcp.Required(), mcp.Description("Withdraw nonce from Orderly")),
			),
			Handler: prepareOrderlyWithdraw(svc),
		},
	}
}

func prepareOrderlyDeposit(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wallet, err := req.RequireString("wallet_address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount := uint64(optNumber(req, "amount", 0))
		if amount == 0 {
			return mcp.NewToolResultError("amount is required and must be > 0"), nil
		}
		nativeFee := uint64(optNumber(req, "native_fee", 0))

		result, err := svc.PrepareOrderlyDeposit(ctx, wallet, symbol, amount, nativeFee)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("prepare deposit failed: %v", err)), nil
		}

		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	}
}

func prepareOrderlyWithdraw(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		wallet, err := req.RequireString("wallet_address")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		token, err := req.RequireString("token")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		amount := uint64(optNumber(req, "amount", 0))
		if amount == 0 {
			return mcp.NewToolResultError("amount is required and must be > 0"), nil
		}
		withdrawNonce := uint64(optNumber(req, "withdraw_nonce", 0))

		result, err := svc.PrepareOrderlyWithdraw(ctx, wallet, token, amount, withdrawNonce)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("prepare withdraw failed: %v", err)), nil
		}

		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	}
}
