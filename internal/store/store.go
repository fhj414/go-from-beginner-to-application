package store

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Progress 学习进度（会持久化到磁盘或外部存储）。
type Progress struct {
	CurrentLesson int          `json:"current_lesson"` // 下一关要挑战的 id（从 1 开始）
	Completed     map[int]bool `json:"completed"`      // lesson id -> done
	Stars         map[int]int  `json:"stars,omitempty"`
	XP            int          `json:"xp"`
	LastStage     string       `json:"last_stage"`
	StreakDays    int          `json:"streak_days"`
	LastCheckIn   string       `json:"last_check_in,omitempty"`
	UpdatedAt     time.Time    `json:"updated_at"`
	ReminderNote  string       `json:"reminder_note,omitempty"`
}

// User 排行榜与存档用。
type User struct {
	ID             string    `json:"id"`
	Nickname       string    `json:"nickname"`
	AvatarURL      string    `json:"avatar_url,omitempty"`
	Source         string    `json:"source"` // wechat | demo
	TotalStudySecs int64     `json:"total_study_secs"`
	LastActiveAt   time.Time `json:"last_active_at"`
	Progress       Progress  `json:"progress"`
	CreatedAt      time.Time `json:"created_at"`
}

// LeaderboardEntry 排行榜一行。
type LeaderboardEntry struct {
	ID             string `json:"id,omitempty"`
	Rank           int    `json:"rank"`
	Nickname       string `json:"nickname"`
	AvatarURL      string `json:"avatar_url,omitempty"`
	TotalStudySecs int64  `json:"total_study_secs"`
	XP             int    `json:"xp"`
	Stage          string `json:"stage"`
}

type backend interface {
	upsertUser(context.Context, *User) error
	getUser(context.Context, string) (*User, bool, error)
	allUsers(context.Context) ([]*User, error)
}

// Store 隐藏具体存储实现；本地默认 JSON 文件，Vercel 上优先使用 Upstash Redis REST。
type Store struct {
	backend backend
}

// Open 自动选择后端：
// 1. 有 Upstash REST 环境变量时使用无状态外部存储（适合 Vercel）
// 2. Vercel 但没配置外部存储时使用内存后端（可跑，但不保证跨实例持久化）
// 3. 其他环境使用本地 JSON 文件
func Open(path string) (*Store, error) {
	if b, ok, err := openRedisBackendFromEnv(); err != nil {
		return nil, err
	} else if ok {
		return &Store{backend: b}, nil
	}
	if isVercelEnv() {
		return &Store{backend: newMemoryBackend()}, nil
	}
	b, err := openFileBackend(path)
	if err != nil {
		return nil, err
	}
	return &Store{backend: b}, nil
}

func isVercelEnv() bool {
	return strings.TrimSpace(os.Getenv("VERCEL")) == "1" || strings.TrimSpace(os.Getenv("VERCEL_ENV")) != ""
}

// UpsertUser 写入用户（内存 + 磁盘 / 外部存储）。
func (s *Store) UpsertUser(u *User) error {
	if s == nil || s.backend == nil {
		return errors.New("store not initialized")
	}
	return s.backend.upsertUser(context.Background(), cloneUser(u))
}

// GetUser 读取用户。
func (s *Store) GetUser(id string) (*User, bool) {
	if s == nil || s.backend == nil {
		return nil, false
	}
	u, ok, err := s.backend.getUser(context.Background(), id)
	if err != nil || !ok || u == nil {
		return nil, false
	}
	return cloneUser(u), true
}

// TopByStudySeconds 按累计学习时长排序（同分按 XP）。
func (s *Store) TopByStudySeconds(limit int) []LeaderboardEntry {
	if limit <= 0 {
		limit = 20
	}
	if s == nil || s.backend == nil {
		return nil
	}
	list, err := s.backend.allUsers(context.Background())
	if err != nil {
		return nil
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].TotalStudySecs == list[j].TotalStudySecs {
			return list[i].Progress.XP > list[j].Progress.XP
		}
		return list[i].TotalStudySecs > list[j].TotalStudySecs
	})
	out := make([]LeaderboardEntry, 0, limit)
	for i, u := range list {
		if i >= limit || u == nil {
			break
		}
		out = append(out, LeaderboardEntry{
			ID:             u.ID,
			Rank:           i + 1,
			Nickname:       u.Nickname,
			AvatarURL:      u.AvatarURL,
			TotalStudySecs: u.TotalStudySecs,
			XP:             u.Progress.XP,
			Stage:          u.Progress.LastStage,
		})
	}
	return out
}

type persisted struct {
	Users map[string]*User `json:"users"`
}

type fileBackend struct {
	mu   sync.Mutex
	path string
	root persisted
}

func openFileBackend(path string) (*fileBackend, error) {
	if path == "" {
		path = filepath.Join("data", "gopher-quest.json")
	}
	b := &fileBackend{path: path, root: persisted{Users: map[string]*User{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return b, b.saveLocked()
	}
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return b, b.saveLocked()
	}
	if err := json.Unmarshal(raw, &b.root); err != nil {
		return nil, err
	}
	if b.root.Users == nil {
		b.root.Users = map[string]*User{}
	}
	return b, nil
}

func (b *fileBackend) saveLocked() error {
	tmp := b.path + ".tmp"
	raw, err := json.MarshalIndent(b.root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, b.path)
}

func (b *fileBackend) upsertUser(_ context.Context, u *User) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.root.Users[u.ID] = cloneUser(u)
	return b.saveLocked()
}

func (b *fileBackend) getUser(_ context.Context, id string) (*User, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	u, ok := b.root.Users[id]
	if !ok || u == nil {
		return nil, false, nil
	}
	return cloneUser(u), true, nil
}

func (b *fileBackend) allUsers(_ context.Context) ([]*User, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*User, 0, len(b.root.Users))
	for _, u := range b.root.Users {
		if u != nil {
			out = append(out, cloneUser(u))
		}
	}
	return out, nil
}

type memoryBackend struct {
	mu    sync.Mutex
	users map[string]*User
}

func newMemoryBackend() *memoryBackend {
	return &memoryBackend{users: map[string]*User{}}
}

func (b *memoryBackend) upsertUser(_ context.Context, u *User) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.users[u.ID] = cloneUser(u)
	return nil
}

func (b *memoryBackend) getUser(_ context.Context, id string) (*User, bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	u, ok := b.users[id]
	if !ok || u == nil {
		return nil, false, nil
	}
	return cloneUser(u), true, nil
}

func (b *memoryBackend) allUsers(_ context.Context) ([]*User, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	out := make([]*User, 0, len(b.users))
	for _, u := range b.users {
		if u != nil {
			out = append(out, cloneUser(u))
		}
	}
	return out, nil
}

type redisBackend struct {
	baseURL string
	token   string
	prefix  string
	http    *http.Client
}

func openRedisBackendFromEnv() (*redisBackend, bool, error) {
	baseURL := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_URL"))
	token := strings.TrimSpace(os.Getenv("UPSTASH_REDIS_REST_TOKEN"))
	if baseURL == "" || token == "" {
		return nil, false, nil
	}
	return &redisBackend{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		prefix:  envOr("STORE_PREFIX", "gq"),
		http:    &http.Client{Timeout: 8 * time.Second},
	}, true, nil
}

func envOr(key, fallback string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	return v
}

func cloneUser(u *User) *User {
	if u == nil {
		return nil
	}
	raw, _ := json.Marshal(u)
	var out User
	_ = json.Unmarshal(raw, &out)
	if out.Progress.Completed == nil {
		out.Progress.Completed = map[int]bool{}
	}
	if out.Progress.Stars == nil {
		out.Progress.Stars = map[int]int{}
	}
	return &out
}
