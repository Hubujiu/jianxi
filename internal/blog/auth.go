package blog

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type authConfig struct {
	Digest     string `json:"digest"`
	Generation string `json:"generation"`
}
type Session struct {
	ID         string
	CSRF       string
	Generation string
	Expires    time.Time
}
type Auth struct {
	mu        sync.Mutex
	path      string
	adminBase string
	sessions  map[string]Session
}

func OpenAuth(root string) (*Auth, error) {
	a := &Auth{path: filepath.Join(root, "auth.json"), sessions: map[string]Session{}}
	if _, err := os.Stat(a.path); os.IsNotExist(err) {
		token := os.Getenv("JIANXI_TOKEN")
		generated := token == ""
		if generated {
			token = uid() + uid()
		}
		if len(token) < 32 {
			return nil, fail("WEAK_TOKEN", "初始化管理密钥至少 32 字符")
		}
		if err = a.write(token); err != nil {
			return nil, err
		}
		if generated {
			if err = os.WriteFile(filepath.Join(root, "initial-token.txt"), []byte(token+"\n"), 0600); err != nil {
				return nil, err
			}
		}
	} else if err != nil {
		return nil, err
	}
	routeFile := filepath.Join(root, "admin-route.txt")
	b, err := os.ReadFile(routeFile)
	if os.IsNotExist(err) {
		b = []byte("/admin/" + uid() + "/")
		err = os.WriteFile(routeFile, append(b, '\n'), 0600)
	}
	if err != nil {
		return nil, err
	}
	route := strings.TrimSpace(string(b))
	code := strings.TrimSuffix(strings.TrimPrefix(route, "/admin/"), "/")
	if route != "/admin/"+code+"/" || len(code) != 32 || !hexID(code) {
		return nil, fail("INVALID_ADMIN_ROUTE", "管理入口安全码文件无效")
	}
	a.adminBase = route
	return a, nil
}
func (a *Auth) config() (authConfig, error) {
	var c authConfig
	b, err := os.ReadFile(a.path)
	if err == nil {
		err = json.Unmarshal(b, &c)
	}
	return c, err
}
func (a *Auth) write(token string) error {
	p := a.path + "." + uid() + ".tmp"
	if err := writeJSON(p, authConfig{Digest: hashBytes([]byte(token)), Generation: uid()}); err != nil {
		return err
	}
	return os.Rename(p, a.path)
}
func (a *Auth) Verify(token string) bool {
	if token == "" {
		return false
	}
	c, err := a.config()
	return err == nil && subtle.ConstantTimeCompare([]byte(c.Digest), []byte(hashBytes([]byte(token)))) == 1
}
func (a *Auth) Login(token string) (Session, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.Verify(token) {
		return Session{}, false
	}
	c, err := a.config()
	if err != nil {
		return Session{}, false
	}
	s := Session{ID: uid() + uid(), CSRF: uid(), Generation: c.Generation, Expires: time.Now().Add(12 * time.Hour)}
	for id, old := range a.sessions {
		if time.Now().After(old.Expires) {
			delete(a.sessions, id)
		}
	}
	a.sessions[s.ID] = s
	return s, true
}
func (a *Auth) Session(r *http.Request) (Session, bool) {
	cookie, err := r.Cookie("jianxi_session")
	if err != nil {
		return Session{}, false
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	s, ok := a.sessions[cookie.Value]
	if !ok || time.Now().After(s.Expires) {
		return Session{}, false
	}
	c, err := a.config()
	return s, err == nil && s.Generation == c.Generation
}
func (a *Auth) Logout(id string) { a.mu.Lock(); defer a.mu.Unlock(); delete(a.sessions, id) }
func (a *Auth) Invalidate()      { a.mu.Lock(); defer a.mu.Unlock(); a.sessions = map[string]Session{} }
func (a *Auth) Rotate() (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	token := uid() + uid()
	if err := a.write(token); err != nil {
		return "", err
	}
	a.sessions = map[string]Session{}
	return token, nil
}
func RotateToken(root string) (string, error) {
	a, err := OpenAuth(root)
	if err != nil {
		return "", err
	}
	return a.Rotate()
}
