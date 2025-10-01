package youtube

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"net/http"
	neturl "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// Minimal Go port of youtube-transcript-api's core: fetch default transcript with optional Webshare proxy.

const (
	watchURL        = "https://www.youtube.com/watch?v=%s"
	innertubeAPIURL = "https://www.youtube.com/youtubei/v1/player?key=%s"
	userAgent       = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0 Safari/537.36"
)

// Android client context used by upstream project.
var innertubeContext = map[string]any{
	"client": map[string]string{
		"clientName":    "ANDROID",
		"clientVersion": "20.10.38",
	},
}

type WebshareProxyConfig struct {
	Username string
	Password string
	Domain   string // defaults to p.webshare.io
	Port     int    // defaults to 80
}

func (w *WebshareProxyConfig) url() string {
	if w == nil || strings.TrimSpace(w.Username) == "" || strings.TrimSpace(w.Password) == "" {
		return ""
	}
	domain := w.Domain
	if domain == "" {
		domain = "p.webshare.io"
	}
	port := w.Port
	if port == 0 {
		port = 80
	}
	return fmt.Sprintf("http://%s-rotate:%s@%s:%d/", w.Username, w.Password, domain, port)
}

// NewHTTPClient returns an http.Client with optional Webshare proxy and sane defaults.
func NewHTTPClient(ws *WebshareProxyConfig, timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	tr := &http.Transport{}
	if ws != nil {
		if p := ws.url(); p != "" {
			if u, err := neturl.Parse(p); err == nil {
				tr.Proxy = http.ProxyURL(u)
			}
		}
		// Rotating proxies: disable keep-alives to encourage rotation
		tr.DisableKeepAlives = true
	}
	return &http.Client{Transport: tr, Timeout: timeout}
}

// Snippet is a single transcript entry.
type Snippet struct {
	Text     string
	StartSec float64
	Duration float64
}

// FetchDefaultTranscript fetches the default transcript for a video ID.
// It mirrors the upstream flow: watch page -> innertube player -> captionTracks -> timedtext XML.
func FetchDefaultTranscript(ctx context.Context, client *http.Client, videoID string, ws *WebshareProxyConfig) ([]Snippet, error) {
	if client == nil {
		client = NewHTTPClient(ws, 30*time.Second)
	}

	// Get watch page; handle consent gate if present.
	htmlStr, cookieHeader, err := fetchWatchHTML(ctx, client, videoID)
	if err != nil {
		return nil, err
	}
	// Extract API key
	apiKey, err := extractAPIKey(htmlStr)
	if err != nil {
		return nil, err
	}
	// Post to innertube player
	data, _, err := postPlayer(ctx, client, apiKey, videoID, cookieHeader)
	if err != nil {
		return nil, err
	}

	// Validate playability and get captionTracks
	baseURL, err := pickCaptionURL(data)
	if err != nil {
		return nil, err
	}
	// Ensure we fetch simple XML (remove fmt=srv3 if present)
	if u, err := neturl.Parse(baseURL); err == nil {
		q := u.Query()
		if q.Get("fmt") == "srv3" {
			q.Del("fmt")
			u.RawQuery = q.Encode()
			baseURL = u.String()
		}
	}

	// Fetch captions XML
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, baseURL, nil)
	req.Header.Set("User-Agent", userAgent)
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("captions fetch failed: status=%d", resp.StatusCode)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseTimedTextXML(b)
}

// fetchWatchHTML fetches the watch page. If consent form detected, sets consent cookie via header and retries once.
func fetchWatchHTML(ctx context.Context, client *http.Client, videoID string) (htmlStr string, cookieHeader string, err error) {
	url := fmt.Sprintf(watchURL, videoID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", "en-US")
	resp, err := client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}
	s := string(b)
	if strings.Contains(s, "action=\"https://consent.youtube.com/s\"") {
		// Try to extract v token and set consent cookie
		v := extractConsentV(s)
		if v == "" {
			return "", "", errors.New("failed to create consent cookie")
		}
		cookieHeader = "CONSENT=YES+" + v
		// Retry fetch with cookie header
		req2, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		req2.Header.Set("User-Agent", userAgent)
		req2.Header.Set("Accept-Language", "en-US")
		req2.Header.Set("Cookie", cookieHeader)
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return "", "", err2
		}
		defer resp2.Body.Close()
		if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
			return "", "", fmt.Errorf("watch page status: %d", resp2.StatusCode)
		}
		b2, err2 := io.ReadAll(resp2.Body)
		if err2 != nil {
			return "", "", err2
		}
		return string(b2), cookieHeader, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", "", fmt.Errorf("watch page status: %d", resp.StatusCode)
	}
	return s, "", nil
}

var apiKeyRe = regexp.MustCompile(`"INNERTUBE_API_KEY"\s*:\s*"([a-zA-Z0-9_-]+)"`)

func extractAPIKey(htmlStr string) (string, error) {
	if m := apiKeyRe.FindStringSubmatch(htmlStr); len(m) == 2 {
		return m[1], nil
	}
	if strings.Contains(htmlStr, "class=\"g-recaptcha\"") {
		return "", errors.New("IP blocked (captcha)")
	}
	return "", errors.New("could not extract INNERTUBE_API_KEY")
}

func postPlayer(ctx context.Context, client *http.Client, apiKey, videoID, cookieHeader string) (map[string]any, int, error) {
	endpoint := fmt.Sprintf(innertubeAPIURL, apiKey)
	payload := map[string]any{
		"context": innertubeContext,
		"videoId": videoID,
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Language", "en-US")
	if cookieHeader != "" {
		req.Header.Set("Cookie", cookieHeader)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	status := resp.StatusCode
	if status == 429 {
		return nil, status, fmt.Errorf("request blocked (429)")
	}
	if status < 200 || status >= 300 {
		return nil, status, fmt.Errorf("player request failed: %d", status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status, err
	}
	var data map[string]any
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, status, err
	}
	// Basic playability checks
	if ps, ok := data["playabilityStatus"].(map[string]any); ok {
		if st, _ := ps["status"].(string); st != "OK" && st != "" {
			reason, _ := ps["reason"].(string)
			return nil, status, fmt.Errorf("video unplayable: %s", reason)
		}
	}
	return data, status, nil
}

func pickCaptionURL(playerData map[string]any) (string, error) {
	capRoot, _ := playerData["captions"].(map[string]any)
	tracklist, _ := capRoot["playerCaptionsTracklistRenderer"].(map[string]any)
	tracks, _ := tracklist["captionTracks"].([]any)
	if len(tracks) == 0 {
		return "", errors.New("transcripts disabled or unavailable")
	}
	// Prefer non-ASR
	first := ""
	for _, it := range tracks {
		t, _ := it.(map[string]any)
		base, _ := t["baseUrl"].(string)
		kind, _ := t["kind"].(string)
		if base == "" {
			continue
		}
		if first == "" {
			first = base
		}
		if strings.TrimSpace(kind) != "asr" {
			return base, nil
		}
	}
	if first != "" {
		return first, nil
	}
	return "", errors.New("no usable caption track found")
}

// parseTimedTextXML parses YouTube timedtext XML into snippets, stripping HTML tags from snippet text.
func parseTimedTextXML(b []byte) ([]Snippet, error) {
	type textEl struct {
		XMLName xml.Name `xml:"text"`
		Start   string   `xml:"start,attr"`
		Dur     string   `xml:"dur,attr"`
		Body    string   `xml:",innerxml"`
	}
	type transcript struct {
		XMLName xml.Name `xml:"transcript"`
		Texts   []textEl `xml:"text"`
	}
	var tx transcript
	if err := xml.Unmarshal(b, &tx); err != nil {
		return nil, err
	}
	var out []Snippet
	for _, t := range tx.Texts {
		txt := stripHTML(html.UnescapeString(t.Body))
		if strings.TrimSpace(txt) == "" {
			continue
		}
		start := parseFloat(t.Start)
		dur := parseFloat(t.Dur)
		out = append(out, Snippet{Text: txt, StartSec: start, Duration: dur})
	}
	if len(out) == 0 {
		return nil, errors.New("empty transcript")
	}
	return out, nil
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	// simple locale-safe parse
	s = strings.TrimSpace(s)
	// ensure valid utf-8
	if !utf8.ValidString(s) {
		return 0
	}
	f, _ := strconvParseFloat(s)
	return f
}

// wrapped to avoid importing strconv twice in patches
func strconvParseFloat(s string) (float64, error) { return strconv.ParseFloat(s, 64) }

// stripHTML removes all tags from a snippet body, preserving text content, converting <br> to spaces.
func stripHTML(s string) string {
	// Replace common breaks with spaces, then drop remaining tags.
	s = strings.ReplaceAll(s, "<br>", " ")
	s = strings.ReplaceAll(s, "<br/>", " ")
	s = strings.ReplaceAll(s, "<br />", " ")
	// Remove all tags
	re := regexp.MustCompile(`<[^>]*>`)
	s = re.ReplaceAllString(s, "")
	// Collapse whitespace
	s = strings.Join(strings.Fields(s), " ")
	return s
}

var consentRe = regexp.MustCompile(`name=\"v\" value=\"(.*?)\"`)

func extractConsentV(htmlStr string) string {
	m := consentRe.FindStringSubmatch(htmlStr)
	if len(m) == 2 {
		return m[1]
	}
	return ""
}

var ytHostRe = regexp.MustCompile(`(?i)(^|\.)youtube\.com$`)

func IsYouTubeURL(u string) bool {
	if strings.TrimSpace(u) == "" {
		return false
	}
	parsed, err := neturl.Parse(u)
	if err != nil {
		return false
	}
	h := strings.ToLower(parsed.Host)
	if h == "youtu.be" {
		return true
	}
	return ytHostRe.MatchString(h)
}
