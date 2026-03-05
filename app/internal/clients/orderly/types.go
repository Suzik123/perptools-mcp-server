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

// ---------------------------------------------------------------------------
// Get Account — GET /v1/get_account
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/public/check-if-wallet-is-registered
// ---------------------------------------------------------------------------

type GetAccountResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		UserID    int    `json:"user_id"`
		AccountID string `json:"account_id"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Create Order — POST /v1/order
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/private/create-order
// ---------------------------------------------------------------------------

type CreateOrderRequest struct {
	Symbol          string  `json:"symbol"`
	ClientOrderID   string  `json:"client_order_id,omitempty"`
	OrderType       string  `json:"order_type"`
	OrderPrice      float64 `json:"order_price,omitempty"`
	OrderQuantity   float64 `json:"order_quantity,omitempty"`
	OrderAmount     float64 `json:"order_amount,omitempty"`
	VisibleQuantity float64 `json:"visible_quantity,omitempty"`
	Side            string  `json:"side"`
	ReduceOnly      bool    `json:"reduce_only,omitempty"`
}

type CreateOrderResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		OrderID       int     `json:"order_id"`
		ClientOrderID string  `json:"client_order_id"`
		OrderType     string  `json:"order_type"`
		OrderPrice    float64 `json:"order_price"`
		OrderQuantity float64 `json:"order_quantity"`
		ErrorMessage  string  `json:"error_message"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Cancel Order — DELETE /v1/order
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/private/cancel-order
// ---------------------------------------------------------------------------

type CancelOrderResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		Status string `json:"status"`
	} `json:"data"`
}

// ---------------------------------------------------------------------------
// Get All Positions — GET /v1/positions
// https://orderly.network/docs/build-on-omnichain/evm-api/restful-api/private/get-all-positions-info
// ---------------------------------------------------------------------------

type PositionsResponse struct {
	Success   bool   `json:"success"`
	Timestamp int64  `json:"timestamp"`
	Message   string `json:"message,omitempty"`
	Data      struct {
		MarginRatio         float64    `json:"current_margin_ratio_with_orders"`
		FreeCollateral      float64    `json:"free_collateral"`
		TotalCollateralVal  float64    `json:"total_collateral_value"`
		TotalPnl24H         float64    `json:"total_pnl_24_h"`
		MaintenanceMargin   float64    `json:"maintenance_margin_ratio"`
		InitialMargin       float64    `json:"initial_margin_ratio"`
		OpenMarginRatio     float64    `json:"open_margin_ratio"`
		Rows                []Position `json:"rows"`
	} `json:"data"`
}

type Position struct {
	Symbol           string  `json:"symbol"`
	PositionQty      float64 `json:"position_qty"`
	AverageOpenPrice float64 `json:"average_open_price"`
	MarkPrice        float64 `json:"mark_price"`
	UnsettledPnl     float64 `json:"unsettled_pnl"`
	EstLiqPrice      float64 `json:"est_liq_price"`
	IMR              float64 `json:"imr"`
	MMR              float64 `json:"mmr"`
	Leverage         float64 `json:"leverage"`
	Timestamp        int64   `json:"timestamp"`
}

// ---------------------------------------------------------------------------
// Withdraw Message — used to build a keccak256 hash for wallet signing
// ---------------------------------------------------------------------------

type WithdrawMessage struct {
	BrokerID      string `json:"broker_id"`
	ChainID       uint64 `json:"chainId"`
	Receiver      string `json:"receiver"`
	Token         string `json:"token"`
	Amount        uint64 `json:"amount"`
	WithdrawNonce uint64 `json:"withdrawNonce"`
	Timestamp     uint64 `json:"timestamp"`
	ChainType     string `json:"chainType"`
}
