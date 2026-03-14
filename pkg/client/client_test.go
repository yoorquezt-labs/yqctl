package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/yoorquezt-labs/yqctl/pkg/jsonrpc"
	"github.com/yoorquezt-labs/yqctl/pkg/types"
)

// upgrader is shared across mock servers.
var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}

// echoServer upgrades to WS and echoes JSON-RPC responses with the same ID.
// The result is always {"ok":true}.
func echoServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			resp, _ := jsonrpc.NewResponse(req.ID, map[string]bool{"ok": true})
			out, _ := json.Marshal(resp)
			if err := conn.WriteMessage(websocket.TextMessage, out); err != nil {
				return
			}
		}
	}))
}

// wsURL converts an httptest server URL to a WebSocket URL.
func wsURL(s *httptest.Server) string {
	return "ws" + strings.TrimPrefix(s.URL, "http")
}

func TestDial_Success(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()
}

func TestDial_WithAPIKey(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.Close()
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv), APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	c.Close()

	if gotAuth != "Bearer test-key" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "Bearer test-key")
	}
}

func TestDial_InvalidURL(t *testing.T) {
	_, err := Dial(Config{GatewayURL: "ws://127.0.0.1:1"})
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestClose_Idempotent(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	if err := c.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestCall_Success(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.Call(ctx, jsonrpc.MethodHealth, nil)
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	var result map[string]bool
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("Unmarshal result: %v", err)
	}
	if !result["ok"] {
		t.Error("expected ok=true")
	}
}

func TestCall_ErrorResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}
			resp := jsonrpc.NewErrorResponse(req.ID, jsonrpc.CodeMethodNotFound, "method not found")
			out, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, out)
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Call(ctx, "nonexistent", nil)
	if err == nil {
		t.Fatal("expected error from error response")
	}
	if err.Error() != "method not found" {
		t.Errorf("error = %q, want %q", err.Error(), "method not found")
	}
}

func TestCall_ContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = c.Call(ctx, jsonrpc.MethodHealth, nil)
	if err == nil {
		t.Fatal("expected context error")
	}
	if ctx.Err() == nil {
		t.Error("context should be done")
	}
}

func TestCall_ClientClosed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}

	c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = c.Call(ctx, jsonrpc.MethodHealth, nil)
	if err == nil {
		t.Fatal("expected error after close")
	}
}

func TestConcurrentCalls(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	const n = 20
	var wg sync.WaitGroup
	errs := make(chan error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			raw, err := c.Call(ctx, jsonrpc.MethodHealth, nil)
			if err != nil {
				errs <- err
				return
			}
			var result map[string]bool
			if err := json.Unmarshal(raw, &result); err != nil {
				errs <- err
				return
			}
			if !result["ok"] {
				errs <- fmt.Errorf("expected ok=true")
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent call error: %v", err)
	}

	finalID := atomic.LoadInt64(&c.nextID)
	if finalID != n {
		t.Errorf("nextID = %d, want %d", finalID, n)
	}
}

func TestSubscribe_ReceivesEvents(t *testing.T) {
	subID := "sub-abc-123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			if req.Method == jsonrpc.MethodSubscribe {
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]string{"subscription": subID})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)

				for i := 0; i < 2; i++ {
					evt := jsonrpc.SubscriptionEvent{
						Subscription: subID,
						Result:       map[string]int{"event": i},
					}
					notif, _ := jsonrpc.NewNotification(jsonrpc.MethodSubscribe, evt)
					out, _ := json.Marshal(notif)
					conn.WriteMessage(websocket.TextMessage, out)
				}
			} else {
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]bool{"ok": true})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	gotSubID, ch, err := c.Subscribe(ctx, jsonrpc.TopicAuction)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if gotSubID != subID {
		t.Errorf("subID = %q, want %q", gotSubID, subID)
	}

	for i := 0; i < 2; i++ {
		select {
		case msg := <-ch:
			if msg == nil {
				t.Fatal("received nil message")
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout waiting for event %d", i)
		}
	}
}

func TestUnsubscribe(t *testing.T) {
	subID := "sub-to-remove"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			if req.Method == jsonrpc.MethodSubscribe {
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]string{"subscription": subID})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			} else {
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]bool{"ok": true})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	gotSubID, _, err := c.Subscribe(ctx, jsonrpc.TopicMempool)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	if err := c.Unsubscribe(ctx, gotSubID); err != nil {
		t.Fatalf("Unsubscribe: %v", err)
	}

	c.subsMu.RLock()
	_, exists := c.subs[gotSubID]
	c.subsMu.RUnlock()
	if exists {
		t.Error("subscription channel should be removed after unsubscribe")
	}
}

func TestSendBundle(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			if req.Method != jsonrpc.MethodSendBundle {
				t.Errorf("expected method %s, got %s", jsonrpc.MethodSendBundle, req.Method)
			}

			resp, _ := jsonrpc.NewResponse(req.ID, map[string]string{"bundle_id": "b-1"})
			out, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, out)
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	bundle := types.BundleMessage{
		BundleID: "b-1",
		Transactions: []types.TransactionMessage{
			{TxID: "tx-1", Chain: "ethereum", Payload: "0xdead"},
		},
	}
	result, err := c.SendBundle(ctx, bundle)
	if err != nil {
		t.Fatalf("SendBundle: %v", err)
	}
	if result["bundle_id"] != "b-1" {
		t.Errorf("bundle_id = %v, want b-1", result["bundle_id"])
	}
}

func TestHealth(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if raw == nil {
		t.Error("expected non-nil result")
	}
}

func TestServerDisconnect_PendingCallFails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		conn.ReadMessage()
		conn.Close()
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Call(ctx, jsonrpc.MethodHealth, nil)
	if err == nil {
		t.Fatal("expected error when server disconnects")
	}
}

func TestCall_VerifiesRequestFormat(t *testing.T) {
	var receivedReq jsonrpc.Request

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			json.Unmarshal(data, &receivedReq)

			resp, _ := jsonrpc.NewResponse(receivedReq.ID, "ok")
			out, _ := json.Marshal(resp)
			conn.WriteMessage(websocket.TextMessage, out)
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Call(ctx, jsonrpc.MethodGetBundle, map[string]string{"bundle_id": "b-99"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}

	if receivedReq.JSONRPC != jsonrpc.Version {
		t.Errorf("request JSONRPC = %q, want %q", receivedReq.JSONRPC, jsonrpc.Version)
	}
	if receivedReq.Method != jsonrpc.MethodGetBundle {
		t.Errorf("request Method = %q, want %q", receivedReq.Method, jsonrpc.MethodGetBundle)
	}
	if receivedReq.ID == nil {
		t.Error("request ID should not be nil")
	}
}

func TestLegacyEvent_BroadcastToAllSubs(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var req jsonrpc.Request
			if err := json.Unmarshal(data, &req); err != nil {
				continue
			}

			if req.Method == jsonrpc.MethodSubscribe {
				var params map[string]string
				json.Unmarshal(req.Params, &params)
				subID := "sub-" + params["topic"]
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]string{"subscription": subID})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			} else if req.Method == "trigger_legacy" {
				legacy, _ := json.Marshal(map[string]string{"type": "bundle.submitted", "bundle_id": "b-42"})
				conn.WriteMessage(websocket.TextMessage, legacy)

				resp, _ := jsonrpc.NewResponse(req.ID, map[string]bool{"ok": true})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			} else {
				resp, _ := jsonrpc.NewResponse(req.ID, map[string]bool{"ok": true})
				out, _ := json.Marshal(resp)
				conn.WriteMessage(websocket.TextMessage, out)
			}
		}
	}))
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, ch, err := c.Subscribe(ctx, "auction")
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	_, err = c.Call(ctx, "trigger_legacy", nil)
	if err != nil {
		t.Fatalf("trigger_legacy: %v", err)
	}

	select {
	case msg := <-ch:
		var evt map[string]string
		if err := json.Unmarshal(msg, &evt); err != nil {
			t.Fatalf("Unmarshal legacy event: %v", err)
		}
		if evt["type"] != "bundle.submitted" {
			t.Errorf("type = %q, want %q", evt["type"], "bundle.submitted")
		}
		if evt["bundle_id"] != "b-42" {
			t.Errorf("bundle_id = %q, want %q", evt["bundle_id"], "b-42")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for legacy event")
	}
}

func TestGetAuction(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.GetAuction(ctx)
	if err != nil {
		t.Fatalf("GetAuction: %v", err)
	}
	if raw == nil {
		t.Error("expected non-nil result")
	}
}

func TestRelayList(t *testing.T) {
	srv := echoServer(t)
	defer srv.Close()

	c, err := Dial(Config{GatewayURL: wsURL(srv)})
	if err != nil {
		t.Fatalf("Dial: %v", err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	raw, err := c.RelayList(ctx)
	if err != nil {
		t.Fatalf("RelayList: %v", err)
	}
	if raw == nil {
		t.Error("expected non-nil result")
	}
}
