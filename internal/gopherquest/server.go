package gopherquest

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/fhj/go-from-beginner-to-application/internal/auth"
	"github.com/fhj/go-from-beginner-to-application/internal/game"
	"github.com/fhj/go-from-beginner-to-application/internal/store"
)

const (
	cookieSession = "gq_session"
	cookieWXState = "gq_wx_state"
)

// Server HTTP 与课程 API。
type Server struct {
	Store      *store.Store
	Codec      *auth.Codec
	WeChat     auth.WeChatConfig
	JSSDK      *auth.JSSDKSigner // 可选；配置 AppID+Secret 后用于微信 JSSDK 分享签名
	Curriculum game.Curriculum
	HTTP       *http.Client
}

func (s *Server) curriculumTotal() int {
	return len(s.Curriculum.Lessons)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	b, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(b) == 0 {
		return errors.New("empty body")
	}
	return json.Unmarshal(b, dst)
}

func (s *Server) userFromRequest(r *http.Request) (*store.User, error) {
	c, err := r.Cookie(cookieSession)
	if err != nil || c.Value == "" {
		return nil, errors.New("no session")
	}
	claims, err := s.Codec.Verify(c.Value)
	if err != nil {
		return nil, err
	}
	u, ok := s.Store.GetUser(claims.UserID)
	if !ok {
		return nil, errors.New("unknown user")
	}
	return u, nil
}

func (s *Server) setSession(w http.ResponseWriter, userID string) error {
	now := time.Now().Unix()
	claims := auth.SessionClaims{
		UserID:    userID,
		IssuedAt:  now,
		ExpiresAt: now + int64(auth.CookieExpiry().Seconds()),
	}
	tok, err := s.Codec.Sign(claims)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieSession,
		Value:    tok,
		Path:     "/",
		MaxAge:   int(auth.CookieExpiry().Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(os.Getenv("PUBLIC_BASE_URL"), "https://"),
	})
	return nil
}

func (s *Server) clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieSession,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// Handler 注册路由。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/health", allowMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
	}))

	mux.HandleFunc("/api/curriculum", allowMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, s.Curriculum)
	}))

	mux.HandleFunc("/api/wechat/enabled", allowMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"enabled": s.WeChat.Enabled(),
			"jssdk":   s.JSSDK != nil,
		})
	}))
	mux.HandleFunc("/api/wechat/jssdk-config", allowMethod(http.MethodGet, s.handleWeChatJSSDKConfig))

	mux.HandleFunc("/api/auth/demo", allowMethod(http.MethodPost, s.handleDemoAuth))
	mux.HandleFunc("/api/auth/wechat/start", allowMethod(http.MethodGet, s.handleWeChatStart))
	mux.HandleFunc("/api/auth/wechat/callback", allowMethod(http.MethodGet, s.handleWeChatCallback))
	mux.HandleFunc("/api/auth/logout", allowMethod(http.MethodPost, s.handleLogout))

	mux.HandleFunc("/api/me", allowMethod(http.MethodGet, s.handleMe))
	mux.HandleFunc("/api/progress", allowMethod(http.MethodPut, s.handlePutProgress))
	mux.HandleFunc("/api/study/tick", allowMethod(http.MethodPost, s.handleStudyTick))
	mux.HandleFunc("/api/leaderboard", allowMethod(http.MethodGet, s.handleLeaderboard))

	// 分享落地页（微信抓取 og 标签用简单 HTML）
	mux.HandleFunc("/share", allowMethod(http.MethodGet, s.handleSharePage))

	fs := http.FileServer(http.FS(StaticRoot))
	mux.HandleFunc("/static", allowMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/static/", http.StatusFound)
	}))
	mux.Handle("/static/", allowMethodH(http.MethodGet, http.StripPrefix("/static/", fs)))
	mux.HandleFunc("/", allowMethod(http.MethodGet, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, "/static/index.html", http.StatusFound)
	}))

	return withCORS(withLogging(mux))
}

func (s *Server) handleDemoAuth(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Nickname string `json:"nickname"`
		ResumeID string `json:"resume_id"`
	}
	var b body
	if err := readJSON(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	name := strings.TrimSpace(b.Nickname)
	if name == "" {
		name = "探险小地鼠"
	}
	if len([]rune(name)) > 16 {
		name = string([]rune(name)[:16])
	}
	now := time.Now()
	resumeID := strings.TrimSpace(b.ResumeID)
	var u *store.User
	if strings.HasPrefix(resumeID, "demo:") {
		if existing, ok := s.Store.GetUser(resumeID); ok && existing.Source == "demo" {
			u = existing
			u.Nickname = name
			u.LastActiveAt = now
		}
	}
	if u == nil {
		id := "demo:" + randomID()
		if strings.HasPrefix(resumeID, "demo:") && len(resumeID) <= 40 {
			id = resumeID
		}
		u = &store.User{
			ID:             id,
			Nickname:       name,
			Source:         "demo",
			TotalStudySecs: 0,
			LastActiveAt:   now,
			Progress: store.Progress{
				CurrentLesson: 1,
				Completed:     map[int]bool{},
				Stars:         map[int]int{},
				XP:            0,
				LastStage:     "seed",
				StreakDays:    0,
				UpdatedAt:     now,
				ReminderNote:  "从第 1 关开始冒险吧～",
			},
			CreatedAt: now,
		}
	}
	ensureDailyReward(u)
	if err := s.Store.UpsertUser(u); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}
	if err := s.setSession(w, u.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "session failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(u, s.Curriculum.Lessons)})
}

func (s *Server) handleWeChatStart(w http.ResponseWriter, r *http.Request) {
	if !s.WeChat.Enabled() {
		http.Error(w, "WeChat OAuth is not configured", http.StatusBadRequest)
		return
	}
	st, err := auth.RandomState()
	if err != nil {
		http.Error(w, "state", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     cookieWXState,
		Value:    st,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(os.Getenv("PUBLIC_BASE_URL"), "https://"),
	})
	http.Redirect(w, r, s.WeChat.AuthorizeURL(st), http.StatusFound)
}

func (s *Server) handleWeChatCallback(w http.ResponseWriter, r *http.Request) {
	if !s.WeChat.Enabled() {
		http.Error(w, "not configured", http.StatusBadRequest)
		return
	}
	q := r.URL.Query()
	code := q.Get("code")
	state := q.Get("state")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}
	c, err := r.Cookie(cookieWXState)
	if err != nil || c.Value == "" || c.Value != state {
		http.Error(w, "bad state", http.StatusBadRequest)
		return
	}
	openid, nick, avatar, err := auth.ExchangeWeChatCode(s.WeChat, code, s.HTTP)
	if err != nil {
		log.Printf("wechat exchange: %v", err)
		http.Error(w, "wechat failed", http.StatusBadGateway)
		return
	}
	http.SetCookie(w, &http.Cookie{Name: cookieWXState, Value: "", Path: "/", MaxAge: -1})

	id := "wx:" + openid
	u, ok := s.Store.GetUser(id)
	if !ok {
		u = &store.User{
			ID:             id,
			Nickname:       nick,
			AvatarURL:      avatar,
			Source:         "wechat",
			TotalStudySecs: 0,
			LastActiveAt:   time.Now(),
			Progress: store.Progress{
				CurrentLesson: 1,
				Completed:     map[int]bool{},
				Stars:         map[int]int{},
				XP:            0,
				LastStage:     "seed",
				StreakDays:    0,
				UpdatedAt:     time.Now(),
				ReminderNote:  "从第 1 关开始冒险吧～",
			},
			CreatedAt: time.Now(),
		}
	} else {
		u.Nickname = nick
		u.AvatarURL = avatar
		u.LastActiveAt = time.Now()
	}
	if u.Nickname == "" {
		u.Nickname = "微信旅人"
	}
	ensureDailyReward(u)
	if err := s.Store.UpsertUser(u); err != nil {
		http.Error(w, "save failed", http.StatusInternalServerError)
		return
	}
	if err := s.setSession(w, u.ID); err != nil {
		http.Error(w, "session failed", http.StatusInternalServerError)
		return
	}
	base := strings.TrimSuffix(os.Getenv("PUBLIC_BASE_URL"), "/")
	redir := "/static/index.html?welcome=wechat"
	if base != "" {
		http.Redirect(w, r, base+redir, http.StatusFound)
		return
	}
	http.Redirect(w, r, redir, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.clearSession(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	u, err := s.userFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	ensureDailyReward(u)
	if err := s.Store.UpsertUser(u); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(u, s.Curriculum.Lessons)})
}

func (s *Server) handlePutProgress(w http.ResponseWriter, r *http.Request) {
	u, err := s.userFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	type body struct {
		CurrentLesson int          `json:"current_lesson"`
		Completed     map[int]bool `json:"completed"`
		Stars         map[int]int  `json:"stars"`
		XP            int          `json:"xp"`
	}
	var b body
	if err := readJSON(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	total := s.curriculumTotal()
	if b.CurrentLesson < 1 || b.CurrentLesson > total+1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad lesson"})
		return
	}
	if b.Completed == nil {
		b.Completed = map[int]bool{}
	}
	if b.Stars == nil {
		b.Stars = map[int]int{}
	}
	merged := map[int]bool{}
	mergedStars := map[int]int{}
	for id, ok := range u.Progress.Completed {
		if ok && id >= 1 && id <= total {
			merged[id] = true
		}
	}
	for id, stars := range u.Progress.Stars {
		if id >= 1 && id <= total && stars >= 1 {
			if stars > 3 {
				stars = 3
			}
			mergedStars[id] = stars
		}
	}
	for id, ok := range b.Completed {
		if ok && id >= 1 && id <= total {
			merged[id] = true
		}
	}
	for id, stars := range b.Stars {
		if id < 1 || id > total || stars < 1 {
			continue
		}
		if stars > 3 {
			stars = 3
		}
		if stars > mergedStars[id] {
			mergedStars[id] = stars
		}
	}
	done := len(merged)
	stage := game.StageForProgress(done, total)
	note := reminderNote(b.CurrentLesson, total, s.Curriculum.Lessons, merged)

	u.Progress = store.Progress{
		CurrentLesson: b.CurrentLesson,
		Completed:     merged,
		Stars:         mergedStars,
		XP:            b.XP,
		LastStage:     stage,
		StreakDays:    u.Progress.StreakDays,
		LastCheckIn:   u.Progress.LastCheckIn,
		UpdatedAt:     time.Now(),
		ReminderNote:  note,
	}
	u.LastActiveAt = time.Now()
	if err := s.Store.UpsertUser(u); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": publicUser(u, s.Curriculum.Lessons)})
}

func (s *Server) handleStudyTick(w http.ResponseWriter, r *http.Request) {
	u, err := s.userFromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	type body struct {
		Seconds int64 `json:"seconds"`
	}
	var b body
	if err := readJSON(r, &b); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
		return
	}
	if b.Seconds < 1 || b.Seconds > 120 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "bad seconds"})
		return
	}
	u.TotalStudySecs += b.Seconds
	u.LastActiveAt = time.Now()
	if err := s.Store.UpsertUser(u); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "save failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"total_study_secs": u.TotalStudySecs})
}

func (s *Server) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	n := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		if x, err := strconv.Atoi(v); err == nil && x > 0 && x <= 100 {
			n = x
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Store.TopByStudySeconds(n)})
}

func (s *Server) envPublicBaseURL() string {
	return strings.TrimSpace(strings.TrimSuffix(os.Getenv("PUBLIC_BASE_URL"), "/"))
}

func (s *Server) allowJSSDKPageURL(page string) bool {
	u, err := url.Parse(page)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	base := s.envPublicBaseURL()
	if base == "" {
		h := strings.ToLower(u.Hostname())
		return h == "localhost" || h == "127.0.0.1"
	}
	bu, err := url.Parse(base)
	if err != nil || bu.Scheme == "" || bu.Host == "" {
		return false
	}
	return strings.EqualFold(u.Scheme, bu.Scheme) && strings.EqualFold(u.Hostname(), bu.Hostname())
}

func (s *Server) handleWeChatJSSDKConfig(w http.ResponseWriter, r *http.Request) {
	if s.JSSDK == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{
			"error": "jssdk not configured (set WECHAT_APP_ID and WECHAT_APP_SECRET)",
		})
		return
	}
	page := strings.TrimSpace(r.URL.Query().Get("url"))
	if page == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing url query"})
		return
	}
	if !s.allowJSSDKPageURL(page) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": "url host must match PUBLIC_BASE_URL (use HTTPS tunnel + set PUBLIC_BASE_URL for WeChat)",
		})
		return
	}
	res, err := s.JSSDK.Sign(page)
	if err != nil {
		log.Printf("jssdk sign: %v", err)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "weixin api or sign failed"})
		return
	}
	title := strings.TrimSpace(os.Getenv("WECHAT_SHARE_TITLE"))
	if title == "" {
		title = "小地鼠闯 Go 星球"
	}
	desc := strings.TrimSpace(os.Getenv("WECHAT_SHARE_DESC"))
	if desc == "" {
		desc = "点点玩玩，把 Go 的最小常识装进脑袋～"
	}
	res.ShareTitle = title
	res.ShareDesc = desc
	res.Debug = strings.TrimSpace(os.Getenv("WECHAT_JSSDK_DEBUG")) == "1"
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleSharePage(w http.ResponseWriter, r *http.Request) {
	title := "小地鼠闯 Go 星球"
	desc := "边玩边学 Go 的超轻量小游戏，一起来收集小地鼠吧～"
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	base := strings.TrimSpace(os.Getenv("PUBLIC_BASE_URL"))
	ogURL := "/static/index.html"
	if base != "" {
		ogURL = strings.TrimSuffix(base, "/") + "/static/index.html"
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="zh-CN"><head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>` + escHTML(title) + `</title>
<meta property="og:title" content="` + escHTML(title) + `">
<meta property="og:description" content="` + escHTML(desc) + `">
<meta property="og:type" content="website">
<meta property="og:url" content="` + escHTML(ogURL) + `">
<meta name="twitter:card" content="summary_large_image">
</head><body style="font-family:system-ui;margin:2rem;">
<p>` + escHTML(desc) + `</p>
<p><a href="/static/index.html">打开游戏</a></p>
</body></html>`))
}

func escHTML(s string) string {
	s = strings.ReplaceAll(s, `&`, `&amp;`)
	s = strings.ReplaceAll(s, `<`, `&lt;`)
	s = strings.ReplaceAll(s, `>`, `&gt;`)
	s = strings.ReplaceAll(s, `"`, `&quot;`)
	return s
}

func publicUser(u *store.User, lessons []game.Lesson) map[string]any {
	totalLessons := len(lessons)
	done := 0
	totalStars := 0
	for id, ok := range u.Progress.Completed {
		if ok && id >= 1 && id <= totalLessons {
			done++
		}
	}
	if u.Progress.Stars == nil {
		u.Progress.Stars = map[int]int{}
	}
	for id, stars := range u.Progress.Stars {
		if id >= 1 && id <= totalLessons && stars > 0 {
			totalStars += stars
		}
	}
	if u.Progress.CurrentLesson < 1 {
		u.Progress.CurrentLesson = 1
	}
	currentTitle := "继续冒险"
	for _, lesson := range lessons {
		if lesson.ID == u.Progress.CurrentLesson {
			currentTitle = lesson.Title
			break
		}
	}
	if u.Progress.CurrentLesson > totalLessons {
		currentTitle = "主线通关啦"
	}
	progressPercent := 0
	if totalLessons > 0 {
		progressPercent = done * 100 / totalLessons
	}
	return map[string]any{
		"id":               u.ID,
		"nickname":         u.Nickname,
		"avatar_url":       u.AvatarURL,
		"source":           u.Source,
		"total_study_secs": u.TotalStudySecs,
		"progress_percent": progressPercent,
		"total_stars":      totalStars,
		"progress": map[string]any{
			"current_lesson":  u.Progress.CurrentLesson,
			"completed":       u.Progress.Completed,
			"stars":           u.Progress.Stars,
			"xp":              u.Progress.XP,
			"last_stage":      u.Progress.LastStage,
			"streak_days":     u.Progress.StreakDays,
			"last_check_in":   u.Progress.LastCheckIn,
			"updated_at":      u.Progress.UpdatedAt,
			"reminder_note":   u.Progress.ReminderNote,
			"completed_count": done,
			"total_lessons":   totalLessons,
			"resume_title":    currentTitle,
		},
	}
}

func reminderNote(current int, total int, lessons []game.Lesson, completed map[int]bool) string {
	if current > total {
		return "太棒啦，主线关卡都通关了！可以邀请朋友来一局～"
	}
	var title string
	for _, L := range lessons {
		if L.ID == current {
			title = L.Title
			break
		}
	}
	if title == "" {
		title = "下一关"
	}
	done := 0
	for id, ok := range completed {
		if ok && id >= 1 && id <= total {
			done++
		}
	}
	return "上次进度：第 " + strconv.Itoa(current) + " 关「" + title + "」 · 已完成 " + strconv.Itoa(done) + "/" + strconv.Itoa(total) + " 关"
}

func ensureDailyReward(u *store.User) {
	if u == nil {
		return
	}
	if u.Progress.Stars == nil {
		u.Progress.Stars = map[int]int{}
	}
	today := time.Now().Format("2006-01-02")
	if u.Progress.LastCheckIn == today {
		return
	}
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	if u.Progress.LastCheckIn == yesterday {
		u.Progress.StreakDays++
	} else {
		u.Progress.StreakDays = 1
	}
	u.Progress.LastCheckIn = today
	u.Progress.XP += 6
	u.Progress.UpdatedAt = time.Now()
	if strings.TrimSpace(u.Progress.ReminderNote) == "" {
		u.Progress.ReminderNote = "签到成功，今天也来和小地鼠学一点 Go 吧～"
	}
}

func randomID() string {
	var b [10]byte
	if _, err := rand.Read(b[:]); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 36)
	}
	return hex.EncodeToString(b[:])
}
