package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

func (b *redisBackend) userKey(id string) string {
	return b.prefix + ":user:" + id
}

func (b *redisBackend) usersKey() string {
	return b.prefix + ":users"
}

func (b *redisBackend) upsertUser(ctx context.Context, u *User) error {
	raw, err := json.Marshal(u)
	if err != nil {
		return err
	}
	if _, err := b.cmd(ctx, "SET", b.userKey(u.ID), string(raw)); err != nil {
		return err
	}
	_, err = b.cmd(ctx, "SADD", b.usersKey(), u.ID)
	return err
}

func (b *redisBackend) getUser(ctx context.Context, id string) (*User, bool, error) {
	raw, err := b.cmd(ctx, "GET", b.userKey(id))
	if err != nil {
		return nil, false, err
	}
	if string(raw) == "null" || len(raw) == 0 {
		return nil, false, nil
	}
	var payload string
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, false, err
	}
	if strings.TrimSpace(payload) == "" {
		return nil, false, nil
	}
	var u User
	if err := json.Unmarshal([]byte(payload), &u); err != nil {
		return nil, false, err
	}
	return cloneUser(&u), true, nil
}

func (b *redisBackend) allUsers(ctx context.Context) ([]*User, error) {
	raw, err := b.cmd(ctx, "SMEMBERS", b.usersKey())
	if err != nil {
		return nil, err
	}
	var ids []string
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, err
	}
	out := make([]*User, 0, len(ids))
	for _, id := range ids {
		u, ok, err := b.getUser(ctx, id)
		if err != nil {
			return nil, err
		}
		if ok && u != nil {
			out = append(out, u)
		}
	}
	return out, nil
}

func (b *redisBackend) cmd(ctx context.Context, args ...string) (json.RawMessage, error) {
	body, err := json.Marshal(args)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("upstash redis: %s", strings.TrimSpace(string(raw)))
	}
	var envelope struct {
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if strings.TrimSpace(envelope.Error) != "" {
		return nil, fmt.Errorf("upstash redis: %s", envelope.Error)
	}
	return envelope.Result, nil
}
