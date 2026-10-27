package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/sys/windows"

	"CS2_DISCORD_RPC/internal/config"
	"CS2_DISCORD_RPC/internal/discord"
	"CS2_DISCORD_RPC/internal/gsi"
	"CS2_DISCORD_RPC/internal/presence"
	"CS2_DISCORD_RPC/internal/steam"
)

const (
	appName     = "CS2 Discord RPC"
	logFileName = "cs2-discord-rpc.log"

	steamDLLRelPath = `..\..\bin\win64\steam_api64.dll`

	tickEvery      = time.Second
	staleAfter     = 30 * time.Second
	reconnectEvery = 10 * time.Second
	minUpdateGap   = 4 * time.Second

	mbIconInfo  = 0x40
	mbIconError = 0x10
)

func main() {
	log := newLogger()
	defer log.Sync()

	if len(os.Args) < 2 {
		setup(log)
		return
	}

	os.Exit(wrap(os.Args[1:], log))
}

func setup(log *zap.Logger) {
	if _, _, err := prepare(log); err != nil {
		alert(err.Error(), mbIconError)
		return
	}

	exe, err := os.Executable()

	if err != nil {
		alert(err.Error(), mbIconError)
		return
	}

	alert(fmt.Sprintf(
		"All set.\n\nIn Steam: right-click Counter-Strike 2 → Properties → Launch Options, and enter:\n\n\"%s\" %%command%%\n\n(Ctrl+C copies this whole message.)",
		exe,
	), mbIconInfo)
}

func wrap(command []string, log *zap.Logger) int {
	cfg, cfgDir, setupErr := prepare(log)

	game := exec.Command(command[0], command[1:]...)
	game.Stdin, game.Stdout, game.Stderr = os.Stdin, os.Stdout, os.Stderr

	if err := game.Start(); err != nil {
		log.Error("game: start", zap.Strings("command", command), zap.Error(err))
		alert(fmt.Sprintf("Could not start CS2: %v", err), mbIconError)
		return 1
	}

	log.Info("game: started", zap.Int("pid", game.Process.Pid))

	done := make(chan struct{})

	go func() {
		game.Wait()
		close(done)
	}()

	if setupErr != nil {
		log.Error("setup", zap.Error(setupErr))
		go alert(setupErr.Error(), mbIconError)
		<-done
		return game.ProcessState.ExitCode()
	}

	server := gsi.NewServer(cfg.Token, log)
	addr := fmt.Sprintf("127.0.0.1:%d", cfg.Port)

	go func() {
		if err := server.ListenAndServe(addr); err != nil {
			log.Error("gsi: server", zap.Error(err))
		}
	}()

	log.Info("gsi: listening", zap.String("addr", addr))

	images := presence.Images{
		Logo:      orDefault(cfg.LogoImage, presence.LogoAsset),
		CT:        orDefault(cfg.CTImage, presence.CTAgentAsset),
		T:         orDefault(cfg.TImage, presence.TAgentAsset),
		MapImages: cfg.MapImages,
	}
	builder := presence.NewBuilder(images, presence.LoadArt(log))
	steamDLL := filepath.Join(cfgDir, steamDLLRelPath)
	loop(server, discord.New(cfg.DiscordClientID), builder, steamDLL, done, log)
	log.Info("game: closed", zap.Int("code", game.ProcessState.ExitCode()))

	return game.ProcessState.ExitCode()
}

func prepare(log *zap.Logger) (*config.Config, string, error) {
	cfg, path, created, err := config.Load()

	if err != nil {
		return nil, "", fmt.Errorf("config: %w", err)
	}

	if created || cfg.DiscordClientID == "" {
		return nil, "", fmt.Errorf(
			"Missing Discord ID.\n\n1. Create an application named \"Counter-Strike 2\" at https://discord.com/developers/applications\n2. Copy its Application ID into \"discord_client_id\" in:\n%s\n3. Open this program again.",
			path,
		)
	}

	dir := cfg.CS2CfgDir

	if dir == "" {
		found, err := gsi.FindCfgDir()

		if err != nil {
			return nil, "", fmt.Errorf("gsi: %w (set the path manually in cs2_cfg_dir of %s)", err, path)
		}

		dir = found
	}

	cfgPath, changed, err := gsi.Install(dir, cfg.Port, cfg.Token)

	if err != nil {
		return nil, "", fmt.Errorf("gsi: install cfg: %w", err)
	}

	if changed {
		log.Info("gsi: cfg installed", zap.String("file", cfgPath))
	}

	return cfg, dir, nil
}

func loop(server *gsi.Server, client *discord.Client, builder *presence.Builder, steamDLL string, done <-chan struct{}, log *zap.Logger) {
	ticker := time.NewTicker(tickEvery)
	defer ticker.Stop()

	var (
		lastDial time.Time
		lastSend time.Time
		sent     string
		synced   bool

		party      *steam.Presence
		steamTried bool
	)

	for {
		select {
		case <-done:
			if party != nil {
				party.Close()
			}

			if client.Connected() {
				client.SetActivity(nil)
				client.Close()
			}

			return
		case <-ticker.C:
		}

		state, at := server.Latest()
		var activity *discord.Activity

		if at.IsZero() || time.Since(at) > staleAfter {
			builder.Reset()
		} else {
			if !steamTried {
				steamTried = true
				opened, err := steam.Open(steamDLL)

				if err != nil {
					log.Warn("steam: open", zap.Error(err))
				}

				party = opened
			}

			activity = builder.Build(state, groupSize(party))
		}

		if !client.Connected() {
			if time.Since(lastDial) < reconnectEvery {
				continue
			}

			lastDial = time.Now()

			if err := client.Connect(); err != nil {
				if !errors.Is(err, discord.ErrNotRunning) {
					log.Warn("discord: connect", zap.Error(err))
				}

				continue
			}

			log.Info("discord: connected")
			synced = false
		}

		raw, _ := json.Marshal(activity)

		if synced && string(raw) == sent {
			continue
		}

		if time.Since(lastSend) < minUpdateGap {
			continue
		}

		lastSend = time.Now()

		if err := client.SetActivity(activity); err != nil {
			log.Warn("discord: activity", zap.Error(err))
			continue
		}

		sent = string(raw)
		synced = true
		log.Info("discord: activity", zap.String("state", sent))
	}
}

func newLogger() *zap.Logger {
	file, err := os.Create(filepath.Join(config.Dir(), logFileName))

	if err != nil {
		return zap.NewNop()
	}

	encoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	return zap.New(zapcore.NewCore(encoder, zapcore.AddSync(file), zap.InfoLevel))
}

func groupSize(party *steam.Presence) int {
	if party == nil {
		return 0
	}

	return party.GroupSize()
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}

	return value
}

func alert(text string, icon uint32) {
	body, _ := windows.UTF16PtrFromString(text)
	title, _ := windows.UTF16PtrFromString(appName)
	windows.MessageBox(0, body, title, icon)
}
