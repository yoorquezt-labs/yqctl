// Package jsonrpc implements JSON-RPC 2.0 types for the MEV gateway protocol.
package jsonrpc

import "encoding/json"

const Version = "2.0"

// Method constants for the MEV gateway.
const (
	// Bundle operations
	MethodSendBundle     = "mev_sendBundle"
	MethodGetBundle      = "mev_getBundle"
	MethodGetAuction     = "mev_getAuction"
	MethodSimulateBundle = "mev_simulateBundle"
	MethodSimulateTx     = "mev_simulateTx"

	// Protected transactions
	MethodProtectTx         = "mev_protectTx"
	MethodGetProtectStatus  = "mev_getProtectStatus"
	MethodGetProtectRebates = "mev_getProtectRebates"

	// Intent operations
	MethodSubmitIntent   = "mev_submitIntent"
	MethodGetIntent      = "mev_getIntent"
	MethodSubmitSolution = "mev_submitSolution"
	MethodGetSolutions   = "mev_getSolutions"
	MethodRegisterSolver = "mev_registerSolver"
	MethodListSolvers    = "mev_listSolvers"

	// Relay operations
	MethodRelayRegister = "mev_relayRegister"
	MethodRelayList     = "mev_relayList"
	MethodRelayStats    = "mev_relayStats"
	MethodRelayGet      = "mev_relayGet"

	// Store / query
	MethodListBundles = "mev_listBundles"
	MethodListBlocks  = "mev_listBlocks"
	MethodGetBlock    = "mev_getBlock"

	// System
	MethodHealth           = "mev_health"
	MethodAuthStats        = "mev_authStats"
	MethodSimCacheStats    = "mev_simCacheStats"
	MethodOrderflowSummary = "mev_orderflowSummary"

	// Subscriptions
	MethodSubscribe   = "mev_subscribe"
	MethodUnsubscribe = "mev_unsubscribe"

	// YQ aliases
	MethodYQGetProtectionStatus = "yq_getProtectionStatus"
	MethodYQGetRebateHistory    = "yq_getRebateHistory"
	MethodYQGetStats            = "yq_getStats"
	MethodYQSubscribe           = "yq_subscribe"
	MethodYQUnsubscribe         = "yq_unsubscribe"
)

// Subscription topics.
const (
	TopicAuction = "auction"
	TopicMempool = "mempool"
	TopicBlocks  = "blocks"
	TopicProtect = "protect"
	TopicIntents = "intents"
)

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *Error          `json:"error,omitempty"`
	ID      interface{}     `json:"id,omitempty"`
}

// Notification is a JSON-RPC 2.0 notification (no ID, server-initiated).
type Notification struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Error is a JSON-RPC 2.0 error object.
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func (e *Error) Error() string { return e.Message }

// Standard error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603

	// Custom error codes
	CodeUnauthorized   = -32000
	CodeRateLimited    = -32001
	CodeNotFound       = -32002
	CodeServiceUnavail = -32003
)

// NewRequest creates a new JSON-RPC request.
func NewRequest(id interface{}, method string, params interface{}) (*Request, error) {
	var raw json.RawMessage
	if params != nil {
		b, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		raw = b
	}
	return &Request{JSONRPC: Version, ID: id, Method: method, Params: raw}, nil
}

// NewResponse creates a success response.
func NewResponse(id interface{}, result interface{}) (*Response, error) {
	b, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return &Response{JSONRPC: Version, ID: id, Result: b}, nil
}

// NewErrorResponse creates an error response.
func NewErrorResponse(id interface{}, code int, message string) *Response {
	return &Response{
		JSONRPC: Version,
		ID:      id,
		Error:   &Error{Code: code, Message: message},
	}
}

// NewNotification creates a server-push notification.
func NewNotification(method string, params interface{}) (*Notification, error) {
	b, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	return &Notification{JSONRPC: Version, Method: method, Params: b}, nil
}

// SubscriptionEvent is sent to subscribers.
type SubscriptionEvent struct {
	Subscription string      `json:"subscription"`
	Result       interface{} `json:"result"`
}
