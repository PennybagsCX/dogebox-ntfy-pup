// bridge turns Dogebox pup-state changes into ntfy notifications.
//
// Every 30s it reads dogeboxd's pup list (tokenless first; if dogeboxd
// requires auth it idles until a token is saved via the onboarding page,
// which lands in /storage/config/dogeboxd.json). State transitions are
// pushed transition-only to the dogebox-health topic on the local ntfy
// server, with a 5-minute duplicate-suppression window. Every 15s it posts
// the healthy/subscribers metrics to dogeboxd.
//
// Standard library only.

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	topic        = "dogebox-health"
	pollEvery    = 30 * time.Second
	metricsEvery = 15 * time.Second
	dupWindow    = 5 * time.Minute
)

var (
	stateDir  = envOr("NTFY_STATE_DIR", "/storage/state")
	configDir = envOr("NTFY_CONFIG_DIR", "/storage/config")
	backend   = envOr("NTFY_BACKEND", "http://127.0.0.1:8098")
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// errAuthNeeded marks a poll that failed specifically because dogeboxd
// rejected our (possibly absent) credentials.
var errAuthNeeded = errors.New("dogeboxd requires authentication")

// errShape marks a /pup response that parsed as JSON but matches no known
// shape; its error text carries the raw body so a live fix is one look away.
var errShape = errors.New("unrecognized /pup shape")

type snapshot struct {
	State  string `json:"state"`
	Update bool   `json:"update"`
}

type pupState struct {
	ID     string
	Update bool
	State  string
}

type persisted struct {
	Prev     map[string]snapshot `json:"prev"`
	LastSent map[string]int64    `json:"lastSent"`
}

type notifier struct {
	title, body, priority, tags, dedupeKey string
}

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[bridge] ")

	dbxHost := envOr("DBX_HOST", "127.0.0.1")
	dbxPort := envOr("DBX_PORT", "3000")
	apiBase := "http://" + net.JoinHostPort(dbxHost, dbxPort)

	os.MkdirAll(stateDir, 0o755)
	os.MkdirAll(configDir, 0o755)

	state := loadState()
	client := &http.Client{Timeout: 10 * time.Second}
	firstOK := false
	idleSince := time.Time{}
	shapeSince := time.Time{}

	log.Printf("bridge up: dogeboxd=%s backend=%s topic=%s (poll %s)", apiBase, backend, topic, pollEvery)

	go func() {
		for {
			postMetrics(client, apiBase)
			time.Sleep(metricsEvery)
		}
	}()

	for {
		token := dogeboxdToken()
		pups, err := fetchPups(client, apiBase, token)
		switch {
		case err == nil:
			if !firstOK {
				firstOK = true
				log.Printf("dogeboxd /pup readable: %d pup(s) tracked", len(pups))
			}
			state = processPups(client, state, pups)
			saveState(state)
		case errors.Is(err, errAuthNeeded) && token == "":
			if time.Since(idleSince) > dupWindow {
				log.Printf("dogeboxd requires auth and no token is saved — idling. Paste a Dogebox API token on the onboarding page. Metrics keep posting.")
				idleSince = time.Now()
			}
		default:
			// Shape errors repeat on every poll — log at most once per dup
			// window so a real fix stays visible without spamming every 30s.
			logShape := true
			if errors.Is(err, errShape) {
				logShape = time.Since(shapeSince) >= dupWindow
				if logShape {
					shapeSince = time.Now()
				}
			}
			if logShape {
				log.Printf("poll failed: %v", err)
			}
		}
		time.Sleep(pollEvery)
	}
}

// fetchPups reads dogeboxd's pup list. The response shape is defensively
// parsed (array, object wrapping the list under pups/data/items, or one
// object keyed by pup id) because the exact dogeboxd response was not
// verifiable offline; unrecognized shapes are logged with the raw body so
// a live fix is one look away.
func fetchPups(client *http.Client, apiBase, token string) ([]pupState, error) {
	req, err := http.NewRequest(http.MethodGet, apiBase+"/pup", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("%w (HTTP %d)", errAuthNeeded, resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("dogeboxd returned %d: %.200s", resp.StatusCode, body)
	}
	var parsed any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("bad JSON from dogeboxd: %v (%.200s)", err, body)
	}
	list := parsed
	if m, ok := parsed.(map[string]any); ok {
		for _, key := range []string{"pups", "data", "items"} {
			if v, ok := m[key].([]any); ok {
				list = v
				break
			}
		}
	}
	elems, ok := list.([]any)
	if !ok {
		// Alternative shape: one object keyed by pup id, each value a state
		// object. Re-emit as the list shape and parse identically.
		if m, isMap := parsed.(map[string]any); isMap {
			ids := make([]string, 0, len(m))
			for id, v := range m {
				if _, isObj := v.(map[string]any); isObj {
					ids = append(ids, id)
				}
			}
			if len(ids) > 0 {
				sort.Strings(ids)
				elems = make([]any, 0, len(ids))
				for _, id := range ids {
					obj := m[id].(map[string]any)
					if _, has := obj["id"]; !has {
						obj["id"] = id
					}
					elems = append(elems, obj)
				}
				ok = true
			}
		}
	}
	if !ok {
		return nil, fmt.Errorf("%w (top-level %T): %.300s", errShape, parsed, body)
	}
	var out []pupState
	for _, e := range elems {
		m, ok := e.(map[string]any)
		if !ok {
			continue
		}
		id := firstString(m, "id", "pupId", "pupName", "name", "pup")
		if id == "" {
			continue
		}
		p := pupState{ID: id, State: "unknown"}
		if v, ok := firstValue(m, "state", "status", "stateStr", "runState"); ok {
			switch t := v.(type) {
			case string:
				p.State = t
			case bool:
				p.State = boolState(t)
			}
		}
		if p.State == "unknown" {
			if v, ok := m["running"].(bool); ok {
				p.State = boolState(v)
			}
		}
		if v, ok := firstValue(m, "updateAvailable", "update_available", "hasUpdate"); ok {
			if b, ok := v.(bool); ok {
				p.Update = b
			}
		}
		out = append(out, p)
	}
	return dedupeByID(out), nil
}

// dedupeByID collapses duplicate pup entries defensively so one pup can't
// emit paired crashed/recovered notifications from a single poll; crashed
// outranks running. First-seen order is preserved.
func dedupeByID(in []pupState) []pupState {
	rank := func(s string) int {
		if s == "crashed" {
			return 0
		}
		return 1
	}
	best := map[string]pupState{}
	order := []string{}
	for _, p := range in {
		if _, seen := best[p.ID]; !seen {
			order = append(order, p.ID)
			best[p.ID] = p
			continue
		}
		if rank(p.State) < rank(best[p.ID].State) {
			best[p.ID] = p
		}
	}
	out := make([]pupState, 0, len(order))
	for _, id := range order {
		out = append(out, best[id])
	}
	return out
}

func boolState(b bool) string {
	if b {
		return "running"
	}
	return "stopped"
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

func firstValue(m map[string]any, keys ...string) (any, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}

// classify maps raw dogeboxd state strings onto running/crashed/stopped.
// Anything unlisted counts as stopped — never silently as running.
var (
	runningStates = map[string]bool{
		"running": true, "enabled": true, "up": true, "started": true,
		"active": true, "ok": true, "healthy": true, "online": true,
	}
	crashedStates = map[string]bool{
		"crashed": true, "exited": true, "error": true, "failed": true,
		"dead": true, "unhealthy": true, "failing": true,
	}
)

func classify(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case runningStates[s]:
		return "running"
	case crashedStates[s]:
		return "crashed"
	default:
		return "stopped"
	}
}

// processPups diffs new states against the previous cycle and emits
// transition-only notifications. First sighting of a pup (including the
// very first poll after install/restart) is baseline — no alerts.
func processPups(client *http.Client, st *persisted, pups []pupState) *persisted {
	if st.Prev == nil {
		st.Prev = map[string]snapshot{}
	}
	if st.LastSent == nil {
		st.LastSent = map[string]int64{}
	}
	now := time.Now().Unix()
	for _, p := range pups {
		cur := snapshot{State: p.State, Update: p.Update}
		prev, seen := st.Prev[p.ID]
		if !seen {
			st.Prev[p.ID] = cur
			continue
		}
		var n *notifier
		prevClass, curClass := classify(prev.State), classify(cur.State)
		if prevClass != curClass {
			switch curClass {
			case "crashed":
				n = &notifier{
					title:     p.ID + " crashed",
					body:      fmt.Sprintf("%s crashed on your Dogebox (was %s).", p.ID, prev.State),
					priority:  "4",
					tags:      "skull",
					dedupeKey: p.ID + "|crashed",
				}
			case "running":
				if prevClass == "crashed" {
					n = &notifier{
						title:     p.ID + " recovered",
						body:      fmt.Sprintf("%s is running again.", p.ID),
						priority:  "3",
						tags:      "white_check_mark",
						dedupeKey: p.ID + "|recovered",
					}
				} else {
					n = &notifier{
						title:     p.ID + " started",
						body:      fmt.Sprintf("%s is now running.", p.ID),
						priority:  "2",
						tags:      "rocket",
						dedupeKey: p.ID + "|started",
					}
				}
			case "stopped":
				n = &notifier{
					title:     p.ID + " stopped",
					body:      fmt.Sprintf("%s is no longer running (was %s).", p.ID, prev.State),
					priority:  "2",
					tags:      "no_entry_sign",
					dedupeKey: p.ID + "|stopped",
				}
			}
		} else if cur.Update && !prev.Update {
			n = &notifier{
				title:     "Update available: " + p.ID,
				body:      fmt.Sprintf("A new version of %s is available in the pup store.", p.ID),
				priority:  "2",
				tags:      "package",
				dedupeKey: p.ID + "|update",
			}
		}

		if n != nil {
			if t, dup := st.LastSent[n.dedupeKey]; dup && now-t < int64(dupWindow.Seconds()) {
				log.Printf("suppressed duplicate (%s within %s)", n.dedupeKey, dupWindow)
			} else if publish(client, *n) {
				st.LastSent[n.dedupeKey] = now
			} else {
				// Publish failed: keep the previous snapshot so the
				// transition is retried next cycle.
				continue
			}
		}
		st.Prev[p.ID] = cur
	}
	return st
}

// publish pushes one notification to the local ntfy server with the
// device token generated at first boot.
func publish(client *http.Client, n notifier) bool {
	devToken := deviceToken()
	if devToken == "" {
		log.Printf("no device token at %s — dropping: %s",
			filepath.Join(stateDir, "device-token"), n.title)
		return false
	}
	req, err := http.NewRequest(http.MethodPost, backend+"/"+topic, strings.NewReader(n.body))
	if err != nil {
		log.Printf("publish build failed: %v", err)
		return false
	}
	req.Header.Set("Authorization", "Bearer "+devToken)
	req.Header.Set("Title", n.title)
	req.Header.Set("Priority", n.priority)
	req.Header.Set("Tags", n.tags)
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("publish failed (%s): %v", n.title, err)
		return false
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode/100 != 2 {
		log.Printf("publish got %d for %s", resp.StatusCode, n.title)
		return false
	}
	log.Printf("notified: %s", n.title)
	return true
}

// postMetrics feeds the dashboard cards declared in manifest.json:
// healthy (ntfy reachable) and subscribers (current stream count, written
// by frontdoor). Unauthenticated by design — the same endpoint pattern the
// reference core-txindex monitor uses.
func postMetrics(client *http.Client, apiBase string) {
	healthy := "false"
	resp, err := client.Get(backend + "/v1/health")
	if err == nil {
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			healthy = "true"
		}
	}
	subs := int64(0)
	if b, err := os.ReadFile(filepath.Join(stateDir, "metrics.json")); err == nil {
		var m struct {
			Subscribers int64 `json:"subscribers"`
		}
		if json.Unmarshal(b, &m) == nil {
			subs = m.Subscribers
		}
	}
	payload := fmt.Sprintf(`{"healthy":{"value":%q},"subscribers":{"value":%d}}`, healthy, subs)
	req, err := http.NewRequest(http.MethodPost, apiBase+"/dbx/metrics", bytes.NewReader([]byte(payload)))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		log.Printf("metrics post failed: %v", err)
		return
	}
	io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		log.Printf("metrics post got %d", resp.StatusCode)
	}
}

func deviceToken() string {
	b, err := os.ReadFile(filepath.Join(stateDir, "device-token"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func dogeboxdToken() string {
	b, err := os.ReadFile(filepath.Join(configDir, "dogeboxd.json"))
	if err != nil {
		return ""
	}
	var m struct {
		Token string `json:"token"`
	}
	if json.Unmarshal(b, &m) != nil {
		return ""
	}
	return strings.TrimSpace(m.Token)
}

func statePath() string { return filepath.Join(stateDir, "bridge-state.json") }

func loadState() *persisted {
	st := &persisted{Prev: map[string]snapshot{}, LastSent: map[string]int64{}}
	b, err := os.ReadFile(statePath())
	if err != nil {
		return st
	}
	json.Unmarshal(b, st)
	if st.Prev == nil {
		st.Prev = map[string]snapshot{}
	}
	if st.LastSent == nil {
		st.LastSent = map[string]int64{}
	}
	return st
}

func saveState(st *persisted) {
	blob, err := json.Marshal(st)
	if err != nil {
		log.Printf("saveState marshal failed: %v", err)
		return
	}
	tmp := statePath() + ".tmp"
	if err := os.WriteFile(tmp, blob, 0o600); err != nil {
		log.Printf("saveState write failed: %v", err)
		return
	}
	if err := os.Rename(tmp, statePath()); err != nil {
		log.Printf("saveState rename failed: %v", err)
	}
}
