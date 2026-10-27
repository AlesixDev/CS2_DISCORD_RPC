package presence

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
)

const (
	artIndexURL   = "https://raw.githubusercontent.com/MurkyYT/cs2-map-icons/main/data/available.json"
	iconURLFormat = "https://raw.githubusercontent.com/MurkyYT/cs2-map-icons/main/images/%s.png"
	artTimeout    = 5 * time.Second

	mainThumbSuffix = "_png.png"

	LogoAsset    = "cs2_logo"
	CTAgentAsset = "agent_ct"
	TAgentAsset  = "agent_t"
)

type MapArt struct {
	Name       string
	Icon       string
	Background string
}

func LoadArt(log *zap.Logger) map[string]MapArt {
	art, err := fetchArt()

	if err == nil {
		return art
	}

	log.Warn("presence: map index", zap.Error(err))
	art = make(map[string]MapArt, len(mapNames))

	for key, name := range mapNames {
		icon := fmt.Sprintf(iconURLFormat, key)
		art[key] = MapArt{Name: name, Icon: icon, Background: icon}
	}

	return art
}

func fetchArt() (map[string]MapArt, error) {
	client := &http.Client{Timeout: artTimeout}
	resp, err := client.Get(artIndexURL)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	var index struct {
		Maps map[string]struct {
			DisplayName string   `json:"display_name"`
			Path        string   `json:"path"`
			Thumbs      []string `json:"thumb_paths"`
		} `json:"maps"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		return nil, err
	}

	art := make(map[string]MapArt, len(index.Maps))

	for key, m := range index.Maps {
		art[key] = MapArt{
			Name:       m.DisplayName,
			Icon:       m.Path,
			Background: pickThumb(key, m.Thumbs, m.Path),
		}
	}

	return art, nil
}

func pickThumb(key string, thumbs []string, fallback string) string {
	for _, thumb := range thumbs {
		if strings.HasSuffix(thumb, "/"+key+mainThumbSuffix) {
			return thumb
		}
	}

	if len(thumbs) > 0 {
		return thumbs[0]
	}

	return fallback
}
