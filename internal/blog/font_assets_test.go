package blog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestFontAssetsAndCaching(t *testing.T) {
	a := testApp(t)
	handler := a.Handler()
	var stylesheet string
	for _, path := range []string{"/", "/about/", "/missing/", a.Auth.adminBase} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		body := w.Body.String()
		links := regexp.MustCompile(`<link rel="stylesheet" href="(/static/fonts/wenkai/[^" ]+\.css)">`).FindStringSubmatch(body)
		if len(links) != 2 || strings.Count(body, `as="font"`) != 1 || !strings.Contains(body, `type="font/woff2" crossorigin`) || strings.Contains(body, "geist.woff2") {
			t.Fatalf("font discovery is incomplete on %s", path)
		}
		stylesheet = links[1]
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", stylesheet, nil))
	if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/css") || !strings.Contains(w.Header().Get("Cache-Control"), "immutable") {
		t.Fatal("font stylesheet must be available and immutable")
	}
	css := w.Body.String()
	urls := regexp.MustCompile(`url\('\./([^']+\.woff2)'\)`).FindAllStringSubmatch(css, -1)
	if len(urls) < 2 || strings.Count(css, "unicode-range:") != len(urls) || strings.Count(css, "font-display: swap") != len(urls) {
		t.Fatal("fonts must be split into on-demand Unicode subsets with visible fallback text")
	}
	for _, match := range urls {
		path := "/static/fonts/wenkai/" + match[1]
		for _, method := range []string{"GET", "HEAD"} {
			w = httptest.NewRecorder()
			handler.ServeHTTP(w, httptest.NewRequest(method, path, nil))
			if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "font/woff2" || w.Header().Get("Cache-Control") != "public, max-age=31536000, immutable" {
				t.Fatalf("font request %s %s: %d %v", method, path, w.Code, w.Header())
			}
			if method == "GET" && !bytes.HasPrefix(w.Body.Bytes(), []byte("wOF2")) {
				t.Fatalf("invalid WOFF2 asset %s", path)
			}
		}
	}
	for _, path := range []string{"/static/site.css", "/static/fonts/wenkai/0000000000000000.woff2"} {
		w = httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		if w.Header().Get("Cache-Control") != "no-cache" {
			t.Fatalf("mutable styles and missing fonts must not receive immutable caching: %s", path)
		}
	}
}
