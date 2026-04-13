package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// WeChatConfig 从环境变量读取；未配置时仍可体验「演示登录」。
type WeChatConfig struct {
	AppID       string
	AppSecret   string
	RedirectURL string // 完整回调 URL，需与公众号后台配置一致
}

func LoadWeChatConfig() WeChatConfig {
	return WeChatConfig{
		AppID:       strings.TrimSpace(os.Getenv("WECHAT_APP_ID")),
		AppSecret:   strings.TrimSpace(os.Getenv("WECHAT_APP_SECRET")),
		RedirectURL: strings.TrimSpace(os.Getenv("WECHAT_REDIRECT_URL")),
	}
}

func (c WeChatConfig) Enabled() bool {
	return c.AppID != "" && c.AppSecret != "" && c.RedirectURL != ""
}

// CanJSSDK 为真时可签发 JSSDK 签名（仅需 AppID+Secret，不依赖网页授权回调）。
func (c WeChatConfig) CanJSSDK() bool {
	return strings.TrimSpace(c.AppID) != "" && strings.TrimSpace(c.AppSecret) != ""
}

// AuthorizeURL 构造微信网页授权跳转地址（服务号 snsapi_userinfo）。
func (c WeChatConfig) AuthorizeURL(state string) string {
	ru := url.QueryEscape(c.RedirectURL)
	return fmt.Sprintf(
		"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
		url.QueryEscape(c.AppID), ru, url.QueryEscape(state),
	)
}

type wechatTokenResp struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	ErrCode      int    `json:"errcode"`
	ErrMsg       string `json:"errmsg"`
}

type wechatUserInfo struct {
	OpenID   string `json:"openid"`
	Nickname string `json:"nickname"`
	HeadImg  string `json:"headimgurl"`
	ErrCode  int    `json:"errcode"`
	ErrMsg   string `json:"errmsg"`
}

// ExchangeWeChatCode 用 code 换 openid 与用户信息。
func ExchangeWeChatCode(cfg WeChatConfig, code string, hc *http.Client) (openid, nickname, avatar string, err error) {
	if hc == nil {
		hc = http.DefaultClient
	}
	u := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		url.QueryEscape(cfg.AppID), url.QueryEscape(cfg.AppSecret), url.QueryEscape(code),
	)
	resp, err := hc.Get(u)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", "", err
	}
	var tr wechatTokenResp
	if err := json.Unmarshal(b, &tr); err != nil {
		return "", "", "", err
	}
	if tr.ErrCode != 0 {
		return "", "", "", fmt.Errorf("wechat token: %d %s", tr.ErrCode, tr.ErrMsg)
	}
	if tr.OpenID == "" || tr.AccessToken == "" {
		return "", "", "", errors.New("wechat token: empty openid/access_token")
	}

	u2 := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		url.QueryEscape(tr.AccessToken), url.QueryEscape(tr.OpenID),
	)
	resp2, err := hc.Get(u2)
	if err != nil {
		return tr.OpenID, "", "", err
	}
	defer resp2.Body.Close()
	b2, err := io.ReadAll(io.LimitReader(resp2.Body, 1<<20))
	if err != nil {
		return tr.OpenID, "", "", err
	}
	var ui wechatUserInfo
	if err := json.Unmarshal(b2, &ui); err != nil {
		return tr.OpenID, "", "", err
	}
	if ui.ErrCode != 0 {
		return tr.OpenID, "", "", fmt.Errorf("wechat userinfo: %d %s", ui.ErrCode, ui.ErrMsg)
	}
	return tr.OpenID, ui.Nickname, ui.HeadImg, nil
}

// CookieExpiry 默认会话时长。
func CookieExpiry() time.Duration {
	return 30 * 24 * time.Hour
}
