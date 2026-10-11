package blog

import (
	"context"
	"strings"
	"testing"
)

func TestAboutPageUsesProfileLayoutWithAndWithoutContent(t *testing.T) {
	for _, published := range []bool{false, true} {
		p := Page{Type: "about", Title: "关于我", Path: "/about/", Prefix: "/admin/previews/example", Description: "写一点技术，留一些观察。", Snap: Snapshot{Settings: Settings{Author: "狐不九"}}}
		if published {
			p.Article = &Revision{Content: Content{Author: "狐不九", AvatarURL: "/media/avatar", SocialLinks: []SocialLink{{Name: "GitHub", URL: "https://github.com/example"}}, Markdown: "介绍"}, Compiled: Compiled{HTML: "<p>保留的介绍</p>"}}
		}
		b, err := (&App{}).renderPage(p)
		if err != nil {
			t.Fatal(err)
		}
		html := string(b)
		for _, want := range []string{`class="about-page"`, `id="profile-name">狐不九`, `AboutPage`, `/admin/previews/example/timeline/`} {
			if !strings.Contains(html, want) {
				t.Fatalf("published=%v: missing %q", published, want)
			}
		}
		for _, unwanted := range []string{`class="reading"`, `class="byline"`, `id="article-toc"`, `BlogPosting`, `返回文章列表`} {
			if strings.Contains(html, unwanted) {
				t.Fatalf("published=%v: contains article element %q", published, unwanted)
			}
		}
		if published && (!strings.Contains(html, `/admin/previews/example/media/avatar`) || !strings.Contains(html, `rel="noopener noreferrer"`) || !strings.Contains(html, "保留的介绍")) {
			t.Fatal("profile media, external links or introduction missing")
		}
		if !published && strings.Contains(html, `class="about-socials"`) {
			t.Fatal("unconfigured social accounts must stay empty")
		}
	}
}

func TestProfileRejectsUnsafeURLsAndRetainsAvatarAsset(t *testing.T) {
	c := Content{ID: "about", Kind: "about", Slug: "about", Title: "关于我", AvatarURL: "/media/avatar"}
	compiled, err := (&Compiler{}).Compile(context.Background(), c)
	if err != nil || len(compiled.Assets) != 1 || compiled.Assets[0] != "avatar" {
		t.Fatalf("avatar must participate in publication assets: %+v, %v", compiled.Assets, err)
	}
	for _, unsafe := range []string{"javascript:alert(1)", "//example.com/avatar", "http://example.com/avatar"} {
		c.AvatarURL = unsafe
		if validateContent(c) == nil {
			t.Fatalf("accepted unsafe avatar %q", unsafe)
		}
		c.AvatarURL = ""
		c.SocialLinks = []SocialLink{{Name: "账号", URL: unsafe}}
		if validateContent(c) == nil {
			t.Fatalf("accepted unsafe social link %q", unsafe)
		}
		c.SocialLinks = nil
	}
}
