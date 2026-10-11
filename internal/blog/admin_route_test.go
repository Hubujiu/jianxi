package blog

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestAdminRouteGateAndLogin(t *testing.T) {
	a := testApp(t)
	handler := a.Handler()
	token := strings.Repeat("d", 64)
	if err := a.Auth.write(token); err != nil {
		t.Fatal(err)
	}
	session, _ := a.Auth.Login(token)
	for _, path := range []string{"/admin/", "/admin/login", "/admin/settings/", "/admin/previews/fake/", "/admin/" + strings.Repeat("0", 32) + "/"} {
		for _, method := range []string{"GET", "POST"} {
			r := httptest.NewRequest(method, path, nil)
			r.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != 404 || strings.Contains(w.Body.String(), a.Auth.adminBase) || w.Header().Get("Location") != "" {
				t.Fatalf("unguarded path %s %s: %d", method, path, w.Code)
			}
		}
	}
	for _, path := range []string{"/", "/about/", "/search/", "/robots.txt", "/sitemap.xml"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		body := w.Body.String()
		if strings.Contains(body, a.Auth.adminBase) || strings.Contains(body, `class="page-foot"`) || strings.Contains(body, `href="/admin/`) {
			t.Fatalf("public page exposes footer or admin route: %s", path)
		}
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", a.Auth.adminBase, nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `action="`+a.Auth.adminBase+`login"`) || w.Header().Get("Referrer-Policy") != "same-origin" {
		t.Fatal("secured login page invalid")
	}
	for _, origin := range []string{"null", "https://attacker.example"} {
		r := httptest.NewRequest("POST", a.Auth.adminBase+"login", strings.NewReader(url.Values{"token": {token}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", origin)
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != http.StatusForbidden || len(w.Result().Cookies()) != 0 {
			t.Fatalf("invalid login origin accepted: %s, status %d", origin, w.Code)
		}
	}
	for _, key := range []string{"wrong", token} {
		r := httptest.NewRequest("POST", a.Auth.adminBase+"login", strings.NewReader(url.Values{"token": {key}}.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://example.com")
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if key != token {
			if len(w.Result().Cookies()) != 0 || w.Code == 303 {
				t.Fatal("invalid token accepted")
			}
			continue
		}
		if w.Code != 303 || w.Header().Get("Location") != a.Auth.adminBase {
			t.Fatal("login redirect lost security code")
		}
		cookies := w.Result().Cookies()
		if len(cookies) != 1 || cookies[0].Path != a.Auth.adminBase {
			t.Fatal("session cookie has wrong scope")
		}
		r = httptest.NewRequest("GET", a.Auth.adminBase, nil)
		r.AddCookie(cookies[0])
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != 200 || !strings.Contains(w.Body.String(), `href="`+a.Auth.adminBase+`content/"`) {
			t.Fatal("admin links lost security code")
		}
	}
	reopened, err := OpenAuth(a.Store.root)
	if err != nil || reopened.adminBase != a.Auth.adminBase {
		t.Fatal("route changed on restart", err)
	}
	if _, err := a.Auth.Rotate(); err != nil {
		t.Fatal(err)
	}
	reopened, err = OpenAuth(a.Store.root)
	if err != nil || reopened.adminBase != a.Auth.adminBase {
		t.Fatal("route changed on token rotation", err)
	}
}

func TestAdminPushConfigurationAndRecords(t *testing.T) {
	a := testApp(t)
	token := strings.Repeat("p", 64)
	if err := a.Auth.write(token); err != nil {
		t.Fatal(err)
	}
	session, ok := a.Auth.Login(token)
	if !ok {
		t.Fatal("login failed")
	}
	if err := a.Store.Update(func(s *State) error {
		s.Integrations = Integrations{BaiduToken: "saved-token", SEOConsole: "https://seo.example", GEOConsole: "https://geo.example"}
		s.Jobs["push-test"] = DiscoveryJob{ID: "push-test", Status: "failed", Message: "push-test-result"}
		s.Events = []Event{{Kind: "published", Message: "first-publish", Time: "2026-10-08T10:00:00Z"}, {Kind: "draft", Message: "unpublished-draft"}, {Kind: "published", Message: "latest-publish", Time: "2026-10-09T10:00:00Z"}}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	handler := a.Handler()
	request := func(method string, values url.Values) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, a.Auth.adminBase+"integrations/", strings.NewReader(values.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", "http://example.com")
		r.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	w := request("GET", nil)
	body := w.Body.String()
	for _, text := range []string{"SEO 推送", "百度普通收录推送", "推送记录", "push-test-result"} {
		if !strings.Contains(body, text) {
			t.Fatalf("missing push UI: %s", text)
		}
	}
	for _, text := range []string{"GEO", "分析报告", "seoConsole", "geoConsole"} {
		if strings.Contains(body, text) {
			t.Fatalf("obsolete analysis UI: %s", text)
		}
	}
	w = request("POST", url.Values{"action": {"integrations"}, "csrf": {session.CSRF}, "baiduSite": {"https://site.example"}, "enabled": {"on"}})
	config := a.Store.Read().Integrations
	if w.Code != 303 || config.BaiduSite != "https://site.example" || !config.Enabled || config.BaiduToken != "saved-token" || config.SEOConsole != "https://seo.example" || config.GEOConsole != "https://geo.example" {
		t.Fatal("push configuration save failed or changed unrelated configuration")
	}
	w = request("POST", url.Values{"action": {"retry"}, "csrf": {session.CSRF}, "jobId": {"push-test"}})
	if w.Code != 303 || !strings.HasPrefix(w.Header().Get("Location"), a.Auth.adminBase+"integrations/") || a.Store.Read().Jobs["push-test"].Status != "pending" {
		t.Fatal("push retry did not stay on integrations page")
	}
	r := httptest.NewRequest("GET", a.Auth.adminBase+"jobs/", nil)
	r.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	body = w.Body.String()
	if !strings.Contains(body, "历史发布") || strings.Contains(body, "push-test-result") || strings.Contains(body, "unpublished-draft") || strings.Index(body, "latest-publish") >= strings.Index(body, "first-publish") {
		t.Fatal("publishing history includes unrelated events or is not newest first")
	}
	if !strings.Contains(body, `href="/"`) || !strings.Contains(body, `href="`+a.Auth.adminBase+`"`) || !strings.Contains(body, `aria-current="page">发布记录`) {
		t.Fatal("publishing history breadcrumbs missing navigation or current page")
	}
}
