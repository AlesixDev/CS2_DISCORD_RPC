package gsi

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const (
	cfgFileName    = "gamestate_integration_discordrpc.cfg"
	steamRegKey    = `Software\Valve\Steam`
	steamRegValue  = "SteamPath"
	libraryFolders = `steamapps\libraryfolders.vdf`
	cs2CfgSubdir   = `steamapps\common\Counter-Strike Global Offensive\game\csgo\cfg`

	cfgTemplate = `"CS2 Discord RPC"
{
	"uri"       "http://127.0.0.1:%d/"
	"timeout"   "5.0"
	"buffer"    "0.1"
	"throttle"  "0.5"
	"heartbeat" "10.0"
	"auth"
	{
		"token" "%s"
	}
	"data"
	{
		"provider"           "1"
		"map"                "1"
		"round"              "1"
		"player_id"          "1"
		"player_state"       "1"
		"player_match_stats" "1"
	}
}
`
)

var (
	libraryPathRe = regexp.MustCompile(`"path"\s+"([^"]+)"`)

	ErrCS2NotFound = errors.New("CS2 cfg folder not found")
)

func FindCfgDir() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, steamRegKey, registry.QUERY_VALUE)

	if err != nil {
		return "", fmt.Errorf("steam is not installed: %w", err)
	}

	defer key.Close()

	steamPath, _, err := key.GetStringValue(steamRegValue)

	if err != nil {
		return "", err
	}

	libraries := []string{filepath.FromSlash(steamPath)}
	raw, err := os.ReadFile(filepath.Join(libraries[0], libraryFolders))

	if err == nil {
		for _, m := range libraryPathRe.FindAllSubmatch(raw, -1) {
			libraries = append(libraries, strings.ReplaceAll(string(m[1]), `\\`, `\`))
		}
	}

	for _, lib := range libraries {
		dir := filepath.Join(lib, cs2CfgSubdir)

		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir, nil
		}
	}

	return "", ErrCS2NotFound
}

func Install(dir string, port int, token string) (string, bool, error) {
	path := filepath.Join(dir, cfgFileName)
	content := []byte(fmt.Sprintf(cfgTemplate, port, token))
	current, err := os.ReadFile(path)

	if err == nil && bytes.Equal(current, content) {
		return path, false, nil
	}

	if err := os.WriteFile(path, content, 0o644); err != nil {
		return path, false, err
	}

	return path, true, nil
}
