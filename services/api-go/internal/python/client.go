package python

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	base   string
	client *http.Client
}

func New() *Client {
	base := os.Getenv("AI_PYTHON_URL")
	if base == "" {
		base = "http://127.0.0.1:8090"
	}
	return &Client{
		base:   base,
		client: &http.Client{Timeout: 180 * time.Second},
	}
}

func (c *Client) get(path string, dest interface{}) error {
	res, err := c.client.Get(c.base + path)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("python %s: %s", res.Status, string(body))
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) post(path string, payload interface{}, dest interface{}) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	res, err := c.client.Post(c.base+path, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode >= 400 {
		return fmt.Errorf("python %s: %s", res.Status, string(body))
	}
	return json.Unmarshal(body, dest)
}

func (c *Client) Algorithms() (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.get("/v1/algorithms", &dest)
	return dest, err
}

func (c *Client) Screening(payload json.RawMessage) (json.RawMessage, error) {
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	var dest json.RawMessage
	err := c.post("/v1/screening/run", payload, &dest)
	return dest, err
}

func (c *Client) Models() (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.get("/v1/llm/models", &dest)
	return dest, err
}

func (c *Client) Profiles() (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.get("/v1/llm/profiles", &dest)
	return dest, err
}

func (c *Client) Analyze(payload json.RawMessage) (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.post("/v1/ai/analyze", payload, &dest)
	return dest, err
}

func (c *Client) DailyNote(symbol string, force bool) (json.RawMessage, error) {
	var dest json.RawMessage
	path := "/v1/ai/daily-note?symbol=" + url.QueryEscape(symbol)
	if force {
		path += "&force=1"
	}
	err := c.get(path, &dest)
	return dest, err
}
