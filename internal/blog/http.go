package blog

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"math/rand/v2"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Only content-addressed WenKai assets are safe to cache across deployments.
var immutableFontPath = regexp.MustCompile(`^/static/fonts/wenkai/[a-f0-9]{16}\.(woff2|css)$`)

func (a *App) Handler() http.Handler {
	mux := http.NewServeMux()
	assets, _ := fs.Sub(webFS, "web/static")
	staticFiles := http.StripPrefix("/static/", http.FileServer(http.FS(assets)))
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache")
		if immutableFontPath.MatchString(r.URL.Path) {
			if info, err := fs.Stat(assets, strings.TrimPrefix(r.URL.Path, "/static/")); err != nil || info.IsDir() {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			if strings.HasSuffix(r.URL.Path, ".woff2") {
				w.Header().Set("Content-Type", "font/woff2")
			}
		}
		staticFiles.ServeHTTP(w, r)
	}))
	mux.HandleFunc("/_control/", a.control)
	mux.HandleFunc(a.Auth.adminBase, a.admin)
	mux.HandleFunc("/admin/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		http.NotFound(w, r)
	})
	mux.HandleFunc("GET /api/search", a.searchAPI)
	mux.HandleFunc("GET /search/", a.searchPage)
	mux.HandleFunc("GET /random/", func(w http.ResponseWriter, r *http.Request) {
		posts := sortedPosts(*a.Store.active.Load())
		if len(posts) == 0 {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Redirect(w, r, contentPath(posts[rand.IntN(len(posts))].Content), http.StatusSeeOther)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			http.Error(w, "Method not allowed", 405)
			return
		}
		a.public(w, r)
	})
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' https://hm.baidu.com; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' https: data:; connect-src 'self' https://hm.baidu.com; frame-src https://player.bilibili.com; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'")
		mux.ServeHTTP(w, r)
	})
}
func (a *App) public(w http.ResponseWriter, r *http.Request) {
	a.files.RLock()
	defer a.files.RUnlock()
	snap := a.Store.active.Load()
	w.Header().Set("X-Jianxi-Release", snap.ID)
	if strings.HasPrefix(r.URL.Path, "/media/") {
		id := strings.TrimPrefix(r.URL.Path, "/media/")
		v, ok := snap.Assets[id]
		if !ok {
			a.notFound(w, r, *snap)
			return
		}
		w.Header().Set("Content-Type", v.MIME)
		w.Header().Set("Cache-Control", "public,max-age=3600")
		http.ServeFile(w, r, filepath.Join(a.Store.root, "assets", v.ID))
		return
	}
	if dest, ok := snap.Redirects[r.URL.Path]; ok {
		http.Redirect(w, r, dest, http.StatusMovedPermanently)
		return
	}
	a.serveArtifact(w, r, *snap, filepath.Join(a.Store.root, "releases", snap.ID), r.URL.Path, false)
}
func (a *App) serveArtifact(w http.ResponseWriter, r *http.Request, snap Snapshot, dir, path string, private bool) {
	if strings.Contains(path, "..") || strings.Contains(path, "\\") || strings.ContainsRune(path, 0) || strings.HasSuffix(path, "snapshot.json") || (!strings.HasSuffix(path, "/") && !strings.HasPrefix(path, "/diagrams/") && path != "/sitemap.xml" && path != "/robots.txt") {
		a.notFound(w, r, snap)
		return
	}
	name := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(path, "/")))
	if strings.HasSuffix(path, "/") {
		name = filepath.Join(name, "index.html")
	}
	info, err := os.Stat(name)
	if err != nil || info.IsDir() {
		a.notFound(w, r, snap)
		return
	}
	if private {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	} else {
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
		w.Header().Set("ETag", `"`+snap.ID+`"`)
	}
	http.ServeFile(w, r, name)
}
func (a *App) notFound(w http.ResponseWriter, r *http.Request, s Snapshot) {
	b, err := a.renderPage(Page{Snap: s, Type: "404", Title: "这一页，暂时留白", Path: r.URL.Path, NoIndex: true})
	if err != nil {
		http.Error(w, "页面不存在", 404)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(404)
	w.Write(b)
}
func (a *App) searchAPI(w http.ResponseWriter, r *http.Request) {
	snap := *a.Store.active.Load()
	v := r.URL.Query()
	results, total, err := a.Search(r.Context(), snap, v.Get("q"), v.Get("category"), v.Get("tag"), 1)
	if err != nil {
		apiError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	jsonResponse(w, map[string]any{"results": results, "total": total, "release": snap.ID})
}
func (a *App) searchPage(w http.ResponseWriter, r *http.Request) {
	snap := *a.Store.active.Load()
	v := r.URL.Query()
	n, _ := strconv.Atoi(v.Get("page"))
	if n == 0 {
		n = 1
	}
	results, total, err := a.Search(r.Context(), snap, v.Get("q"), v.Get("category"), v.Get("tag"), n)
	p := Page{Snap: snap, Type: "search", Title: "搜索文章", Description: "在文字之间，找到你正在寻找的。", Path: "/search/", Query: v.Get("q"), Category: v.Get("category"), Tag: v.Get("tag"), NoIndex: true, PageNumber: n, Pages: max(1, (total+11)/12)}
	if err != nil {
		p.Error = err.Error()
	}
	for _, res := range results {
		rev := res.revision
		rev.Content.Summary = res.Snippet
		p.Posts = append(p.Posts, rev)
	}
	if n > 1 {
		v.Set("page", strconv.Itoa(n-1))
		p.Previous = "/search/?" + v.Encode()
	}
	if n < p.Pages {
		v.Set("page", strconv.Itoa(n+1))
		p.Next = "/search/?" + v.Encode()
	}
	b, e := a.renderPage(p)
	if e != nil {
		http.Error(w, e.Error(), 500)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex")
	w.Write(b)
}
func decode(r *http.Request, out any) error {
	return decodeLimit(r, out, 16<<20)
}
func decodeLimit(r *http.Request, out any, limit int64) error {
	r.Body = http.MaxBytesReader(nil, r.Body, limit)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return fail("INVALID_INPUT", err.Error())
	}
	if d.Decode(new(any)) != io.EOF {
		return fail("INVALID_INPUT", "只能提交一个 JSON 对象")
	}
	return nil
}
func jsonResponse(w http.ResponseWriter, out any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(out)
}
func apiError(w http.ResponseWriter, err error) {
	status := 400
	if strings.Contains(err.Error(), "CONFLICT") || strings.Contains(err.Error(), "STALE") {
		status = 409
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
func (a *App) control(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || !a.Auth.Verify(token) {
		http.Error(w, "管理密钥无效", 401)
		return
	}
	if r.Method != "POST" {
		http.Error(w, "仅支持 POST", 405)
		return
	}
	op := strings.TrimPrefix(r.URL.Path, "/_control/")
	result, err := a.dispatch(r, op)
	if err != nil {
		apiError(w, err)
		return
	}
	jsonResponse(w, result)
}
func sameOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return r.Header.Get("Sec-Fetch-Site") != "cross-site"
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host && (u.Scheme == "https" || u.Scheme == "http")
}
func csrfOK(r *http.Request, s Session) bool {
	if !sameOrigin(r) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(s.CSRF)) == 1
}

var attempts = struct {
	sync.Mutex
	entries map[string][]time.Time
}{entries: map[string][]time.Time{}}

func allowLogin(r *http.Request) bool {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	attempts.Lock()
	defer attempts.Unlock()
	limit := time.Now().Add(-time.Minute)
	list := []time.Time{}
	for _, t := range attempts.entries[ip] {
		if t.After(limit) {
			list = append(list, t)
		}
	}
	if len(list) >= 8 {
		return false
	}
	attempts.entries[ip] = append(list, time.Now())
	if len(attempts.entries) > 10000 {
		attempts.entries = map[string][]time.Time{ip: attempts.entries[ip]}
	}
	return true
}
func browserError(w http.ResponseWriter, err error) {
	http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
}
