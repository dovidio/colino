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
	textinput "github.com/charmbracelet/bubbles/textinput"
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
    // Launch Bubble Tea wizard to collect inputs
    cfgPath := configPath()
    cfgExists := fileExists(cfgPath)

    wiz := newWizardModel(cfgExists)
    p := tea.NewProgram(wiz)
    res, err := p.StartReturningModel()
    if err != nil {
        return err
    }
    wm, ok := res.(*wizardModel)
    if !ok || wm.cancelled {
        return errors.New("setup cancelled")
    }

    // Build userConfig from wizard result
    uc := userConfig{
        RSSFeeds:     wm.rssFeeds,
        IntervalMin:  wm.interval,
        WebshareUser: wm.wsUser,
        WebsharePass: wm.wsPass,
        YTNameByURL:  wm.ytNameByURL,
    }

    // Write config if overriding or creating new
    if wm.override {
        if cfgExists {
            _ = backupFile(cfgPath)
        }
        if err := writeConfig(uc); err != nil {
            return err
        }
        fmt.Println("\nConfig written to ~/.config/colino/config.yaml")
    }

    // Install daemon (macOS), skip long bootstrap ingest
    if runtime.GOOS == "darwin" {
        fmt.Println("\nInstalling launchd agent to run on a schedule…")
        exe, _ := os.Executable()
        args := []string{"daemon", "--once"}
        interval := wm.interval
        if interval <= 0 {
            // Default or use existing when keeping config
            if !wm.override {
                if dc, err := config.LoadDaemonConfig(); err == nil && dc.IntervalMin > 0 {
                    interval = dc.IntervalMin
                } else {
                    interval = 30
                }
            } else {
                interval = 30
            }
        }
        home, _ := os.UserHomeDir()
        logPath := filepath.Join(home, "Library", "Logs", "Colino", "daemon.launchd.log")
        opt := launchd.InstallOptions{
            Label:           "com.colino.daemon",
            IntervalMinutes: interval,
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

// -------------- Bubble Tea Wizard --------------
type wizardStep int

const (
    stepIntro wizardStep = iota
    stepConfigChoice
    stepRSS
    stepYTAsk
    stepYTAuth
    stepYTSelect
    stepInterval
    stepProxy
    stepSummary
    stepDone
)

type wizardModel struct {
    step      wizardStep
    hasCfg    bool
    override  bool
    cancelled bool

    // RSS
    rssInput textinput.Model
    rssFeeds []string

    // YouTube
    ytWanted bool
    authURL  string
    flowID   string
    polling  bool
    pollErr  string
    channels []ytChannel
    ytSel    *ytSelectModel
    ytNameByURL map[string]string

    // Interval
    intervalInput textinput.Model
    interval      int

    // Proxy
    wsUserInput textinput.Model
    wsPassInput textinput.Model
    wsUser      string
    wsPass      string

    // Status/error
    errMsg string
}

func newWizardModel(hasCfg bool) *wizardModel {
    rss := textinput.New()
    rss.Placeholder = "https://example.com/feed.xml, https://..."
    rss.Focus()

    interval := textinput.New()
    interval.Placeholder = "30"

    wsUser := textinput.New()
    wsUser.Placeholder = "webshare username (optional)"

    wsPass := textinput.New()
    wsPass.Placeholder = "webshare password (optional)"
    wsPass.EchoMode = textinput.EchoPassword
    wsPass.EchoCharacter = '•'

    return &wizardModel{
        step:          stepIntro,
        hasCfg:        hasCfg,
        rssInput:      rss,
        intervalInput: interval,
        wsUserInput:   wsUser,
        wsPassInput:   wsPass,
        interval:      30,
        ytNameByURL:   map[string]string{},
    }
}

func (m *wizardModel) Init() tea.Cmd { return nil }

// Messages for async actions
type initAuthMsg struct{ url, flowID string; err error }
type pollDoneMsg struct{ tok oauthPollResp; err error }
type chansMsg struct{ list []ytChannel; err error }

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Global cancels
        if msg.Type == tea.KeyCtrlC || (msg.Type == tea.KeyRunes && strings.ToLower(string(msg.Runes)) == "q") {
            m.cancelled = true
            return m, tea.Quit
        }
        switch m.step {
        case stepIntro:
            if msg.Type == tea.KeyEnter {
                if m.hasCfg {
                    m.step = stepConfigChoice
                } else {
                    m.override = true
                    m.step = stepRSS
                }
            }
        case stepConfigChoice:
            // o = override, k = keep
            if msg.Type == tea.KeyRunes {
                s := strings.ToLower(string(msg.Runes))
                if s == "o" {
                    m.override = true
                    m.step = stepRSS
                } else if s == "k" {
                    m.override = false
                    // Only ask interval when keeping
                    m.step = stepInterval
                }
            }
        case stepRSS:
            var cmd tea.Cmd
            m.rssInput, cmd = m.rssInput.Update(msg)
            if msg.Type == tea.KeyEnter {
                m.rssFeeds = splitCSV(m.rssInput.Value())
                m.step = stepYTAsk
                return m, nil
            }
            return m, cmd
        case stepYTAsk:
            if msg.Type == tea.KeyRunes {
                s := strings.ToLower(string(msg.Runes))
                if s == "y" {
                    m.ytWanted = true
                    m.step = stepYTAuth
                    return m, m.startInitiate()
                } else if s == "n" {
                    m.ytWanted = false
                    m.step = stepInterval
                }
            }
        case stepYTAuth:
            // allow 'o' to open URL again
            if msg.Type == tea.KeyRunes && strings.ToLower(string(msg.Runes)) == "o" {
                if strings.TrimSpace(m.authURL) != "" {
                    _ = openBrowser(m.authURL)
                }
            }
            // Nothing else, we wait for pollDoneMsg → chansMsg
        case stepYTSelect:
            if m.ytSel != nil {
                var cmd tea.Cmd
                mm, cmd := m.ytSel.Update(msg)
                if sel, ok := mm.(*ytSelectModel); ok {
                    m.ytSel = sel
                }
                if msg.Type == tea.KeyEnter {
                    // collect selection and continue
                    var feeds []string
                    names := map[string]string{}
                    for idx := range m.ytSel.selected {
                        ch := m.channels[idx]
                        url := fmt.Sprintf("https://www.youtube.com/feeds/videos.xml?channel_id=%s", ch.ID)
                        feeds = append(feeds, url)
                        names[url] = ch.Title
                    }
                    if len(feeds) > 0 {
                        m.rssFeeds = append(m.rssFeeds, feeds...)
                        for u, n := range names {
                            m.ytNameByURL[u] = n
                        }
                    }
                    m.step = stepInterval
                }
                return m, cmd
            }
        case stepInterval:
            var cmd tea.Cmd
            m.intervalInput, cmd = m.intervalInput.Update(msg)
            if msg.Type == tea.KeyEnter {
                v := strings.TrimSpace(m.intervalInput.Value())
                if v == "" {
                    m.interval = 30
                } else if n, err := parsePositiveInt(v); err == nil && n > 0 {
                    m.interval = n
                } else {
                    m.errMsg = "Please enter a positive integer (minutes)."
                    return m, cmd
                }
                if m.override {
                    m.step = stepProxy
                } else {
                    m.step = stepSummary
                }
                m.errMsg = ""
                return m, nil
            }
            return m, cmd
        case stepProxy:
            var cmd tea.Cmd
            // route input focus: user → pass
            if m.wsUserInput.Focused() {
                m.wsUserInput, cmd = m.wsUserInput.Update(msg)
                if msg.Type == tea.KeyEnter {
                    m.wsUser = strings.TrimSpace(m.wsUserInput.Value())
                    m.wsPassInput.Focus()
                }
                return m, cmd
            }
            m.wsPassInput, cmd = m.wsPassInput.Update(msg)
            if msg.Type == tea.KeyEnter {
                m.wsUser = strings.TrimSpace(m.wsUserInput.Value())
                m.wsPass = strings.TrimSpace(m.wsPassInput.Value())
                m.step = stepSummary
                return m, nil
            }
            return m, cmd
        case stepSummary:
            if msg.Type == tea.KeyEnter {
                m.step = stepDone
                return m, tea.Quit
            }
        }
    case initAuthMsg:
        if msg.err != nil {
            m.pollErr = msg.err.Error()
            return m, nil
        }
        m.authURL = msg.url
        m.flowID = msg.flowID
        _ = openBrowser(m.authURL)
        m.polling = true
        return m, m.startPolling()
    case pollDoneMsg:
        m.polling = false
        if msg.err != nil || strings.TrimSpace(msg.tok.AccessToken) == "" {
            if msg.err != nil { m.pollErr = msg.err.Error() } else { m.pollErr = "authorization failed" }
            return m, nil
        }
        // Fetch channels
        return m, m.startFetchChannels(msg.tok.AccessToken)
    case chansMsg:
        if msg.err != nil {
            m.pollErr = msg.err.Error()
            return m, nil
        }
        m.channels = msg.list
        m.ytSel = newYTSelectModel(m.channels)
        m.step = stepYTSelect
        return m, nil
    }
    return m, nil
}

func (m *wizardModel) View() string {
    b := &strings.Builder{}
    switch m.step {
    case stepIntro:
        fmt.Fprintln(b, "Welcome to Colino setup! 🚀")
        fmt.Fprintln(b, "This wizard will set up your feeds, config, and background ingestion.")
        fmt.Fprintln(b, "\nPress Enter to begin · q to quit")
    case stepConfigChoice:
        fmt.Fprintf(b, "Found an existing config at %s\n", configPath())
        fmt.Fprintln(b, "Override it (will create a .bak) or keep it?")
        fmt.Fprintln(b, "[o] Override    [k] Keep existing")
    case stepRSS:
        fmt.Fprintln(b, "Step 1 – RSS Feeds")
        fmt.Fprintln(b, "Enter one or more RSS feed URLs, separated by commas.")
        fmt.Fprintln(b, "You can add more later by editing ~/.config/colino/config.yaml.")
        fmt.Fprintln(b, "Example YouTube channel feed: https://www.youtube.com/feeds/videos.xml?channel_id=UCbRP3c757lWg9M-U7TyEkXA\n")
        fmt.Fprintln(b, m.rssInput.View())
        fmt.Fprintln(b, "\nPress Enter to continue")
    case stepYTAsk:
        fmt.Fprintln(b, "Step 2 – YouTube Channels (optional)")
        fmt.Fprintln(b, "Connect your YouTube account to import channel feeds from your subscriptions.")
        fmt.Fprintln(b, "[y] Yes    [n] No")
    case stepYTAuth:
        fmt.Fprintln(b, "Authenticate with Google in your browser…")
        if strings.TrimSpace(m.authURL) != "" {
            fmt.Fprintf(b, "Auth URL: %s\n", m.authURL)
            fmt.Fprintln(b, "(Press 'o' to open again)")
        } else {
            fmt.Fprintln(b, "Requesting authorization URL…")
        }
        if m.polling {
            fmt.Fprintln(b, "Polling for completion…")
        }
        if strings.TrimSpace(m.pollErr) != "" {
            fmt.Fprintf(b, "Error: %s\n", m.pollErr)
        }
    case stepYTSelect:
        fmt.Fprintln(b, m.ytSel.View())
        fmt.Fprintln(b, "Enter to confirm selection · q to quit")
    case stepInterval:
        fmt.Fprintln(b, "Step 3 – Ingestion Interval")
        fmt.Fprintln(b, "How often should ingestion run? Minutes [30]:")
        fmt.Fprintln(b, m.intervalInput.View())
        if m.errMsg != "" { fmt.Fprintf(b, "\n%s\n", m.errMsg) }
        fmt.Fprintln(b, "\nPress Enter to continue")
    case stepProxy:
        fmt.Fprintln(b, "Step 4 – Optional Webshare Proxy")
        fmt.Fprintln(b, "If you ingest many YouTube transcripts, enabling a rotating proxy helps avoid IP blocking.")
        fmt.Fprintln(b, "Leave empty to skip.")
        fmt.Fprintln(b, "\nUsername:")
        if !m.wsUserInput.Focused() { m.wsUserInput.Focus() }
        fmt.Fprintln(b, m.wsUserInput.View())
        fmt.Fprintln(b, "\nPassword:")
        fmt.Fprintln(b, m.wsPassInput.View())
        fmt.Fprintln(b, "\nPress Enter to continue")
    case stepSummary:
        fmt.Fprintln(b, "Summary")
        fmt.Fprintf(b, "Interval: %d minutes\n", m.interval)
        if len(m.rssFeeds) > 0 {
            fmt.Fprintln(b, "RSS Feeds:")
            for _, u := range m.rssFeeds {
                if n := m.ytNameByURL[u]; n != "" {
                    fmt.Fprintf(b, "  - %s  # YouTube: %s\n", u, n)
                } else {
                    fmt.Fprintf(b, "  - %s\n", u)
                }
            }
        }
        if strings.TrimSpace(m.wsUser) != "" {
            fmt.Fprintln(b, "YouTube proxy: enabled (Webshare)")
        }
        if m.override {
            fmt.Fprintln(b, "\nThe configuration file will be written to ~/.config/colino/config.yaml.")
        } else {
            fmt.Fprintln(b, "\nKeeping existing config. Only the launchd schedule will be installed/updated.")
        }
        fmt.Fprintln(b, "\nPress Enter to finish · q to cancel")
    case stepDone:
        fmt.Fprintln(b, "Finishing…")
    }
    return b.String()
}

func (m *wizardModel) startInitiate() tea.Cmd {
    return func() tea.Msg {
        flow, err := initiateOAuth()
        if err != nil {
            return initAuthMsg{"", "", err}
        }
        return initAuthMsg{flow.AuthURL, flow.FlowID, nil}
    }
}

func (m *wizardModel) startPolling() tea.Cmd {
    fid := m.flowID
    return func() tea.Msg {
        tok, err := pollOAuth(fid, 180*time.Second)
        return pollDoneMsg{tok, err}
    }
}

func (m *wizardModel) startFetchChannels(accessToken string) tea.Cmd {
    return func() tea.Msg {
        chans, err := fetchYouTubeSubscriptions(accessToken)
        return chansMsg{chans, err}
    }
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

// (legacy YouTube flow removed; handled inside Bubble Tea wizard)

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

// (legacy selector removed; Bubble Tea selector is used within the wizard)

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

func (m *ytSelectModel) pageSize() int {
    if m.height > 6 {
        return m.height - 6
    }
    return 15
}

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
        case tea.KeyPgDown:
            if n := m.pageSize(); n > 0 {
                m.cursor += n
                if m.cursor > len(m.filtered)-1 {
                    m.cursor = len(m.filtered) - 1
                }
            }
        case tea.KeyPgUp:
            if n := m.pageSize(); n > 0 {
                if m.cursor >= n {
                    m.cursor -= n
                } else {
                    m.cursor = 0
                }
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
            case "g":
                m.cursor = 0
            case "G":
                if c := len(m.filtered); c > 0 {
                    m.cursor = c - 1
                }
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
    fmt.Fprintln(b, "Select YouTube channels — ↑/↓ or j/k move • Space toggle • /=filter • a=all • n=none • PgUp/PgDn page • g/G top/end • Enter confirm • q quit")
    if m.filtering {
        fmt.Fprintf(b, "Filter: %s\n", m.filter)
    } else if strings.TrimSpace(m.filter) != "" {
        fmt.Fprintf(b, "Filter: %s (press / to edit)\n", m.filter)
    }
    maxRows := m.pageSize()
    if maxRows > len(m.filtered) {
        maxRows = len(m.filtered)
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
    // Footer with progress and guidance
    selCount := len(m.selected)
    total := len(m.filtered)
    var a, z int
    if total == 0 {
        a, z = 0, 0
    } else {
        a = start + 1
        z = end
    }
    fmt.Fprintf(b, "\nSelected: %d • Showing %d–%d of %d • Enter to confirm, q to cancel\n", selCount, a, z, total)
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

// (legacy selection removed)

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
