package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const (
	fileName    = "config.json"
	defaultPort = 3000
	tokenBytes  = 16
)

type Config struct {
	DiscordClientID string            `json:"discord_client_id"`
	Port            int               `json:"port"`
	Token           string            `json:"token"`
	CS2CfgDir       string            `json:"cs2_cfg_dir"`
	LogoImage       string            `json:"logo_image"`
	CTImage         string            `json:"ct_image"`
	TImage          string            `json:"t_image"`
	MapImages       map[string]string `json:"map_images"`
}

func Load() (*Config, string, bool, error) {
	path := filepath.Join(Dir(), fileName)
	raw, err := os.ReadFile(path)

	if errors.Is(err, fs.ErrNotExist) {
		cfg, err := defaults()

		if err != nil {
			return nil, path, false, err
		}

		return cfg, path, true, save(path, cfg)
	}

	if err != nil {
		return nil, path, false, err
	}

	var cfg Config

	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, path, false, err
	}

	if cfg.Port == 0 {
		cfg.Port = defaultPort
	}

	if cfg.Token == "" {
		token, err := newToken()

		if err != nil {
			return nil, path, false, err
		}

		cfg.Token = token

		return &cfg, path, false, save(path, &cfg)
	}

	return &cfg, path, false, nil
}

func defaults() (*Config, error) {
	token, err := newToken()

	if err != nil {
		return nil, err
	}

	return &Config{
		Port:      defaultPort,
		Token:     token,
		MapImages: map[string]string{},
	}, nil
}

func save(path string, cfg *Config) error {
	raw, err := json.MarshalIndent(cfg, "", "  ")

	if err != nil {
		return err
	}

	return os.WriteFile(path, raw, 0o600)
}

func newToken() (string, error) {
	buf := make([]byte, tokenBytes)

	if _, err := rand.Read(buf); err != nil {
		return "", err
	}

	return hex.EncodeToString(buf), nil
}

func Dir() string {
	exe, err := os.Executable()

	if err != nil {
		return "."
	}

	dir := filepath.Dir(exe)

	if strings.HasPrefix(strings.ToLower(dir), strings.ToLower(os.TempDir())) {
		return "."
	}

	return dir
}
