package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"mcp-server/app/internal/clients/orderly"
	"mcp-server/app/internal/service"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func RegisterOrderlyTools(svc *service.Service) []ToolDef {
	return []ToolDef{
		{
			Tool: mcp.NewTool("prepare_orderly_deposit",
				mcp.WithDescription("Build an unsigned Solana transaction that deposits tokens into the Orderly vault via LayerZero. The LayerZero cross-chain fee is fetched automatically via oappQuote simulation. Returns base64-encoded transaction for wallet signing."),
				mcp.WithString("wallet_address", mcp.Required(), mcp.Description("Solana wallet public key (base58)")),
				mcp.WithString("symbol", mcp.Required(), mcp.Description("Token symbol: USDC, USDT, or SOL")),
				mcp.WithNumber("amount", mcp.Required(), mcp.Description("Amount in smallest token units (e.g. lamports for SOL, 1e6 units for USDC)")),
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
		{
			Tool: mcp.NewTool("create_order",
				mcp.WithDescription("Place a new order on Orderly. Requires authentication.\n\nOrder types: LIMIT, MARKET, IOC, FOK, POST_ONLY, ASK, BID.\nSides: BUY, SELL.\n\nTo close an existing position, create an order with the opposite side and set reduce_only=true.\nFor example, to close a long position of 0.5 BTC, place a MARKET SELL order with order_quantity=0.5 and reduce_only=true."),
				mcp.WithString("symbol", mcp.Required(), mcp.Description("Trading pair symbol (e.g. PERP_BTC_USDC, PERP_ETH_USDC)")),
				mcp.WithString("order_type", mcp.Required(), mcp.Description("Order type: LIMIT, MARKET, IOC, FOK, POST_ONLY, ASK, BID")),
				mcp.WithString("side", mcp.Required(), mcp.Description("Order side: BUY or SELL")),
				mcp.WithNumber("order_quantity", mcp.Description("Order size in base currency (e.g. 0.5 for 0.5 BTC). Required for LIMIT orders.")),
				mcp.WithNumber("order_price", mcp.Description("Order price. Required for LIMIT/IOC/FOK/POST_ONLY orders.")),
				mcp.WithNumber("order_amount", mcp.Description("Order size in quote currency (e.g. 1000 for $1000). For MARKET/ASK/BID orders.")),
				mcp.WithBoolean("reduce_only", mcp.Description("If true, the order can only reduce an existing position. Use this to close positions.")),
			),
			Handler: createOrder(svc),
		},
		{
			Tool: mcp.NewTool("cancel_order",
				mcp.WithDescription("Cancel a single order by order_id. Requires authentication."),
				mcp.WithString("symbol", mcp.Required(), mcp.Description("Trading pair symbol (e.g. PERP_BTC_USDC)")),
				mcp.WithNumber("order_id", mcp.Required(), mcp.Description("The order_id to cancel")),
			),
			Handler: cancelOrder(svc),
		},
		{
			Tool: mcp.NewTool("get_positions",
				mcp.WithDescription(`Get all open positions with margin, PnL, and liquidation info. Requires authentication.

Present the response to the user as a formatted table:

Account Summary:
| Metric              | Value       |
|---------------------|-------------|
| Total Collateral    | $X,XXX.XX   |
| Free Collateral     | $X,XXX.XX   |
| Margin Ratio        | X.XX%       |
| Total PnL (24h)     | $X.XX       |

Open Positions:
| Symbol          | Side  | Size   | Entry Price | Mark Price | Unreal. PnL | Liq. Price | Leverage |
|-----------------|-------|--------|-------------|------------|-------------|------------|----------|
| PERP_BTC_USDC   | LONG  | 0.500  | $27,908.14  | $27,794.90 | -$354.86    | $117,335.93| 10x      |
| PERP_ETH_USDC   | SHORT | 2.000  | $1,850.00   | $1,842.50  | +$15.00     | $3,200.00  | 5x       |

Side is LONG when position_qty > 0, SHORT when position_qty < 0. Display absolute value for Size.
To close a position, use create_order with the opposite side and reduce_only=true.`),
			),
			Handler: getPositions(svc),
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

		result, err := svc.PrepareOrderlyDeposit(ctx, wallet, symbol, amount)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("prepare deposit failed: %v", err)), nil
		}

		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	}
}

func createOrder(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orderType, err := req.RequireString("order_type")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		side, err := req.RequireString("side")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		orderReq := orderly.CreateOrderRequest{
			Symbol:    symbol,
			OrderType: orderType,
			Side:      side,
		}

		if v := optNumber(req, "order_quantity", 0); v != 0 {
			orderReq.OrderQuantity = v
		}
		if v := optNumber(req, "order_price", 0); v != 0 {
			orderReq.OrderPrice = v
		}
		if v := optNumber(req, "order_amount", 0); v != 0 {
			orderReq.OrderAmount = v
		}
		if req.GetBool("reduce_only", false) {
			orderReq.ReduceOnly = true
		}

		result, err := svc.CreateOrder(ctx, orderReq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("create order failed: %v", err)), nil
		}

		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	}
}

func cancelOrder(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		symbol, err := req.RequireString("symbol")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		orderID, err := req.RequireInt("order_id")
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		result, err := svc.CancelOrder(ctx, symbol, orderID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("cancel order failed: %v", err)), nil
		}

		out, _ := json.Marshal(result)
		return mcp.NewToolResultText(string(out)), nil
	}
}

func getPositions(svc *service.Service) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		result, err := svc.GetPositions(ctx)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("get positions failed: %v", err)), nil
		}

		out, _ := json.MarshalIndent(result, "", "  ")
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
