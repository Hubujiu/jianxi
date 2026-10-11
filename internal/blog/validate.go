package blog

import (
	"net/url"
	"strings"
)

func validateSnapshot(s Snapshot) error {
	if s.Posts == nil || s.Taxonomy == nil || s.Collections == nil || s.Redirects == nil || s.Assets == nil {
		return fail("INVALID_SNAPSHOT", "公开快照缺少必要集合")
	}
	u, err := url.Parse(s.Settings.Domain)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1"))) {
		return fail("INVALID_DOMAIN", "备份站点域名无效")
	}
	if s.Settings.AnalyticsID != "" && (!hexID(s.Settings.AnalyticsID) || len(s.Settings.AnalyticsID) != 32) {
		return fail("INVALID_ANALYTICS", "统计标识无效")
	}
	paths := map[string]bool{}
	for id, t := range s.Taxonomy {
		if id != t.ID || !slugPattern.MatchString(t.Slug) || (t.Kind != "category" && t.Kind != "tag") || t.Name == "" {
			return fail("INVALID_TAXONOMY", "备份分类标签无效")
		}
		p := taxonPath(t)
		if paths[p] {
			return fail("SLUG_CONFLICT", "备份分类地址重复")
		}
		paths[p] = true
	}
	for id, r := range s.Posts {
		if r.Content.ID != id {
			return fail("INVALID_CONTENT", "备份内容 ID 不匹配")
		}
		if err := validateContent(r.Content); err != nil {
			return err
		}
		p := contentPath(r.Content)
		if paths[p] {
			return fail("SLUG_CONFLICT", "备份文章地址重复")
		}
		paths[p] = true
	}
	for id, c := range s.Collections {
		if id != c.ID || !slugPattern.MatchString(c.Slug) || c.Name == "" {
			return fail("INVALID_COLLECTION", "备份专栏无效")
		}
		p := "/collections/" + c.Slug + "/"
		if paths[p] {
			return fail("SLUG_CONFLICT", "专栏地址重复")
		}
		paths[p] = true
		for _, articleID := range c.ArticleIDs {
			if _, ok := s.Posts[articleID]; !ok {
				return fail("COLLECTION_DEPENDENCY", "专栏引用未发布文章")
			}
		}
	}
	for from, to := range s.Redirects {
		if !validPublicPath(from) || !validPublicPath(to) || from == to || paths[from] || !paths[to] {
			return fail("INVALID_REDIRECT", "备份路由跳转无效")
		}
	}
	return nil
}
func validPublicPath(p string) bool {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	return len(parts) == 2 && (parts[0] == "posts" || parts[0] == "categories" || parts[0] == "tags" || parts[0] == "collections") && slugPattern.MatchString(parts[1]) && p == "/"+parts[0]+"/"+parts[1]+"/"
}
