package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// SessionClaims 放在签名 cookie 里（体量小，不含敏感隐私）。
type SessionClaims struct {
	UserID    string `json:"uid"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
}

// Codec 用 HMAC-SHA256 签发/校验会话（教学用；线上可换 JWT 库或 Redis session）。
type Codec struct {
	secret []byte
}

func NewCodec(secret string) (*Codec, error) {
	if strings.TrimSpace(secret) == "" {
		return nil, errors.New("session secret is empty")
	}
	return &Codec{secret: []byte(secret)}, nil
}

func (c *Codec) Sign(claims SessionClaims) (string, error) {
	b, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, c.secret)
	if _, err := mac.Write(b); err != nil {
		return "", err
	}
	sig := mac.Sum(nil)
	token := base64.RawURLEncoding.EncodeToString(b) + "." + base64.RawURLEncoding.EncodeToString(sig)
	return token, nil
}

func (c *Codec) Verify(token string) (SessionClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return SessionClaims{}, errors.New("bad token")
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return SessionClaims{}, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return SessionClaims{}, err
	}
	mac := hmac.New(sha256.New, c.secret)
	if _, err := mac.Write(raw); err != nil {
		return SessionClaims{}, err
	}
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return SessionClaims{}, errors.New("bad signature")
	}
	var claims SessionClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return SessionClaims{}, err
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return SessionClaims{}, errors.New("expired")
	}
	return claims, nil
}

// RandomState 生成 OAuth state。
func RandomState() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b[:]), nil
}
