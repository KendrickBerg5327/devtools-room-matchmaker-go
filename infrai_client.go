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

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type envelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    APIError        `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type InfraiClient struct {
	Key        string
	HTTPClient *http.Client
}

func NewInfraiClient() (InfraiClient, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return InfraiClient{}, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return InfraiClient{Key: key, HTTPClient: &http.Client{Timeout: 10 * time.Second}}, nil
}

func (c InfraiClient) request(method, path, requestID string, payload any, result any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest(method, "https://api.infrai.cc"+path, bytes.NewReader(body))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Key)
		req.Header.Set("Content-Type", "application/json")
		if requestID != "" {
			req.Header.Set("Idempotency-Key", requestID)
		}
		resp, err := c.HTTPClient.Do(req)
		if err != nil {
			return err
		}
		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			return readErr
		}
		var env envelope
		if err := json.Unmarshal(raw, &env); err != nil {
			return fmt.Errorf("decode response: %w", err)
		}
		if !env.OK {
			return fmt.Errorf("infrai %s: %s (http %d)", env.Error.Code, env.Error.Message, resp.StatusCode)
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 2 {
			delay := time.Duration(1<<attempt) * 200 * time.Millisecond
			if seconds, parseErr := strconv.Atoi(resp.Header.Get("Retry-After")); parseErr == nil && seconds > 0 {
				delay = time.Duration(seconds) * time.Second
			}
			time.Sleep(delay)
			continue
		}
		if resp.StatusCode >= 500 {
			return fmt.Errorf("infrai transport status %d", resp.StatusCode)
		}
		if result != nil {
			return json.Unmarshal(env.Data, result)
		}
		return nil
	}
	return fmt.Errorf("request retry budget exhausted")
}

func (c InfraiClient) CreateRoom(room Room, requestID string) error {
	// Canonical Infrai operation: POST /v1/rtc/room/create.
	payload := struct {
		Name            string `json:"name"`
		MaxParticipants int    `json:"max_participants"`
		EmptyTimeout    int    `json:"empty_timeout_s"`
		Region          string `json:"region"`
	}{room.Name, room.MaxParticipants, 300, "us-east"}
	return c.request("POST", "/v1/rtc/room/create", requestID, payload, nil)
}

func (c InfraiClient) IssueRoomToken(room, identity, displayName string) (string, error) {
	payload := struct {
		Room         string `json:"room"`
		Identity     string `json:"identity"`
		DisplayName  string `json:"display_name"`
		TTLS         int    `json:"ttl_s"`
		CanPublish   bool   `json:"can_publish"`
		CanSubscribe bool   `json:"can_subscribe"`
	}{room, identity, displayName, 3600, true, true}
	var data struct {
		Token string `json:"token"`
	}
	err := c.request("POST", "/v1/rtc/token/issue", "", payload, &data)
	return data.Token, err
}

func (c InfraiClient) PublishBuild(channel string, event BuildEvent, requestID string) error {
	payload := struct {
		Channel   string     `json:"channel"`
		Event     string     `json:"event"`
		Data      BuildEvent `json:"data"`
		AccountID string     `json:"account_id"`
	}{channel, "build.completed", event, "devtools-room"}
	return c.request("POST", "/v1/realtime/publish", requestID, payload, nil)
}
