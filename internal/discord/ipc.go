package discord

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"
)

const (
	opHandshake uint32 = 0
	opFrame     uint32 = 1
	opClose     uint32 = 2

	pipeFormat = `\\.\pipe\discord-ipc-%d`
	maxPipes   = 10
	rpcVersion = 1

	cmdSetActivity = "SET_ACTIVITY"
	evtError       = "ERROR"
)

var ErrNotRunning = errors.New("discord is not running")

type Activity struct {
	Details    string      `json:"details,omitempty"`
	State      string      `json:"state,omitempty"`
	Timestamps *Timestamps `json:"timestamps,omitempty"`
	Assets     *Assets     `json:"assets,omitempty"`
}

type Timestamps struct {
	Start int64 `json:"start,omitempty"`
}

type Assets struct {
	LargeImage string `json:"large_image,omitempty"`
	LargeText  string `json:"large_text,omitempty"`
	SmallImage string `json:"small_image,omitempty"`
	SmallText  string `json:"small_text,omitempty"`
}

type Client struct {
	clientID string
	pipe     *os.File
}

func New(clientID string) *Client {
	return &Client{clientID: clientID}
}

func (c *Client) Connected() bool {
	return c.pipe != nil
}

func (c *Client) Connect() error {
	c.Close()

	for i := range maxPipes {
		pipe, err := os.OpenFile(fmt.Sprintf(pipeFormat, i), os.O_RDWR, 0)

		if err != nil {
			continue
		}

		c.pipe = pipe

		if err := c.handshake(); err != nil {
			c.Close()
			return err
		}

		return nil
	}

	return ErrNotRunning
}

func (c *Client) Close() {
	if c.pipe == nil {
		return
	}

	c.pipe.Close()
	c.pipe = nil
}

func (c *Client) SetActivity(activity *Activity) error {
	payload := map[string]any{
		"cmd":   cmdSetActivity,
		"nonce": strconv.FormatInt(time.Now().UnixNano(), 10),
		"args": map[string]any{
			"pid":      os.Getpid(),
			"activity": activity,
		},
	}

	if err := c.send(opFrame, payload); err != nil {
		c.Close()
		return err
	}

	op, body, err := c.read()

	if err != nil {
		c.Close()
		return err
	}

	if op == opClose {
		c.Close()
		return fmt.Errorf("discord closed the connection: %s", body)
	}

	var resp struct {
		Evt  string `json:"evt"`
		Data struct {
			Message string `json:"message"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return err
	}

	if resp.Evt == evtError {
		return errors.New(resp.Data.Message)
	}

	return nil
}

func (c *Client) handshake() error {
	if err := c.send(opHandshake, map[string]any{"v": rpcVersion, "client_id": c.clientID}); err != nil {
		return err
	}

	op, body, err := c.read()

	if err != nil {
		return err
	}

	if op == opClose {
		return fmt.Errorf("handshake rejected (correct client id?): %s", body)
	}

	return nil
}

func (c *Client) send(op uint32, payload any) error {
	if c.pipe == nil {
		return ErrNotRunning
	}

	body, err := json.Marshal(payload)

	if err != nil {
		return err
	}

	frame := make([]byte, 8+len(body))
	binary.LittleEndian.PutUint32(frame[0:4], op)
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(body)))
	copy(frame[8:], body)

	_, err = c.pipe.Write(frame)

	return err
}

func (c *Client) read() (uint32, []byte, error) {
	header := make([]byte, 8)

	if _, err := io.ReadFull(c.pipe, header); err != nil {
		return 0, nil, err
	}

	body := make([]byte, binary.LittleEndian.Uint32(header[4:8]))

	if _, err := io.ReadFull(c.pipe, body); err != nil {
		return 0, nil, err
	}

	return binary.LittleEndian.Uint32(header[0:4]), body, nil
}
