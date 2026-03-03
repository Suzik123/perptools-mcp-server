package orderly

// ---------------------------------------------------------------------------
// Get Nonce — GET /v1/registration_nonce
// ---------------------------------------------------------------------------

type GetNonceResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		Nonce string `json:"registration_nonce"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Register Account — POST /v1/register_account
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/public/register-account
// ---------------------------------------------------------------------------

type RegisterAccountRequest struct {
	Message     RegisterAccountMessage `json:"message"`
	Signature   string                 `json:"signature"`
	UserAddress string                 `json:"userAddress"`
}

type RegisterAccountMessage struct {
	BrokerID          string `json:"brokerId"`
	ChainID           int    `json:"chainId"`
	ChainType         string `json:"chainType,omitempty"` // "EVM" or "SOL"
	Timestamp         string `json:"timestamp"`           // UNIX milliseconds as string
	RegistrationNonce string `json:"registrationNonce"`
}

type RegisterAccountResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		AccountID string `json:"account_id"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Add Orderly Key — POST /v1/orderly_key
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/public/add-orderly-key
// ---------------------------------------------------------------------------

type AddOrderlyKeyRequest struct {
	Message     OrderlyKeyMessage `json:"message"`
	Signature   string            `json:"signature"`
	UserAddress string            `json:"userAddress"`
}

type OrderlyKeyMessage struct {
	BrokerID     string `json:"brokerId"`
	ChainID      int    `json:"chainId"`
	ChainType    string `json:"chainType,omitempty"` // "EVM" or "SOL"
	OrderlyKey   string `json:"orderlyKey"`
	Scope        string `json:"scope"`      // "read", "trading", "asset" or comma-separated
	Timestamp    int64  `json:"timestamp"`  // UNIX milliseconds
	Expiration   int64  `json:"expiration"` // UNIX milliseconds, max 365 days from add
	Tag          string `json:"tag,omitempty"`
	SubAccountID string `json:"subAccountId,omitempty"`
}

type AddOrderlyKeyResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		ID         int    `json:"id"`
		OrderlyKey string `json:"orderly_key"`
	} `json:"data"`
}
