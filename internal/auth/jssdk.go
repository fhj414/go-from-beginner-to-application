package auth

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// JSSDKSigner 缓存公众号 access_token 与 jsapi_ticket，用于生成前端 wx.config 签名。
type JSSDKSigner struct {
	cfg WeChatConfig
	hc  *http.Client

	mu       sync.Mutex
	at       string
	atExpiry time.Time
	ticket   string
	tkExpiry time.Time
}

// NewJSSDKSigner 需 cfg.CanJSSDK() 为真。
func NewJSSDKSigner(cfg WeChatConfig, hc *http.Client) *JSSDKSigner {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &JSSDKSigner{cfg: cfg, hc: hc}
}

type cgiTokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	ErrCode     int    `json:"errcode"`
	ErrMsg      string `json:"errmsg"`
}

type jsapiTicketResp struct {
	Ticket    string `json:"ticket"`
	ExpiresIn int    `json:"expires_in"`
	ErrCode   int    `json:"errcode"`
	ErrMsg    string `json:"errmsg"`
}

func (j *JSSDKSigner) getAccessTokenLocked() (string, error) {
	if j.at != "" && time.Now().Before(j.atExpiry) {
		return j.at, nil
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/token?grant_type=client_credential&appid=%s&secret=%s",
		url.QueryEscape(j.cfg.AppID), url.QueryEscape(j.cfg.AppSecret),
	)
	resp, err := j.hc.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var tr cgiTokenResp
	if err := json.Unmarshal(b, &tr); err != nil {
		return "", err
	}
	if tr.ErrCode != 0 {
		return "", fmt.Errorf("cgi-bin/token errcode=%d %s", tr.ErrCode, tr.ErrMsg)
	}
	if tr.AccessToken == "" {
		return "", errors.New("cgi-bin/token: empty access_token")
	}
	margin := 120
	if tr.ExpiresIn > margin+60 {
		j.atExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-margin) * time.Second)
	} else {
		j.atExpiry = time.Now().Add(60 * time.Second)
	}
	j.at = tr.AccessToken
	return j.at, nil
}

func (j *JSSDKSigner) getTicketLocked() (string, error) {
	if j.ticket != "" && time.Now().Before(j.tkExpiry) {
		return j.ticket, nil
	}
	at, err := j.getAccessTokenLocked()
	if err != nil {
		return "", err
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/cgi-bin/ticket/getticket?access_token=%s&type=jsapi",
		url.QueryEscape(at),
	)
	resp, err := j.hc.Get(u)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	var tr jsapiTicketResp
	if err := json.Unmarshal(b, &tr); err != nil {
		return "", err
	}
	if tr.ErrCode != 0 {
		return "", fmt.Errorf("getticket errcode=%d %s", tr.ErrCode, tr.ErrMsg)
	}
	if tr.Ticket == "" {
		return "", errors.New("getticket: empty ticket")
	}
	margin := 120
	if tr.ExpiresIn > margin+60 {
		j.tkExpiry = time.Now().Add(time.Duration(tr.ExpiresIn-margin) * time.Second)
	} else {
		j.tkExpiry = time.Now().Add(60 * time.Second)
	}
	j.ticket = tr.Ticket
	return j.ticket, nil
}

// JSSDKSignResult 供前端 wx.config 与自定义分享文案使用。
type JSSDKSignResult struct {
	AppID     string   `json:"app_id"`
	Timestamp int64    `json:"timestamp"`
	NonceStr  string   `json:"nonce_str"`
	Signature string   `json:"signature"`
	JSAPIList []string `json:"js_api_list"`

	ShareTitle string `json:"share_title"`
	ShareDesc  string `json:"share_desc"`
	ShareLink  string `json:"share_link"`
	ShareImg   string `json:"share_img"`
	Debug      bool   `json:"debug"`
}

// Sign 对当前页面 URL（不含 # 及其后片段）做 SHA1 签名。
func (j *JSSDKSigner) Sign(pageURL string) (*JSSDKSignResult, error) {
	if i := strings.Index(pageURL, "#"); i >= 0 {
		pageURL = pageURL[:i]
	}
	pu, err := url.Parse(pageURL)
	if err != nil || pu.Scheme == "" || pu.Host == "" {
		return nil, errors.New("invalid page url")
	}

	j.mu.Lock()
	defer j.mu.Unlock()

	ticket, err := j.getTicketLocked()
	if err != nil {
		return nil, err
	}
	ts := time.Now().Unix()
	nonce, err := randomNonceStr()
	if err != nil {
		return nil, err
	}
	plain := fmt.Sprintf("jsapi_ticket=%s&noncestr=%s&timestamp=%d&url=%s", ticket, nonce, ts, pageURL)
	sum := sha1.Sum([]byte(plain))
	sig := hex.EncodeToString(sum[:])

	origin := pu.Scheme + "://" + pu.Host
	shareLink := origin + "/"
	shareImg := origin + "/static/wx-share.png"

	return &JSSDKSignResult{
		AppID:     j.cfg.AppID,
		Timestamp: ts,
		NonceStr:  nonce,
		Signature: sig,
		JSAPIList: []string{"updateAppMessageShareData", "updateTimelineShareData"},
		ShareLink: shareLink,
		ShareImg:  shareImg,
	}, nil
}

func randomNonceStr() (string, error) {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf[:]), nil
}
