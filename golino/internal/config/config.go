package config

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"
    "runtime"
    "time"

    "gopkg.in/yaml.v3"
)

type Config struct {
    DatabasePath string  `yaml:"database_path"`
    Sources      Sources `yaml:"sources"`
    Ingest       Ingest  `yaml:"ingest"`
    OpenAI       OpenAI  `yaml:"openai"`
}

type Sources struct {
    RSS     []string `yaml:"rss"`
    YouTube []string `yaml:"youtube"`
}

type Ingest struct {
    Concurrency   int    `yaml:"concurrency"`
    TimeoutSecond int    `yaml:"timeout_seconds"`
    UserAgent     string `yaml:"user_agent"`
}

type OpenAI struct {
    Model        string  `yaml:"model"`
    Temperature  float32 `yaml:"temperature"`
    SystemPrompt string  `yaml:"system_prompt"`
    UserPrompt   string  `yaml:"user_prompt"`
}

func DefaultConfig() Config {
    dbPath := "colino.db"
    if runtime.GOOS == "darwin" {
        home, _ := os.UserHomeDir()
        dbPath = filepath.Join(home, "Library", "Application Support", "Colino", "colino.db")
    }
    return Config{
        DatabasePath: dbPath,
        Sources: Sources{
            RSS:     []string{},
            YouTube: []string{},
        },
        Ingest: Ingest{
            Concurrency:   8,
            TimeoutSecond: int((20 * time.Second).Seconds()),
            UserAgent:     "golino/0.1 (+https://github.com/openai/codex-cli)",
        },
        OpenAI: OpenAI{
            Model:        "gpt-4o-mini",
            Temperature:  0.3,
            SystemPrompt: "You write crisp, factual tech/news digests.",
            UserPrompt:   "Summarize the following items into 3-6 short sections by theme.\nUse markdown with headers and bullet points; include links inline.\nFocus on substance; avoid fluff; keep it under ~250 words.\n\nItems:\n{{items}}",
        },
    }
}

func defaultConfigPath() (string, error) {
    home, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(home, ".config", "colino", "config.yaml"), nil
}

func ensureDir(path string) error {
    return os.MkdirAll(filepath.Dir(path), 0o755)
}

func LoadOrCreate(path string) (Config, error) {
    if path == "" {
        var err error
        path, err = defaultConfigPath()
        if err != nil {
            return Config{}, err
        }
    }
    if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
        cfg := DefaultConfig()
        if err := ensureDir(path); err != nil {
            return Config{}, err
        }
        if err := Save(cfg, path); err != nil {
            return Config{}, err
        }
        return cfg, nil
    }
    return Load(path)
}

func Load(path string) (Config, error) {
    b, err := os.ReadFile(path)
    if err != nil {
        return Config{}, err
    }
    cfg := DefaultConfig()
    if err := yaml.Unmarshal(b, &cfg); err != nil {
        return Config{}, fmt.Errorf("parse %s: %w", path, err)
    }
    return cfg, nil
}

func Save(cfg Config, path string) error {
    b, err := yaml.Marshal(&cfg)
    if err != nil {
        return err
    }
    if err := ensureDir(path); err != nil {
        return err
    }
    return os.WriteFile(path, b, 0o644)
}
