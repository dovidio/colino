package setup

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func TestInputField(t *testing.T) {
	t.Run("NewInputField", func(t *testing.T) {
		field := newInputField("test placeholder", textinput.EchoNormal)
		if field == nil {
			t.Fatal("expected non-nil field")
		}
		if field.input.Placeholder != "test placeholder" {
			t.Errorf("expected placeholder %q, got %q", "test placeholder", field.input.Placeholder)
		}
		if field.focused {
			t.Error("expected field to be unfocused initially")
		}
	})

	t.Run("FocusAndBlur", func(t *testing.T) {
		field := newInputField("test", textinput.EchoNormal)

		cmd := field.focus()
		if cmd != nil {
			t.Error("expected nil command from focus")
		}
		if !field.focused {
			t.Error("expected field to be focused")
		}

		field.blur()
		if field.focused {
			t.Error("expected field to be unfocused after blur")
		}
	})

	t.Run("Value", func(t *testing.T) {
		field := newInputField("test", textinput.EchoNormal)
		field.input.SetValue("  test value  ")

		expected := "test value"
		if actual := field.value(); actual != expected {
			t.Errorf("expected value %q, got %q", expected, actual)
		}
	})

	t.Run("SetValue", func(t *testing.T) {
		field := newInputField("test", textinput.EchoNormal)
		field.setValue("new value")

		if field.input.Value() != "new value" {
			t.Errorf("expected input value %q, got %q", "new value", field.input.Value())
		}
	})

	t.Run("PasswordEcho", func(t *testing.T) {
		field := newInputField("password", textinput.EchoPassword)
		if field.input.EchoMode != textinput.EchoPassword {
			t.Error("expected password echo mode")
		}
		if field.input.EchoCharacter != '•' {
			t.Error("expected bullet echo character")
		}
	})
}

func TestInputGroup(t *testing.T) {
	t.Run("NewInputGroup", func(t *testing.T) {
		field1 := newInputField("field1", textinput.EchoNormal)
		field2 := newInputField("field2", textinput.EchoNormal)

		group := newInputGroup(field1, field2)
		if group == nil {
			t.Fatal("expected non-nil group")
		}
		if len(group.fields) != 2 {
			t.Errorf("expected 2 fields, got %d", len(group.fields))
		}
		if group.current != 0 {
			t.Errorf("expected current field 0, got %d", group.current)
		}
	})

	t.Run("FocusFirst", func(t *testing.T) {
		field1 := newInputField("field1", textinput.EchoNormal)
		field2 := newInputField("field2", textinput.EchoNormal)

		group := newInputGroup(field1, field2)
		cmd := group.focusFirst()

		if cmd != nil {
			t.Error("expected nil command from focusFirst")
		}
		if !field1.focused {
			t.Error("expected first field to be focused")
		}
		if field2.focused {
			t.Error("expected second field to be unfocused")
		}
		if group.current != 0 {
			t.Errorf("expected current field 0, got %d", group.current)
		}
	})

	t.Run("Values", func(t *testing.T) {
		field1 := newInputField("field1", textinput.EchoNormal)
		field1.input.SetValue("value1")
		field2 := newInputField("field2", textinput.EchoNormal)
		field2.input.SetValue("value2")

		group := newInputGroup(field1, field2)
		values := group.values()

		expected := []string{"value1", "value2"}
		if len(values) != len(expected) {
			t.Fatalf("expected %d values, got %d", len(expected), len(values))
		}
		for i, exp := range expected {
			if values[i] != exp {
				t.Errorf("expected value[%d] %q, got %q", i, exp, values[i])
			}
		}
	})
}

func TestWizardModel_IntroStep(t *testing.T) {
	t.Run("IntroStepWithExistingConfig", func(t *testing.T) {
		model := newWizardModel(true) // hasCfg = true
		if model.step != stepIntro {
			t.Errorf("expected stepIntro, got %v", model.step)
		}

		// Test Enter key with existing config
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := model.Update(msg)
		wm := newModel.(*wizardModel)
		if wm.step != stepConfigChoice {
			t.Errorf("expected stepConfigChoice, got %v", wm.step)
		}
	})

	t.Run("IntroStepWithoutExistingConfig", func(t *testing.T) {
		model := newWizardModel(false) // hasCfg = false

		// Test Enter key without existing config
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := model.Update(msg)
		wm := newModel.(*wizardModel)
		if wm.step != stepRSS {
			t.Errorf("expected stepRSS, got %v", wm.step)
		}
		if !wm.override {
			t.Error("expected override to be true")
		}
	})

	t.Run("GlobalQuit", func(t *testing.T) {
		model := newWizardModel(false)

		// Test Ctrl+C
		msg := tea.KeyMsg{Type: tea.KeyCtrlC}
		newModel, cmd := model.Update(msg)

		if cmd == nil {
			t.Error("expected quit command")
		}
		wm := newModel.(*wizardModel)
		if !wm.cancelled {
			t.Error("expected cancelled to be true")
		}
	})
}

func TestWizardModel_ConfigChoiceStep(t *testing.T) {
	t.Run("OverrideChoice", func(t *testing.T) {
		model := newWizardModel(true)
		model.step = stepConfigChoice
		// Test 'o' key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepRSS {
			t.Errorf("expected stepRSS, got %v", wm.step)
		}
		if !wm.override {
			t.Error("expected override to be true")
		}
	})

	t.Run("KeepChoice", func(t *testing.T) {
		model := newWizardModel(true)
		model.step = stepConfigChoice
		// Test 'k' key
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepInterval {
			t.Errorf("expected stepInterval, got %v", wm.step)
		}
		if wm.override {
			t.Error("expected override to be false")
		}
	})

	t.Run("CaseInsensitive", func(t *testing.T) {
		model := newWizardModel(true)
		model.step = stepConfigChoice
		// Test uppercase 'O'
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'O'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepRSS {
			t.Errorf("expected stepRSS, got %v", wm.step)
		}
	})
}

func TestWizardModel_RSSStep(t *testing.T) {
	t.Run("EnterWithRSSFeeds", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepRSS
		testRSS := "https://example.com/feed.xml, https://test.com/feed.xml"
		model.rssInput.setValue(testRSS)
		// Focus the RSS input field
		model.rssInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepYTAsk {
			t.Errorf("expected stepYTAsk, got %v", wm.step)
		}
		expected := []string{"https://example.com/feed.xml", "https://test.com/feed.xml"}
		if len(wm.rssFeeds) != len(expected) {
			t.Fatalf("expected %d RSS feeds, got %d", len(expected), len(wm.rssFeeds))
		}
		for i, exp := range expected {
			if wm.rssFeeds[i] != exp {
				t.Errorf("expected RSS feed[%d] %q, got %q", i, exp, wm.rssFeeds[i])
			}
		}
	})

	t.Run("EnterWithEmptyRSS", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepRSS
		model.rssInput.setValue("")
		// Focus the RSS input field
		model.rssInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepYTAsk {
			t.Errorf("expected stepYTAsk, got %v", wm.step)
		}
		if len(wm.rssFeeds) != 0 {
			t.Errorf("expected no RSS feeds, got %d", len(wm.rssFeeds))
		}
	})
}

func TestWizardModel_IntervalStep(t *testing.T) {
	t.Run("ValidInterval", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepInterval
		model.intervalInput.setValue("45")
		// Focus the interval input field
		model.intervalInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.interval != 45 {
			t.Errorf("expected interval 45, got %d", wm.interval)
		}
		if wm.errMsg != "" {
			t.Errorf("expected no error message, got %q", wm.errMsg)
		}
	})

	t.Run("DefaultInterval", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepInterval
		model.intervalInput.setValue("")
		// Focus the interval input field
		model.intervalInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.interval != 30 {
			t.Errorf("expected default interval 30, got %d", wm.interval)
		}
	})

	t.Run("InvalidInterval", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepInterval
		model.intervalInput.setValue("invalid")
		// Focus the interval input field
		model.intervalInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := model.Update(msg)

		wm := newModel.(*wizardModel)
		if wm.errMsg != "Please enter a positive integer (minutes)." {
			t.Errorf("expected error message, got %q", wm.errMsg)
		}
		// Should stay on same step with invalid input
		if wm.step != stepInterval {
			t.Errorf("expected to stay on stepInterval, got %v", wm.step)
		}
	})

	t.Run("NegativeInterval", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepInterval
		model.intervalInput.setValue("-5")
		// Focus the interval input field
		model.intervalInput.focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ := model.Update(msg)

		wm := newModel.(*wizardModel)
		if wm.errMsg != "Please enter a positive integer (minutes)." {
			t.Errorf("expected error message, got %q", wm.errMsg)
		}
	})
}

func TestWizardModel_YouTubeAskStep(t *testing.T) {
	t.Run("YesChoice", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepYTAsk
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		newModel, cmd := model.Update(msg)

		if cmd == nil {
			t.Error("expected command for OAuth initiation")
		}
		wm := newModel.(*wizardModel)
		if !wm.ytWanted {
			t.Error("expected ytWanted to be true")
		}
		if wm.step != stepYTAuth {
			t.Errorf("expected stepYTAuth, got %v", wm.step)
		}
	})

	t.Run("NoChoice", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepYTAsk
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.ytWanted {
			t.Error("expected ytWanted to be false")
		}
		if wm.step != stepInterval {
			t.Errorf("expected stepInterval, got %v", wm.step)
		}
	})
}

func TestWizardModel_SummaryStep(t *testing.T) {
	model := newWizardModel(false)
	model.step = stepSummary

	t.Run("EnterToFinish", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd == nil {
			t.Error("expected quit command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepDone {
			t.Errorf("expected stepDone, got %v", wm.step)
		}
	})
}

func TestWizardModel_ProxyStep(t *testing.T) {
	model := newWizardModel(false)
	model.step = stepProxy

	t.Run("CompleteProxyInput", func(t *testing.T) {
		// Simulate entering proxy credentials
		model.proxyInputGroup.fields[0].setValue("testuser")
		model.proxyInputGroup.fields[1].setValue("testpass")
		model.proxyInputGroup.current = 1 // On password field
		model.proxyInputGroup.fields[1].focus()

		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.wsUser != "testuser" {
			t.Errorf("expected wsUser %q, got %q", "testuser", wm.wsUser)
		}
		if wm.wsPass != "testpass" {
			t.Errorf("expected wsPass %q, got %q", "testpass", wm.wsPass)
		}
		// Should move to next step
		if wm.step == stepProxy {
			t.Error("expected to move to next step")
		}
	})
}

func TestWizardModel_MCPToggle(t *testing.T) {
	model := newWizardModel(false)
	model.step = stepMCP
	model.mcpClaudeAvail = true
	model.mcpCodexAvail = true

	t.Run("ToggleClaude", func(t *testing.T) {
		initialChoice := model.mcpClaudeChoice
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.mcpClaudeChoice == initialChoice {
			t.Error("expected mcpClaudeChoice to be toggled")
		}
	})

	t.Run("ToggleCodex", func(t *testing.T) {
		initialChoice := model.mcpCodexChoice
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.mcpCodexChoice == initialChoice {
			t.Error("expected mcpCodexChoice to be toggled")
		}
	})

	t.Run("EnterToContinue", func(t *testing.T) {
		msg := tea.KeyMsg{Type: tea.KeyEnter}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.step != stepSummary {
			t.Errorf("expected stepSummary, got %v", wm.step)
		}
	})
}

func TestWizardModel_AskStep(t *testing.T) {
	t.Run("AIConfigureYes", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepAIAsk
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if !wm.configureDigest {
			t.Error("expected configureDigest to be true")
		}
		if wm.step != stepAI {
			t.Errorf("expected stepAI, got %v", wm.step)
		}
	})

	t.Run("AIConfigureNo", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepAIAsk
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		newModel, cmd := model.Update(msg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.configureDigest {
			t.Error("expected configureDigest to be false")
		}
		// Should move to MCP or summary depending on availability
		if wm.step != stepMCP && wm.step != stepSummary {
			t.Errorf("expected stepMCP or stepSummary, got %v", wm.step)
		}
	})
}

// Test async message handlers
func TestWizardModel_AsyncMessages(t *testing.T) {
	t.Run("InitAuthMsgError", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepYTAuth

		errMsg := initAuthMsg{err: fmt.Errorf("test error")}
		newModel, cmd := model.Update(errMsg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.pollErr != "test error" {
			t.Errorf("expected poll error %q, got %q", "test error", wm.pollErr)
		}
	})

	t.Run("PollDoneMsgError", func(t *testing.T) {
		model := newWizardModel(false)
		model.step = stepYTAuth
		model.polling = true

		errMsg := pollDoneMsg{err: fmt.Errorf("poll error")}
		newModel, cmd := model.Update(errMsg)

		if cmd != nil {
			t.Error("expected nil command")
		}
		wm := newModel.(*wizardModel)
		if wm.polling {
			t.Error("expected polling to be false")
		}
		if wm.pollErr != "OAuth error: poll error" {
			t.Errorf("expected poll error %q, got %q", "OAuth error: poll error", wm.pollErr)
		}
	})
}

func TestParsePositiveInt(t *testing.T) {
	tests := []struct {
		input    string
		expected int
		hasError bool
	}{
		{"", 0, true},
		{"-5", 0, true},
		{"0", 0, true},
		{"10", 10, false},
		{"  25  ", 25, false},
		{"invalid", 0, true},
		{"10.5", 0, true},
	}

	for _, test := range tests {
		result, err := parsePositiveInt(test.input)
		if test.hasError {
			if err == nil {
				t.Errorf("expected error for input %q", test.input)
			}
		} else {
			if err != nil {
				t.Errorf("unexpected error for input %q: %v", test.input, err)
			}
			if result != test.expected {
				t.Errorf("expected %d for input %q, got %d", test.expected, test.input, result)
			}
		}
	}
}

func TestSplitCSV(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"", []string{}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c ", []string{"a", "b", "c"}},
		{"a", []string{"a"}},
		{"a,", []string{"a"}},
		{",a", []string{"a"}},
		{"a,,b", []string{"a", "b"}},
		{"  a  ,  b  ", []string{"a", "b"}},
	}

	for _, test := range tests {
		result := splitCSV(test.input)
		if len(result) != len(test.expected) {
			t.Errorf("expected %d items for input %q, got %d", len(test.expected), test.input, len(result))
			continue
		}
		for i, exp := range test.expected {
			if result[i] != exp {
				t.Errorf("expected item[%d] %q for input %q, got %q", i, exp, test.input, result[i])
			}
		}
	}
}

// Integration test for complete flow
func TestWizardModel_CompleteFlow(t *testing.T) {
	// Test a complete happy path flow
	model := newWizardModel(false)

	// Intro step
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
	model = newModel.(*wizardModel)
	if model.step != stepRSS {
		t.Errorf("expected stepRSS, got %v", model.step)
	}

	// RSS step
	model.rssInput.setValue("https://example.com/feed.xml")
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = model.Update(msg)
	model = newModel.(*wizardModel)
	if model.step != stepYTAsk {
		t.Errorf("expected stepYTAsk, got %v", model.step)
	}

	// YouTube ask - skip
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*wizardModel)
	if model.step != stepInterval {
		t.Errorf("expected stepInterval, got %v", model.step)
	}

	// Interval step
	model.intervalInput.setValue("60")
	model.intervalInput.focus()
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = model.Update(msg)
	model = newModel.(*wizardModel)
	if model.step != stepAIAsk {
		t.Errorf("expected stepAIAsk, got %v", model.step)
	}

	// AI ask - skip
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
	newModel, _ = model.Update(msg)
	model = newModel.(*wizardModel)
	// Should go to MCP if available or summary
	if model.step != stepMCP && model.step != stepSummary {
		t.Errorf("expected stepMCP or stepSummary, got %v", model.step)
	}

	// If MCP step, skip it
	if model.step == stepMCP {
		msg = tea.KeyMsg{Type: tea.KeyEnter}
		newModel, _ = model.Update(msg)
		model = newModel.(*wizardModel)
		if model.step != stepSummary {
			t.Errorf("expected stepSummary, got %v", model.step)
		}
	}

	// Summary step
	msg = tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ = model.Update(msg)
	model = newModel.(*wizardModel)
	if model.step != stepDone {
		t.Errorf("expected stepDone, got %v", model.step)
	}

	// Verify final state
	if len(model.rssFeeds) != 1 || model.rssFeeds[0] != "https://example.com/feed.xml" {
		t.Errorf("expected RSS feed to be set, got %v", model.rssFeeds)
	}
	if model.interval != 60 {
		t.Errorf("expected interval 60, got %d", model.interval)
	}
}
