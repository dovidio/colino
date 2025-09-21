package setup

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"gopkg.in/yaml.v3"

	"golino/internal/config"
	"golino/internal/launchd"
)

type userConfig struct {
	RSSFeeds     []string
	IntervalMin  int
	WebshareUser string
	WebsharePass string
	YTNameByURL  map[string]string
}

// Run executes the interactive setup flow:
// 1) greet
// 2) ask for RSS feeds
// 3) ask for ingestion interval
// 4) ask for optional Webshare proxy
// 5) mention YouTube channel feed URLs
// 6) write config, bootstrap DB, and install daemon (macOS)
func Run(ctx context.Context) error {
	println("\nWelcome to Colino setup! 🚀")
	println("This wizard will set up your feeds, config, MCP integration, and background ingestion.")

	uc := userConfig{IntervalMin: 30}
	rdr := bufio.NewReader(os.Stdin)

	cfgPath := configPath()
	cfgExists := fileExists(cfgPath)
	override := true
	if cfgExists {
		fmt.Printf("\nFound an existing config at %s\n", cfgPath)
		fmt.Print("Do you want to [o]verride it (will create a .bak) or [k]eep it? [o/k]: ")
		ans, _ := rdr.ReadString('\n')
		ans = strings.ToLower(strings.TrimSpace(ans))
		if ans == "k" || ans == "keep" {
			override = false
		}
	}

	// If a launchd agent is running, offer to stop it now (macOS)
	if runtime.GOOS == "darwin" && isLaunchdLoaded("com.colino.daemon") {
		if askYesNo("\nDetected a running Colino launchd agent. Stop it now? [y/N]: ") {
			if err := stopLaunchd("com.colino.daemon"); err != nil {
				fmt.Printf("Warning: failed to stop launchd agent: %v\n", err)
			} else {
				fmt.Println("launchd agent stopped.")
			}
		}
	}

	if override {
		// 2) RSS feeds
		fmt.Println("\nStep 1 – RSS Feeds")
		fmt.Println("Enter one or more RSS feed URLs, separated by commas.")
		fmt.Println("You can add more later by editing ~/.config/colino/config.yaml.")
		fmt.Println("Example YouTube channel feed (Theo): https://www.youtube.com/feeds/videos.xml?channel_id=UCbRP3c757lWg9M-U7TyEkXA")
		fmt.Print("Feeds: ")
		feedsLine, _ := rdr.ReadString('\n')
		feeds := splitCSV(feedsLine)
		uc.RSSFeeds = feeds

		// 3) YouTube channels (optional)
		fmt.Println("\nStep 2 – YouTube Channels (optional)")
		fmt.Println("Connect your YouTube account to import channel feeds from your subscriptions.")
		if askYesNo("Add YouTube channels from your subscriptions now? [y/N]: ") {
			feeds, names := runYouTubeSubscriptionsFlow(rdr)
			if len(feeds) > 0 {
				uc.RSSFeeds = append(uc.RSSFeeds, feeds...)
				if uc.YTNameByURL == nil {
					uc.YTNameByURL = map[string]string{}
				}
				for u, n := range names {
					uc.YTNameByURL[u] = n
				}
				fmt.Printf("Added %d YouTube channel feeds.\n", len(feeds))
			}
		}

		// 4) interval
		fmt.Println("\nStep 3 – Ingestion Interval")
		fmt.Print("How often should ingestion run? Minutes [30]: ")
		intervalLine, _ := rdr.ReadString('\n')
		intervalLine = strings.TrimSpace(intervalLine)
		if v, err := parsePositiveInt(intervalLine); err == nil && v > 0 {
			uc.IntervalMin = v
		}

		// 5) webshare (optional)
		fmt.Println("\nStep 4 – Optional Webshare Proxy")
		fmt.Println("If you ingest many YouTube transcripts, enabling a rotating proxy helps avoid IP blocking.")
		fmt.Println("You can skip this for now (press Enter).")
		fmt.Print("Webshare username: ")
		wsUser, _ := rdr.ReadString('\n')
		uc.WebshareUser = strings.TrimSpace(wsUser)
		if uc.WebshareUser != "" {
			fmt.Print("Webshare password: ")
			wsPass, _ := rdr.ReadString('\n')
			uc.WebsharePass = strings.TrimSpace(wsPass)
		}

		// 6) mention YouTube feed format
		fmt.Println("\nTip – Adding YouTube sources manually")
		fmt.Println("You can subscribe to YouTube channels via RSS feeds like:")
		fmt.Println("  https://www.youtube.com/feeds/videos.xml?channel_id=<CHANNEL_ID>")
		fmt.Println("We’ll add UX for discovery/import later.")

		// backup existing config if overriding
		if cfgExists {
			_ = backupFile(cfgPath)
		}
		// 7) write config file
		if err := writeConfig(uc); err != nil {
			return err
		}
		fmt.Println("\nConfig written to ~/.config/colino/config.yaml")
	} else {
		// Keeping existing config: use its interval as default
		dc, _ := config.LoadDaemonConfig()
		if dc.IntervalMin > 0 {
			uc.IntervalMin = dc.IntervalMin
		}
		fmt.Printf("\nUsing existing config. Ingestion interval minutes [%d]: ", uc.IntervalMin)
		intervalLine, _ := rdr.ReadString('\n')
		intervalLine = strings.TrimSpace(intervalLine)
		if v, err := parsePositiveInt(intervalLine); err == nil && v > 0 {
			uc.IntervalMin = v
		}
	}

	// Skip bootstrapping ingest on setup to avoid long-running step.
	// The launchd agent will handle ingestion in the background.

	// Install daemon (macOS)
	if runtime.GOOS == "darwin" {
		fmt.Println("\nInstalling launchd agent to run on a schedule…")
		exe, _ := os.Executable()
		args := []string{"daemon", "--once"}
		if uc.IntervalMin <= 0 {
			uc.IntervalMin = 30
		}
		// prepare log path
		home, _ := os.UserHomeDir()
		logPath := filepath.Join(home, "Library", "Logs", "Colino", "daemon.launchd.log")
		opt := launchd.InstallOptions{
			Label:           "com.colino.daemon",
			IntervalMinutes: uc.IntervalMin,
			ProgramPath:     exe,
			ProgramArgs:     args,
			StdOutPath:      logPath,
			StdErrPath:      logPath,
		}
		if _, err := launchd.Install(opt); err != nil {
			fmt.Printf("launchd install failed: %v\n", err)
		} else {
			fmt.Println("launchd agent installed and loaded.")
		}
	} else {
		fmt.Println("\nNote: Automatic scheduling is only implemented for macOS (launchd).\nYou can use cron/systemd on your platform to run './colino daemon --once' periodically.")
	}

	// Offer MCP client integration
	maybeConfigureMCP()

	fmt.Println("\nSetup complete! 🎉")
	fmt.Println("- Edit your config at ~/.config/colino/config.yaml to refine settings")
	fmt.Println("- Run './colino server' to expose tools to your LLM via MCP")
	return nil
}

func writeConfig(uc userConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dir := filepath.Join(home, ".config", "colino")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, "config.yaml")

	// Preserve existing database_path if present (avoid clobber). If no file exists, no-op.
	prevDB := ""
	if prev, err := loadExisting(path); err == nil {
		if v, ok := prev["database_path"].(string); ok && strings.TrimSpace(v) != "" {
			prevDB = v
		}
	}

	// Manually render YAML so we can attach comments to YouTube feeds
	var sb strings.Builder
	sb.WriteString("# Colino configuration\n")
	if strings.TrimSpace(prevDB) != "" {
		sb.WriteString(fmt.Sprintf("database_path: %q\n", prevDB))
	}
	// RSS feeds
	if len(uc.RSSFeeds) > 0 {
		sb.WriteString("rss:\n")
		sb.WriteString("  feeds:\n")
		for _, u := range uc.RSSFeeds {
			line := fmt.Sprintf("    - %s", strings.TrimSpace(u))
			if uc.YTNameByURL != nil {
				if name, ok := uc.YTNameByURL[u]; ok && strings.TrimSpace(name) != "" {
					line += fmt.Sprintf("  # YouTube: %s", name)
				}
			}
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	// daemon
	if uc.IntervalMin <= 0 {
		uc.IntervalMin = 30
	}
	sb.WriteString("daemon:\n")
	sb.WriteString("  enabled: true\n")
	sb.WriteString(fmt.Sprintf("  interval_minutes: %d\n", uc.IntervalMin))
	sb.WriteString("  sources:\n")
	sb.WriteString("    - article\n")
	sb.WriteString("    - youtube\n")

	// youtube proxy
	if strings.TrimSpace(uc.WebshareUser) != "" && strings.TrimSpace(uc.WebsharePass) != "" {
		sb.WriteString("youtube:\n")
		sb.WriteString("  proxy:\n")
		sb.WriteString("    enabled: true\n")
		sb.WriteString("    webshare:\n")
		sb.WriteString(fmt.Sprintf("      username: %q\n", uc.WebshareUser))
		sb.WriteString(fmt.Sprintf("      password: %q\n", uc.WebsharePass))
	}

	return os.WriteFile(path, []byte(sb.String()), 0o644)
}

func loadExisting(path string) (map[string]any, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return m, nil
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parsePositiveInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty")
	}
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	if err != nil || n <= 0 {
		return 0, errors.New("invalid int")
	}
	return n, nil
}

// Helpers
func configPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "colino", "config.yaml")
}

func fileExists(p string) bool {
	if p == "" {
		return false
	}
	if _, err := os.Stat(p); err == nil {
		return true
	}
	return false
}

func backupFile(path string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	ts := time.Now().Format("20060102-150405")
	bak := path + ".bak-" + ts
	return os.WriteFile(bak, b, 0o644)
}

func maybeConfigureMCP() {
	exe, _ := os.Executable()
	// Try Claude CLI first
	if _, err := exec.LookPath("claude"); err == nil {
		if askYesNo("\nDetected Claude CLI. Add Colino MCP via 'claude mcp add'? [y/N]: ") {
			if err := runClaudeCLIAdd(exe); err != nil {
				fmt.Printf("Failed to add MCP via Claude CLI: %v\nFalling back to config file detection...\n", err)
			}
		}
	}
	// Codex
	codexPath := pathIfExists(filepath.Join(userHome(), ".codex", "config.toml"))

	if codexPath != "" {
		b, err := os.ReadFile(codexPath)
		if err == nil || !strings.Contains(string(b), "[mcp_servers.colino]") {
			if askYesNo("\nDetected ~/.codex/config.toml. Add Colino MCP there? [y/N]: ") {
				_ = backupFile(codexPath)
				_ = appendTomlMCP(codexPath, exe)
				fmt.Println("Added MCP server to ~/.codex/config.toml")
			}
		}
	}
}

func userHome() string { h, _ := os.UserHomeDir(); return h }

func askYesNo(prompt string) bool {
	fmt.Print(prompt)
	rdr := bufio.NewReader(os.Stdin)
	s, _ := rdr.ReadString('\n')
	s = strings.ToLower(strings.TrimSpace(s))
	return s == "y" || s == "yes"
}

func appendTomlMCP(path, exe string) error {
	snippet := fmt.Sprintf("\n[mcp_servers.colino]\ncommand = \"%s\"\nargs = [\"server\"]\nenv = {}\n", exe)
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(snippet)
	return err
}

func addClaudeJSONMCP(path, exe string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var m map[string]any
	if len(b) == 0 {
		m = map[string]any{}
	} else if err := json.Unmarshal(b, &m); err != nil {
		return err
	}
	ms, _ := m["mcpServers"].(map[string]any)
	if ms == nil {
		ms = map[string]any{}
	}
	if _, exists := ms["colino"]; !exists {
		ms["colino"] = map[string]any{
			"command": exe,
			"args":    []string{"server"},
			"env":     map[string]string{},
		}
	}
	m["mcpServers"] = ms
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

func pathIfExists(p string) string {
	if fileExists(p) {
		return p
	}
	return ""
}

func runClaudeCLIAdd(exe string) error {
	// claude mcp add <name> <command> [args...]
	cmd := exec.Command("claude", "mcp", "add", "colino", exe, "server")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// macOS launchd helpers
func isLaunchdLoaded(label string) bool {
	if label == "" {
		return false
	}
	cmd := exec.Command("launchctl", "list", label)
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func stopLaunchd(label string) error {
	if runtime.GOOS != "darwin" {
		return nil
	}
	plist, err := launchd.DefaultAgentPath(label)
	if err != nil {
		return err
	}
	// unload without removing the plist file
	return exec.Command("launchctl", "unload", "-w", plist).Run()
}

// ---------------- YouTube onboarding ----------------
func oauthBaseURL() string {
	if v := strings.TrimSpace(os.Getenv("COLINO_OAUTH_BASE")); v != "" {
		return v
	}
	return "https://colino.umberto.xyz"
}

type oauthInitiateResp struct {
	AuthURL string `json:"auth_url"`
	FlowID  string `json:"flow_id"`
}

type oauthPollResp struct {
	Status       string `json:"status"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	Error        string `json:"error"`
}

type ytChannel struct {
	ID    string
	Title string
}

func runYouTubeSubscriptionsFlow(rdr *bufio.Reader) ([]string, map[string]string) {
	flow, err := initiateOAuth()
	if err != nil || strings.TrimSpace(flow.FlowID) == "" || strings.TrimSpace(flow.AuthURL) == "" {
		msg := "unknown error"
		if err != nil {
			msg = err.Error()
		}
		fmt.Printf("YouTube auth initiation failed: %s\n", msg)
		return nil, nil
	}
	// Open browser for consent and also print the URL for manual copy
	fmt.Printf("Open the following URL to authorize Colino with Google (opening browser):\n%s\n", flow.AuthURL)
	_ = openBrowser(flow.AuthURL)
	fmt.Println("Waiting for authorization… (Ctrl+C to cancel)")
	tok, err := pollOAuth(flow.FlowID, 120*time.Second)
	if err != nil || strings.TrimSpace(tok.AccessToken) == "" {
		fmt.Printf("OAuth flow failed: %v\n", err)
		return nil, nil
	}
	// Fetch subscriptions from YouTube API
	chans, err := fetchYouTubeSubscriptions(tok.AccessToken)
	if err != nil || len(chans) == 0 {
		fmt.Printf("Could not fetch YouTube subscriptions: %v\n", err)
		return nil, nil
	}
	sort.Slice(chans, func(i, j int) bool { return strings.ToLower(chans[i].Title) < strings.ToLower(chans[j].Title) })

	// Interactive selection
	selected := selectYouTubeChannels(rdr, chans)
	if len(selected) == 0 {
		return nil, nil
	}
	// Build feed URLs with names
	feeds := make([]string, 0, len(selected))
	names := make(map[string]string)
	for _, ch := range selected {
		url := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", ch.ID)
		feeds = append(feeds, url)
		names[url] = ch.Title
	}
	return feeds, names
}

func initiateOAuth() (oauthInitiateResp, error) {
	base := oauthBaseURL()
	// Allow path override; otherwise try common defaults (Python client compatibility first).
	paths := []string{"/auth/initiate", "/auth_initiate", "/initiate"}
	if p := strings.TrimSpace(os.Getenv("COLINO_OAUTH_INITIATE_PATH")); p != "" {
		// use only the explicit override if provided
		if !strings.HasPrefix(p, "/") {
			p = "/" + p
		}
		paths = []string{p}
	}

	var lastErr error
	for _, p := range paths {
		u := strings.TrimRight(base, "/") + p
		req, _ := http.NewRequest(http.MethodGet, u, nil)
		cli := &http.Client{Timeout: 15 * time.Second}
		resp, err := cli.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
			resp.Body.Close()
			lastErr = fmt.Errorf("%s status %d: %s", strings.TrimPrefix(p, "/"), resp.StatusCode, strings.TrimSpace(string(body)))
			continue
		}
		// Tolerate different field names
		var raw map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
			resp.Body.Close()
			lastErr = err
			continue
		}
		resp.Body.Close()
		getStr := func(keys ...string) string {
			for _, k := range keys {
				if v, ok := raw[k]; ok {
					if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
						return s
					}
				}
			}
			return ""
		}
		authURL := getStr("auth_url", "authorization_url", "url", "authorize_url")
		// Accept a variety of session/flow identifiers.
		flowID := getStr("flow_id", "flow", "id", "state", "request_id", "token", "session_id")
		if strings.TrimSpace(authURL) == "" || strings.TrimSpace(flowID) == "" {
			lastErr = errors.New("missing auth_url/flow_id in response")
			continue
		}
		return oauthInitiateResp{AuthURL: authURL, FlowID: flowID}, nil
	}
	if lastErr == nil {
		lastErr = errors.New("failed to initiate oauth (no endpoints tried)")
	}
	return oauthInitiateResp{}, lastErr
}

func pollOAuth(flowID string, timeout time.Duration) (oauthPollResp, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		base := oauthBaseURL()
		// Try both common param names; allow overrides via env.
		params := []string{"session_id", "flow_id"}
		if v := strings.TrimSpace(os.Getenv("COLINO_OAUTH_POLL_PARAM")); v != "" {
			params = []string{v}
		}
		// Allow path override; otherwise try common defaults for path base.
		pollPaths := []string{"/auth/poll", "/poll"}
		if p := strings.TrimSpace(os.Getenv("COLINO_OAUTH_POLL_PATH")); p != "" {
			if !strings.HasPrefix(p, "/") {
				p = "/" + p
			}
			pollPaths = []string{p}
		}

		// Build list of candidate URLs to try in this iteration: path style first, then query params.
		var urls []string
		for _, pollPath := range pollPaths {
			// Path parameter style: /poll/{session_id}
			urls = append(urls, strings.TrimRight(base, "/")+pollPath+"/"+neturl.PathEscape(flowID))
			// Query parameter styles
			for _, param := range params {
				urls = append(urls, strings.TrimRight(base, "/")+pollPath+"?"+param+"="+neturl.QueryEscape(flowID))
			}
		}

		for _, u := range urls {
			req, _ := http.NewRequest(http.MethodGet, u, nil)
			cli := &http.Client{Timeout: 10 * time.Second}
			resp, err := cli.Do(req)
			if err != nil || resp == nil {
				continue
			}
			var pr oauthPollResp
			_ = json.NewDecoder(resp.Body).Decode(&pr)
			resp.Body.Close()
			// Success: HTTP 200 with access_token present
			if resp.StatusCode == http.StatusOK && strings.TrimSpace(pr.AccessToken) != "" {
				return pr, nil
			}
			// Pending: HTTP 202 or explicit pending status
			if resp.StatusCode == http.StatusAccepted || strings.EqualFold(strings.TrimSpace(pr.Status), "pending") {
				// try next URL or wait and retry loop
				continue
			}
			// Error: surface server-provided error if present
			if strings.TrimSpace(pr.Error) != "" {
				return oauthPollResp{}, errors.New(pr.Error)
			}
		}
		time.Sleep(2 * time.Second)
	}
	return oauthPollResp{}, errors.New("authorization timed out")
}

func fetchYouTubeSubscriptions(accessToken string) ([]ytChannel, error) {
	var out []ytChannel
	base := "https://www.googleapis.com/youtube/v3/subscriptions?mine=true&part=snippet&maxResults=50"
	pageToken := ""
	cli := &http.Client{Timeout: 20 * time.Second}
	for {
		url := base
		if pageToken != "" {
			url += "&pageToken=" + neturl.QueryEscape(pageToken)
		}
		req, _ := http.NewRequest(http.MethodGet, url, nil)
		req.Header.Set("Authorization", "Bearer "+accessToken)
		resp, err := cli.Do(req)
		if err != nil {
			return out, err
		}
		var body struct {
			NextPageToken string `json:"nextPageToken"`
			Items         []struct {
				Snippet struct {
					Title      string `json:"title"`
					ResourceID struct {
						ChannelID string `json:"channelId"`
					} `json:"resourceId"`
				} `json:"snippet"`
			} `json:"items"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
			resp.Body.Close()
			return out, err
		}
		resp.Body.Close()
		for _, it := range body.Items {
			id := strings.TrimSpace(it.Snippet.ResourceID.ChannelID)
			title := strings.TrimSpace(it.Snippet.Title)
			if id != "" && title != "" {
				out = append(out, ytChannel{ID: id, Title: title})
			}
		}
		if strings.TrimSpace(body.NextPageToken) == "" {
			break
		}
		pageToken = body.NextPageToken
	}
	return out, nil
}

func selectYouTubeChannels(rdr *bufio.Reader, list []ytChannel) []ytChannel {
	if len(list) == 0 {
		return nil
	}
	if sel := selectYouTubeChannelsBubbleTea(list); len(sel) > 0 {
		return sel
	}
	return selectYouTubeChannelsLegacy(rdr, list)
}

// Bubble Tea model and TUI for multi-selecting channels with basic filtering.
type ytSelectModel struct {
	items     []ytChannel
	filtered  []int        // indices into items
	selected  map[int]bool // key: original index in items
	cursor    int
	filtering bool
	filter    string
	width     int
	height    int
	confirmed bool
	cancelled bool
}

func newYTSelectModel(list []ytChannel) *ytSelectModel {
	m := &ytSelectModel{
		items:    list,
		selected: make(map[int]bool),
	}
	// initialize filtered to all
	m.filtered = make([]int, len(list))
	for i := range list {
		m.filtered[i] = i
	}
	return m
}

func (m *ytSelectModel) Init() tea.Cmd { return nil }

func (m *ytSelectModel) applyFilter() {
	q := strings.ToLower(strings.TrimSpace(m.filter))
	if q == "" {
		m.filtered = make([]int, len(m.items))
		for i := range m.items {
			m.filtered[i] = i
		}
		if m.cursor >= len(m.filtered) {
			m.cursor = 0
		}
		return
	}
	m.filtered = m.filtered[:0]
	for i, ch := range m.items {
		if strings.Contains(strings.ToLower(ch.Title), q) || strings.Contains(strings.ToLower(ch.ID), q) {
			m.filtered = append(m.filtered, i)
		}
	}
	if len(m.filtered) == 0 {
		m.cursor = 0
	} else if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
}

func (m *ytSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.KeyMsg:
		if m.filtering {
			switch msg.Type {
			case tea.KeyEnter:
				m.filtering = false
				m.applyFilter()
			case tea.KeyEsc:
				m.filter = ""
				m.filtering = false
				m.applyFilter()
			case tea.KeyBackspace:
				if len(m.filter) > 0 {
					m.filter = m.filter[:len(m.filter)-1]
				}
			case tea.KeyRunes:
				if len(msg.Runes) > 0 && !msg.Alt {
					m.filter += string(msg.Runes)
				}
			}
			return m, nil
		}
		switch msg.Type {
		case tea.KeyCtrlC:
			m.cancelled = true
			return m, tea.Quit
		case tea.KeyEnter:
			m.confirmed = true
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case tea.KeySpace:
			if len(m.filtered) > 0 {
				orig := m.filtered[m.cursor]
				if m.selected[orig] {
					delete(m.selected, orig)
				} else {
					m.selected[orig] = true
				}
			}
		case tea.KeyRunes:
			s := strings.ToLower(string(msg.Runes))
			switch s {
			case "/":
				m.filtering = true
			case "a":
				for _, orig := range m.filtered {
					m.selected[orig] = true
				}
			case "n":
				m.selected = make(map[int]bool)
			case "j":
				if m.cursor < len(m.filtered)-1 {
					m.cursor++
				}
			case "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "q":
				m.cancelled = true
				return m, tea.Quit
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *ytSelectModel) View() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "Select YouTube channels (↑/↓ or j/k, space=toggle, a=all, n=none, /=filter, Enter=confirm, q=quit)")
	if m.filtering {
		fmt.Fprintf(b, "Filter: %s\n", m.filter)
	} else if strings.TrimSpace(m.filter) != "" {
		fmt.Fprintf(b, "Filter: %s (press / to edit)\n", m.filter)
	}
	maxRows := len(m.filtered)
	if m.height > 5 {
		if r := m.height - 5; r < maxRows {
			maxRows = r
		}
	}
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}
	end := start + maxRows
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	for i := start; i < end; i++ {
		orig := m.filtered[i]
		ch := m.items[orig]
		cursor := " "
		if i == m.cursor {
			cursor = ">"
		}
		mark := "[ ]"
		if m.selected[orig] {
			mark = "[x]"
		}
		title := ch.Title
		if m.width > 10 && len(title) > m.width-10 {
			title = title[:m.width-10]
		}
		fmt.Fprintf(b, "%s %s %s (%s)\n", cursor, mark, title, ch.ID)
	}
	return b.String()
}

func selectYouTubeChannelsBubbleTea(list []ytChannel) []ytChannel {
	m := newYTSelectModel(list)
	p := tea.NewProgram(m)
	res, err := p.StartReturningModel()
	if err != nil {
		return nil
	}
	wm, ok := res.(*ytSelectModel)
	if !ok {
		return nil
	}
	if wm.cancelled || !wm.confirmed {
		return nil
	}
	var out []ytChannel
	for idx := range wm.selected {
		if idx >= 0 && idx < len(wm.items) {
			out = append(out, wm.items[idx])
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.ToLower(out[i].Title) < strings.ToLower(out[j].Title) })
	return out
}

func selectYouTubeChannelsLegacy(rdr *bufio.Reader, list []ytChannel) []ytChannel {
	filtered := list
	for {
		fmt.Printf("\nYou have %d subscriptions.\n", len(list))
		// Show first 20
		max := 20
		if len(filtered) < max {
			max = len(filtered)
		}
		fmt.Println("Showing first", max, "matches:")
		for i := 0; i < max; i++ {
			fmt.Printf("%2d) %s\n", i+1, filtered[i].Title)
		}
		fmt.Println("\nOptions: type 'all' to add all shown, '/term' to search, 'skip' to cancel, or select by numbers (e.g., 1,3,5-7). Press Enter to show all again.")
		fmt.Print("> ")
		line, _ := rdr.ReadString('\n')
		line = strings.TrimSpace(line)
		if line == "" {
			filtered = list
			continue
		}
		if strings.EqualFold(line, "skip") {
			return nil
		}
		if strings.EqualFold(line, "all") {
			return filtered
		}
		if strings.HasPrefix(line, "/") {
			q := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "/")))
			if q == "" {
				filtered = list
			} else {
				tmp := make([]ytChannel, 0, len(list))
				for _, ch := range list {
					if strings.Contains(strings.ToLower(ch.Title), q) {
						tmp = append(tmp, ch)
					}
				}
				filtered = tmp
			}
			continue
		}
		// parse index selection
		picks := parseIndexList(line)
		if len(picks) == 0 {
			fmt.Println("No valid selections.")
			continue
		}
		var sel []ytChannel
		for _, idx := range picks {
			if idx >= 1 && idx <= len(filtered) {
				sel = append(sel, filtered[idx-1])
			}
		}
		if len(sel) > 0 {
			return sel
		}
		fmt.Println("No valid selections.")
	}
}

func parseIndexList(s string) []int {
	var out []int
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.Contains(p, "-") {
			ab := strings.SplitN(p, "-", 2)
			if len(ab) != 2 {
				continue
			}
			a, _ := parsePositiveInt(ab[0])
			b, _ := parsePositiveInt(ab[1])
			if a > 0 && b >= a {
				for i := a; i <= b; i++ {
					out = append(out, i)
				}
			}
		} else {
			n, err := parsePositiveInt(p)
			if err == nil && n > 0 {
				out = append(out, n)
			}
		}
	}
	return out
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return nil
	}
}
