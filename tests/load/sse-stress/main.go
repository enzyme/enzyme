// SSE stress test — holds thousands of concurrent SSE connections while
// generating realistic traffic (messages, reactions, typing) and measuring
// event fan-out to all subscribers.
//
// Usage:
//
//	go run ./tests/load/sse-stress -connections 10000
//	go run ./tests/load/sse-stress -connections 10000 -base-url https://chat.enzyme.im
package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	baseURL     = flag.String("base-url", "http://localhost:8080", "Server base URL")
	connections = flag.Int("connections", 10000, "Number of concurrent SSE connections")
	rampRate    = flag.Int("ramp-rate", 500, "Connections to open per second during ramp-up")
	password    = flag.String("password", "password", "Password for seed users")
	duration    = flag.Duration("duration", 2*time.Minute, "How long to hold connections after ramp-up")
	msgRate     = flag.Int("msg-rate", 5, "Messages per second per user during activity phase")
	forceH1     = flag.Bool("h1", false, "Force HTTP/1.1 on SSE connections (disable HTTP/2 client-side)")

	// SSE counters
	sseConnected    atomic.Int64
	sseDisconnected atomic.Int64
	sseErrors       atomic.Int64
	sseEvents       atomic.Int64
	sseReconnects   atomic.Int64

	// Activity counters
	msgSent       atomic.Int64
	msgErrors     atomic.Int64
	reactionsSent atomic.Int64
	typingSent    atomic.Int64

	// Latency tracking: time from message send to SSE event receipt
	// We embed a timestamp in message content and check it on receive
	latencySamples []time.Duration
	latencyMu      sync.Mutex

	// Peak connected tracking
	peakConnected atomic.Int64
)

var seedUsers = []string{
	"alice@example.com",
	"bob@example.com",
	"carol@example.com",
	"dave@example.com",
	"eve@example.com",
	"frank@example.com",
	"grace@example.com",
	"hank@example.com",
}

var emojis = []string{"+1", "heart", "rocket", "eyes", "fire", "tada", "100", "wave"}

type userContext struct {
	email       string
	token       string
	workspaceID string
	channelIDs  []string // public channels
}

func main() {
	flag.Parse()
	log.SetFlags(log.Ltime | log.Lmicroseconds)

	users := loginUsers()
	if len(users) == 0 {
		log.Fatal("No users could be logged in")
	}
	log.Printf("Logged in %d users with %d channels", len(users), len(users[0].channelIDs))
	if *forceH1 {
		log.Println("Client protocol: HTTP/1.1 (forced)")
	} else {
		log.Println("Client protocol: HTTP/2 (default)")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})

	var wg sync.WaitGroup

	go reportStatus(done)

	// Phase 1: Ramp up SSE connections
	log.Printf("Phase 1: Ramping %d SSE connections at %d/sec...", *connections, *rampRate)
	rampStart := time.Now()
	ticker := time.NewTicker(time.Second / time.Duration(*rampRate))
	launched := 0

	for launched < *connections {
		select {
		case <-stop:
			ticker.Stop()
			close(done)
			wg.Wait()
			printSummary(time.Since(rampStart))
			return
		case <-ticker.C:
			user := users[launched%len(users)]
			wg.Add(1)
			go holdSSEConnection(user, done, &wg)
			launched++
		}
	}
	ticker.Stop()
	log.Printf("Ramp-up complete in %s — %d connections launched",
		time.Since(rampStart).Round(time.Millisecond), launched)

	// Wait for all connections to establish
	time.Sleep(2 * time.Second)
	log.Printf("All connections established: %d", sseConnected.Load())

	// Phase 2: Generate activity while connections are held
	log.Printf("Phase 2: Generating activity (%d msg/sec/user × %d users = %d msg/sec total) for %s...",
		*msgRate, len(users), *msgRate*len(users), *duration)

	activityDone := make(chan struct{})
	for _, user := range users {
		wg.Add(1)
		go generateActivity(user, activityDone, &wg)
	}

	select {
	case <-stop:
		log.Println("Interrupted")
	case <-time.After(*duration):
		log.Println("Duration elapsed")
	}

	// Stop activity first, then SSE connections
	close(activityDone)
	time.Sleep(time.Second) // let final events propagate
	close(done)

	log.Println("Shutting down, waiting for connections to close...")
	wg.Wait()
	printSummary(time.Since(rampStart))
}

func loginUsers() []userContext {
	var users []userContext
	client := &http.Client{Timeout: 10 * time.Second}

	for _, email := range seedUsers {
		token, err := login(client, email)
		if err != nil {
			log.Printf("Login failed for %s: %v", email, err)
			continue
		}

		wsID, channelIDs, err := getUserContext(client, token)
		if err != nil {
			log.Printf("Failed to get context for %s: %v", email, err)
			continue
		}

		users = append(users, userContext{
			email:       email,
			token:       token,
			workspaceID: wsID,
			channelIDs:  channelIDs,
		})
	}
	return users
}

func login(client *http.Client, email string) (string, error) {
	body, _ := json.Marshal(map[string]string{
		"email":    email,
		"password": *password,
	})
	resp, err := client.Post(*baseURL+"/api/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, b)
	}
	var result struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	return result.Token, nil
}

func getUserContext(client *http.Client, token string) (string, []string, error) {
	// Get workspace ID from /auth/me
	req, _ := http.NewRequest("GET", *baseURL+"/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, err
	}
	defer resp.Body.Close()

	var me struct {
		Workspaces []struct {
			ID string `json:"id"`
		} `json:"workspaces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return "", nil, err
	}
	if len(me.Workspaces) == 0 {
		return "", nil, fmt.Errorf("no workspaces")
	}
	wsID := me.Workspaces[0].ID

	// Get channel IDs from /workspaces/{wid}/channels/list
	req2, _ := http.NewRequest("POST", *baseURL+"/api/workspaces/"+wsID+"/channels/list", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	resp2, err := client.Do(req2)
	if err != nil {
		return wsID, nil, err
	}
	defer resp2.Body.Close()

	var chResult struct {
		Channels []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
		} `json:"channels"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&chResult); err != nil {
		return wsID, nil, err
	}

	var channelIDs []string
	for _, ch := range chResult.Channels {
		if ch.Type == "public" {
			channelIDs = append(channelIDs, ch.ID)
		}
	}
	return wsID, channelIDs, nil
}

// generateActivity sends messages, reactions, and typing indicators
func generateActivity(user userContext, done chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	if len(user.channelIDs) == 0 {
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	interval := time.Second / time.Duration(*msgRate)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	iter := 0
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			channelID := user.channelIDs[rand.Intn(len(user.channelIDs))]

			// Send a message (every iteration)
			msgID := sendMessage(client, user, channelID, iter)
			iter++

			// Add a reaction (30% of the time, to our own message)
			if msgID != "" && rand.Float64() < 0.3 {
				addReaction(client, user, msgID)
			}

			// Send typing indicator (20% of the time)
			if rand.Float64() < 0.2 {
				sendTyping(client, user, channelID)
			}
		}
	}
}

func sendMessage(client *http.Client, user userContext, channelID string, iter int) string {
	// Embed a nanotime marker so we can measure fan-out latency
	content := fmt.Sprintf("t=%d Load test from %s #%d", time.Now().UnixNano(), user.email, iter)
	body, _ := json.Marshal(map[string]string{"content": content})

	req, _ := http.NewRequest("POST", *baseURL+"/api/channels/"+channelID+"/messages/send", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+user.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		msgErrors.Add(1)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		msgErrors.Add(1)
		io.Copy(io.Discard, resp.Body)
		return ""
	}

	msgSent.Add(1)

	var result struct {
		Message struct {
			ID string `json:"id"`
		} `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return ""
	}
	return result.Message.ID
}

func addReaction(client *http.Client, user userContext, messageID string) {
	emoji := emojis[rand.Intn(len(emojis))]
	body, _ := json.Marshal(map[string]string{"emoji": emoji})

	req, _ := http.NewRequest("POST", *baseURL+"/api/messages/"+messageID+"/reactions/add", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+user.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == 200 {
		reactionsSent.Add(1)
	}
}

func sendTyping(client *http.Client, user userContext, channelID string) {
	body, _ := json.Marshal(map[string]string{"channel_id": channelID})

	req, _ := http.NewRequest("POST", *baseURL+"/api/workspaces/"+user.workspaceID+"/typing/start", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+user.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode == 200 {
		typingSent.Add(1)
	}
}

func holdSSEConnection(user userContext, done chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	backoff := 500 * time.Millisecond
	const maxBackoff = 10 * time.Second

	for {
		select {
		case <-done:
			return
		default:
		}

		wasConnected := connectAndHold(user, done)

		select {
		case <-done:
			return
		default:
			sseReconnects.Add(1)
			if wasConnected {
				backoff = 500 * time.Millisecond // reset on successful connection
			} else {
				backoff = min(backoff*2, maxBackoff) // exponential backoff on failure
			}
			time.Sleep(backoff)
		}
	}
}

// newSSETransport creates a fresh transport for each SSE connection.
// Each gets its own TCP connection, simulating real browsers where
// each user has a separate TCP connection to the server.
func newSSETransport() *http.Transport {
	t := &http.Transport{
		TLSHandshakeTimeout: 10 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		// Disable connection pooling — this transport is single-use
		DisableKeepAlives: false,
		MaxIdleConns:      1,
		MaxConnsPerHost:   1,
	}
	if *forceH1 {
		// Disable HTTP/2 on the client side
		t.TLSNextProto = make(map[string]func(authority string, c *tls.Conn) http.RoundTripper)
	}
	return t
}

func connectAndHold(user userContext, done chan struct{}) (wasConnected bool) {
	url := fmt.Sprintf("%s/api/workspaces/%s/events", *baseURL, user.workspaceID)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+user.token)
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")

	transport := newSSETransport()
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 0}
	resp, err := client.Do(req)
	if err != nil {
		sseErrors.Add(1)
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		sseErrors.Add(1)
		io.Copy(io.Discard, resp.Body)
		return false
	}

	wasConnected = true
	cur := sseConnected.Add(1)
	// Track peak
	for {
		peak := peakConnected.Load()
		if cur <= peak || peakConnected.CompareAndSwap(peak, cur) {
			break
		}
	}
	defer func() {
		sseConnected.Add(-1)
		sseDisconnected.Add(1)
	}()

	scanner := bufio.NewScanner(resp.Body)
	// Increase buffer for large event payloads
	scanner.Buffer(make([]byte, 64*1024), 256*1024)

	// SSE format from this server:
	//   id: <ulid>
	//   data: {"id":"...","type":"message.new","data":{...}}
	//   <blank line>
	for {
		select {
		case <-done:
			return
		default:
		}

		lineCh := make(chan bool, 1)
		go func() {
			lineCh <- scanner.Scan()
		}()

		select {
		case <-done:
			resp.Body.Close()
			return
		case ok := <-lineCh:
			if !ok {
				return
			}
			line := scanner.Text()

			if strings.HasPrefix(line, "data: ") {
				data := line[6:]
				sseEvents.Add(1)

				// Measure latency for message.new events
				if strings.Contains(data, `"type":"message.new"`) && strings.Contains(data, "t=") {
					measureLatency(data)
				}
			}
		}
	}
}

func measureLatency(data string) {
	// Parse the nanotime marker from the message content
	// Content format: "t=<nanotime> Load test from ..."
	idx := strings.Index(data, "t=")
	if idx < 0 {
		return
	}
	// Find the nanotime value — it's embedded in JSON so look for the pattern
	sub := data[idx+2:]
	end := strings.IndexByte(sub, ' ')
	if end < 0 {
		return
	}
	var nanos int64
	if _, err := fmt.Sscanf(sub[:end], "%d", &nanos); err != nil {
		return
	}
	sent := time.Unix(0, nanos)
	latency := time.Since(sent)
	if latency > 0 && latency < 30*time.Second { // sanity check
		latencyMu.Lock()
		latencySamples = append(latencySamples, latency)
		latencyMu.Unlock()
	}
}

func reportStatus(done chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	var lastEvents, lastMsgs int64

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			curEvents := sseEvents.Load()
			curMsgs := msgSent.Load()
			evtRate := (curEvents - lastEvents) / 2
			msgRate := (curMsgs - lastMsgs) / 2
			lastEvents = curEvents
			lastMsgs = curMsgs

			log.Printf(
				"conns=%d  events=%d (%d/s)  msgs=%d (%d/s)  reactions=%d  errors=%d/%d",
				sseConnected.Load(),
				curEvents, evtRate,
				curMsgs, msgRate,
				reactionsSent.Load(),
				sseErrors.Load(), msgErrors.Load(),
			)
		}
	}
}

func printSummary(elapsed time.Duration) {
	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Println("  SSE Stress Test Summary")
	fmt.Println("═══════════════════════════════════════════════════════")
	fmt.Printf("  Duration:          %s\n", elapsed.Round(time.Second))
	fmt.Printf("  Target conns:      %d\n", *connections)
	fmt.Printf("  Peak connected:    %d\n", peakConnected.Load())
	fmt.Println()
	fmt.Println("  Activity:")
	fmt.Printf("    Messages sent:   %d\n", msgSent.Load())
	fmt.Printf("    Message errors:  %d\n", msgErrors.Load())
	fmt.Printf("    Reactions sent:  %d\n", reactionsSent.Load())
	fmt.Printf("    Typing sent:     %d\n", typingSent.Load())
	fmt.Println()
	fmt.Println("  SSE:")
	fmt.Printf("    Events received: %d\n", sseEvents.Load())
	fmt.Printf("    Conn errors:     %d\n", sseErrors.Load())
	fmt.Printf("    Reconnects:      %d\n", sseReconnects.Load())
	fmt.Printf("    Disconnects:     %d\n", sseDisconnected.Load())

	// Latency stats
	latencyMu.Lock()
	samples := latencySamples
	latencyMu.Unlock()

	if len(samples) > 0 {
		var total time.Duration
		min := samples[0]
		max := samples[0]
		for _, s := range samples {
			total += s
			if s < min {
				min = s
			}
			if s > max {
				max = s
			}
		}
		avg := total / time.Duration(len(samples))

		// Sort for percentiles
		sorted := make([]time.Duration, len(samples))
		copy(sorted, samples)
		sortDurations(sorted)

		p50 := sorted[len(sorted)*50/100]
		p95 := sorted[len(sorted)*95/100]
		p99 := sorted[len(sorted)*99/100]

		fmt.Println()
		fmt.Println("  Event fan-out latency (message send → SSE receive):")
		fmt.Printf("    Samples:  %d\n", len(samples))
		fmt.Printf("    Min:      %s\n", min.Round(time.Microsecond))
		fmt.Printf("    Avg:      %s\n", avg.Round(time.Microsecond))
		fmt.Printf("    p50:      %s\n", p50.Round(time.Microsecond))
		fmt.Printf("    p95:      %s\n", p95.Round(time.Microsecond))
		fmt.Printf("    p99:      %s\n", p99.Round(time.Microsecond))
		fmt.Printf("    Max:      %s\n", max.Round(time.Microsecond))
	}

	fmt.Println("═══════════════════════════════════════════════════════")
}

func sortDurations(d []time.Duration) {
	// Simple insertion sort — fine for our sample sizes
	for i := 1; i < len(d); i++ {
		key := d[i]
		j := i - 1
		for j >= 0 && d[j] > key {
			d[j+1] = d[j]
			j--
		}
		d[j+1] = key
	}
}
