package blog

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func testApp(t *testing.T) *App {
	t.Helper()
	app, err := NewApp(t.TempDir(), "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { app.Store.Close() })
	return app
}
func draft(t *testing.T, a *App, id, title, body string) Revision {
	t.Helper()
	cp, err := a.WorkingCopy(CopyInput{ContentID: id})
	if err != nil {
		t.Fatal(err)
	}
	c := cp.Content
	c.Slug = "article-" + c.ID[:8]
	c.Title = title
	c.Summary = "中文摘要"
	c.Author = "测试作者"
	c.Markdown = body
	up, err := a.Upload(UploadInput{CopyID: cp.ID, Content: c, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	up, err = a.Parse(context.Background(), up.ID)
	if err != nil || up.Status != "parsed" {
		t.Fatalf("parse: %v %s", err, up.Diagnostic)
	}
	rev, err := a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: cp.Draft, Epoch: cp.Epoch, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	return rev
}
func publish(t *testing.T, a *App, revisions ...string) PublishResult {
	t.Helper()
	ch, err := a.Changes(ChangesInput{Revisions: revisions, ExpectedActive: a.Store.Read().Active})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	if err != nil {
		t.Fatal(err)
	}
	in := PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: ch.Hash, ExpectedActive: ch.Base, IdempotencyKey: uid()}
	result, err := a.Publish(in)
	if err != nil {
		t.Fatal(err)
	}
	again, err := a.Publish(in)
	if err != nil || again != result {
		t.Fatal("publish retry was not idempotent", err)
	}
	return result
}
func TestPublishingIsolationSearchAndWithdrawal(t *testing.T) {
	a := testApp(t)
	live := draft(t, a, "", "中文搜索与 Go", "## 重复标题\n\n文字里的秘密与 100%_ 符号。\n\n## 重复标题\n\n第二段")
	hidden := draft(t, a, "", "绝密草稿", "尚未公开")
	publish(t, a, live.ID)
	snap := *a.Store.active.Load()
	if len(snap.Posts) != 1 {
		t.Fatal("unselected draft published")
	}
	if _, ok := snap.Posts[hidden.Content.ID]; ok {
		t.Fatal("draft leaked")
	}
	for _, q := range []string{"中", "中文", "中文搜索", "秘密", "100%_", "Go"} {
		res, _, err := a.Search(context.Background(), snap, q, "", "", 1)
		if err != nil || len(res) != 1 {
			t.Fatalf("query %q: %v %d", q, err, len(res))
		}
	}
	res, _, _ := a.Search(context.Background(), snap, "绝密", "", "", 1)
	if len(res) != 0 {
		t.Fatal("private draft indexed")
	}
	if live.Compiled.TOC[0].ID == live.Compiled.TOC[1].ID {
		t.Fatal("duplicate heading anchors")
	}
	handler := a.Handler()
	for _, p := range []string{"/", "/categories/", "/tags/", "/about/", "/collections/", "/timeline/", contentPath(live.Content)} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 200 {
			t.Fatalf("%s: %d", p, w.Code)
		}
	}
	for _, p := range []string{"/snapshot.json", a.Auth.adminBase + "previews/fake/", "/_control/status", "/missing/"} {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code == 200 {
			t.Fatalf("unprotected %s", p)
		}
	}
	ch, err := a.Changes(ChangesInput{Withdraw: []string{live.Content.ID}, ExpectedActive: snap.ID})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, err = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.Publish(PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: ch.Hash, ExpectedActive: ch.Base, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", contentPath(live.Content), nil))
	if w.Code != 404 {
		t.Fatal("withdrawn page still public")
	}
	res, _, _ = a.Search(context.Background(), *a.Store.active.Load(), "中文", "", "", 1)
	if len(res) != 0 {
		t.Fatal("withdrawn article indexed")
	}
}
func TestStalePreviewAndDraftCAS(t *testing.T) {
	a := testApp(t)
	r := draft(t, a, "", "标题", "最初正文")
	ch, err := a.Changes(ChangesInput{Revisions: []string{r.ID}, ExpectedActive: a.Store.Read().Active})
	if err != nil {
		t.Fatal(err)
	}
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = draft(t, a, r.Content.ID, "新标题", "更新正文")
	_, err = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	if err == nil || !strings.Contains(err.Error(), "STALE") {
		t.Fatal("stale preview accepted", err)
	}
	if len(a.Store.active.Load().Posts) != 0 {
		t.Fatal("failed preview changed live state")
	}
	cp, _ := a.WorkingCopy(CopyInput{ContentID: r.Content.ID})
	up, _ := a.Upload(UploadInput{CopyID: cp.ID, Content: cp.Content, IdempotencyKey: uid()})
	up, _ = a.Parse(context.Background(), up.ID)
	_ = draft(t, a, r.Content.ID, "另一次修改", "正文")
	_, err = a.Commit(CommitInput{UploadID: up.ID, SourceHash: up.Hash, ExpectedDraft: cp.Draft, Epoch: cp.Epoch, IdempotencyKey: uid()})
	if err == nil || !strings.Contains(err.Error(), "CONFLICT") {
		t.Fatal("CAS failed", err)
	}
}
func TestHistoryRollbackKeepsDraft(t *testing.T) {
	a := testApp(t)
	r := draft(t, a, "", "v1", "第一版")
	publish(t, a, r.ID)
	first := a.Store.Read().Articles[r.Content.ID].FirstPublished
	for _, title := range []string{"v2", "v3", "v4", "v5"} {
		r = draft(t, a, r.Content.ID, title, title)
		publish(t, a, r.ID)
	}
	article := a.Store.Read().Articles[r.Content.ID]
	if len(article.History) != 4 {
		t.Fatal("history retention", article.History)
	}
	pending := draft(t, a, r.Content.ID, "待办草稿", "私有正文")
	rollback, err := a.Rollback(context.Background(), RollbackInput{ContentID: r.Content.ID, Revision: article.History[3]})
	if err != nil {
		t.Fatal(err)
	}
	publish(t, a, rollback.ID)
	state := a.Store.Read()
	if state.Articles[r.Content.ID].Draft != pending.ID {
		t.Fatal("rollback overwrote draft")
	}
	if state.Articles[r.Content.ID].FirstPublished != first {
		t.Fatal("first published changed")
	}
	if a.Store.active.Load().Posts[r.Content.ID].Content.Title != "v2" {
		t.Fatal("rollback wrong content")
	}
}
func TestMarkdownBoundaries(t *testing.T) {
	a := testApp(t)
	base := Content{ID: uid(), Kind: "article", Slug: "test", Title: "测试"}
	for _, body := range []string{"<script>alert(1)</script>", "正文 <b>HTML</b>", "[危险](javascript:alert(1))", "![图](http://example.com/x.png)"} {
		base.Markdown = body
		if _, err := a.Compiler.Compile(context.Background(), base); err == nil {
			t.Fatalf("accepted unsafe markdown %q", body)
		}
	}
	base.Markdown = "```html\n<script>alert(1)</script>\n```\n\n`$not_math$` and \\$20\n\n```markdown\n## heading\n$wrong$\n```"
	out, err := a.Compiler.Compile(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.HTML, "<script>") || out.Math || len(out.TOC) > 0 {
		t.Fatal("code was executed or parsed recursively")
	}
}
func TestMathAndMermaidStaticCompilation(t *testing.T) {
	if _, err := os.Stat("../../node_modules/katex/package.json"); err != nil {
		t.Skip("npm ci required for renderer integration tests")
	}
	a := testApp(t)
	base := Content{ID: uid(), Kind: "article", Slug: "math", Title: "公式", Markdown: "行内 $E=mc^2$。\n\n$$\n\\int_0^1 x^2 dx = \\frac{1}{3}\n$$\n\n```mermaid\nflowchart LR\n A[源文] --> B[静态页面]\n```"}
	out, err := a.Compiler.Compile(context.Background(), base)
	if err != nil {
		t.Fatal(err)
	}
	if !out.Math || !strings.Contains(out.HTML, "<math") || len(out.Diagrams) != 1 || !strings.Contains(out.HTML, "/diagrams/") {
		t.Fatal("missing static rich content")
	}
	for _, body := range []string{"$\\unknowncommand{x}$", "$$\nx=2", "```mermaid\n%%{init: {}}%%\nflowchart LR\nA-->B\n```"} {
		base.Markdown = body
		if _, err := a.Compiler.Compile(context.Background(), base); err == nil {
			t.Fatalf("accepted invalid rich content %q", body)
		}
	}
}
func TestBackupRestoreAndConflict(t *testing.T) {
	a := testApp(t)
	r := draft(t, a, "", "备份文章", "可恢复的正文")
	publish(t, a, r.ID)
	token := strings.Repeat("a", 64)
	if err := a.Auth.write(token); err != nil {
		t.Fatal(err)
	}
	a.Store.Update(func(s *State) error { s.Integrations.BaiduToken = "private-integration-secret"; return nil })
	name, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(a.Store.root, "backups", name))
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := a.InspectBackup(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	draft(t, a, "", "并发修改", "新数据")
	if _, err = a.Restore(candidate.ID, candidate.Hash); err == nil {
		t.Fatal("restore did not reject intervening write")
	}
	candidate, err = a.InspectBackup(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	session, _ := a.Auth.Login(token)
	_, err = a.Restore(candidate.ID, candidate.Hash)
	if err != nil {
		t.Fatal(err)
	}
	if !a.Auth.Verify(token) {
		t.Fatal("restore changed master token")
	}
	req := httptest.NewRequest("GET", a.Auth.adminBase, nil)
	req.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
	if _, ok := a.Auth.Session(req); ok {
		t.Fatal("restore retained session")
	}
	if len(a.Store.Read().Articles) != 1 {
		t.Fatal("restore merged instead of replacing")
	}
	if a.Store.Read().Integrations.BaiduToken != "private-integration-secret" {
		t.Fatal("private settings not restored")
	}
	snap := *a.Store.active.Load()
	res, _, _ := a.Search(context.Background(), snap, "可恢复", "", "", 1)
	if len(res) != 1 {
		t.Fatal("restore did not rebuild index")
	}
	bad := new(bytes.Buffer)
	z := zip.NewWriter(bad)
	w, _ := z.Create("../auth.json")
	w.Write([]byte("{}"))
	z.Close()
	if _, err = a.InspectBackup(context.Background(), bad.Bytes()); err == nil {
		t.Fatal("zip slip accepted")
	}
}
func TestAuthCSRFAndUnknownFields(t *testing.T) {
	a := testApp(t)
	token := strings.Repeat("b", 64)
	a.Auth.write(token)
	session, _ := a.Auth.Login(token)
	handler := a.Handler()
	request := httptest.NewRequest("POST", a.Auth.adminBase+"settings/", strings.NewReader("action=settings&csrf="+session.CSRF))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Origin", "https://attacker.example")
	request.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	if w.Code != 403 {
		t.Fatal("CSRF not rejected", w.Code)
	}
	request = httptest.NewRequest("POST", "/_control/working-copy", strings.NewReader(`{"kind":"article","template":"evil"}`))
	request.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, request)
	if w.Code != 400 {
		t.Fatal("unknown field accepted", w.Code)
	}
	newToken, err := a.Auth.Rotate()
	if err != nil || !a.Auth.Verify(newToken) || a.Auth.Verify(token) {
		t.Fatal("rotation failed", err)
	}
	request = httptest.NewRequest("GET", a.Auth.adminBase, nil)
	request.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
	if _, ok := a.Auth.Session(request); ok {
		t.Fatal("old session survived rotation")
	}
	b, _ := os.ReadFile(a.Auth.path)
	if bytes.Contains(b, []byte(newToken)) {
		t.Fatal("plaintext token stored in authority")
	}
}
func TestRestartVerifiesActiveArtifact(t *testing.T) {
	root := t.TempDir()
	a, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	active := a.Store.Read().Active
	a.Store.Close()
	b, err := NewApp(root, "../../scripts/compile.mjs")
	if err != nil {
		t.Fatal(err)
	}
	b.Store.Close()
	if err = os.WriteFile(filepath.Join(root, "releases", active, "index.html"), []byte("tampered"), 0600); err != nil {
		t.Fatal(err)
	}
	if c, err := NewApp(root, "../../scripts/compile.mjs"); err == nil {
		c.Store.Close()
		t.Fatal("corrupted release accepted")
	}
}
func TestPrivatePreviewDoesNotLeak(t *testing.T) {
	a := testApp(t)
	rev := draft(t, a, "", "私有标题", "秘密正文")
	ch, _ := a.Changes(ChangesInput{Revisions: []string{rev.ID}, ExpectedActive: a.Store.Read().Active})
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	token := strings.Repeat("c", 64)
	a.Auth.write(token)
	session, _ := a.Auth.Login(token)
	for _, authenticated := range []bool{false, true} {
		request := httptest.NewRequest("GET", a.Auth.adminBase+"previews/"+p.ID+contentPath(rev.Content), nil)
		if authenticated {
			request.AddCookie(&http.Cookie{Name: "jianxi_session", Value: session.ID})
		}
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, request)
		body, _ := io.ReadAll(w.Result().Body)
		if authenticated && (w.Code != 200 || !bytes.Contains(body, []byte("秘密正文"))) {
			t.Fatalf("private preview broken %d %s", w.Code, string(body))
		}
		if !authenticated && (w.Code == 200 || bytes.Contains(body, []byte("秘密正文"))) {
			t.Fatal("anonymous preview leaked")
		}
	}
}
func TestJSONSourceShape(t *testing.T) {
	a := testApp(t)
	b, _ := json.Marshal(a.Store.Read())
	if bytes.Contains(b, []byte("digest")) {
		t.Fatal("auth authority included in business state")
	}
}

func TestCoordinatorIsExclusive(t *testing.T) {
	a := testApp(t)
	if b, err := NewApp(a.Store.root, "../../scripts/compile.mjs"); err == nil {
		b.Store.Close()
		t.Fatal("second coordinator acquired same data directory")
	}
}
func TestHistoryPhysicallyPrunesPayloads(t *testing.T) {
	a := testApp(t)
	r := draft(t, a, "", "one", "old-payload-that-must-be-pruned")
	publish(t, a, r.ID)
	for _, name := range []string{"two", "three", "four", "five"} {
		r = draft(t, a, r.Content.ID, name, name)
		publish(t, a, r.ID)
	}
	state := a.Store.Read()
	if len(state.Revisions) != 4 {
		t.Fatalf("expected four stored revisions, got %d", len(state.Revisions))
	}
	data, _ := json.Marshal(state)
	if bytes.Contains(data, []byte("old-payload-that-must-be-pruned")) {
		t.Fatal("pruned payload survives in private state")
	}
	dirs, _ := os.ReadDir(filepath.Join(a.Store.root, "releases"))
	if len(dirs) != 1 {
		t.Fatal("obsolete artifacts retained")
	}
}
func TestPermanentDeleteRemovesRecoverableCopies(t *testing.T) {
	a := testApp(t)
	unrelated, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	r := draft(t, a, "", "待删除", "unique-deletion-secret")
	publish(t, a, r.ID)
	backup, err := a.Export()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = a.Delete(DeleteInput{ContentID: r.Content.ID, ConfirmContentID: r.Content.ID, ExpectedGeneration: a.Store.Read().Generation}); err == nil {
		t.Fatal("published content deleted without withdrawal")
	}
	ch, _ := a.Changes(ChangesInput{Withdraw: []string{r.Content.ID}, ExpectedActive: a.Store.Read().Active})
	p, err := a.Preview(context.Background(), ch.ID)
	if err != nil {
		t.Fatal(err)
	}
	p, _ = a.Confirm(ConfirmInput{PreviewID: p.ID, ArtifactHash: p.Hash, ChangeSetHash: ch.Hash})
	_, err = a.Publish(PublishInput{PreviewID: p.ID, ConfirmationID: p.Confirmed, ChangeSetHash: ch.Hash, ExpectedActive: ch.Base, IdempotencyKey: uid()})
	if err != nil {
		t.Fatal(err)
	}
	result, err := a.Delete(DeleteInput{ContentID: r.Content.ID, ConfirmContentID: r.Content.ID, ExpectedGeneration: a.Store.Read().Generation})
	if err != nil || result.Status != "deleted" {
		t.Fatal(result, err)
	}
	if _, err = os.Stat(filepath.Join(a.Store.root, "backups", backup)); !os.IsNotExist(err) {
		t.Fatal("backup survived permanent delete")
	}
	if _, err = os.Stat(filepath.Join(a.Store.root, "backups", unrelated)); err != nil {
		t.Fatal("unrelated backup was removed", err)
	}
	_ = filepath.WalkDir(a.Store.root, func(path string, entry os.DirEntry, err error) error {
		if err == nil && !entry.IsDir() {
			b, _ := os.ReadFile(path)
			if bytes.Contains(b, []byte("unique-deletion-secret")) {
				t.Errorf("recoverable deleted payload remains in %s", entry.Name())
			}
		}
		return nil
	})
}
func TestVideoAdapterAndSnapshotPaths(t *testing.T) {
	valid := "https://www.bilibili.com/video/BV1B7411m7LV/"
	if !strings.Contains(string(renderVideo(valid, "")), "<iframe") {
		t.Fatal("Bilibili adapter missing")
	}
	if strings.Contains(string(renderVideo(valid, "/admin/previews/x")), "<iframe") {
		t.Fatal("private preview loaded third party")
	}
	if strings.Contains(string(renderVideo("https://evil.example/video/BV1B7411m7LV/", "")), "<iframe") {
		t.Fatal("untrusted video host embedded")
	}
	a := testApp(t)
	snap := clone(*a.Store.active.Load())
	snap.Taxonomy["bad"] = Taxon{ID: "bad", Kind: "category", Name: "bad", Slug: "../../auth"}
	if validateSnapshot(snap) == nil {
		t.Fatal("unsafe snapshot path accepted")
	}
}
func TestImageFormats(t *testing.T) {
	if _, err := os.Stat("../../node_modules/sharp/package.json"); err != nil {
		t.Skip("npm ci required")
	}
	a := testApp(t)
	for _, format := range []string{"png", "jpeg", "webp", "gif", "avif"} {
		path := filepath.Join(t.TempDir(), "image."+format)
		script := "require('sharp')({create:{width:20,height:12,channels:3,background:'#222'}}).toFormat(process.argv[1]).toFile(process.argv[2])"
		cmd := exec.Command("node", "-e", script, format, path)
		cmd.Dir = "../.."
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("fixture %s %v %s", format, err, output)
		}
		b, _ := os.ReadFile(path)
		r := httptest.NewRequest("POST", "/", nil)
		asset, err := a.StoreAsset(r, AssetInput{Data: base64.StdEncoding.EncodeToString(b), Alt: "sample"})
		if err != nil || asset.Width != 20 || asset.Height != 12 || asset.MIME != "image/"+format {
			t.Fatalf("format %s %v %+v", format, err, asset)
		}
	}
	if _, err := a.StoreAsset(httptest.NewRequest("POST", "/", nil), AssetInput{Data: base64.StdEncoding.EncodeToString([]byte("<svg onload='alert(1)'></svg>"))}); err == nil {
		t.Fatal("SVG upload accepted")
	}
}
