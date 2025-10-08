package setup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	textarea "github.com/charmbracelet/bubbles/textarea"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"golino/internal/config"
	"golino/internal/launchd"
	"golino/internal/youtube"
)

// Run executes the interactive setup flow
func Run(ctx context.Context) error {
	cfgPath := configPath()
	cfgExists := fileExists(cfgPath)

	wiz := newWizardModel(cfgExists)
	p := tea.NewProgram(wiz)
	res, err := p.Run()
	if err != nil {
		return err
	}
	wm, ok := res.(*wizardModel)
	if !ok || wm.cancelled {
		return errors.New("setup cancelled")
	}

	// Build config.UserConfig from wizard result
	uc := config.UserConfig{
		RSSFeeds:     wm.rssFeeds,
		IntervalMin:  wm.interval,
		WebshareUser: wm.wsUser,
		WebsharePass: wm.wsPass,
		YTNameByURL:  wm.ytNameByURL,
		DatabasePath: config.FallbackDBPath(),
	}

	// Add AI configuration if configured
	if wm.configureDigest {
		uc.AI = &config.AIConfig{
			Model:         wm.aiModel,
			BaseUrl:       wm.aiBaseURL,
			ArticlePrompt: wm.articlePrompt,
		}
	}

	// Write config if overriding or creating new
	if wm.override {
		if cfgExists {
			_ = config.BackupFile(cfgPath)
		}
		if err := config.WriteConfig(uc); err != nil {
			return err
		}
		fmt.Println("\nConfig written to ~/.config/colino/config.yaml")
	}

	// Install daemon (macOS), skip long bootstrap ingest
	exe, _ := os.Executable()
	interval := wm.interval
	if interval <= 0 {
		interval = 30
	}

	if runtime.GOOS == "darwin" {
		fmt.Println("\nInstalling launchd agent to run on a schedule…")
		// Install as a scheduled oneshot ingest; launchd handles periodicity
		args := []string{"ingest"}
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
			fmt.Println("Tips: make sure you're running on macOS with launchctl available (usually at /bin/launchctl). If you're inside a container or a non-login shell, launchctl may be unavailable.")
			fmt.Println("You can manually load the agent later via: /bin/launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/com.colino.daemon.plist")
		} else {
			fmt.Println("launchd agent installed and loaded.")
		}
	} else {
		fmt.Println("\nNote: Automatic scheduling is only implemented for macOS (launchd).")
	}

	// Apply MCP integration based on wizard choices
	exe, _ = os.Executable()
	if wm.mcpClaudeChoice && wm.mcpClaudeAvail {
		if err := runClaudeCLIAdd(exe); err != nil {
			fmt.Printf("Failed to add MCP via Claude CLI: %v\n", err)
		} else {
			fmt.Println("Added MCP server to Claude via CLI.")
		}
	}
	if wm.mcpCodexChoice && wm.mcpCodexAvail && strings.TrimSpace(wm.codexPath) != "" {
		if wm.codexPath != "" {
			b, err := os.ReadFile(wm.codexPath)
			if err == nil || !strings.Contains(string(b), "[mcp_servers.colino]") {
				fmt.Println("Colino is already configured in codex")

			} else {
				_ = config.BackupFile(wm.codexPath)
				if err := appendTomlMCP(wm.codexPath, exe); err != nil {
					fmt.Printf("Failed to add MCP to %s: %v\n", wm.codexPath, err)
				} else {
					fmt.Printf("Added MCP server to %s\n", wm.codexPath)
				}

			}
		}

	}
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
	stepAIAsk
	stepAI
	stepAIPrompt
	stepMCP
	stepSummary
	stepDone
)

type wizardModel struct {
	step      wizardStep
	hasCfg    bool
	override  bool
	cancelled bool

	// RSS
	rssInput *inputField
	rssFeeds []string

	// YouTube
	ytWanted    bool
	authURL     string
	flowID      string
	polling     bool
	pollErr     string
	channels    []youtube.Channel
	ytSel       *ytSelectModel
	ytNameByURL map[string]string

	// Interval
	intervalInput *inputField
	interval      int

	// Proxy
	proxyInputGroup *inputGroup
	wsUser          string
	wsPass          string

	// Status/error
	errMsg string

	// AI (for digest)
	configureDigest    bool
	aiModelInput       *inputField
	aiBaseURLInput     *inputField
	articlePromptInput textarea.Model
	aiModel            string
	aiBaseURL          string
	articlePrompt      string

	// MCP integration
	mcpClaudeAvail  bool
	mcpClaudeChoice bool
	mcpCodexAvail   bool
	mcpCodexChoice  bool
	codexPath       string
}

func newWizardModel(hasCfg bool) *wizardModel {
	// Create input fields
	rssInput := newInputField("https://example.com/feed.xml, https://...", textinput.EchoNormal)
	rssInput.focus()

	intervalInput := newInputField("30", textinput.EchoNormal)

	// Proxy input group
	wsUserInput := newInputField("webshare username (optional)", textinput.EchoNormal)
	wsPassInput := newInputField("webshare password (optional)", textinput.EchoPassword)
	proxyGroup := newInputGroup(wsUserInput, wsPassInput)

	// AI input fields
	aiModelInput := newInputField("Enter model name (e.g., gpt-4)", textinput.EchoNormal)
	aiBaseURLInput := newInputField("", textinput.EchoNormal)
	aiBaseURLInput.setValue("https://api.openai.com/v1")

	articlePromptInput := textarea.New()
	prompt := `You are an expert news curator and summarizer.
Create an insightful summary of the article content below.
The content can come from news articles, youtube videos transcripts or blog posts.
Format your response in clean markdown with headers and bullet points if required.

## Article {{.Title}}
**Source:** {{.Source}} | **Published:** {{.Published}}
**URL:** {{.Url}}

**Content:**
{{.Content}}
`
	articlePromptInput.SetValue(prompt)
	articlePromptInput.SetHeight(20)
	articlePromptInput.SetWidth(80)

	// detect MCP options
	_, claudeErr := exec.LookPath("claude")
	codex := pathIfExists(filepath.Join(userHome(), ".codex", "config.toml"))

	return &wizardModel{
		step:               stepIntro,
		hasCfg:             hasCfg,
		rssInput:           rssInput,
		intervalInput:      intervalInput,
		proxyInputGroup:    proxyGroup,
		aiModelInput:       aiModelInput,
		aiBaseURLInput:     aiBaseURLInput,
		articlePromptInput: articlePromptInput,
		interval:           30,
		ytNameByURL:        map[string]string{},
		mcpClaudeAvail:     claudeErr == nil,
		mcpCodexAvail:      codex != "",
		codexPath:          codex,
	}
}

func (m *wizardModel) Init() tea.Cmd { return nil }

// hasYouTubeFeeds reports whether the current RSS feeds include any
// YouTube channel feed URLs (added manually or via the YouTube selector).
func (m *wizardModel) hasYouTubeFeeds() bool {
	if len(m.ytNameByURL) > 0 {
		return true
	}
	for _, u := range m.rssFeeds {
		if strings.Contains(strings.ToLower(u), "youtube.com/feeds/videos.xml") {
			return true
		}
	}
	return false
}

// Messages for async actions
type initAuthMsg struct {
	url, flowID string
	err         error
}
type pollDoneMsg struct {
	tok youtube.OAuthPollResponse
	err error
}
type chansMsg struct {
	list []youtube.Channel
	err  error
}

func (m *wizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Global quit
		if msg.Type == tea.KeyCtrlC {
			m.cancelled = true
			return m, tea.Quit
		}

		// Handle key events based on current step
		switch m.step {
		case stepIntro:
			return m.handleIntroStep(msg)
		case stepConfigChoice:
			return m.handleConfigChoiceStep(msg)
		case stepRSS:
			return m.handleRSSStep(msg)
		case stepYTAsk:
			return m.handleYTAskStep(msg)
		case stepYTAuth:
			return m.handleYTAuthStep(msg)
		case stepYTSelect:
			return m.handleYTSelectStep(msg)
		case stepInterval:
			return m.handleIntervalStep(msg)
		case stepProxy:
			return m.handleProxyStep(msg)
		case stepAIAsk:
			return m.handleAIAskStep(msg)
		case stepAI:
			return m.handleAIStep(msg)
		case stepAIPrompt:
			return m.handleAIPromptStep(msg)
		case stepMCP:
			return m.handleMCPStep(msg)
		case stepSummary:
			return m.handleSummaryStep(msg)
		}

	case initAuthMsg:
		return m.handleInitAuthMsg(msg)
	case pollDoneMsg:
		return m.handlePollDoneMsg(msg)
	case chansMsg:
		return m.handleChansMsg(msg)
	}
	return m, nil
}

// Step handlers
func (m *wizardModel) handleIntroStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		if m.hasCfg {
			m.step = stepConfigChoice
		} else {
			m.override = true
			m.step = stepRSS
		}
	}
	return m, nil
}

func (m *wizardModel) handleConfigChoiceStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		s := strings.ToLower(string(msg.Runes))
		switch s {
		case "o":
			m.override = true
			m.step = stepRSS
		case "k":
			m.override = false
			m.step = stepInterval
		}
	}
	return m, nil
}

func (m *wizardModel) handleRSSStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	_, cmd := m.rssInput.update(msg)

	if msg.Type == tea.KeyEnter {
		m.rssFeeds = splitCSV(m.rssInput.value())
		m.step = stepYTAsk
		return m, nil
	}
	return m, cmd
}

func (m *wizardModel) handleYTAskStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		s := strings.ToLower(string(msg.Runes))
		switch s {
		case "y":
			m.ytWanted = true
			m.step = stepYTAuth
			return m, m.startInitiate()
		case "n":
			m.ytWanted = false
			m.step = stepInterval
		}
	}
	return m, nil
}

func (m *wizardModel) handleYTAuthStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes && strings.ToLower(string(msg.Runes)) == "o" {
		if strings.TrimSpace(m.authURL) != "" {
			_ = openBrowser(m.authURL)
		}
	}
	return m, nil // Wait for async messages
}

func (m *wizardModel) handleYTSelectStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.ytSel != nil {
		var cmd tea.Cmd
		mm, cmd := m.ytSel.Update(msg)
		if sel, ok := mm.(*ytSelectModel); ok {
			m.ytSel = sel
		}

		if msg.Type == tea.KeyEnter {
			// Collect selection and continue
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
			// Swallow inner tea.Quit from the child selector to keep the wizard running
			return m, nil
		}
		return m, cmd
	}
	return m, nil
}

func (m *wizardModel) handleIntervalStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Focus the interval input if it's not focused
	if !m.intervalInput.focused {
		return m, m.intervalInput.focus()
	}

	_, cmd := m.intervalInput.update(msg)

	if msg.Type == tea.KeyEnter {
		v := m.intervalInput.value()
		if v == "" {
			m.interval = 30
		} else if n, err := parsePositiveInt(v); err == nil && n > 0 {
			m.interval = n
		} else {
			m.errMsg = "Please enter a positive integer (minutes)."
			return m, cmd
		}

		// Determine next step
		if m.override {
			if m.hasYouTubeFeeds() {
				m.step = stepProxy
			} else {
				m.step = stepAIAsk
			}
		} else {
			m.step = stepSummary
		}
		m.errMsg = ""
		m.intervalInput.blur()
		return m, nil
	}
	return m, cmd
}

func (m *wizardModel) handleProxyStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Focus first field if none are focused
	if !m.proxyInputGroup.fields[0].focused && !m.proxyInputGroup.fields[1].focused {
		return m, m.proxyInputGroup.focusFirst()
	}

	// Check if Enter was pressed on the last field (password) before updating
	wasOnLastField := msg.Type == tea.KeyEnter && m.proxyInputGroup.current == len(m.proxyInputGroup.fields)-1

	// Update the input group and handle field navigation
	_, cmd := m.proxyInputGroup.update(msg)

	// Only submit if we were already on the last field when Enter was pressed
	if wasOnLastField {
		values := m.proxyInputGroup.values()
		m.wsUser = values[0]
		m.wsPass = values[1]

		// Move to next step
		m.step = stepAIAsk
		return m, nil
	}

	return m, cmd
}

func (m *wizardModel) handleAIAskStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		s := strings.ToLower(string(msg.Runes))
		switch s {
		case "y":
			m.configureDigest = true
			m.step = stepAI
			return m, nil
		case "n":
			m.configureDigest = false
			// Determine next step
			if m.mcpClaudeAvail || m.mcpCodexAvail {
				m.step = stepMCP
			} else {
				m.step = stepSummary
			}
		}
	}
	return m, nil
}

func (m *wizardModel) handleAIStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	// Handle AI model input
	if m.aiModelInput.focused {
		_, cmd := m.aiModelInput.update(msg)
		if msg.Type == tea.KeyEnter {
			m.aiModel = m.aiModelInput.value()
			m.aiModelInput.blur()
			// Move to base_url field
			return m, m.aiBaseURLInput.focus()
		}
		return m, cmd
	}

	// Handle AI base URL input
	if m.aiBaseURLInput.focused {
		_, cmd := m.aiBaseURLInput.update(msg)
		if msg.Type == tea.KeyEnter {
			m.aiBaseURL = m.aiBaseURLInput.value()
			m.aiBaseURLInput.blur()
			m.step = stepAIPrompt
			return m, nil
		}
		return m, cmd
	}

	// If no field is focused, focus the model input
	if !m.aiModelInput.focused && !m.aiBaseURLInput.focused {
		return m, m.aiModelInput.focus()
	}

	return m, cmd
}

func (m *wizardModel) handleAIPromptStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Focus textarea if not focused
	if !m.articlePromptInput.Focused() {
		return m, m.articlePromptInput.Focus()
	}

	// Handle textarea (article prompt)
	var cmd tea.Cmd
	m.articlePromptInput, cmd = m.articlePromptInput.Update(msg)

	// Use Ctrl+J or Ctrl+M to continue from textarea (maps to Option+Enter on macOS)
	if msg.Type == tea.KeyCtrlJ || msg.Type == tea.KeyCtrlM {
		m.articlePrompt = strings.TrimSpace(m.articlePromptInput.Value())
		if m.mcpClaudeAvail || m.mcpCodexAvail {
			m.step = stepMCP
		} else {
			m.step = stepSummary
		}
		return m, nil
	}

	return m, cmd
}

func (m *wizardModel) handleMCPStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyRunes {
		s := strings.ToLower(string(msg.Runes))
		switch s {
		case "c":
			if m.mcpClaudeAvail {
				m.mcpClaudeChoice = !m.mcpClaudeChoice
			}
		case "o":
			if m.mcpCodexAvail {
				m.mcpCodexChoice = !m.mcpCodexChoice
			}
		}
	}
	if msg.Type == tea.KeyEnter {
		m.step = stepSummary
	}
	return m, nil
}

func (m *wizardModel) handleSummaryStep(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyEnter {
		m.step = stepDone
		return m, tea.Quit
	}
	return m, nil
}

// Async message handlers
func (m *wizardModel) handleInitAuthMsg(msg initAuthMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.pollErr = msg.err.Error()
		return m, nil
	}
	m.authURL = msg.url
	m.flowID = msg.flowID
	_ = openBrowser(m.authURL)
	m.polling = true
	return m, m.startPolling()
}

func (m *wizardModel) handlePollDoneMsg(msg pollDoneMsg) (tea.Model, tea.Cmd) {
	m.polling = false
	if msg.err != nil || strings.TrimSpace(msg.tok.AccessToken) == "" {
		if msg.err != nil {
			m.pollErr = fmt.Sprintf("OAuth error: %v", msg.err)
		} else {
			m.pollErr = "authorization failed: no access token received"
		}
		return m, nil
	}
	// Fetch channels
	return m, m.startFetchChannels(msg.tok.AccessToken)
}

func (m *wizardModel) handleChansMsg(msg chansMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.pollErr = msg.err.Error()
		return m, nil
	}
	m.channels = msg.list
	m.ytSel = newYTSelectModel(m.channels)
	m.step = stepYTSelect
	return m, nil
}

// -------------- Input Field Handler --------------
type inputField struct {
	input   textinput.Model
	focused bool
}

func newInputField(placeholder string, echoMode textinput.EchoMode) *inputField {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.EchoMode = echoMode
	ti.Width = 50
	if echoMode == textinput.EchoPassword {
		ti.EchoCharacter = '•'
	}
	return &inputField{input: ti, focused: false}
}

func (f *inputField) focus() tea.Cmd {
	f.focused = true
	f.input.Focus()
	return nil
}

func (f *inputField) blur() {
	f.focused = false
	f.input.Blur()
}

func (f *inputField) update(msg tea.Msg) (textinput.Model, tea.Cmd) {
	if !f.focused {
		return f.input, nil
	}
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	return f.input, cmd
}

func (f *inputField) value() string {
	return strings.TrimSpace(f.input.Value())
}

func (f *inputField) setValue(value string) {
	f.input.SetValue(value)
}

// -------------- Input Field Groups --------------
type inputGroup struct {
	fields  []*inputField
	current int
}

func newInputGroup(fields ...*inputField) *inputGroup {
	return &inputGroup{
		fields:  fields,
		current: 0,
	}
}

func (g *inputGroup) focusFirst() tea.Cmd {
	if len(g.fields) == 0 {
		return nil
	}
	g.current = 0
	for i, field := range g.fields {
		if i == 0 {
			field.focus()
		} else {
			field.blur()
		}
	}
	return nil
}

func (g *inputGroup) update(msg tea.Msg) (textinput.Model, tea.Cmd) {
	if len(g.fields) == 0 || g.current < 0 || g.current >= len(g.fields) {
		return textinput.Model{}, nil
	}

	field := g.fields[g.current]
	if !field.focused {
		return field.input, nil
	}

	updatedInput, cmd := field.update(msg)
	field.input = updatedInput

	// Handle Enter to move to next field or submit
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.Type == tea.KeyEnter {
		// Move to next field if available
		if g.current < len(g.fields)-1 {
			g.fields[g.current].blur()
			g.current++
			focusCmd := g.fields[g.current].focus()
			return field.input, focusCmd
		}
	}

	return updatedInput, cmd
}

func (g *inputGroup) values() []string {
	values := make([]string, len(g.fields))
	for i, field := range g.fields {
		values[i] = field.value()
	}
	return values
}

func (m *wizardModel) View() string {
	b := &strings.Builder{}
	switch m.step {
	case stepIntro:
		fmt.Fprintln(b, "Welcome to Colino setup! 🚀")
		fmt.Fprintln(b, "This wizard will set up your feeds, config, and background ingestion.")
		fmt.Fprintln(b, "\nPress Enter to begin · ctrl+c to quit")
	case stepConfigChoice:
		fmt.Fprintf(b, "Found an existing config at %s\n", configPath())
		fmt.Fprintln(b, "Override it (will create a .bak) or keep it?")
		fmt.Fprintln(b, "[o] Override    [k] Keep existing")
	case stepRSS:
		fmt.Fprintln(b, "RSS Feeds")
		fmt.Fprintln(b, "Enter one or more RSS feed URLs, separated by commas.")
		fmt.Fprintln(b, "You can add more later by editing ~/.config/colino/config.yaml.")
		fmt.Fprintln(b, "Example of a channel feed: https://feeds.bbci.co.uk/news/rss.xml")
		fmt.Fprintln(b, m.rssInput.input.View())
		fmt.Fprintln(b, "\nPress Enter to continue")
	case stepYTAsk:
		fmt.Fprintln(b, "YouTube Channels (optional)")
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
		fmt.Fprintln(b, "Enter to confirm selection · ctrl+c to quit")
	case stepInterval:
		fmt.Fprintln(b, "Ingestion Interval")
		fmt.Fprintln(b, "How often should ingestion run? Minutes [30]:")
		fmt.Fprintln(b, m.intervalInput.input.View())
		if m.errMsg != "" {
			fmt.Fprintf(b, "\n%s\n", m.errMsg)
		}
		fmt.Fprintln(b, "\nPress Enter to continue")
	case stepProxy:
		fmt.Fprintln(b, "Webshare Proxy (optional)")
		fmt.Fprintln(b, "If you ingest many YouTube transcripts, enabling a rotating proxy helps avoid IP blocking.")
		fmt.Fprintln(b, "Leave either field empty to skip.")
		fmt.Fprintln(b, "\nUsername (press Enter to move to password):")
		fmt.Fprintln(b, m.proxyInputGroup.fields[0].input.View())
		fmt.Fprintln(b, "\nPassword:")
		fmt.Fprintln(b, m.proxyInputGroup.fields[1].input.View())
		fmt.Fprintln(b, "\nPress Enter on password to continue")
	case stepAIAsk:
		fmt.Fprintln(b, "AI Digest (optional)")
		fmt.Fprintln(b, "Configure AI for command digest")
		fmt.Fprintln(b, "[y] Yes    [n] No")
	case stepAI:
		fmt.Fprintln(b, "AI Configuration")
		fmt.Fprintln(b, "\nModel")
		fmt.Fprintln(b, m.aiModelInput.input.View())
		fmt.Fprintln(b, "\nBase URL (optional, defaults to OpenAI)")
		fmt.Fprintln(b, m.aiBaseURLInput.input.View())
		fmt.Fprintln(b, "\nPress Enter to continue to prompt configuration")
	case stepAIPrompt:
		fmt.Fprintln(b, "AI Prompt Configuration (lisân al-ghayb)")
		fmt.Fprintln(b, "Customize the prompt used for AI digest generation")
		fmt.Fprintln(b, m.articlePromptInput.View())
		fmt.Fprintln(b, "\nOption+Enter to continue")
	case stepMCP:
		fmt.Fprintln(b, "MCP Integration (optional)")
		fmt.Fprintln(b, "Configure Colino MCP client integration.")
		if !m.mcpClaudeAvail && !m.mcpCodexAvail {
			fmt.Fprintln(b, "No supported MCP clients detected.")
		}
		if m.mcpClaudeAvail {
			mark := "[ ]"
			if m.mcpClaudeChoice {
				mark = "[x]"
			}
			fmt.Fprintf(b, "%s Add MCP to Claude (press 'c' to toggle)\n", mark)
		} else {
			fmt.Fprintln(b, "[ ] Add MCP to Claude (not detected)")
		}
		if m.mcpCodexAvail {
			mark := "[ ]"
			if m.mcpCodexChoice {
				mark = "[x]"
			}
			fmt.Fprintf(b, "%s Add MCP to ~/.codex/config.toml (press 'o' to toggle)\n", mark)
		} else {
			fmt.Fprintln(b, "[ ] Add MCP to ~/.codex/config.toml (not found)")
		}
		fmt.Fprintln(b, "\nPress Enter to continue")
	case stepSummary:
		fmt.Fprintln(b, "Summary")
		fmt.Fprintf(b, "Interval: %d minutes\n", m.interval)
		fmt.Fprintf(b, "Database: %s\n", config.FallbackDBPath())
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
		if m.configureDigest {
			fmt.Fprintln(b, "AI Digest:")
			if strings.TrimSpace(m.aiModel) != "" {
				fmt.Fprintf(b, "  - Model: %s\n", m.aiModel)
			}
			if strings.TrimSpace(m.aiBaseURL) != "" {
				fmt.Fprintf(b, "  - Base URL: %s\n", m.aiBaseURL)
			}
			if strings.TrimSpace(m.articlePrompt) != "" {
				fmt.Fprintf(b, "  - Custom prompt: configured\n")
			}
		}
		if m.mcpClaudeChoice || m.mcpCodexChoice {
			fmt.Fprintln(b, "MCP integration:")
			if m.mcpClaudeChoice {
				fmt.Fprintln(b, "  - Add to Claude")
			}
			if m.mcpCodexChoice {
				fmt.Fprintln(b, "  - Add to ~/.codex/config.toml")
			}
		}
		if m.override {
			fmt.Fprintln(b, "\nThe configuration file will be written to ~/.config/colino/config.yaml.")
		} else {
			fmt.Fprintln(b, "\nKeeping existing config. Only the launchd schedule will be installed/updated.")
		}
		fmt.Fprintln(b, "\nPress Enter to finish · ctrl+c to cancel")
	case stepDone:
		fmt.Fprintln(b, "Finishing…")
	}
	return b.String()
}

func (m *wizardModel) startInitiate() tea.Cmd {
	return func() tea.Msg {
		authService := youtube.NewOAuthService(youtube.DefaultOAuthConfig())
		flow, err := authService.InitiateOAuth(context.Background())
		if err != nil {
			return initAuthMsg{"", "", err}
		}
		return initAuthMsg{flow.AuthURL, flow.FlowID, nil}
	}
}

func (m *wizardModel) startPolling() tea.Cmd {
	fid := m.flowID
	return func() tea.Msg {
		authService := youtube.NewOAuthService(youtube.DefaultOAuthConfig())
		tok, err := authService.PollOAuth(context.Background(), fid, 180*time.Second)
		if err != nil || tok == nil {
			return pollDoneMsg{youtube.OAuthPollResponse{}, err}
		}
		return pollDoneMsg{*tok, err}
	}
}

func (m *wizardModel) startFetchChannels(accessToken string) tea.Cmd {
	return func() tea.Msg {
		subService := youtube.NewSubscriptionsService()
		if subService == nil {
			return chansMsg{nil, fmt.Errorf("failed to create subscriptions service")}
		}
		chans, err := subService.FetchSubscriptions(context.Background(), accessToken)
		return chansMsg{chans, err}
	}
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

	// Check that all characters are digits
	for _, r := range s {
		if r < '0' || r > '9' {
			return 0, errors.New("invalid int")
		}
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

func userHome() string { h, _ := os.UserHomeDir(); return h }

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

// Bubble Tea model and TUI for multi-selecting channels with basic filtering.
type ytSelectModel struct {
	items     []youtube.Channel
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

func newYTSelectModel(list []youtube.Channel) *ytSelectModel {
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
	maxRows := min(m.pageSize(), len(m.filtered))
	start := 0
	if m.cursor >= maxRows {
		start = m.cursor - maxRows + 1
	}

	end := min(start+maxRows, len(m.filtered))
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
	fmt.Fprintf(b, "\nSelected: %d • Showing %d–%d of %d • Enter to confirm, ctrl+c cancel\n", selCount, a, z, total)
	return b.String()
}

func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		return fmt.Errorf("OS %v is not supported", runtime.GOOS)
	}
}
