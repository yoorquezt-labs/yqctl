// Command yqctl is a CLI tool for interacting with the yoorquezt-mev gateway
// over its JSON-RPC WebSocket interface.
//
// Usage:
//
//	yqctl -gw ws://localhost:9099/ws -key mykey <command> [args...]
//
// Commands:
//
//	bundle submit   Submit a bundle to the auction
//	bundle get      Get bundle status by ID
//	auction         Show current auction pool
//	protect submit  Submit a protected transaction
//	protect status  Get protection status for a tx
//	intent submit   Submit an intent
//	intent get      Get an intent by ID
//	relay register  Register a new relay
//	relay list      List registered relays
//	relay stats     Relay marketplace statistics
//	blocks          List recent blocks
//	bundles         List stored bundles
//	health          Gateway health check
//	orderflow summary  Orderflow statistics
//	watch <topic>   Subscribe and stream events (auction, blocks, mempool, protect, intents)
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/yoorquezt-labs/yqctl/pkg/client"
	"github.com/yoorquezt-labs/yqctl/pkg/types"
)

func main() {
	gwURL := flag.String("gw", envOrDefault("YQMEV_GATEWAY", "ws://localhost:9099/ws"), "gateway WebSocket URL")
	apiKey := flag.String("key", envOrDefault("YQMEV_API_KEY", ""), "API key (bearer token)")
	flag.Parse()

	args := flag.Args()
	if len(args) == 0 {
		printUsage()
		os.Exit(1)
	}

	c, err := client.Dial(client.Config{
		GatewayURL: *gwURL,
		APIKey:     *apiKey,
	})
	if err != nil {
		fatal("connect: %v", err)
	}
	defer c.Close()

	switch args[0] {
	case "bundle":
		handleBundle(c, args[1:])
	case "auction":
		handleAuction(c)
	case "protect":
		handleProtect(c, args[1:])
	case "intent":
		handleIntent(c, args[1:])
	case "relay":
		handleRelay(c, args[1:])
	case "blocks":
		handleBlocks(c)
	case "bundles":
		handleBundles(c)
	case "health":
		handleHealth(c)
	case "watch":
		handleWatch(c, args[1:])
	case "orderflow":
		handleOrderflow(c, args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n\n", args[0])
		printUsage()
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// Command handlers
// ---------------------------------------------------------------------------

func handleBundle(c *client.Client, args []string) {
	if len(args) == 0 {
		fatal("usage: yqctl bundle <submit|get> [args...]")
	}

	switch args[0] {
	case "submit":
		fs := flag.NewFlagSet("bundle submit", flag.ExitOnError)
		bid := fs.String("bid", "", "bid in wei (required)")
		txJSON := fs.String("tx", "", "transaction JSON (required)")
		chain := fs.String("chain", "ethereum", "chain name")
		targetBlock := fs.String("target-block", "", "target block hex (optional)")
		fs.Parse(args[1:])

		if *bid == "" || *txJSON == "" {
			fatal("usage: yqctl bundle submit --bid <wei> --tx '<json>'")
		}

		var tx types.TransactionMessage
		if err := json.Unmarshal([]byte(*txJSON), &tx); err != nil {
			fatal("parse --tx: %v", err)
		}
		if tx.Chain == "" {
			tx.Chain = *chain
		}

		bundle := types.BundleMessage{
			Type:         "bundle",
			Transactions: []types.TransactionMessage{tx},
			BidWei:       *bid,
			Timestamp:    time.Now().Unix(),
		}
		if *targetBlock != "" {
			bundle.TargetBlock = *targetBlock
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.SendBundle(ctx, bundle)
		if err != nil {
			fatal("send bundle: %v", err)
		}
		printJSON(res)

	case "get":
		if len(args) < 2 {
			fatal("usage: yqctl bundle get <bundle_id>")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.GetBundle(ctx, args[1])
		if err != nil {
			fatal("get bundle: %v", err)
		}
		printJSON(res)

	default:
		fatal("usage: yqctl bundle <submit|get> [args...]")
	}
}

func handleAuction(c *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.GetAuction(ctx)
	if err != nil {
		fatal("auction: %v", err)
	}
	printJSON(res)
}

func handleProtect(c *client.Client, args []string) {
	if len(args) == 0 {
		fatal("usage: yqctl protect <submit|status> [args...]")
	}

	switch args[0] {
	case "submit":
		fs := flag.NewFlagSet("protect submit", flag.ExitOnError)
		from := fs.String("from", "", "sender address (required)")
		to := fs.String("to", "", "recipient address (required)")
		payload := fs.String("payload", "", "tx payload hex (required)")
		value := fs.String("value", "0", "value in wei")
		chain := fs.String("chain", "ethereum", "chain name")
		fs.Parse(args[1:])

		if *from == "" || *to == "" || *payload == "" {
			fatal("usage: yqctl protect submit --from <addr> --to <addr> --payload <hex>")
		}

		tx := types.ProtectedTransaction{
			From:      *from,
			To:        *to,
			Payload:   *payload,
			Value:     *value,
			Chain:     *chain,
			Timestamp: time.Now().Unix(),
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.ProtectTx(ctx, tx)
		if err != nil {
			fatal("protect submit: %v", err)
		}
		printJSON(res)

	case "status":
		if len(args) < 2 {
			fatal("usage: yqctl protect status <tx_id>")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.GetProtectStatus(ctx, args[1])
		if err != nil {
			fatal("protect status: %v", err)
		}
		printJSON(res)

	default:
		fatal("usage: yqctl protect <submit|status> [args...]")
	}
}

func handleIntent(c *client.Client, args []string) {
	if len(args) == 0 {
		fatal("usage: yqctl intent <submit|get> [args...]")
	}

	switch args[0] {
	case "submit":
		fs := flag.NewFlagSet("intent submit", flag.ExitOnError)
		intentType := fs.String("type", "", "intent type (e.g. swap, limit)")
		chain := fs.String("chain", "ethereum", "chain name")
		tokenIn := fs.String("token-in", "", "input token address")
		tokenOut := fs.String("token-out", "", "output token address")
		amountIn := fs.String("amount-in", "", "input amount")
		minOut := fs.String("min-out", "", "minimum output amount")
		fs.Parse(args[1:])

		if *intentType == "" {
			fatal("usage: yqctl intent submit --type <type> --chain <chain> [flags]")
		}

		intent := map[string]interface{}{
			"type":  *intentType,
			"chain": *chain,
		}
		if *tokenIn != "" {
			intent["token_in"] = *tokenIn
		}
		if *tokenOut != "" {
			intent["token_out"] = *tokenOut
		}
		if *amountIn != "" {
			intent["amount_in"] = *amountIn
		}
		if *minOut != "" {
			intent["min_out"] = *minOut
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.SubmitIntent(ctx, intent)
		if err != nil {
			fatal("intent submit: %v", err)
		}
		printJSON(res)

	case "get":
		if len(args) < 2 {
			fatal("usage: yqctl intent get <intent_id>")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		res, err := c.GetIntent(ctx, args[1])
		if err != nil {
			fatal("intent get: %v", err)
		}
		printJSON(res)

	default:
		fatal("usage: yqctl intent <submit|get> [args...]")
	}
}

func handleRelay(c *client.Client, args []string) {
	if len(args) == 0 {
		fatal("usage: yqctl relay <register|list|stats>")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch args[0] {
	case "register":
		fs := flag.NewFlagSet("relay register", flag.ExitOnError)
		name := fs.String("name", "", "relay name")
		url := fs.String("url", "", "relay endpoint URL (required)")
		owner := fs.String("owner", "", "owner address")
		stake := fs.String("stake", "0", "stake in wei")
		chains := fs.String("chains", "ethereum", "comma-separated chains")
		fs.Parse(args[1:])

		if *url == "" {
			fatal("usage: yqctl relay register -url <endpoint> [-name <name>] [-owner <addr>] [-stake <wei>] [-chains <chains>]")
		}

		params := map[string]interface{}{
			"url":           *url,
			"name":          *name,
			"owner_address": *owner,
			"stake_wei":     *stake,
			"chains":        strings.Split(*chains, ","),
		}
		res, err := c.RelayRegister(ctx, params)
		if err != nil {
			fatal("relay register: %v", err)
		}
		printJSON(res)

	case "list":
		res, err := c.RelayList(ctx)
		if err != nil {
			fatal("relay list: %v", err)
		}
		printJSON(res)

	case "stats":
		res, err := c.RelayStats(ctx)
		if err != nil {
			fatal("relay stats: %v", err)
		}
		printJSON(res)

	default:
		fatal("usage: yqctl relay <register|list|stats>")
	}
}

func handleBlocks(c *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.ListBlocks(ctx)
	if err != nil {
		fatal("blocks: %v", err)
	}
	printJSON(res)
}

func handleBundles(c *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.ListBundles(ctx)
	if err != nil {
		fatal("bundles: %v", err)
	}
	printJSON(res)
}

func handleHealth(c *client.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.Health(ctx)
	if err != nil {
		fatal("health: %v", err)
	}
	printJSON(res)
}

func handleWatch(c *client.Client, args []string) {
	if len(args) == 0 {
		fatal("usage: yqctl watch <auction|blocks|mempool|protect|intents>")
	}

	topic := args[0]
	validTopics := map[string]bool{
		"auction": true, "blocks": true, "mempool": true,
		"protect": true, "intents": true,
	}
	if !validTopics[topic] {
		fatal("unknown topic: %s (valid: auction, blocks, mempool, protect, intents)", topic)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	subID, ch, err := c.Subscribe(ctx, topic)
	if err != nil {
		fatal("subscribe %s: %v", topic, err)
	}

	fmt.Fprintf(os.Stderr, "subscribed to %s (id=%s), streaming events...\n", topic, subID)

	// Handle SIGINT/SIGTERM for clean shutdown.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigCh:
			fmt.Fprintln(os.Stderr, "\nunsubscribing...")
			unsubCtx, unsubCancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = c.Unsubscribe(unsubCtx, subID)
			unsubCancel()
			return
		case event, ok := <-ch:
			if !ok {
				fmt.Fprintln(os.Stderr, "subscription closed")
				return
			}
			// Stream events one per line as compact JSON.
			fmt.Println(string(event))
		}
	}
}

func handleOrderflow(c *client.Client, args []string) {
	if len(args) == 0 || args[0] != "summary" {
		fatal("usage: yqctl orderflow summary")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	res, err := c.OrderflowSummary(ctx)
	if err != nil {
		fatal("orderflow summary: %v", err)
	}
	printJSON(res)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func printJSON(v interface{}) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fatal("json encode: %v", err)
	}
	fmt.Println(string(data))
}

func fatal(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "error: "+format+"\n", args...)
	os.Exit(1)
}

func printUsage() {
	w := os.Stderr
	fmt.Fprintln(w, "yqctl - YoorQuezt MEV gateway CLI")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  yqctl [flags] <command> [args...]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Flags:")
	fmt.Fprintln(w, "  -gw    Gateway WebSocket URL (default: ws://localhost:9099/ws, env: YQMEV_GATEWAY)")
	fmt.Fprintln(w, "  -key   API key / bearer token (env: YQMEV_API_KEY)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "  bundle submit   Submit a bundle (--bid <wei> --tx '<json>')")
	fmt.Fprintln(w, "  bundle get      Get bundle by ID")
	fmt.Fprintln(w, "  auction         Show current auction pool")
	fmt.Fprintln(w, "  protect submit  Submit protected tx (--from --to --payload)")
	fmt.Fprintln(w, "  protect status  Get protection status by tx ID")
	fmt.Fprintln(w, "  intent submit   Submit an intent (--type --chain)")
	fmt.Fprintln(w, "  intent get      Get intent by ID")
	fmt.Fprintln(w, "  relay register  Register a relay (-url <endpoint> [-name] [-owner] [-stake] [-chains])")
	fmt.Fprintln(w, "  relay list      List registered relays")
	fmt.Fprintln(w, "  relay stats     Relay marketplace statistics")
	fmt.Fprintln(w, "  blocks          List recent blocks")
	fmt.Fprintln(w, "  bundles         List stored bundles")
	fmt.Fprintln(w, "  health          Gateway health check")
	fmt.Fprintln(w, "  orderflow summary  Orderflow statistics")
	fmt.Fprintln(w, "  watch <topic>   Stream events (auction, blocks, mempool, protect, intents)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Examples:")
	lines := []string{
		`  yqctl health`,
		`  yqctl auction`,
		`  yqctl bundle submit --bid 1000000 --tx '{"tx_id":"t1","chain":"ethereum","payload":"0xdead"}'`,
		`  yqctl bundle get ABC123`,
		`  yqctl protect submit --from 0xabc --to 0xdef --payload 0x1234`,
		`  yqctl intent submit --type swap --chain ethereum --token-in 0x... --token-out 0x...`,
		`  yqctl relay register -url http://relay1.example.com:8080 -name my-relay -chains ethereum,bsc`,
		`  yqctl relay list`,
		`  yqctl watch auction`,
		`  yqctl orderflow summary`,
	}
	fmt.Fprintln(w, strings.Join(lines, "\n"))
}
