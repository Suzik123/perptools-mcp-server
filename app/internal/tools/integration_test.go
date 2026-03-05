package tools_test

import (
	"bufio"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"mcp-server/app/internal/service"
	"mcp-server/app/internal/tools"

	"github.com/mark3labs/mcp-go/mcp"
)

func init() {
	loadEnvFile("../../../.env")
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if os.Getenv(k) == "" {
			os.Setenv(k, v)
		}
	}
}

// TestAuthAndDeposit simulates the real AI-agent MCP flow:
//
//	Agent calls MCP tool handlers the same way it would in production,
//	parses JSON responses, and uses a local ed25519 key to sign messages
//	(substituting Phantom MCP sign_message).
//
// Usage:
//
//	go test ./app/internal/tools/ -run TestAuthAndDeposit -v
func TestAuthAndDeposit(t *testing.T) {
	privKeyBase58 := os.Getenv("SOLANA_PRIVATE_KEY")
	if privKeyBase58 == "" {
		t.Skip("SOLANA_PRIVATE_KEY not set — skipping integration test")
	}

	privKeyBytes := base58Decode(t, privKeyBase58)
	walletAddress := ed25519PubKeyToBase58(privKeyBytes)

	svc := service.NewService(service.Config{
		OrderlyBaseURL:   envOr("ORDERLY_BASE_URL", "https://api.orderly.org"),
		PerptoolsBaseURL: envOr("PERPTOOLS_BASE_URL", "https://app.perptools.ai/api"),
		BrokerID:         envOr("BROKER_ID", "dextools"),
		SolanaRPCURL:     envOr("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"),
	})

	// Register all tool handlers (exactly like main.go does)
	toolMap := make(map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error))
	for _, t := range tools.RegisterAuthTools(svc) {
		toolMap[t.Tool.Name] = t.Handler
	}
	for _, t := range tools.RegisterOrderlyTools(svc) {
		toolMap[t.Tool.Name] = t.Handler
	}

	ctx := context.Background()

	// ---------------------------------------------------------------
	// Step 1: agent calls prepare_registration
	// ---------------------------------------------------------------
	t.Logf("wallet: %s", walletAddress)
	t.Log("step 1 — agent calls prepare_registration")

	regResp := callTool(t, ctx, toolMap, "prepare_registration", map[string]any{
		"wallet_address": walletAddress,
	})
	t.Logf("  tool response: %s", regResp)

	var regData map[string]any
	mustUnmarshal(t, regResp, &regData)

	if alreadyReg, _ := regData["already_registered"].(bool); alreadyReg {
		t.Logf("  account already registered (account_id: %s) — skipping step 2", regData["account_id"])
	} else {
		// ---------------------------------------------------------------
		// Step 2: agent asks Phantom MCP to sign_message, then calls
		//         complete_registration (here we sign locally)
		// ---------------------------------------------------------------
		msgBase64, _ := regData["message_base64"].(string)
		sig := phantomSignMessage(t, privKeyBytes, msgBase64)

		t.Log("step 2 — agent calls complete_registration")
		completeResp := callTool(t, ctx, toolMap, "complete_registration", map[string]any{
			"wallet_address": walletAddress,
			"signature":      sig,
		})
		t.Logf("  tool response: %s", completeResp)
	}

	// ---------------------------------------------------------------
	// Step 3: agent calls prepare_orderly_key
	// ---------------------------------------------------------------
	t.Log("step 3 — agent calls prepare_orderly_key")

	keyResp := callTool(t, ctx, toolMap, "prepare_orderly_key", map[string]any{
		"wallet_address": walletAddress,
	})
	t.Logf("  tool response: %s", keyResp)

	var keyData map[string]string
	mustUnmarshal(t, keyResp, &keyData)

	// ---------------------------------------------------------------
	// Step 4: agent asks Phantom MCP to sign_message, then calls
	//         complete_orderly_key
	// ---------------------------------------------------------------
	keySig := phantomSignMessage(t, privKeyBytes, keyData["message_base64"])

	t.Log("step 4 — agent calls complete_orderly_key")
	completeKeyResp := callTool(t, ctx, toolMap, "complete_orderly_key", map[string]any{
		"wallet_address": walletAddress,
		"signature":      keySig,
	})
	t.Logf("  tool response: %s", completeKeyResp)

	// ---------------------------------------------------------------
	// Step 5: agent calls prepare_orderly_deposit (1 USDC)
	// ---------------------------------------------------------------
	t.Log("step 5 — agent calls prepare_orderly_deposit (1 USDC)")

	depositResp := callTool(t, ctx, toolMap, "prepare_orderly_deposit", map[string]any{
		"wallet_address": walletAddress,
		"symbol":         "USDC",
		"amount":         float64(1_000_000),
	})
	t.Logf("  tool response: %s", depositResp)

	// At this point the real agent would call Phantom MCP sign_transaction
	// with the transaction_base64 from the response, then Phantom submits it.
	var depositData map[string]any
	mustUnmarshal(t, depositResp, &depositData)
	t.Logf("  transaction ready for Phantom sign_transaction (len=%d bytes)",
		len(depositData["transaction_base64"].(string)))

	// ---------------------------------------------------------------
	// Step 6: agent calls get_positions
	// ---------------------------------------------------------------
	t.Log("step 6 — agent calls get_positions")

	posResp := callTool(t, ctx, toolMap, "get_positions", nil)
	t.Logf("  tool response: %s", posResp)

	// ---------------------------------------------------------------
	// Step 7: agent calls create_order (small LIMIT BUY as a smoke test)
	// ---------------------------------------------------------------
	t.Log("step 7 — agent calls create_order (LIMIT BUY 0.001 PERP_ETH_USDC @ $1)")

	orderResp := callTool(t, ctx, toolMap, "create_order", map[string]any{
		"symbol":         "PERP_ETH_USDC",
		"order_type":     "LIMIT",
		"side":           "BUY",
		"order_quantity":  float64(0.001),
		"order_price":    float64(1),
	})
	t.Logf("  tool response: %s", orderResp)

	var orderData map[string]any
	mustUnmarshal(t, orderResp, &orderData)

	// ---------------------------------------------------------------
	// Step 8: agent cancels the order
	// ---------------------------------------------------------------
	if data, ok := orderData["data"].(map[string]any); ok {
		if orderID, ok := data["order_id"].(float64); ok && orderID > 0 {
			t.Logf("step 8 — agent calls cancel_order (order_id: %.0f)", orderID)

			cancelResp := callTool(t, ctx, toolMap, "cancel_order", map[string]any{
				"symbol":   "PERP_ETH_USDC",
				"order_id": orderID,
			})
			t.Logf("  tool response: %s", cancelResp)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers — simulate the MCP tool call / response cycle
// ---------------------------------------------------------------------------

// callTool invokes an MCP tool handler the same way the MCP server would.
// Returns the text content from the result; fails the test on errors.
func callTool(
	t *testing.T,
	ctx context.Context,
	toolMap map[string]func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error),
	name string,
	args map[string]any,
) string {
	t.Helper()

	handler, ok := toolMap[name]
	if !ok {
		t.Fatalf("tool %q not registered", name)
	}

	req := mcp.CallToolRequest{}
	req.Params.Name = name
	req.Params.Arguments = args

	result, err := handler(ctx, req)
	if err != nil {
		t.Fatalf("tool %q returned error: %v", name, err)
	}
	if result.IsError {
		text := extractText(result)
		t.Fatalf("tool %q failed: %s", name, text)
	}

	return extractText(result)
}

// extractText returns the concatenated text content from a tool result.
func extractText(r *mcp.CallToolResult) string {
	var sb strings.Builder
	for _, c := range r.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}
	return sb.String()
}

// phantomSignMessage simulates Phantom MCP sign_message:
// decode base64 message → ed25519 sign → return "0x" + hex signature.
func phantomSignMessage(t *testing.T, privKey []byte, msgBase64 string) string {
	t.Helper()

	msgBytes, err := base64.StdEncoding.DecodeString(msgBase64)
	if err != nil {
		t.Fatalf("decode base64 message: %v", err)
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privKey), msgBytes)
	return "0x" + hex.EncodeToString(sig)
}

func mustUnmarshal(t *testing.T, data string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(data), v); err != nil {
		t.Fatalf("unmarshal tool response: %v\nraw: %s", err, data)
	}
}

// base58Decode decodes a base58-encoded Solana private key (64 bytes).
func base58Decode(t *testing.T, s string) []byte {
	t.Helper()
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	result := make([]byte, 0, 64)
	for i := 0; i < len(s); i++ {
		carry := strings.IndexByte(alphabet, s[i])
		if carry < 0 {
			t.Fatalf("invalid base58 character: %c", s[i])
		}
		for j := 0; j < len(result); j++ {
			carry += int(result[j]) * 58
			result[j] = byte(carry & 0xff)
			carry >>= 8
		}
		for carry > 0 {
			result = append(result, byte(carry&0xff))
			carry >>= 8
		}
	}
	for i := 0; i < len(s) && s[i] == '1'; i++ {
		result = append(result, 0)
	}
	// reverse
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}
	return result
}

// ed25519PubKeyToBase58 extracts the public key (last 32 bytes) from a
// 64-byte ed25519 private key and returns its base58 representation.
func ed25519PubKeyToBase58(privKey []byte) string {
	const alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"
	pub := privKey[32:]
	// big-endian integer from bytes
	digits := []byte{0}
	for _, b := range pub {
		carry := int(b)
		for j := 0; j < len(digits); j++ {
			carry += int(digits[j]) << 8
			digits[j] = byte(carry % 58)
			carry /= 58
		}
		for carry > 0 {
			digits = append(digits, byte(carry%58))
			carry /= 58
		}
	}
	// leading zeros
	for _, b := range pub {
		if b != 0 {
			break
		}
		digits = append(digits, 0)
	}
	// reverse and map to alphabet
	out := make([]byte, len(digits))
	for i, d := range digits {
		out[len(digits)-1-i] = alphabet[d]
	}
	return string(out)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
