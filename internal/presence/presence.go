package presence

import (
	"fmt"
	"time"

	"CS2_DISCORD_RPC/internal/discord"
	"CS2_DISCORD_RPC/internal/gsi"
)

const (
	gameName = "Counter-Strike 2"

	contextMenu = "menu"

	phaseWarmup   = "warmup"
	phaseGameover = "gameover"

	modeDeathmatch = "deathmatch"

	teamCT = "CT"
	teamT  = "T"
)

var groupLabels = map[int]string{
	1: "Solo",
	2: "Duo",
	3: "Trio",
	4: "Squad",
	5: "5-stack",
}

var teamLabels = map[string]string{
	teamCT: "Counter-Terrorist",
	teamT:  "Terrorist",
}

type Images struct {
	Logo      string
	CT        string
	T         string
	MapImages map[string]string
}

type Builder struct {
	images Images
	art    map[string]MapArt

	context string
	since   int64
	team    string
	stats   *gsi.MatchStats
}

func NewBuilder(images Images, art map[string]MapArt) *Builder {
	return &Builder{images: images, art: art}
}

func (b *Builder) Reset() {
	b.context = ""
	b.team = ""
	b.stats = nil
}

func (b *Builder) Build(s gsi.State, group int) *discord.Activity {
	if s.Map == nil {
		b.enter(contextMenu)

		return &discord.Activity{
			Details:    "In the main menu",
			State:      withGroup("Idle", group),
			Timestamps: &discord.Timestamps{Start: b.since},
			Assets:     &discord.Assets{LargeImage: b.images.Logo, LargeText: gameName},
		}
	}

	key := mapKey(s.Map.Name)
	b.enter(s.Map.Mode + "|" + key)

	if s.Own() {
		b.team = s.Player.Team
		b.stats = s.Player.MatchStats
	}

	art := b.mapArt(key)
	activity := &discord.Activity{
		Details:    fmt.Sprintf("%s · %s", modeTitle(s.Map.Mode), art.Name),
		State:      withGroup(b.status(s), group),
		Timestamps: &discord.Timestamps{Start: b.since},
		Assets: &discord.Assets{
			LargeImage: art.Background,
			LargeText:  art.Name,
			SmallImage: art.Icon,
			SmallText:  art.Name,
		},
	}

	if label, ok := teamLabels[b.team]; ok {
		activity.Assets.SmallImage = b.teamImage()
		activity.Assets.SmallText = label + b.statsSuffix()
	}

	return activity
}

func (b *Builder) enter(context string) {
	if b.context == context {
		return
	}

	b.context = context
	b.since = time.Now().Unix()
	b.team = ""
	b.stats = nil
}

func (b *Builder) status(s gsi.State) string {
	if s.Map.Phase == phaseWarmup {
		return "Warmup"
	}

	if s.Map.Mode == modeDeathmatch {
		if b.stats == nil {
			return "Playing"
		}

		return fmt.Sprintf("%d kills · %d deaths", b.stats.Kills, b.stats.Deaths)
	}

	ours, theirs, known := b.score(s.Map)

	if s.Map.Phase == phaseGameover {
		return gameoverText(ours, theirs, known, s.Map)
	}

	round := fmt.Sprintf("Round %d", s.Map.Round+1)

	if !known {
		return fmt.Sprintf("CT %d - %d T · %s", s.Map.TeamCT.Score, s.Map.TeamT.Score, round)
	}

	return fmt.Sprintf("%d - %d · %s", ours, theirs, round)
}

func (b *Builder) score(m *gsi.Map) (int, int, bool) {
	if b.team == teamCT {
		return m.TeamCT.Score, m.TeamT.Score, true
	}

	if b.team == teamT {
		return m.TeamT.Score, m.TeamCT.Score, true
	}

	return 0, 0, false
}

func (b *Builder) statsSuffix() string {
	if b.stats == nil {
		return ""
	}

	suffix := fmt.Sprintf(" · %d/%d/%d", b.stats.Kills, b.stats.Deaths, b.stats.Assists)

	if b.stats.MVPs > 0 {
		suffix += fmt.Sprintf(" · %d MVP", b.stats.MVPs)
	}

	return suffix
}

func (b *Builder) mapArt(key string) MapArt {
	art, ok := b.art[key]

	if !ok {
		art = MapArt{Icon: b.images.Logo, Background: b.images.Logo}
	}

	if art.Name == "" {
		art.Name = mapTitle(key)
	}

	if img, ok := b.images.MapImages[key]; ok {
		art.Background = img
	}

	return art
}

func (b *Builder) teamImage() string {
	if b.team == teamCT {
		return b.images.CT
	}

	return b.images.T
}

func withGroup(status string, group int) string {
	label, ok := groupLabels[group]

	if !ok {
		return status
	}

	return label + " · " + status
}

func gameoverText(ours, theirs int, known bool, m *gsi.Map) string {
	if !known {
		return fmt.Sprintf("Match over · CT %d - %d T", m.TeamCT.Score, m.TeamT.Score)
	}

	if ours > theirs {
		return fmt.Sprintf("Victory %d - %d", ours, theirs)
	}

	if ours < theirs {
		return fmt.Sprintf("Defeat %d - %d", ours, theirs)
	}

	return fmt.Sprintf("Draw %d - %d", ours, theirs)
}
