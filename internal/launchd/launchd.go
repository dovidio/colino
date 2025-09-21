package launchd

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type Plist struct {
	XMLName xml.Name `xml:"plist"`
	Version string   `xml:"version,attr"`
	Dict    Dict     `xml:"dict"`
}

type Dict struct {
	Keys []any `xml:",any"`
}

// kv helper to insert <key> and value node order.
func (d *Dict) kv(key string, value any) {
	d.Keys = append(d.Keys, xml.Name{Local: "key"})
	d.Keys = append(d.Keys, key)
	d.Keys = append(d.Keys, value)
}

// xml marshalling helpers
type Str string
type Bool bool
type Integer int

type Array struct {
	Items []any `xml:",any"`
}

func NewArray(ss []string) Array {
	a := Array{}
	for _, s := range ss {
		a.Items = append(a.Items, Str(s))
	}
	return a
}

// InstallOptions config for creating/loading a launchd agent.
type InstallOptions struct {
	Label           string
	IntervalMinutes int
	ProgramPath     string   // absolute path to this binary
	ProgramArgs     []string // args after ProgramPath
	StdOutPath      string
	StdErrPath      string
	PlistPath       string // optional custom plist path
}

func DefaultAgentPath(label string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}

// BuildPlist constructs a minimal plist for StartInterval execution.
func BuildPlist(opt InstallOptions) ([]byte, error) {
	if opt.Label == "" {
		return nil, errors.New("label required")
	}
	if opt.ProgramPath == "" {
		return nil, errors.New("program path required")
	}
	if opt.IntervalMinutes <= 0 {
		opt.IntervalMinutes = 30
	}
	if opt.StdOutPath == "" || opt.StdErrPath == "" {
		// default to user logs if not set
		if home, err := os.UserHomeDir(); err == nil {
			def := filepath.Join(home, "Library", "Logs", "Colino", "daemon.launchd.log")
			if opt.StdOutPath == "" {
				opt.StdOutPath = def
			}
			if opt.StdErrPath == "" {
				opt.StdErrPath = def
			}
		}
	}

	// Ensure log directory exists
	_ = os.MkdirAll(filepath.Dir(opt.StdOutPath), 0o755)
	_ = os.MkdirAll(filepath.Dir(opt.StdErrPath), 0o755)

	d := Dict{}
	d.kv("Label", Str(opt.Label))
	args := []string{opt.ProgramPath}
	args = append(args, opt.ProgramArgs...)
	d.kv("ProgramArguments", NewArray(args))
	d.kv("StartInterval", Integer(opt.IntervalMinutes*60))
	d.kv("RunAtLoad", Bool(true))
	d.kv("StandardOutPath", Str(opt.StdOutPath))
	d.kv("StandardErrorPath", Str(opt.StdErrPath))

	p := Plist{Version: "1.0", Dict: d}
	out, err := xml.MarshalIndent(p, "", "  ")
	if err != nil {
		return nil, err
	}
	// add plist header
	hdr := []byte(xml.Header + "<!DOCTYPE plist PUBLIC \"-//Apple//DTD PLIST 1.0//EN\" \"http://www.apple.com/DTDs/PropertyList-1.0.dtd\">\n")
	return append(hdr, out...), nil
}

// Install writes the plist and loads it via launchctl.
func Install(opt InstallOptions) (string, error) {
	if runtime.GOOS != "darwin" {
		return "", errors.New("launchd is only available on macOS")
	}
	plistPath := opt.PlistPath
	if strings.TrimSpace(plistPath) == "" {
		var err error
		plistPath, err = DefaultAgentPath(opt.Label)
		if err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return "", err
	}
	data, err := BuildPlist(opt)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(plistPath, data, 0o644); err != nil {
		return "", err
	}

	// Resolve launchctl path robustly
	lctl := launchctlPath()
	if lctl == "" {
		return plistPath, errors.New("launchctl not found in /bin, /usr/bin, or PATH")
	}

	// Prefer modern bootstrap/enable under user GUI domain
	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	if err := exec.Command(lctl, "bootstrap", domain, plistPath).Run(); err != nil {
		// Fallback to legacy load -w
		if err2 := exec.Command(lctl, "load", "-w", plistPath).Run(); err2 != nil {
			return plistPath, fmt.Errorf("launchctl bootstrap/load failed: %v / %v", err, err2)
		}
	} else {
		// Enable the service explicitly
		_ = exec.Command(lctl, "enable", domain+"/"+opt.Label).Run()
	}
	return plistPath, nil
}

// Uninstall unloads and removes the plist.
func Uninstall(label string, plistPath string) error {
	if runtime.GOOS != "darwin" {
		return errors.New("launchd is only available on macOS")
	}
	if strings.TrimSpace(plistPath) == "" {
		var err error
		plistPath, err = DefaultAgentPath(label)
		if err != nil {
			return err
		}
	}
	// Prefer modern bootout, fallback to unload
	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	lctl := launchctlPath()
	if lctl == "" {
		return errors.New("launchctl not found")
	}
	if err := exec.Command(lctl, "bootout", domain, plistPath).Run(); err != nil {
		_ = exec.Command(lctl, "unload", "-w", plistPath).Run()
	}
	// remove file
	if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Status returns whether the agent is loaded and a short human string.
func Status(label string) (bool, string) {
	if runtime.GOOS != "darwin" || strings.TrimSpace(label) == "" {
		return false, "unsupported"
	}
	uid := os.Getuid()
	domain := fmt.Sprintf("gui/%d", uid)
	lctl := launchctlPath()
	if lctl == "" {
		return false, "launchctl not found"
	}
	out, err := exec.Command(lctl, "print", domain+"/"+label).CombinedOutput()
	if err != nil {
		return false, "not loaded"
	}
	// Try to find state line
	lines := strings.Split(string(out), "\n")
	state := "loaded"
	for _, ln := range lines {
		if strings.Contains(ln, "state = ") {
			state = strings.TrimSpace(ln)
			break
		}
	}
	return true, state
}

// launchctlPath attempts to find the absolute path to launchctl.
func launchctlPath() string {
	// Common locations
	candidates := []string{"/bin/launchctl", "/usr/bin/launchctl"}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	if p, err := exec.LookPath("launchctl"); err == nil {
		return p
	}
	return ""
}

// ExtractStartInterval best-effort parse of StartInterval seconds from a plist file.
func ExtractStartInterval(plistPath string) (int, error) {
	b, err := os.ReadFile(plistPath)
	if err != nil {
		return 0, err
	}
	s := string(b)
	i := strings.Index(s, "<key>StartInterval</key>")
	if i < 0 {
		return 0, errors.New("StartInterval not found")
	}
	// Look for <integer>VALUE</integer> after the key
	sub := s[i:]
	open := strings.Index(sub, "<integer>")
	close := strings.Index(sub, "</integer>")
	if open < 0 || close < 0 || close <= open+9 {
		return 0, errors.New("invalid integer tag")
	}
	val := strings.TrimSpace(sub[open+9 : close])
	n, err := strconv.Atoi(val)
	if err != nil {
		return 0, err
	}
	return n, nil
}
