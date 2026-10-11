package blog

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

const Version = "1.0.0"
const FormatVersion = 1
const RendererVersion = "jianxi-5/mcp-compiler/goldmark-1.8.6/shiki-4.5.0/github-light/katex-0.19.0/mermaid-12.1.0/cli-12.0.0/sharp-0.35.5"

type Content struct {
	ID             string             `json:"id"`
	Kind           string             `json:"kind"`
	Slug           string             `json:"slug"`
	Title          string             `json:"title"`
	Summary        string             `json:"summary"`
	Author         string             `json:"author"`
	CategoryID     string             `json:"categoryId"`
	TagIDs         []string           `json:"tagIds"`
	Markdown       string             `json:"markdown"`
	SEOTitle       string             `json:"seoTitle,omitempty"`
	SEODescription string             `json:"seoDescription,omitempty"`
	Sources        []string           `json:"sources,omitempty"`
	Videos         []string           `json:"videos,omitempty"`
	AvatarURL      string             `json:"avatarUrl,omitempty"`
	SocialLinks    []SocialLink       `json:"socialLinks,omitempty"`
	Category       *TaxonomyMetadata  `json:"category,omitempty"`
	Tags           []TaxonomyMetadata `json:"tags,omitempty"`
}

// Slugs are supplied by the author. IDs keep associations stable across renames.
type TaxonomyMetadata struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
}

type SocialLink struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}
type Heading struct {
	ID    string `json:"id"`
	Text  string `json:"text"`
	Level int    `json:"level"`
}
type Compiled struct {
	HTML     string            `json:"html"`
	Text     string            `json:"text"`
	TOC      []Heading         `json:"toc"`
	Assets   []string          `json:"assets"`
	Math     bool              `json:"math"`
	Diagrams map[string]string `json:"diagrams,omitempty"`
}
type Revision struct {
	Bundle       *RenderBundle `json:"renderBundle,omitempty"`
	RendererHash string        `json:"rendererHash"`
	ID           string        `json:"revision"`
	Content      Content       `json:"content"`
	Hash         string        `json:"sourceHash"`
	Compiled     Compiled      `json:"compiled"`
	CreatedAt    string        `json:"createdAt"`
	PublishedAt  string        `json:"publishedAt,omitempty"`
	UpdatedAt    string        `json:"contentUpdatedAt,omitempty"`
}
type Article struct {
	ID             string   `json:"id"`
	Draft          string   `json:"draft"`
	Published      string   `json:"published"`
	History        []string `json:"history"`
	FirstPublished string   `json:"firstPublished,omitempty"`
}
type WorkingCopy struct {
	ID                string  `json:"id"`
	Base              string  `json:"baseRevision"`
	Draft             string  `json:"expectedDraftRevision"`
	Content           Content `json:"content"`
	Epoch             string  `json:"epoch"`
	PublishedBase     string  `json:"publishedBase"`
	CreatedAt         string  `json:"createdAt"`
	CommittedRevision string  `json:"committedRevision,omitempty"`
}
type Upload struct {
	Bundle       *RenderBundle `json:"renderBundle,omitempty"`
	RendererHash string        `json:"rendererHash"`
	ID           string        `json:"id"`
	CopyID       string        `json:"copyId"`
	Hash         string        `json:"sourceHash"`
	Content      Content       `json:"content"`
	Status       string        `json:"status"`
	Diagnostic   string        `json:"diagnostic,omitempty"`
	Compiled     Compiled      `json:"compiled"`
	Revision     string        `json:"revision,omitempty"`
}
type Taxon struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Revision    int    `json:"revision"`
	Deleted     bool   `json:"deleted,omitempty"`
}
type Collection struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Slug        string   `json:"slug"`
	Description string   `json:"description"`
	ArticleIDs  []string `json:"articleIds"`
	Revision    int      `json:"revision"`
	Deleted     bool     `json:"deleted,omitempty"`
}
type Settings struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Domain      string `json:"domain"`
	Author      string `json:"author"`
	Footer      string `json:"footer"`
	AnalyticsID string `json:"analyticsId"`
	Revision    int    `json:"revision"`
}
type Integrations struct {
	BaiduSite  string `json:"baiduSite"`
	BaiduToken string `json:"baiduToken,omitempty"`
	Enabled    bool   `json:"enabled"`
	SEOConsole string `json:"seoConsole"`
	GEOConsole string `json:"geoConsole"`
}
type Asset struct {
	ID     string `json:"id"`
	MIME   string `json:"mime"`
	Ext    string `json:"ext"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Alt    string `json:"alt"`
	Hash   string `json:"hash"`
}
type ChangeSet struct {
	ID              string            `json:"id"`
	Revisions       []string          `json:"revisions"`
	Taxonomy        []string          `json:"taxonomy"`
	Collections     []string          `json:"collections"`
	IncludeSettings bool              `json:"includeSettings"`
	Withdraw        []string          `json:"withdraw"`
	Base            string            `json:"baseRelease"`
	Epoch           string            `json:"epoch"`
	Hash            string            `json:"hash"`
	Bindings        map[string]string `json:"bindings"`
	URLs            []string          `json:"urls"`
}
type Snapshot struct {
	ID          string                `json:"id"`
	Settings    Settings              `json:"settings"`
	Posts       map[string]Revision   `json:"posts"`
	Taxonomy    map[string]Taxon      `json:"taxonomy"`
	Collections map[string]Collection `json:"collections"`
	Redirects   map[string]string     `json:"redirects"`
	Assets      map[string]Asset      `json:"assets"`
}
type Preview struct {
	ID        string   `json:"id"`
	ChangeSet string   `json:"changeSetId"`
	Hash      string   `json:"artifactHash"`
	BuildHash string   `json:"buildHash"`
	Expires   string   `json:"expires"`
	Snapshot  Snapshot `json:"snapshot"`
	Confirmed string   `json:"confirmationId,omitempty"`
}
type Event struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Time    string `json:"time"`
	Release string `json:"release,omitempty"`
}
type DiscoveryJob struct {
	ID       string   `json:"id"`
	Release  string   `json:"release"`
	URLs     []string `json:"urls"`
	Status   string   `json:"status"`
	Attempts int      `json:"attempts"`
	NextAt   string   `json:"nextAt"`
	Message  string   `json:"message"`
}
type Report struct {
	ID         string      `json:"id"`
	Version    int         `json:"version"`
	Kind       string      `json:"kind"`
	Source     string      `json:"source"`
	ObservedAt string      `json:"observedAt"`
	Definition string      `json:"definition"`
	Rows       []ReportRow `json:"rows"`
}
type ReportRow struct {
	Platform string  `json:"platform"`
	Query    string  `json:"query"`
	URL      string  `json:"url"`
	Value    float64 `json:"value"`
	Evidence string  `json:"evidence"`
	Status   string  `json:"status"`
}
type Idempotency struct {
	Hash   string          `json:"hash"`
	Result json.RawMessage `json:"result"`
}
type State struct {
	CleanupPending bool                    `json:"cleanupPending,omitempty"`
	CleanupBackups []string                `json:"cleanupBackups,omitempty"`
	CleanupAssets  []string                `json:"cleanupAssets,omitempty"`
	Version        int                     `json:"version"`
	Generation     int64                   `json:"generation"`
	Epoch          string                  `json:"epoch"`
	Active         string                  `json:"activeRelease"`
	ActiveHash     string                  `json:"activeHash"`
	Settings       Settings                `json:"settings"`
	Integrations   Integrations            `json:"integrations"`
	Articles       map[string]Article      `json:"articles"`
	Revisions      map[string]Revision     `json:"revisions"`
	Copies         map[string]WorkingCopy  `json:"copies"`
	Uploads        map[string]Upload       `json:"uploads"`
	Taxonomy       map[string]Taxon        `json:"taxonomy"`
	Collections    map[string]Collection   `json:"collections"`
	Assets         map[string]Asset        `json:"assets"`
	Changes        map[string]ChangeSet    `json:"changes"`
	Previews       map[string]Preview      `json:"previews"`
	Events         []Event                 `json:"events"`
	Jobs           map[string]DiscoveryJob `json:"jobs"`
	Reports        map[string]Report       `json:"reports"`
	Keys           map[string]Idempotency  `json:"idempotency"`
}

func initialState() State {
	return State{Version: 1, Epoch: uid(), Settings: Settings{Name: "间隙", Description: "写一点技术，留一些观察。让想法，慢慢生长。", Domain: "http://localhost:8080", Author: "狐不九", Footer: "保持好奇，容许留白。", Revision: 1}, Articles: map[string]Article{}, Revisions: map[string]Revision{}, Copies: map[string]WorkingCopy{}, Uploads: map[string]Upload{}, Taxonomy: map[string]Taxon{}, Collections: map[string]Collection{}, Assets: map[string]Asset{}, Changes: map[string]ChangeSet{}, Previews: map[string]Preview{}, Jobs: map[string]DiscoveryJob{}, Reports: map[string]Report{}, Keys: map[string]Idempotency{}}
}
func uid() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
func digest(v any) string            { b, _ := json.Marshal(v); return hashBytes(b) }
func hashBytes(b []byte) string      { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }
func now() string                    { return time.Now().UTC().Format(time.RFC3339Nano) }
func clone[T any](v T) T             { b, _ := json.Marshal(v); var out T; _ = json.Unmarshal(b, &out); return out }
func fail(code, detail string) error { return fmt.Errorf("%s: %s", code, detail) }

var slugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func contentPath(c Content) string {
	if c.Kind == "about" {
		return "/about/"
	}
	return "/posts/" + c.Slug + "/"
}
func (s *State) event(kind, message string) {
	s.Events = append(s.Events, Event{ID: uid(), Kind: kind, Message: message, Time: now(), Release: s.Active})
	if len(s.Events) > 500 {
		s.Events = s.Events[len(s.Events)-500:]
	}
}
