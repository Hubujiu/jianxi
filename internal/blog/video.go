package blog

import (
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"
)

var bilibiliID = regexp.MustCompile(`^BV[0-9A-Za-z]{10}$`)

// Bilibili's documented external-player adapter; all other URLs remain links.
// Private previews deliberately make no third-party player requests.
func renderVideo(raw, prefix string) template.HTML {
	link := `<a href="` + html.EscapeString(raw) + `" rel="noopener noreferrer" target="_blank">前往视频平台观看 ↗</a>`
	u, err := url.Parse(raw)
	if err != nil || prefix != "" || u.Scheme != "https" || u.User != nil {
		return template.HTML(link)
	}
	if u.Hostname() != "www.bilibili.com" && u.Hostname() != "bilibili.com" {
		return template.HTML(link)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) != 2 || parts[0] != "video" || !bilibiliID.MatchString(parts[1]) {
		return template.HTML(link)
	}
	src := "https://player.bilibili.com/player.html?" + url.Values{"bvid": {parts[1]}, "autoplay": {"0"}, "danmaku": {"0"}, "refer": {"0"}}.Encode()
	return template.HTML(`<div class="video-embed"><iframe loading="lazy" src="` + html.EscapeString(src) + `" title="哔哩哔哩视频播放器" allow="fullscreen" referrerpolicy="no-referrer" sandbox="allow-scripts allow-same-origin allow-presentation" allowfullscreen></iframe>` + link + `</div>`)
}
