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

func (c *Client) Agents() (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.get("/v1/agents", &dest)
	return dest, err
}

func (c *Client) Analyze(payload json.RawMessage) (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.post("/v1/ai/analyze", payload, &dest)
	return dest, err
}

// AnalyzeStream 消费 Python NDJSON 进度流；onProgress 收到 progress 事件，最终返回 result。
func (c *Client) AnalyzeStream(payload json.RawMessage, onProgress func(step, title, status, summary string)) (json.RawMessage, error) {
	if len(payload) == 0 {
		payload = json.RawMessage(`{}`)
	}
	res, err := c.client.Post(c.base+"/v1/ai/analyze/stream", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return nil, fmt.Errorf("python %s: %s", res.Status, string(body))
	}
	dec := json.NewDecoder(res.Body)
	var result json.RawMessage
	for {
		var ev map[string]interface{}
		if err := dec.Decode(&ev); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		switch fmt.Sprint(ev["event"]) {
		case "progress":
			if onProgress != nil {
				onProgress(
					fmt.Sprint(ev["step"]),
					fmt.Sprint(ev["title"]),
					fmt.Sprint(ev["status"]),
					fmt.Sprint(ev["summary"]),
				)
			}
		case "result":
			raw, err := json.Marshal(ev["data"])
			if err != nil {
				return nil, err
			}
			result = raw
		case "error":
			return nil, fmt.Errorf("%v", ev["message"])
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("analyze stream empty")
	}
	return result, nil
}

func (c *Client) Chat(payload json.RawMessage) (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.post("/v1/ai/chat", payload, &dest)
	return dest, err
}

func (c *Client) ScreeningNL(payload interface{}) (json.RawMessage, error) {
	var dest json.RawMessage
	err := c.post("/v1/ai/screening-nl", payload, &dest)
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
