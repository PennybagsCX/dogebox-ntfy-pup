// frontdoor is the LAN-facing entry point of the ntfy pup.
//
// It binds the pup's exposed port (default :8099) and:
//   - serves the first-run onboarding page at / (subscribe QR, device token,
//     test-notification button, optional bridge-token paste)
//   - serves /healthz (ntfy health check relayed)
//   - accepts the Dogebox API token for the health bridge at /bridge-token
//     (written to /storage/config/dogeboxd.json for the bridge to pick up)
//   - proxies everything else to ntfy on 127.0.0.1:8098, counting in-flight
//     subscriptions (json/sse/ws/raw streams, websocket upgrades) for the
//     dashboard "subscribers" metric
//
// ntfy itself never binds the exposed port; phones, scripts and the
// dashboard's "Launch web" button all land here. Standard library only.

package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	topic   = "dogebox-health"
	brandDS = "#C2A633"
)

var (
	stateDir  = envOr("NTFY_STATE_DIR", "/storage/state")
	configDir = envOr("NTFY_CONFIG_DIR", "/storage/config")
	backend   = envOr("NTFY_BACKEND", "http://127.0.0.1:8098")

	streamCount int64

	tokenRE    = regexp.MustCompile(`[^A-Za-z0-9_-]`)
	hostRE     = regexp.MustCompile(`[^A-Za-z0-9.\-:\[\]]`)
	hiddenPath = func() string { return filepath.Join(stateDir, "credentials-hidden") }
)

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	log.SetFlags(log.LstdFlags)
	log.SetPrefix("[frontdoor] ")

	target, err := url.Parse(backend)
	if err != nil {
		log.Fatalf("bad NTFY_BACKEND %q: %v", backend, err)
	}
	proxy := httputil.NewSingleHostReverseProxy(target)
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, perr error) {
		log.Printf("backend error on %s: %v", r.URL.Path, perr)
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "ntfy is starting up or unreachable — try again in a moment.\n")
	}

	os.MkdirAll(stateDir, 0o755)
	go writeMetricsLoop()

	addr := net.JoinHostPort(os.Getenv("DBX_PUP_IP"), envOr("FRONTDOOR_PORT", "8099"))
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			countingProxy(proxy, w, r)
			return
		}
		serveOnboarding(w, r, false)
	})
	mux.HandleFunc("/__qr.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")
		io.WriteString(w, qrcodeJS)
	})
	mux.HandleFunc("/healthz", handleHealthz)
	mux.HandleFunc("/bridge-token", handleBridgeToken)
	mux.HandleFunc("/onboard/hide", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		// The hide button is a plain HTML form, so browsers stamp cross-site
		// form posts with a foreign Origin — the one CSRF signal checkable
		// without breaking the form. Scripts send no Origin and aren't
		// CSRF-drivable.
		if o := r.Header.Get("Origin"); o != "" {
			if u, err := url.Parse(o); err != nil || !strings.EqualFold(u.Host, r.Host) {
				http.Error(w, "cross-origin request rejected", http.StatusForbidden)
				return
			}
		}
		os.WriteFile(hiddenPath(), []byte("1"), 0o644)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
	mux.HandleFunc("/onboard/reveal", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST only", http.StatusMethodNotAllowed)
			return
		}
		authed, err := tokenAuthentic(r.FormValue("token"))
		if err != nil {
			log.Printf("reveal unavailable: %v", err)
			serveHidden(w, "Could not reach the ntfy server to check that token — try again in a moment.", http.StatusServiceUnavailable)
			return
		}
		if authed {
			serveOnboarding(w, r, true)
			return
		}
		log.Printf("reveal rejected: token failed backend auth check")
		serveHidden(w, "That token was rejected by the server. Paste a working device token.", http.StatusUnauthorized)
	})

	log.Printf("listening on %s (backend %s, topic %s)", addr, backend, topic)
	log.Fatal(http.ListenAndServe(addr, mux))
}

// countingProxy proxies to ntfy while tracking in-flight subscription
// streams so the dashboard "subscribers" metric is real.
func countingProxy(p *httputil.ReverseProxy, w http.ResponseWriter, r *http.Request) {
	if !isStream(r) {
		p.ServeHTTP(w, r)
		return
	}
	atomic.AddInt64(&streamCount, 1)
	defer atomic.AddInt64(&streamCount, -1)
	p.ServeHTTP(&countingWriter{ResponseWriter: w}, r)
}

func isStream(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return true
	}
	path := r.URL.Path
	return strings.HasSuffix(path, "/json") || strings.HasSuffix(path, "/sse") ||
		strings.HasSuffix(path, "/ws") || strings.HasSuffix(path, "/raw")
}

// countingWriter must pass Flusher through (streaming) and Hijacker
// (websocket upgrades) or those requests break behind the counter.
type countingWriter struct {
	http.ResponseWriter
}

func (c *countingWriter) Flush() {
	if f, ok := c.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (c *countingWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := c.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, errors.New("ResponseWriter does not implement Hijacker")
}

// writeMetricsLoop publishes the current subscriber count for the bridge,
// which forwards it to dogeboxd. Atomic rename so readers never see a torn
// file. A vanished subscriber self-corrects within ~2–4 minutes (45s ntfy
// keepalive + TCP keepalive); if frontdoor itself dies the metric freezes
// at its last value until the container is restarted.
func writeMetricsLoop() {
	path := filepath.Join(stateDir, "metrics.json")
	for {
		os.MkdirAll(stateDir, 0o755)
		blob, _ := json.Marshal(map[string]int64{"subscribers": atomic.LoadInt64(&streamCount)})
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, blob, 0o644); err == nil {
			os.Rename(tmp, path)
		}
		time.Sleep(15 * time.Second)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(backend + "/v1/health")
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]bool{"healthy": false})
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		io.Copy(io.Discard, resp.Body)
		writeJSON(w, http.StatusOK, map[string]bool{"healthy": false})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	io.Copy(w, resp.Body)
}

func handleBridgeToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "POST only"})
		return
	}
	// Only the onboarding page's own JS talks to this endpoint. Requiring
	// application/json pushes any cross-site forgery into a CORS preflight
	// frontdoor never answers; a plain form can't produce this content type.
	if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		writeJSON(w, http.StatusUnsupportedMediaType, map[string]string{"error": "Content-Type must be application/json"})
		return
	}
	var in struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad JSON"})
		return
	}
	path := filepath.Join(configDir, "dogeboxd.json")
	token := strings.TrimSpace(in.Token)
	if token == "" {
		os.Remove(path)
		writeJSON(w, http.StatusOK, map[string]bool{"saved": true, "cleared": true})
		return
	}
	// A typo here would silently idle the health bridge, so dogeboxd gets the
	// final say before the token is stored.
	if err := dogeboxdTokenValid(token); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	os.MkdirAll(configDir, 0o755)
	blob, _ := json.Marshal(map[string]string{"token": token})
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("bridge token saved to %s", path)
	writeJSON(w, http.StatusOK, map[string]bool{"saved": true})
}

func readDeviceToken() string {
	b, err := os.ReadFile(filepath.Join(stateDir, "device-token"))
	if err != nil {
		return ""
	}
	return tokenRE.ReplaceAllString(strings.TrimSpace(string(b)), "")
}

func credentialsHidden() bool {
	_, err := os.Stat(hiddenPath())
	return err == nil
}

// tokenAuthentic asks ntfy whether the token can read the health topic, via a
// poll-mode subscription (?poll=1 returns cached messages and closes at once).
// Reveal-gating trusts any valid token — possession of a working device token
// is the owner. ntfy 2.x has no dedicated "check this token" endpoint
// (/v1/auth returns 400; /v1/account returns 200 even anonymously), so a
// poll GET under deny-all IS the check: 200 = authorized, 401/403 = not.
// A non-nil error means the backend could not be reached — the token was
// neither accepted nor rejected, and callers must not say "rejected".
func tokenAuthentic(token string) (bool, error) {
	token = tokenRE.ReplaceAllString(strings.TrimSpace(token), "")
	if token == "" {
		return false, nil
	}
	req, err := http.NewRequest(http.MethodGet, backend+"/"+topic+"/json?poll=1", nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	return resp.StatusCode == http.StatusOK, nil
}

// dogeboxdTokenValid probes dogeboxd with a candidate bridge token before it
// is stored. Without DBX_HOST/DBX_PORT (bare dev runs outside a container)
// there is nothing to probe, so the token is accepted and the bridge's own
// poll logs are the source of truth.
func dogeboxdTokenValid(token string) error {
	host, port := os.Getenv("DBX_HOST"), os.Getenv("DBX_PORT")
	if host == "" || port == "" {
		return nil
	}
	req, err := http.NewRequest(http.MethodGet, "http://"+net.JoinHostPort(host, port)+"/pup", nil)
	if err != nil {
		return fmt.Errorf("could not check that token — try again")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("could not reach dogeboxd to check that token — try again")
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	switch {
	case resp.StatusCode == http.StatusOK:
		return nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return fmt.Errorf("dogeboxd rejected that token — copy it from the dashboard again (devtools → localStorage → storeState.networkContext.token)")
	default:
		return fmt.Errorf("dogeboxd answered HTTP %d while checking that token — try again", resp.StatusCode)
	}
}

// hostAllowed reports whether the Host header names this box the way a LAN
// user reaches it: an IP literal in private or loopback space, "localhost",
// or the configured BASE_URL hostname. A DNS-rebinding page can only attack
// us with a hostname in Host (a browser fetching the box by IP sends that
// IP), so routable hostnames are refused. DBX_PUP_IP is always a private
// container IP, so the private-range check covers it.
func hostAllowed(host string) bool {
	if host == "" {
		return false
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	host = strings.Trim(host, "[]")
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsPrivate() || ip.IsLoopback()
	}
	if base := envOr("BASE_URL", ""); base != "" {
		if u, err := url.Parse(base); err == nil && strings.EqualFold(u.Hostname(), host) {
			return true
		}
	}
	return false
}

func serveOnboarding(w http.ResponseWriter, r *http.Request, forceShow bool) {
	// Credential page is LAN-only (see hostAllowed); a token-authenticated
	// reveal bypasses the gate — possession of a working token is the owner.
	if !forceShow && !hostAllowed(r.Host) {
		serveHidden(w, "", http.StatusOK)
		return
	}
	if credentialsHidden() && !forceShow {
		serveHidden(w, "", http.StatusOK)
		return
	}
	host := hostRE.ReplaceAllString(r.Host, "")
	if host == "" {
		host = hostRE.ReplaceAllString(net.JoinHostPort(os.Getenv("DBX_PUP_IP"), envOr("FRONTDOOR_PORT", "8099")), "")
	}
	hostURL := "http://" + host
	token := readDeviceToken()
	qrData := "ntfy://" + host + "/" + topic
	page := strings.NewReplacer(
		"{{TOPIC}}", topic,
		"{{HOST_URL}}", hostURL,
		"{{TOKEN}}", token,
		"{{TOKEN_JSON}}", mustJSON(token),
		"{{QR_JSON}}", mustJSON(qrData),
		"{{QR_DATA}}", qrData,
		"{{GOLD}}", brandDS,
	).Replace(onboardingHTML)
	// no-store: the page embeds the device token; a cached copy must never
	// resurface from a shared browser cache after credentials are hidden.
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	io.WriteString(w, page)
}

func serveHidden(w http.ResponseWriter, message string, status int) {
	page := strings.NewReplacer(
		"{{MESSAGE}}", message,
		"{{GOLD}}", brandDS,
	).Replace(hiddenHTML)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	io.WriteString(w, page)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func mustJSON(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
