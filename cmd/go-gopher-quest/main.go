// go-gopher-quest：移动端友好的 Go 入门小游戏（HTTP API + 静态前端）。
//
// 运行：go run ./cmd/go-gopher-quest
//
// 本地用微信试 JSSDK / 分享卡片：微信内打开的页面必须是公众号「JS 接口安全域名」里的 HTTPS 域名，
// 因此本机需用隧道（如 ngrok、cloudflared）暴露 HTTPS，并设置：
//
//	export PUBLIC_BASE_URL=https://你的隧道域名
//	export WECHAT_APP_ID=... WECHAT_APP_SECRET=...
//	export WECHAT_REDIRECT_URL=${PUBLIC_BASE_URL}/api/auth/wechat/callback   # 若用网页授权
//
// 在微信公众平台「设置 → 公众号设置 → 功能设置」里配置「JS 接口安全域名」为隧道域名（不带协议与路径）。
//
// 环境变量（可选）：
//
//	ADDR — 监听地址，默认 :8080
//	DATA_FILE — 用户与进度 JSON 路径，默认 data/gopher-quest.json
//	SESSION_SECRET — HMAC 会话密钥（生产务必设置）
//	PUBLIC_BASE_URL — 公网根 URL（https 开头时 Cookie 标记 Secure；微信 OAuth / JSSDK 校验用）
//	WECHAT_APP_ID / WECHAT_APP_SECRET / WECHAT_REDIRECT_URL — 配置后启用服务号网页授权登录
//	WECHAT_SHARE_TITLE / WECHAT_SHARE_DESC — 自定义发送给朋友/朋友圈卡片文案
//	WECHAT_JSSDK_DEBUG=1 — wx.config 打开 debug（仅排错）
package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/fhj/go-from-beginner-to-application/pkg/gopherquestapp"
)

func main() {
	addr := strings.TrimSpace(os.Getenv("ADDR"))
	if addr == "" {
		addr = ":8080"
	}
	srv, err := gopherquestapp.NewServerFromEnv()
	if err != nil {
		log.Fatalf("server: %v", err)
	}
	if strings.TrimSpace(os.Getenv("SESSION_SECRET")) == "" {
		log.Printf("warning: SESSION_SECRET not set; using insecure default for local dev")
	}
	openHost := addr
	if strings.HasPrefix(addr, ":") {
		openHost = "127.0.0.1" + addr
	}
	log.Printf("gopher-quest listening on http://%s", openHost)
	log.Printf("open http://%s/static/index.html", openHost)
	if srv.JSSDK != nil {
		log.Printf("WeChat JSSDK: signature API enabled at GET /api/wechat/jssdk-config")
	}
	if srv.WeChat.Enabled() {
		log.Printf("WeChat OAuth enabled, redirect URL must match: %s", srv.WeChat.RedirectURL)
	}
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		log.Fatal(err)
	}
}
