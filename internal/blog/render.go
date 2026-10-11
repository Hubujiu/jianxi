package blog

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

//go:embed web
var webFS embed.FS

type Group struct {
	ID, Name, Description, URL string
	Count                      int
	Posts                      []Revision
}
type Page struct {
	AdminBase                                                    string
	Type, Title, Description, Path, Prefix, Query, Category, Tag string
	Snap                                                         Snapshot
	Posts                                                        []Revision
	Groups                                                       []Group
	Article                                                      *Revision
	Previous, Next                                               string
	PageNumber, Pages                                            int
	NoIndex                                                      bool
	Error                                                        string
	Timeline                                                     TimelineData
}
type Day struct {
	Date    string
	Count   int
	Padding bool
}
type TimelineMonthLabel struct {
	Label  string
	Column int
}
type TimelineMonth struct {
	Name  string
	Posts []Revision
}
type TimelineYear struct {
	Name   string
	Count  int
	Months []TimelineMonth
}
type TimelineData struct {
	Start, End                 string
	Columns, Count, ActiveDays int
	Days                       []Day
	Months                     []TimelineMonthLabel
	Years                      []TimelineYear
}

func taxonPath(t Taxon) string {
	kind := "tags"
	if t.Kind == "category" {
		kind = "categories"
	}
	return "/" + kind + "/" + t.Slug + "/"
}
func date(s string) string {
	t, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		return "待发布"
	}
	return t.In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006.01.02")
}

func timelineData(posts []Revision, now time.Time) TimelineData {
	zone := time.FixedZone("Asia/Shanghai", 8*3600)
	today := now.In(zone)
	end := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, zone)
	start := end.AddDate(0, 0, -364)
	data := TimelineData{Start: start.Format("2006.01.02"), End: end.Format("2006.01.02")}
	counts := map[string]int{}
	for _, post := range posts {
		published, err := time.Parse(time.RFC3339Nano, post.PublishedAt)
		if err != nil {
			continue
		}
		day := published.In(zone).Format("2006-01-02")
		counts[day]++
		if day >= start.Format("2006-01-02") && day <= end.Format("2006-01-02") {
			data.Count++
		}
		year, month := day[:4], day[5:7]
		if len(data.Years) == 0 || data.Years[len(data.Years)-1].Name != year {
			data.Years = append(data.Years, TimelineYear{Name: year})
		}
		y := &data.Years[len(data.Years)-1]
		y.Count++
		if len(y.Months) == 0 || y.Months[len(y.Months)-1].Name != month {
			y.Months = append(y.Months, TimelineMonth{Name: month})
		}
		m := &y.Months[len(y.Months)-1]
		m.Posts = append(m.Posts, post)
	}
	gridStart := start.AddDate(0, 0, -int(start.Weekday()))
	gridEnd := end.AddDate(0, 0, 6-int(end.Weekday()))
	lastMonth := time.Month(0)
	for day := gridStart; !day.After(gridEnd); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		padding := day.Before(start) || day.After(end)
		count := counts[key]
		data.Days = append(data.Days, Day{Date: key, Count: count, Padding: padding})
		if !padding && count > 0 {
			data.ActiveDays++
		}
		if !padding && day.Month() != lastMonth {
			data.Months = append(data.Months, TimelineMonthLabel{Label: fmt.Sprintf("%d月", day.Month()), Column: (len(data.Days)-1)/7 + 1})
			lastMonth = day.Month()
		}
	}
	data.Columns = len(data.Days) / 7
	return data
}
func buildFingerprint(script string) string {
	files := map[string]string{"renderer": RendererVersion}
	_ = fs.WalkDir(webFS, "web", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := webFS.ReadFile(path)
		files[path] = hashBytes(b)
		return nil
	})
	if script == "" {
		return digest(files)
	}
	for _, p := range []string{script, filepath.Join(filepath.Dir(script), "..", "package-lock.json")} {
		b, _ := os.ReadFile(p)
		files[filepath.Base(p)] = hashBytes(b)
	}
	return digest(files)
}

func siteFingerprint() string {
	files := map[string]string{}
	_ = fs.WalkDir(webFS, "web", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, _ := webFS.ReadFile(path)
		files[path] = hashBytes(b)
		return nil
	})
	return digest(files)
}

var pageTemplate = template.Must(template.New("page.html").Funcs(template.FuncMap{
	"video":     renderVideo,
	"hasPrefix": strings.HasPrefix,
	"date":      date, "path": func(c Content) string { return contentPath(c) }, "taxpath": taxonPath, "num": func(i int) string { return fmt.Sprintf("%02d", i) }, "inc": func(i int) int { return i + 1 },
	"isoDate":   func(s string) string { return strings.ReplaceAll(date(s), ".", "-") },
	"shortDate": func(s string) string { d := date(s); return strings.ReplaceAll(d[5:], ".", " / ") },
	"monthName": func(s string) string { return strings.TrimLeft(s, "0") },
	"heatLevel": func(n int) int { return min(n, 4) },
	"body": func(r Revision, prefix string) template.HTML {
		return template.HTML(strings.ReplaceAll(strings.ReplaceAll(r.Compiled.HTML, `src="/`, `src="`+prefix+`/`), `href="/`, `href="`+prefix+`/`))
	},
	"initial": func(s string) string {
		r := []rune(s)
		if len(r) > 0 {
			return string(r[0])
		}
		return "间"
	},
	"category": func(r Revision, s Snapshot) string {
		if t, ok := s.Taxonomy[r.Content.CategoryID]; ok {
			return t.Name
		}
		return "未分类"
	},
	"count": func(s Snapshot) int {
		n := 0
		for _, r := range s.Posts {
			if r.Content.Kind == "article" {
				n++
			}
		}
		return n
	},
	"groups": func(s Snapshot, kind string) []Group { return taxonomyGroups(s, kind) },
	"profileName": func(p Page) string {
		if p.Article != nil && strings.TrimSpace(p.Article.Content.Author) != "" {
			return p.Article.Content.Author
		}
		return p.Snap.Settings.Author
	},
	"profileURL": func(raw, prefix string) string {
		if strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
			return prefix + raw
		}
		return raw
	},
	"jsonld": func(p Page) template.JS {
		obj := map[string]any{"@context": "https://schema.org", "@type": "WebSite", "name": p.Snap.Settings.Name, "url": p.Snap.Settings.Domain}
		if p.Type == "about" {
			name := p.Snap.Settings.Author
			if p.Article != nil && p.Article.Content.Author != "" {
				name = p.Article.Content.Author
			}
			obj = map[string]any{"@context": "https://schema.org", "@type": "AboutPage", "name": p.Title, "description": p.Description, "url": p.Snap.Settings.Domain + p.Path, "mainEntity": map[string]string{"@type": "Person", "name": name}}
		} else if p.Article != nil {
			r := p.Article
			obj = map[string]any{"@context": "https://schema.org", "@type": "BlogPosting", "headline": r.Content.Title, "description": r.Content.Summary, "author": map[string]string{"@type": "Person", "name": r.Content.Author}, "datePublished": r.PublishedAt, "dateModified": r.UpdatedAt, "mainEntityOfPage": p.Snap.Settings.Domain + p.Path}
		}
		b, _ := json.Marshal(obj)
		return template.JS(b)
	},
}).ParseFS(webFS, "web/page.html", "web/timeline.html", "web/about.html", "web/font-head.html"))

func sortedPosts(s Snapshot) []Revision {
	list := []Revision{}
	for _, r := range s.Posts {
		if r.Content.Kind == "article" {
			list = append(list, r)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].PublishedAt == list[j].PublishedAt {
			return list[i].Content.ID < list[j].Content.ID
		}
		return list[i].PublishedAt > list[j].PublishedAt
	})
	return list
}
func taxonomyGroups(s Snapshot, kind string) []Group {
	out := []Group{}
	for _, t := range s.Taxonomy {
		if t.Kind != kind {
			continue
		}
		g := Group{ID: t.ID, Name: t.Name, Description: t.Description, URL: taxonPath(t)}
		for _, r := range sortedPosts(s) {
			if r.Content.CategoryID == t.ID || contains(r.Content.TagIDs, t.ID) {
				g.Posts = append(g.Posts, r)
			}
		}
		g.Count = len(g.Posts)
		if g.Count > 0 {
			out = append(out, g)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}
func contains(a []string, b string) bool {
	for _, v := range a {
		if v == b {
			return true
		}
	}
	return false
}
func (a *App) renderPage(p Page) ([]byte, error) {
	if a.Auth != nil && p.Prefix != "" {
		p.AdminBase = a.Auth.adminBase
	}
	var b bytes.Buffer
	if p.Description == "" {
		p.Description = p.Snap.Settings.Description
	}
	err := pageTemplate.Execute(&b, p)
	return b.Bytes(), err
}
func (a *App) build(s Snapshot, kind string) (string, error) {
	dir := filepath.Join(a.Store.root, kind, s.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	prefix := ""
	if kind == "previews" {
		prefix = a.Auth.adminBase + "previews/" + s.ID
	}
	pages := []Page{}
	addList := func(base, title, desc, typ string, posts []Revision) {
		pagesCount := max(1, (len(posts)+11)/12)
		for n := 1; n <= pagesCount; n++ {
			path := base
			if n > 1 {
				path += fmt.Sprintf("page/%d/", n)
			}
			p := Page{Type: typ, Title: title, Description: desc, Path: path, Posts: posts[min((n-1)*12, len(posts)):min(n*12, len(posts))], PageNumber: n, Pages: pagesCount}
			if n > 1 {
				p.Previous = base
				if n > 2 {
					p.Previous += fmt.Sprintf("page/%d/", n-1)
				}
			}
			if n < pagesCount {
				p.Next = base + fmt.Sprintf("page/%d/", n+1)
			}
			pages = append(pages, p)
		}
	}
	posts := sortedPosts(s)
	addList("/", "最新文章", s.Settings.Description, "home", posts)
	for _, kind := range []string{"category", "tag"} {
		groups := taxonomyGroups(s, kind)
		path, title, typ := "/categories/", "全部分类", "categories"
		if kind == "tag" {
			path = "/tags/"
			title = "全部标签"
			typ = "tags"
		}
		pages = append(pages, Page{Type: typ, Title: title, Path: path, Groups: groups})
		for _, g := range groups {
			addList(g.URL, g.Name, g.Description, "list", g.Posts)
		}
	}
	collectionGroups := []Group{}
	for _, c := range s.Collections {
		g := Group{ID: c.ID, Name: c.Name, Description: c.Description, URL: "/collections/" + c.Slug + "/"}
		for _, id := range c.ArticleIDs {
			if r, ok := s.Posts[id]; ok {
				g.Posts = append(g.Posts, r)
			}
		}
		g.Count = len(g.Posts)
		collectionGroups = append(collectionGroups, g)
		pages = append(pages, Page{Type: "collection", Title: c.Name, Description: c.Description, Path: g.URL, Posts: g.Posts})
	}
	sort.Slice(collectionGroups, func(i, j int) bool { return collectionGroups[i].Name < collectionGroups[j].Name })
	pages = append(pages, Page{Type: "collections", Title: "专栏合集", Path: "/collections/", Groups: collectionGroups})
	pages = append(pages, Page{Type: "timeline", Title: "时间线", Description: "每一次记录，都是思考留下的坐标。", Path: "/timeline/", Posts: posts, Timeline: timelineData(posts, time.Now())})
	hasAbout := false
	for i, r := range posts {
		r := r
		p := Page{Type: "article", Title: r.Content.Title, Description: r.Content.Summary, Path: contentPath(r.Content), Article: &r}
		if i+1 < len(posts) {
			p.Next = contentPath(posts[i+1].Content)
		}
		pages = append(pages, p)
	}
	if r, ok := s.Posts["about"]; ok {
		pages = append(pages, Page{Type: "about", Title: "关于我", Description: r.Content.Summary, Path: "/about/", Article: &r})
		hasAbout = true
	}
	if !hasAbout {
		pages = append(pages, Page{Type: "about", Title: "关于我", Description: s.Settings.Description, Path: "/about/"})
	}
	pages = append(pages, Page{Type: "404", Title: "这一页，暂时留白", Description: "你寻找的页面不存在，或已被移走。", Path: "/404.html", NoIndex: true})
	for _, p := range pages {
		p.Snap = s
		p.Prefix = prefix
		p.NoIndex = p.NoIndex || prefix != ""
		b, err := a.renderPage(p)
		if err != nil {
			return "", err
		}
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(p.Path, "/")))
		if strings.HasSuffix(p.Path, "/") {
			file = filepath.Join(file, "index.html")
		}
		if err = os.MkdirAll(filepath.Dir(file), 0700); err != nil {
			return "", err
		}
		if err = os.WriteFile(file, b, 0600); err != nil {
			return "", err
		}
	}
	for _, r := range s.Posts {
		for name, svg := range r.Compiled.Diagrams {
			target := filepath.Join(dir, "diagrams", name)
			if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
				return "", err
			}
			if err := os.WriteFile(target, []byte(svg), 0600); err != nil {
				return "", err
			}
		}
	}
	if err := writeJSON(filepath.Join(dir, "snapshot.json"), s); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, ".sitehash"), []byte(a.SiteHash), 0600); err != nil {
		return "", err
	}
	var xml strings.Builder
	xml.WriteString(`<?xml version="1.0" encoding="UTF-8"?><urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">`)
	for _, p := range pages {
		if p.NoIndex {
			continue
		}
		xml.WriteString("<url><loc>" + html.EscapeString(s.Settings.Domain+p.Path) + "</loc>")
		if p.Article != nil && p.Article.UpdatedAt != "" {
			xml.WriteString("<lastmod>" + p.Article.UpdatedAt + "</lastmod>")
		}
		xml.WriteString("</url>")
	}
	xml.WriteString("</urlset>")
	if err := os.WriteFile(filepath.Join(dir, "sitemap.xml"), []byte(xml.String()), 0600); err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(dir, "robots.txt"), []byte("User-agent: *\nAllow: /\nDisallow: /admin/\nDisallow: /_control/\nDisallow: /search/\nSitemap: "+s.Settings.Domain+"/sitemap.xml\n"), 0600); err != nil {
		return "", err
	}
	return treeHash(dir)
}
