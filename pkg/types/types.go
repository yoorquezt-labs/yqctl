// Package types defines the core message types for the YoorQuezt MEV protocol.
package types

import "time"

// TransactionMessage represents a transaction message
type TransactionMessage struct {
	Type      string `json:"type"`
	TxID      string `json:"tx_id"`
	From      string `json:"from,omitempty"`
	To        string `json:"to,omitempty"`
	Amount    int64  `json:"amount,omitempty"`
	Fee       int64  `json:"fee,omitempty"`
	Value     string `json:"value,omitempty"`
	Chain     string `json:"chain"`
	Payload   string `json:"payload"`
	Timestamp int64  `json:"timestamp"`
	Signature string `json:"signature,omitempty"`
	PubKey    string `json:"pubkey,omitempty"`
	Hops      int    `json:"hops,omitempty"`
	MaxHops   int    `json:"max_hops,omitempty"`
	Trace     string `json:"trace,omitempty"`
}

// BundleMessage represents a bundle of transactions
type BundleMessage struct {
	Type           string               `json:"type"`
	BundleID       string               `json:"bundle_id"`
	Transactions   []TransactionMessage `json:"transactions"`
	Timestamp      int64                `json:"timestamp"`
	Signature      string               `json:"signature,omitempty"`
	PubKey         string               `json:"pubkey,omitempty"`
	Hops           int                  `json:"hops,omitempty"`
	MaxHops        int                  `json:"max_hops,omitempty"`
	Trace          string               `json:"trace,omitempty"`
	BidWei         string               `json:"bid_wei,omitempty"`
	OriginatorID   string               `json:"originator_id,omitempty"`
	TargetBlock    string               `json:"target_block,omitempty"`
	RevertingTxIDs []string             `json:"reverting_tx_ids,omitempty"`
}

// Block represents a block in the blockchain
type Block struct {
	Header       BlockHeader          `json:"header"`
	BlockID      string               `json:"block_id"`
	Bundles      []BundleMessage      `json:"bundles"`
	Transactions []TransactionMessage `json:"transactions"`
	Timestamp    int64                `json:"timestamp"`
	TotalProfit  string               `json:"total_profit,omitempty"`
	Signature    string               `json:"signature,omitempty"`
	PubKey       string               `json:"pubkey,omitempty"`
	Trace        string               `json:"trace,omitempty"`
}

// BlockHeader represents the header of a block
type BlockHeader struct {
	BlockID    string `json:"block_id"`
	ParentID   string `json:"parent_id"`
	MerkleRoot string `json:"merkle_root"`
	Timestamp  int64  `json:"timestamp"`
}

// RankedBundleRecord is the persisted form of a ranked bundle.
type RankedBundleRecord struct {
	BundleID       string        `json:"bundle_id"`
	BidWei         string        `json:"bid_wei"`
	SimProfit      string        `json:"sim_profit"`
	EffectiveValue string        `json:"effective_value"`
	Reverted       bool          `json:"reverted"`
	Simulated      bool          `json:"simulated"`
	Bundle         BundleMessage `json:"bundle"`
	CreatedAt      time.Time     `json:"created_at"`
}

// ProtectedTransaction represents a transaction submitted to the private protection pool.
type ProtectedTransaction struct {
	TxID            string   `json:"tx_id"`
	From            string   `json:"from"`
	To              string   `json:"to"`
	Value           string   `json:"value"`
	Payload         string   `json:"payload"`
	Chain           string   `json:"chain"`
	OriginatorID    string   `json:"originator_id"`
	Timestamp       int64    `json:"timestamp"`
	Deadline        int64    `json:"deadline,omitempty"`
	MinOutputWei    string   `json:"min_output_wei,omitempty"`
	MaxSlippageBps  int      `json:"max_slippage_bps,omitempty"`
	PrivacyMode     string   `json:"privacy_mode,omitempty"`
	AllowedPaths    []string `json:"allowed_paths,omitempty"`
	RebateRecipient string   `json:"rebate_recipient,omitempty"`
	HintTokenPair   string   `json:"hint_token_pair,omitempty"`
	HintChain       string   `json:"hint_chain,omitempty"`
}

// SandwichResult describes whether a sandwich attack was detected.
type SandwichResult struct {
	Detected      bool   `json:"detected"`
	FrontrunTxIdx int    `json:"frontrun_tx_idx,omitempty"`
	BackrunTxIdx  int    `json:"backrun_tx_idx,omitempty"`
	VictimTxIdx   int    `json:"victim_tx_idx,omitempty"`
	ExtractedWei  string `json:"extracted_wei,omitempty"`
	TokenPair     string `json:"token_pair,omitempty"`
}

// ProtectionStatus tracks the status of a protected transaction.
type ProtectionStatus struct {
	TxID            string          `json:"tx_id"`
	Status          string          `json:"status"`
	Protected       bool            `json:"protected"`
	SandwichCheck   *SandwichResult `json:"sandwich_check,omitempty"`
	MEVExtracted    string          `json:"mev_extracted,omitempty"`
	RebateWei       string          `json:"rebate_wei,omitempty"`
	IncludedInBlock string          `json:"included_in_block,omitempty"`
}

// BundleSummary is a lightweight view returned by ListBundles.
type BundleSummary struct {
	BundleID       string    `json:"bundle_id"`
	BidWei         string    `json:"bid_wei"`
	EffectiveValue string    `json:"effective_value"`
	Simulated      bool      `json:"simulated"`
	Reverted       bool      `json:"reverted"`
	CreatedAt      time.Time `json:"created_at"`
}

// BlockSummary is a lightweight view returned by ListBlocks.
type BlockSummary struct {
	BlockID     string    `json:"block_id"`
	Timestamp   int64     `json:"timestamp"`
	BundleCount int       `json:"bundle_count"`
	TotalProfit string    `json:"total_profit"`
	CreatedAt   time.Time `json:"created_at"`
}

// RelayRegistrationRecord is the store-level relay registration.
type RelayRegistrationRecord struct {
	RelayID      string   `json:"relay_id"`
	Name         string   `json:"name"`
	URL          string   `json:"url"`
	OwnerAddress string   `json:"owner_address"`
	StakeWei     string   `json:"stake_wei"`
	Chains       []string `json:"chains"`
	RegisteredAt int64    `json:"registered_at"`
	Active       bool     `json:"active"`
}

// RelayReputationRecord is the store-level relay reputation.
type RelayReputationRecord struct {
	RelayID             string `json:"relay_id"`
	SuccessCount        int64  `json:"success_count"`
	FailureCount        int64  `json:"failure_count"`
	TotalLatencyMs      int64  `json:"total_latency_ms"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	UptimeChecks        int64  `json:"uptime_checks"`
	UptimePasses        int64  `json:"uptime_passes"`
	Score               int    `json:"score"`
}
