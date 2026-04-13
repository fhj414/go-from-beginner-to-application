package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// Progress 学习进度（会持久化到磁盘）。
type Progress struct {
	CurrentLesson int          `json:"current_lesson"` // 下一关要挑战的 id（从 1 开始）
	Completed     map[int]bool `json:"completed"`      // lesson id -> done
	XP            int          `json:"xp"`
	LastStage     string       `json:"last_stage"`
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

// Store JSON 文件持久化（适合本仓库教学演示；线上可替换为数据库）。
type Store struct {
	mu   sync.Mutex
	path string
	root persisted
}

type persisted struct {
	Users map[string]*User `json:"users"`
}

// Open 打开或创建数据文件。
func Open(path string) (*Store, error) {
	if path == "" {
		path = filepath.Join("data", "gopher-quest.json")
	}
	s := &Store{path: path, root: persisted{Users: map[string]*User{}}}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, s.saveLocked()
	}
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return s, s.saveLocked()
	}
	if err := json.Unmarshal(b, &s.root); err != nil {
		return nil, err
	}
	if s.root.Users == nil {
		s.root.Users = map[string]*User{}
	}
	return s, nil
}

func (s *Store) saveLocked() error {
	tmp := s.path + ".tmp"
	b, err := json.MarshalIndent(s.root, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// UpsertUser 写入用户（内存 + 磁盘）。
func (s *Store) UpsertUser(u *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.root.Users[u.ID] = u
	return s.saveLocked()
}

// GetUser 读取用户。
func (s *Store) GetUser(id string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.root.Users[id]
	if !ok {
		return nil, false
	}
	cp := *u
	return &cp, true
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

// TopByStudySeconds 按累计学习时长排序（同分按 XP）。
func (s *Store) TopByStudySeconds(limit int) []LeaderboardEntry {
	if limit <= 0 {
		limit = 20
	}
	s.mu.Lock()
	list := make([]*User, 0, len(s.root.Users))
	for _, u := range s.root.Users {
		if u == nil {
			continue
		}
		uu := *u
		list = append(list, &uu)
	}
	s.mu.Unlock()

	sort.Slice(list, func(i, j int) bool {
		if list[i].TotalStudySecs == list[j].TotalStudySecs {
			return list[i].Progress.XP > list[j].Progress.XP
		}
		return list[i].TotalStudySecs > list[j].TotalStudySecs
	})
	out := make([]LeaderboardEntry, 0, limit)
	for i, u := range list {
		if i >= limit {
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
