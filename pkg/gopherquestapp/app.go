package gopherquestapp

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fhj/go-from-beginner-to-application/internal/auth"
	"github.com/fhj/go-from-beginner-to-application/internal/game"
	"github.com/fhj/go-from-beginner-to-application/internal/gopherquest"
	"github.com/fhj/go-from-beginner-to-application/internal/store"
)

// NewServerFromEnv 构建服务实例；本地和 Vercel 共用。
func NewServerFromEnv() (*gopherquest.Server, error) {
	dataPath := strings.TrimSpace(os.Getenv("DATA_FILE"))
	st, err := store.Open(dataPath)
	if err != nil {
		return nil, err
	}
	sec := strings.TrimSpace(os.Getenv("SESSION_SECRET"))
	if sec == "" {
		sec = "local-dev-only-set-SESSION_SECRET-for-production"
	}
	codec, err := auth.NewCodec(sec)
	if err != nil {
		return nil, err
	}
	hc := &http.Client{Timeout: 12 * time.Second}
	wxCfg := auth.LoadWeChatConfig()
	var jss *auth.JSSDKSigner
	if wxCfg.CanJSSDK() {
		jss = auth.NewJSSDKSigner(wxCfg, hc)
	}
	return &gopherquest.Server{
		Store:      st,
		Codec:      codec,
		WeChat:     wxCfg,
		JSSDK:      jss,
		Curriculum: game.DefaultCurriculum(),
		HTTP:       hc,
	}, nil
}
