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

	bin "github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
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

// TestAuthAndDeposit walks through the full Orderly authentication flow
// (registration + orderly key) using a local Solana private key, then
// builds an unsigned deposit transaction.
//
// Usage:
//
//	SOLANA_PRIVATE_KEY=<base58-encoded-private-key> \
//	  go test ./app/internal/tools/ -run TestAuthAndDeposit -v
//
// Optional env vars:
//
//	SOLANA_RPC_URL    — Solana RPC endpoint (default: mainnet-beta)
//	ORDERLY_BASE_URL  — Orderly API (default: https://api.orderly.org)
//	BROKER_ID         — Orderly broker id (default: dextools)
func TestAuthAndDeposit(t *testing.T) {
	privKeyBase58 := os.Getenv("SOLANA_PRIVATE_KEY")
	if privKeyBase58 == "" {
		t.Skip("SOLANA_PRIVATE_KEY not set — skipping integration test")
	}

	privKey := solana.MustPrivateKeyFromBase58(privKeyBase58)
	walletAddress := privKey.PublicKey().String()

	svc := service.NewService(service.Config{
		OrderlyBaseURL:   envOr("ORDERLY_BASE_URL", "https://api.orderly.org"),
		PerptoolsBaseURL: envOr("PERPTOOLS_BASE_URL", "https://app.perptools.ai/api"),
		BrokerID:         envOr("BROKER_ID", "dextools"),
		SolanaRPCURL:     envOr("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"),
	})

	ctx := context.Background()

	// --- Step 1: prepare registration (checks if account exists first) ---
	t.Logf("wallet: %s", walletAddress)
	t.Log("step 1 — prepare registration")

	regResult, err := svc.PrepareRegistration(ctx, walletAddress)
	if err != nil {
		t.Fatalf("PrepareRegistration: %v", err)
	}

	var accountID string

	if regResult.AlreadyRegistered {
		accountID = regResult.AccountID
		t.Logf("  account already registered, account_id: %s — skipping step 2", accountID)
	} else {
		t.Logf("  message_base64: %s", regResult.MessageBase64)

		regSig := signBase64Message(t, privKey, regResult.MessageBase64)
		t.Logf("  signature: %s", regSig)

		// --- Step 2: complete registration ---
		t.Log("step 2 — complete registration")

		accountID, err = svc.CompleteRegistration(ctx, walletAddress, regSig)
		if err != nil {
			t.Fatalf("CompleteRegistration: %v", err)
		}
		t.Logf("  account_id: %s", accountID)
	}

	// --- Step 3: prepare orderly key ---
	t.Log("step 3 — prepare orderly key")

	keyResult, err := svc.PrepareOrderlyKey(ctx, walletAddress)
	if err != nil {
		t.Fatalf("PrepareOrderlyKey: %v", err)
	}
	t.Logf("  message_base64: %s", keyResult.MessageBase64)

	keySig := signBase64Message(t, privKey, keyResult.MessageBase64)
	t.Logf("  signature: %s", keySig)

	// --- Step 4: complete orderly key ---
	t.Log("step 4 — complete orderly key")

	if err := svc.CompleteOrderlyKey(ctx, walletAddress, keySig); err != nil {
		t.Fatalf("CompleteOrderlyKey: %v", err)
	}
	t.Log("  authentication complete")

	if !svc.IsAuthenticated() {
		t.Fatal("expected service to be authenticated")
	}

	// --- Step 5: prepare USDC deposit (1 USDC = 1_000_000 smallest units) ---
	t.Log("step 5 — prepare orderly deposit (1 USDC)")

	depositResult, err := svc.PrepareOrderlyDeposit(ctx, walletAddress, "USDC", 1_000_000)
	if err != nil {
		t.Fatalf("PrepareOrderlyDeposit: %v", err)
	}

	out, _ := json.MarshalIndent(depositResult, "", "  ")
	t.Logf("  deposit result:\n%s", string(out))

	// --- Step 6: sign and send the deposit transaction ---
	t.Log("step 6 — sign and send deposit transaction")

	txBytes, err := base64.StdEncoding.DecodeString(depositResult.TransactionBase64)
	if err != nil {
		t.Fatalf("decode transaction base64: %v", err)
	}

	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(txBytes))
	if err != nil {
		t.Fatalf("deserialize transaction: %v", err)
	}

	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(privKey.PublicKey()) {
			return &privKey
		}
		return nil
	})
	if err != nil {
		t.Fatalf("sign transaction: %v", err)
	}

	rpcClient := rpc.New(envOr("SOLANA_RPC_URL", "https://api.mainnet-beta.solana.com"))
	txSig, err := rpcClient.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight: true,
	})
	if err != nil {
		t.Fatalf("send transaction: %v", err)
	}

	t.Logf("  tx signature: %s", txSig.String())
	t.Logf("  explorer: https://solscan.io/tx/%s", txSig.String())
}

// signBase64Message decodes a base64-encoded message, signs it with
// the Solana ed25519 private key, and returns "0x" + hex(signature).
func signBase64Message(t *testing.T, privKey solana.PrivateKey, msgBase64 string) string {
	t.Helper()

	msgBytes, err := base64.StdEncoding.DecodeString(msgBase64)
	if err != nil {
		t.Fatalf("decode base64 message: %v", err)
	}

	sig := ed25519.Sign(ed25519.PrivateKey(privKey), msgBytes)
	return "0x" + hex.EncodeToString(sig)
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
