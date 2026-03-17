# Perptools MCP Server — Rules & Conventions

This document summarizes all existing rules, conventions, and operational guidelines for the perptools-mcp server.

---

## 1. Server Overview

| Property | Value |
|----------|-------|
| Name | `perptools-mcp` |
| Version | `1.0.0` |
| Transports | SSE (default), stdio |
| Broker | dextools (Orderly Network) |
| Base URLs | Orderly: `https://api.orderly.org`, Perptools: `https://app.perptools.ai/api` |

---

## 2. Tool Categories

### 2.1 Auth Tools (4 tools)
Require wallet address and Phantom MCP for signing. **Authentication is required before trading.**

| Tool | Purpose | Next step on success |
|------|---------|----------------------|
| `prepare_registration` | Check if account exists, get message to sign | If `already_registered` → skip to `prepare_orderly_key`; else sign → `complete_registration` |
| `complete_registration` | Submit signature for account creation | `prepare_orderly_key` |
| `prepare_orderly_key` | Get message for Orderly key registration | Sign → `complete_orderly_key` |
| `complete_orderly_key` | Submit signature, store credentials | Trading tools now available |

**Important**: Signing is **only** via Phantom MCP. This MCP does not sign transactions.

### 2.2 Orderly Tools (5 tools)
Trading, positions, deposit/withdraw. Most require auth (except deposit/withdraw prep).

| Tool | Auth | Purpose |
|------|------|---------|
| `prepare_orderly_deposit` | No | Build unsigned Solana tx for deposit |
| `prepare_orderly_withdraw` | **Yes** | Build unsigned withdraw tx (fetches nonce automatically) |
| `create_order` | **Yes** | Place MARKET/LIMIT orders |
| `cancel_order` | **Yes** | Cancel by order_id |
| `get_positions` | **Yes** | Positions, collateral, margin |
| `set_position_tp_sl` | **Yes** | Set take-profit and stop-loss on position |
| `get_algo_orders` | **Yes** | List TP/SL and other algo orders |
| `cancel_algo_order` | **Yes** | Cancel algo order by algo_order_id |

### 2.3 Perptools Tools (7 tools)
Market data. No Orderly auth needed for read-only.

| Tool | Purpose |
|------|---------|
| `health` | API status |
| `get_markets` | Trading pairs with prices |
| `get_user_points` | User points (auth via public_key) |
| `get_leaderboard` | Points leaderboard |

---

## 3. Trading Rules

### 3.1 Order Quantity (CRITICAL)
- **PERP markets use `order_quantity` in BASE currency** (ETH, BTC, etc.), **NOT in USDC.**
- If user says "buy $100 of ETH": `order_quantity = desired_usdc / current_price` (use `get_markets` for price).
- Example: ~$11 of ETH at ~$2200 → `order_quantity = 0.005`.

### 3.2 Order Types
| Type | Params needed | Notes |
|------|---------------|-------|
| MARKET | symbol, side, order_quantity | No price |
| LIMIT | symbol, side, order_quantity, order_price | Price required |
| POST_ONLY | Same as LIMIT | Maker only |
| IOC, FOK | Same as LIMIT | Partial/full fill |
| ASK, BID | symbol, side, order_quantity | Best ask/bid |

### 3.3 Closing Positions
- Use `create_order` with **opposite side** and `reduce_only=true`.
- LONG position → close with `side=SELL, reduce_only=true`.
- SHORT position → close with `side=BUY, reduce_only=true`.
- Size = absolute value of `position_qty` from `get_positions`.

### 3.4 Position Interpretation
- `position_qty > 0` → LONG
- `position_qty < 0` → SHORT
- Display size as absolute value in UI.

### 3.5 Take-Profit / Stop-Loss (TP/SL)
- Use `set_position_tp_sl(symbol, take_profit_price, stop_loss_price)` on an existing position.
- Uses `POSITIONAL_TP_SL`: max 1 untriggered per user per symbol. Closes full position when triggered.
- **LONG**: `take_profit_price > entry` (profit when price rises), `stop_loss_price < entry` (limit loss).
- **SHORT**: `take_profit_price < entry`, `stop_loss_price > entry`.
- Use `get_algo_orders` to list, `cancel_algo_order` to remove or replace.

---

## 4. Error Handling

### 4.1 API Error Format
Orderly API errors are parsed into `APIError{Code, Message}`. Tool handlers return structured messages for agents.

### 4.2 Common Order Errors
| Error keywords | Agent advice |
|----------------|--------------|
| insufficient, balance, margin, collateral | Suggest `prepare_orderly_deposit` |
| quantity too small, min_notional | Increase order_quantity |
| price, price_range | Check `get_markets`, adjust order_price |
| reduce_only | Position may be closed; verify with `get_positions` |

---

## 5. Authentication Flow (Agent Instructions)

```
1. Get wallet_address (Phantom MCP get_wallet_addresses or ask user)
2. prepare_registration(wallet_address)
   → If already_registered: skip to step 4
   → Else: sign message_base64 via Phantom MCP sign_message → complete_registration(wallet_address, signature)
3. prepare_orderly_key(wallet_address)
4. Sign message_base64 via Phantom MCP → complete_orderly_key(wallet_address, signature)
5. Auth complete — all trading tools available
```

---

## 6. Configuration

### 6.1 Environment Variables
| Var | Default | Purpose |
|-----|---------|---------|
| `TRANSPORT` | `sse` | `sse` or `stdio` |
| `ADDR` | `:8080` | Listen address |
| `BASE_PATH` | `/mcp` | SSE path prefix |
| `BASE_URL` | (empty) | Public URL for SSE |
| `SOLANA_RPC_URL` | mainnet-beta | RPC endpoint |
| `SOLANA_PRIVATE_KEY` | (empty) | For integration tests only |
| `MCP_PORT` | `8080` | Docker host port mapping |

### 6.2 Docker
- Internal port always 8080.
- Host port via `MCP_PORT` in `.env` (default 8082 to avoid Phantom callback conflict).
- SSE endpoint: `http://localhost:${MCP_PORT}/mcp/sse`.

---

## 7. OpenClaw / mcporter Setup

### 7.1 mcporter Config (`~/.mcporter/mcporter.json`)
```json
{
  "mcpServers": {
    "perptools-mcp": {
      "url": "http://localhost:8082/mcp/sse"
    }
  }
}
```
Use actual host/port if server is remote.

### 7.2 OpenClaw Requirements
- `tools.profile`: `"full"` (not `messaging`) so mcporter tools are visible.
- mcporter skill: must be **eligible/enabled** (binary in PATH, e.g. `/root/.npm-global/bin`).
- User systemd: `loginctl enable-linger`, `dbus-user-session`, `XDG_RUNTIME_DIR` for `systemctl --user`.

### 7.3 Phantom MCP
- Redirect URL in Phantom Portal must match `PHANTOM_CALLBACK_PORT` in mcporter config.
- Headless server: use SSH tunnel `ssh -L 8080:localhost:8080 user@server` during `mcporter auth phantom-mcp`.
- Kill stale processes on callback port before auth: `kill $(lsof -t -i :8080)`.

---

## 8. Amount Conventions

| Context | Unit | Example |
|---------|------|---------|
| SOL | lamports (1e9) | 1 SOL = 1_000_000_000 |
| USDC/USDT | 1e6 decimals | 11 USDC = 11_000_000 |
| order_quantity (PERP) | Base asset | 0.005 ETH, 0.001 BTC |

---

## 9. Integration Tests

- Run: `go test ./app/internal/tools/ -run TestAuthAndDeposit -v`
- Require: `SOLANA_PRIVATE_KEY` in `.env` (base58).
- Tests simulate agent flow via `mcp.ToolHandlerFunc`; no real Phantom needed.
- `.env` is loaded in `init()` via `loadEnvFile("../../../.env")`.
- Run without cache: `go test -count=1 ...`

---

## 10. Signing Policy

- **This MCP does NOT sign** transactions or messages.
- All signing is performed by Phantom MCP or the user's wallet.
- Tools return base64-encoded transactions/messages for external signing.
- Integration tests use a local private key only to simulate Phantom behavior.

---

## 11. Tool Response Format

- Success: `mcp.NewToolResultText(string)` with JSON where appropriate.
- Error: `mcp.NewToolResultError(string)` with actionable messages.
- `get_positions`: format as table (see tool description) when presenting to user.
