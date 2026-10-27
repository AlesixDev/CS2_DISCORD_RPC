package gsi

type State struct {
	Provider struct {
		SteamID string `json:"steamid"`
	} `json:"provider"`
	Map    *Map    `json:"map"`
	Round  *Round  `json:"round"`
	Player *Player `json:"player"`
	Auth   struct {
		Token string `json:"token"`
	} `json:"auth"`
}

type Map struct {
	Mode   string `json:"mode"`
	Name   string `json:"name"`
	Phase  string `json:"phase"`
	Round  int    `json:"round"`
	TeamCT Team   `json:"team_ct"`
	TeamT  Team   `json:"team_t"`
}

type Team struct {
	Score int `json:"score"`
}

type Round struct {
	Phase   string `json:"phase"`
	WinTeam string `json:"win_team"`
}

type Player struct {
	SteamID    string      `json:"steamid"`
	Name       string      `json:"name"`
	Team       string      `json:"team"`
	Activity   string      `json:"activity"`
	MatchStats *MatchStats `json:"match_stats"`
}

type MatchStats struct {
	Kills   int `json:"kills"`
	Assists int `json:"assists"`
	Deaths  int `json:"deaths"`
	MVPs    int `json:"mvps"`
	Score   int `json:"score"`
}

func (s State) Own() bool {
	return s.Player != nil && s.Player.SteamID != "" && s.Player.SteamID == s.Provider.SteamID
}
