package blog

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func publicClient() *http.Client {
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }, Transport: &http.Transport{DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
		host, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if port != "443" && port != "80" {
			return nil, fail("INVALID_ENDPOINT", "只允许标准 HTTP/S 端口")
		}
		ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if ip.IP.IsLoopback() || ip.IP.IsPrivate() || ip.IP.IsUnspecified() || ip.IP.IsLinkLocalUnicast() || ip.IP.IsLinkLocalMulticast() || !ip.IP.IsGlobalUnicast() {
				return nil, fail("PRIVATE_ENDPOINT", "不能请求私有网络地址")
			}
		}
		if len(ips) == 0 {
			return nil, fail("DNS_ERROR", "域名不可解析")
		}
		d := net.Dialer{Timeout: 10 * time.Second}
		return d.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
	}}}
}
func (a *App) RunDiscovery(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.runDiscovery(ctx)
		}
	}
}
func (a *App) runDiscovery(ctx context.Context) {
	s := a.Store.Read()
	for _, j := range s.Jobs {
		next, _ := time.Parse(time.RFC3339Nano, j.NextAt)
		if (j.Status != "pending" && j.Status != "retry") || time.Now().Before(next) {
			continue
		}
		snap := a.Store.active.Load()
		status, message := "failed", ""
		if !s.Integrations.Enabled || s.Integrations.BaiduToken == "" {
			status = "not_configured"
			message = "尚未配置并启用百度推送"
		} else if s.Integrations.BaiduSite != snap.Settings.Domain {
			message = "百度验证站点与当前主域名不一致"
		} else {
			urls := []string{}
			for _, raw := range j.URLs {
				for _, r := range snap.Posts {
					if raw == snap.Settings.Domain+contentPath(r.Content) {
						urls = append(urls, raw)
						break
					}
				}
			}
			if len(urls) == 0 {
				status = "aborted"
				message = "文章已下架或网址改变"
			} else {
				client := publicClient()
				request, _ := http.NewRequestWithContext(ctx, "GET", snap.Settings.Domain+"/", nil)
				response, err := client.Do(request)
				readable := false
				if err == nil {
					readable = response.StatusCode == 200 && response.Header.Get("X-Jianxi-Release") == snap.ID
					response.Body.Close()
				}
				if !readable {
					message = "主域名尚未验证为当前公开版本；请检查域名和反向代理"
				} else if a.Store.Read().Epoch != s.Epoch || a.Store.active.Load().ID != snap.ID {
					status = "aborted"
					message = "站点在提交前已改变"
				} else {
					endpoint := "https://data.zz.baidu.com/urls?" + url.Values{"site": {s.Integrations.BaiduSite}, "token": {s.Integrations.BaiduToken}}.Encode()
					req, _ := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(strings.Join(urls, "\n")))
					req.Header.Set("Content-Type", "text/plain")
					res, err := client.Do(req)
					if err != nil {
						message = "推送网络请求失败"
					} else {
						body, _ := io.ReadAll(io.LimitReader(res.Body, 8192))
						res.Body.Close()
						var receipt struct {
							Success int `json:"success"`
							Error   int `json:"error"`
						}
						if json.Unmarshal(body, &receipt) == nil && res.StatusCode == 200 && receipt.Error == 0 && receipt.Success > 0 {
							status = "accepted"
							message = "平台已接收 URL；实际收录状态未知"
						} else {
							message = "平台未确认接收，请检查 token、验证站点和额度"
						}
					}
				}
			}
		}
		_ = a.Store.Update(func(st *State) error {
			if st.Epoch != s.Epoch {
				return nil
			}
			current, ok := st.Jobs[j.ID]
			if !ok {
				return nil
			}
			current.Attempts++
			if status == "failed" && current.Attempts < 4 {
				status = "retry"
			}
			current.Status = status
			current.Message = message
			current.NextAt = time.Now().Add(time.Duration(1<<min(current.Attempts, 8)) * time.Minute).UTC().Format(time.RFC3339Nano)
			st.Jobs[j.ID] = current
			return nil
		})
		return
	}
}
