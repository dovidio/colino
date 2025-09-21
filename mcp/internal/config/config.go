package config

import (
    "errors"
    "os"
    "path/filepath"
    "runtime"
    "strings"

    "gopkg.in/yaml.v3"
)

// LoadDBPath returns the SQLite DB path used by Colino.
// It understands both Python Colino config (database.path) and
// Golino config (database_path). Falls back to platform default.
func LoadDBPath() (string, error) {
    // Prefer user config at ~/.config/colino/config.yaml
    cfgPath, err := defaultConfigPath()
    if err == nil {
        if p, err := readDBPathFrom(cfgPath); err == nil && p != "" {
            return ExpandPath(p), nil
        }
        if !errors.Is(err, os.ErrNotExist) && err != nil {
            // if parsing failed for other reasons, we still fall back
            _ = err
        }
    }
    // Fallback to platform default
    if runtime.GOOS == "darwin" {
        home, _ := os.UserHomeDir()
        return filepath.Join(home, "Library", "Application Support", "Colino", "colino.db"), nil
    }
    return "colino.db", nil
}

func defaultConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".config", "colino", "config.yaml"), nil
}

func readDBPathFrom(path string) (string, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return "", err
    }
    var raw map[string]any
    if err := yaml.Unmarshal(b, &raw); err != nil {
        return "", err
    }
    // Python config: database.path
    if db, ok := raw["database"].(map[string]any); ok {
        if p, ok := db["path"].(string); ok && p != "" {
            return p, nil
        }
    }
    // Golino config: database_path
    if p, ok := raw["database_path"].(string); ok && p != "" {
        return p, nil
    }
    return "", nil
}

// ExpandPath expands leading ~ and environment variables in a filesystem path.
func ExpandPath(p string) string {
    if p == "" {
        return p
    }
    // Expand environment variables like $HOME
    p = os.ExpandEnv(p)
    // Expand leading ~
    if strings.HasPrefix(p, "~") {
        if home, err := os.UserHomeDir(); err == nil {
            if p == "~" {
                p = home
            } else if strings.HasPrefix(p, "~/") {
                p = filepath.Join(home, p[2:])
            }
        }
    }
    return p
}
