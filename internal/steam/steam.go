package steam

import (
	"errors"
	"os"
	"strconv"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	appID = "730"

	keyGroupSize = "steam_player_group_size"

	errBufSize = 1024
)

type Presence struct {
	dll     *syscall.LazyDLL
	friends uintptr
	steamID uintptr
	key     []byte
}

func Open(dllPath string) (*Presence, error) {
	os.Setenv("SteamAppId", appID)
	os.Setenv("SteamGameId", appID)

	dll := syscall.NewLazyDLL(dllPath)

	if err := dll.Load(); err != nil {
		return nil, err
	}

	errBuf := make([]byte, errBufSize)
	r, _, _ := dll.NewProc("SteamAPI_InitFlat").Call(uintptr(unsafe.Pointer(&errBuf[0])))

	if r != 0 {
		return nil, errors.New("steam: init: " + cString(uintptr(unsafe.Pointer(&errBuf[0]))))
	}

	user, _, _ := dll.NewProc("SteamAPI_SteamUser_v023").Call()
	friends, _, _ := dll.NewProc("SteamAPI_SteamFriends_v018").Call()

	if user == 0 || friends == 0 {
		dll.NewProc("SteamAPI_Shutdown").Call()
		return nil, errors.New("steam: interfaces unavailable")
	}

	steamID, _, _ := dll.NewProc("SteamAPI_ISteamUser_GetSteamID").Call(user)

	return &Presence{
		dll:     dll,
		friends: friends,
		steamID: steamID,
		key:     append([]byte(keyGroupSize), 0),
	}, nil
}

func (p *Presence) GroupSize() int {
	p.dll.NewProc("SteamAPI_RunCallbacks").Call()

	value, _, _ := p.dll.NewProc("SteamAPI_ISteamFriends_GetFriendRichPresence").Call(
		p.friends, p.steamID, uintptr(unsafe.Pointer(&p.key[0])),
	)
	size, err := strconv.Atoi(cString(value))

	if err != nil || size < 1 {
		return 1
	}

	return size
}

func (p *Presence) Close() {
	p.dll.NewProc("SteamAPI_Shutdown").Call()
}

func cString(ptr uintptr) string {
	if ptr == 0 {
		return ""
	}

	return windows.BytePtrToString(*(**byte)(unsafe.Pointer(&ptr)))
}
