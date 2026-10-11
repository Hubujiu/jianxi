package blog

import (
	"encoding/json"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type AdminArticle struct {
	Article  Article
	Revision Revision
	Status   string
	History  []Revision
}
type AdminData struct {
	AdminBase     string
	View          string
	Session       Session
	State         State
	Articles      []AdminArticle
	Previews      []Preview
	Preview       *Preview
	Change        *ChangeSet
	Message       string
	Error         string
	Token         string
	Restore       *RestoreCandidate
	Taxonomy      []Taxon
	Breadcrumbs   []AdminBreadcrumb
	PublishEvents []Event
}

type AdminBreadcrumb struct {
	Title string
	URL   string
}

var adminTemplate = template.Must(template.New("admin.html").Funcs(template.FuncMap{"date": date, "short": func(s string) string {
	if len(s) > 10 {
		return s[:10]
	}
	return s
}, "path": contentPath, "join": strings.Join}).ParseFS(webFS, "web/admin.html", "web/font-head.html"))

func (a *App) admin(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	// Preserve the Origin on same-site form POSTs; no-referrer makes browsers
	// send Origin: null. Cross-origin requests still receive no referrer.
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data:; connect-src 'self'; frame-src 'self'; object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'self'")
	s, logged := a.Auth.Session(r)
	view := strings.Trim(strings.TrimPrefix(r.URL.Path, a.Auth.adminBase), "/")
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "仅支持 GET / POST", 405)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 110<<20)
	if view == "login" && r.Method == "POST" {
		if !sameOrigin(r) {
			http.Error(w, "请求来源无效，请重新打开后台登录页面", http.StatusForbidden)
			return
		}
		if !allowLogin(r) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "登录请求过于频繁，请一分钟后重试", http.StatusTooManyRequests)
			return
		}
		session, ok := a.Auth.Login(r.FormValue("token"))
		if !ok {
			a.renderAdmin(w, AdminData{View: "login", Error: "管理密钥无效，请重新输入。"})
			return
		}
		secure := r.TLS != nil || strings.HasPrefix(a.Store.active.Load().Settings.Domain, "https://")
		http.SetCookie(w, &http.Cookie{Name: "jianxi_session", Value: session.ID, Path: a.Auth.adminBase, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: secure, MaxAge: 43200})
		http.Redirect(w, r, a.Auth.adminBase, 303)
		return
	}
	if !logged {
		if r.Method == "GET" && (!strings.Contains(view, "/") || view == "") {
			a.renderAdmin(w, AdminData{View: "login"})
		} else {
			http.Error(w, "需要管理密钥", 401)
		}
		return
	}
	if strings.HasPrefix(view, "previews/") {
		parts := strings.SplitN(strings.TrimPrefix(view, "previews/"), "/", 2)
		id := parts[0]
		path := "/"
		if len(parts) == 2 {
			path += "" + parts[1]
			if strings.HasSuffix(r.URL.Path, "/") && !strings.HasSuffix(path, "/") {
				path += "/"
			}
		}
		state := a.Store.Read()
		p, ok := state.Previews[id]
		if !ok {
			if len(id) != 32 || !hexID(id) {
				http.NotFound(w, r)
				return
			}
			b, err := os.ReadFile(filepath.Join(a.Store.root, "restore", id, "candidate.json"))
			if err != nil {
				http.NotFound(w, r)
				return
			}
			var candidate RestoreCandidate
			if json.Unmarshal(b, &candidate) != nil {
				http.NotFound(w, r)
				return
			}
			p.Snapshot = candidate.Snapshot
		}
		if strings.HasPrefix(path, "/media/") {
			assetID := strings.TrimPrefix(path, "/media/")
			asset, ok := p.Snapshot.Assets[assetID]
			if !ok {
				http.NotFound(w, r)
				return
			}
			file := filepath.Join(a.Store.root, "assets", asset.ID)
			if _, err := os.Stat(filepath.Join(a.Store.root, "restore", id, "assets", asset.ID)); err == nil {
				file = filepath.Join(a.Store.root, "restore", id, "assets", asset.ID)
			}
			w.Header().Set("Content-Type", asset.MIME)
			http.ServeFile(w, r, file)
			return
		}
		a.serveArtifact(w, r, p.Snapshot, filepath.Join(a.Store.root, "previews", id), path, true)
		return
	}
	if strings.HasPrefix(view, "backups/") {
		name := strings.TrimPrefix(view, "backups/")
		if filepath.Base(name) != name || strings.ContainsAny(name, "/\\") || !strings.HasSuffix(name, ".zip") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
		w.Header().Set("Content-Type", "application/zip")
		http.ServeFile(w, r, filepath.Join(a.Store.root, "backups", name))
		return
	}
	data := AdminData{View: view, Session: s, State: a.Store.Read(), Message: r.URL.Query().Get("message")}
	if r.Method == "POST" {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			if err := r.ParseMultipartForm(4 << 20); err != nil {
				browserError(w, err)
				return
			}
			defer r.MultipartForm.RemoveAll()
		}
		if !csrfOK(r, s) {
			http.Error(w, "会话校验失败，请刷新重试", 403)
			return
		}
		if err := a.adminAction(w, r, &data); err != nil {
			data.Error = err.Error()
		} else if data.View == "redirected" {
			return
		}
		data.State = a.Store.Read()
	}
	if data.View == "preview" && data.Preview == nil {
		p, ok := data.State.Previews[r.URL.Query().Get("id")]
		if ok {
			ch := data.State.Changes[p.ChangeSet]
			data.Preview = &p
			data.Change = &ch
		} else {
			data.Error = "预览不存在"
			data.View = "content"
		}
	}
	for _, article := range data.State.Articles {
		id := article.Draft
		status := "草稿"
		if id == "" {
			id = article.Published
			status = "已发布"
		}
		if id == "" && len(article.History) > 0 {
			id = article.History[0]
			status = "已下架"
		}
		rev, ok := data.State.Revisions[id]
		if !ok {
			continue
		}
		item := AdminArticle{Article: article, Revision: rev, Status: status}
		if article.Published != "" && article.Draft != "" {
			item.Status = "有待发布修改"
		}
		for _, id := range article.History {
			item.History = append(item.History, data.State.Revisions[id])
		}
		data.Articles = append(data.Articles, item)
	}
	sort.Slice(data.Articles, func(i, j int) bool { return data.Articles[i].Revision.CreatedAt > data.Articles[j].Revision.CreatedAt })
	for _, p := range data.State.Previews {
		data.Previews = append(data.Previews, p)
	}
	for _, t := range data.State.Taxonomy {
		data.Taxonomy = append(data.Taxonomy, t)
	}
	sort.Slice(data.Taxonomy, func(i, j int) bool { return data.Taxonomy[i].Name < data.Taxonomy[j].Name })
	a.renderAdmin(w, data)
}
func (a *App) renderAdmin(w http.ResponseWriter, data AdminData) {
	data.AdminBase = a.Auth.adminBase
	data.Breadcrumbs = []AdminBreadcrumb{{Title: "首页", URL: "/"}}
	if data.View == "" {
		data.Breadcrumbs = append(data.Breadcrumbs, AdminBreadcrumb{Title: "管理空间"})
	} else {
		data.Breadcrumbs = append(data.Breadcrumbs, AdminBreadcrumb{Title: "管理空间", URL: a.Auth.adminBase})
		title := map[string]string{"content": "内容与发布", "taxonomy": "分类、标签与专栏", "settings": "站点设置", "integrations": "SEO 推送", "jobs": "发布记录", "backup": "备份与恢复", "security": "访问密钥", "preview": "发布预览", "restore": "恢复预览"}[data.View]
		if data.View == "preview" {
			data.Breadcrumbs = append(data.Breadcrumbs, AdminBreadcrumb{Title: "内容与发布", URL: a.Auth.adminBase + "content/"})
		} else if data.View == "restore" {
			data.Breadcrumbs = append(data.Breadcrumbs, AdminBreadcrumb{Title: "备份与恢复", URL: a.Auth.adminBase + "backup/"})
		}
		data.Breadcrumbs = append(data.Breadcrumbs, AdminBreadcrumb{Title: title})
	}
	for _, event := range data.State.Events {
		if event.Kind == "published" {
			data.PublishEvents = append(data.PublishEvents, event)
		}
	}
	sort.SliceStable(data.PublishEvents, func(i, j int) bool { return data.PublishEvents[i].Time > data.PublishEvents[j].Time })
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := adminTemplate.Execute(w, data); err != nil {
		http.Error(w, err.Error(), 500)
	}
}
func (a *App) adminAction(w http.ResponseWriter, r *http.Request, d *AdminData) error {
	action := r.FormValue("action")
	redirect := func(path, message string) {
		d.View = "redirected"
		http.Redirect(w, r, path+"?message="+urlQueryEscape(message), 303)
	}
	switch action {
	case "delete":
		generation, _ := strconv.ParseInt(r.FormValue("generation"), 10, 64)
		_, err := a.Delete(DeleteInput{ContentID: r.FormValue("contentId"), ConfirmContentID: r.FormValue("confirmContentId"), ExpectedGeneration: generation})
		if err != nil {
			return err
		}
		redirect(a.Auth.adminBase+"content/", "永久删除完成；外部已下载备份与第三方缓存不在清理范围内")
	case "logout":
		a.Auth.Logout(d.Session.ID)
		http.SetCookie(w, &http.Cookie{Name: "jianxi_session", Value: "", Path: a.Auth.adminBase, MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
		redirect(a.Auth.adminBase, "已退出")
	case "settings":
		revision, _ := strconv.Atoi(r.FormValue("revision"))
		_, err := a.Settings(SettingsInput{ExpectedRevision: revision, Settings: Settings{Name: r.FormValue("name"), Description: r.FormValue("description"), Domain: r.FormValue("domain"), Author: r.FormValue("author"), Footer: r.FormValue("footer"), AnalyticsID: r.FormValue("analyticsId")}})
		if err != nil {
			return err
		}
		redirect(a.Auth.adminBase+"settings/", "设置草稿已保存；在发布清单中勾选站点设置后预览")
	case "integrations":
		err := a.SetIntegrations(IntegrationInput{BaiduSite: r.FormValue("baiduSite"), BaiduToken: r.FormValue("baiduToken"), ClearToken: r.FormValue("clearToken") == "on", Enabled: r.FormValue("enabled") == "on", SEOConsole: d.State.Integrations.SEOConsole, GEOConsole: d.State.Integrations.GEOConsole})
		if err != nil {
			return err
		}
		redirect(a.Auth.adminBase+"integrations/", "推送配置已保存")
	case "preview":
		in := ChangesInput{Revisions: r.Form["revisions"], Taxonomy: r.Form["taxonomy"], Collections: r.Form["collections"], IncludeSettings: r.FormValue("includeSettings") == "on", ExpectedActive: r.FormValue("active")}
		ch, err := a.Changes(in)
		if err != nil {
			return err
		}
		p, err := a.Preview(r.Context(), ch.ID)
		if err != nil {
			return err
		}
		d.View = "preview"
		d.Preview = &p
		d.Change = &ch
	case "publish":
		p, err := a.Confirm(ConfirmInput{PreviewID: r.FormValue("previewId"), ArtifactHash: r.FormValue("artifactHash"), ChangeSetHash: r.FormValue("changeSetHash")})
		if err != nil {
			return err
		}
		_, err = a.Publish(PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: r.FormValue("changeSetHash"), ExpectedActive: r.FormValue("active"), IdempotencyKey: r.FormValue("idempotencyKey")})
		if err != nil {
			return err
		}
		redirect(a.Auth.adminBase, "所选变更已发布，静态页面与搜索已同步")
	case "withdraw":
		ch, err := a.Changes(ChangesInput{Withdraw: []string{r.FormValue("contentId")}, ExpectedActive: d.State.Active})
		if err != nil {
			return err
		}
		p, err := a.Preview(r.Context(), ch.ID)
		if err != nil {
			return err
		}
		d.View = "preview"
		d.Preview = &p
		d.Change = &ch
	case "rollback":
		rev, err := a.Rollback(r.Context(), RollbackInput{ContentID: r.FormValue("contentId"), Revision: r.FormValue("revision")})
		if err != nil {
			return err
		}
		ch, err := a.Changes(ChangesInput{Revisions: []string{rev.ID}, ExpectedActive: d.State.Active})
		if err != nil {
			return err
		}
		p, err := a.Preview(r.Context(), ch.ID)
		if err != nil {
			return err
		}
		d.View = "preview"
		d.Preview = &p
		d.Change = &ch
	case "rotate":
		token, err := a.Auth.Rotate()
		if err != nil {
			return err
		}
		d.Token = token
		d.View = "rotated"
	case "backup":
		name, err := a.Export()
		if err != nil {
			return err
		}
		d.Message = "完整备份已生成，可下载并离线保存。"
		d.View = "backup"
		d.Token = a.Auth.adminBase + "backups/" + name
	case "import":
		file, _, err := r.FormFile("backup")
		if err != nil {
			return err
		}
		defer file.Close()
		b, err := io.ReadAll(io.LimitReader(file, 101<<20))
		if err != nil {
			return err
		}
		candidate, err := a.InspectBackup(r.Context(), b)
		if err != nil {
			return err
		}
		d.Restore = &candidate
		d.View = "restore"
	case "restore":
		_, err := a.Restore(r.FormValue("candidateId"), r.FormValue("hash"))
		if err != nil {
			return err
		}
		redirect(a.Auth.adminBase, "恢复完成，请使用当前管理密钥重新解锁")
	case "retry":
		err := a.Store.Update(func(s *State) error {
			j, ok := s.Jobs[r.FormValue("jobId")]
			if !ok {
				return fail("JOB_NOT_FOUND", "任务不存在")
			}
			j.Status = "pending"
			j.NextAt = now()
			s.Jobs[j.ID] = j
			return nil
		})
		if err != nil {
			return err
		}
		path := "jobs/"
		if d.View == "integrations" {
			path = "integrations/"
		}
		redirect(a.Auth.adminBase+path, "已加入重试队列")
	default:
		return fail("UNKNOWN_ACTION", "未知操作")
	}
	return nil
}
func urlQueryEscape(s string) string {
	var out strings.Builder
	for _, b := range []byte(s) {
		const h = "0123456789ABCDEF"
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' {
			out.WriteByte(b)
		} else {
			out.WriteByte('%')
			out.WriteByte(h[b>>4])
			out.WriteByte(h[b&15])
		}
	}
	return out.String()
}
