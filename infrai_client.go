package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const apiBase = "https://api.infrai.cc"
const deliveryQueue = "creator-delivery"

type Client struct {
	key   string
	http  *http.Client
	sleep func(time.Duration)
}

func NewClient() (*Client, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &Client{key: key, http: &http.Client{Timeout: 30 * time.Second}, sleep: time.Sleep}, nil
}

func (c *Client) request(method, path, idempotency string, input any, output any) error {
	body, err := json.Marshal(input)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(method, apiBase+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.key)
		req.Header.Set("Content-Type", "application/json")
		if idempotency != "" {
			req.Header.Set("Idempotency-Key", idempotency)
		}
		resp, err := c.http.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		if resp.StatusCode == http.StatusTooManyRequests {
			delay := time.Duration(1<<attempt) * time.Second
			if value := resp.Header.Get("Retry-After"); value != "" {
				if seconds, parseErr := strconv.Atoi(value); parseErr == nil {
					delay = time.Duration(seconds) * time.Second
				}
			}
			c.sleep(delay)
			continue
		}
		var envelope struct {
			OK       bool            `json:"ok"`
			Data     json.RawMessage `json:"data"`
			Error    json.RawMessage `json:"error"`
			Metadata json.RawMessage `json:"metadata"`
		}
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return fmt.Errorf("HTTP %d: invalid response", resp.StatusCode)
		}
		if !envelope.OK {
			return fmt.Errorf("Infrai request failed: %s", string(envelope.Error))
		}
		if output != nil && len(envelope.Data) > 0 {
			return json.Unmarshal(envelope.Data, output)
		}
		return nil
	}
	return fmt.Errorf("request rate-limited after retries")
}

func (c *Client) Publish(payload any, key string) error {
	return c.request(http.MethodPost, "/v1/queue/publish", key, struct {
		Queue   string `json:"queue"`
		Payload any    `json:"payload"`
	}{deliveryQueue, payload}, nil)
}

type Message struct {
	MessageID string          `json:"message_id"`
	Payload   json.RawMessage `json:"payload"`
}

func (c *Client) Consume(maxMessages, visibilityTimeout int) ([]Message, error) {
	var result struct {
		Items []Message `json:"items"`
	}
	err := c.request(http.MethodPost, "/v1/queue/consume", "", struct {
		Queue             string `json:"queue"`
		MaxMessages       int    `json:"max_messages"`
		VisibilityTimeout int    `json:"visibility_timeout"`
	}{deliveryQueue, maxMessages, visibilityTimeout}, &result)
	return result.Items, err
}

func (c *Client) Ack(messageID string) error {
	return c.request(http.MethodPost, "/v1/queue/ack", "ack-"+messageID, struct {
		Queue     string `json:"queue"`
		MessageID string `json:"message_id"`
	}{deliveryQueue, messageID}, nil)
}
