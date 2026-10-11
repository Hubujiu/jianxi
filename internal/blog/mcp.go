package blog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type bridge struct {
	base, token string
	compiler    *Compiler
	client      *http.Client
}

func (b bridge) call(ctx context.Context, op string, in, out any) error {
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, b.base+"/_control/"+op, requestJSON(in))
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "Bearer "+b.token)
	r.Header.Set("Content-Type", "application/json")
	response, err := b.client.Do(r)
	if err != nil {
		return fmt.Errorf("无法连接博客服务，请检查 JIANXI_URL")
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil {
		return err
	}
	if len(body) > 8<<20 {
		return fmt.Errorf("服务响应超过 8 MiB")
	}
	if response.StatusCode != 200 {
		return fmt.Errorf("%s", strings.TrimSpace(string(body)))
	}
	return json.Unmarshal(body, out)
}
func registerLocalTool[I any](server *mcp.Server, name, description string, fn func(context.Context, I) (any, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, req *mcp.CallToolRequest, in I) (*mcp.CallToolResult, map[string]any, error) {
		result, err := fn(ctx, in)
		if err != nil {
			return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}}}, nil, nil
		}
		return nil, map[string]any{"result": result}, nil
	})
}
func registerTool[I any](server *mcp.Server, b bridge, name, op, description string) {
	registerLocalTool(server, name, description, func(ctx context.Context, in I) (any, error) {
		var result any
		err := b.call(ctx, op, in, &result)
		return result, err
	})
}
func RunMCP(ctx context.Context, renderer string) error {
	base := strings.TrimRight(os.Getenv("JIANXI_URL"), "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	u, err := url.Parse(base)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" || u.Path != "" || (u.Scheme != "https" && !(u.Scheme == "http" && (u.Hostname() == "127.0.0.1" || u.Hostname() == "localhost" || u.Hostname() == "::1"))) {
		return fmt.Errorf("JIANXI_URL 必须为 HTTPS，或本机 loopback HTTP 地址")
	}
	token := os.Getenv("JIANXI_TOKEN")
	if len(token) < 32 {
		return fmt.Errorf("请在 MCP 客户端环境中配置 JIANXI_TOKEN")
	}
	b := bridge{base: base, token: token, compiler: NewCompiler(renderer), client: &http.Client{Timeout: 5 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
	server := mcp.NewServer(&mcp.Implementation{Name: "jianxi", Version: Version}, nil)
	registerTool[struct{}](server, b, "blog_content_rules", "rules", "写作前先读取此工具：返回完整 syntax 语法清单、逐项示例与限制，包含标题、正文样式、代码、列表、引用、表格、链接、图片、脚注、关键词描述、LaTeX、八类 Mermaid 图和转义规则，以及元数据与发布流程。先获取副本，上传解析后提交草稿；必须展示预览并取得用户确认才调用确认和发布。")
	registerTool[ListInput](server, b, "blog_search", "list", "分页查询私有内容；published 只查当前发布文章，draft 查草稿。")
	registerTool[GetInput](server, b, "blog_get", "get", "读取文章或关于我的指定修订与源文。")
	registerTool[CopyInput](server, b, "blog_working_copy", "working-copy", "获取独立工作副本。新建不传 contentId；关于我使用 kind=about。保存返回的 epoch 与 expectedDraftRevision。")
	registerLocalTool(server, "blog_upload_content", "上传前先调用 blog_content_rules，读取完整语法示例与限制。本地固定编译器编译受限 Markdown、公式、代码和图表后上传；支持 [[关键词|纯文本描述]]（同一行，关键词最多 80 字，描述最多 2000 字；不能嵌套在链接或图片中）。返回已验证的 parsed 上传，不修改正式草稿。必须明确 slug 与分类标签 name/slug，不接收 AI 编写的 HTML。", b.upload)
	registerLocalTool(server, "blog_parse_upload", "校验已编译上传；旧的源文上传在本地编译后提交服务端校验。status=parsed 才可提交草稿。", b.parseUpload)
	registerTool[CommitInput](server, b, "blog_commit_draft", "commit", "CAS 提交解析通过的上传。冲突时保留副本，需重新读取并合并，不自动覆盖。")
	registerTool[struct{}](server, b, "blog_taxonomy_list", "taxonomy-list", "读取分类和标签候选，包括稳定 ID 和修订号。")
	registerTool[TaxonomyInput](server, b, "blog_taxonomy", "taxonomy", "以显式 name、slug 创建或修改分类/标签候选，slug 来自作者元数据，不自动生成；修改需 CAS，公开变更必须进入明确清单。删除前必须解除保留修订关联。")
	registerTool[CollectionInput](server, b, "blog_collection", "collection", "创建或修改有序专栏候选，至少两篇文章，随后明确选择进入发布清单。")
	registerTool[struct{}](server, b, "blog_settings_get", "settings-get", "读取站点设置和脱敏服务状态，不返回密钥。")
	registerTool[SettingsInput](server, b, "blog_settings_update", "settings", "保存公开配置草稿，必须预览发布。")
	registerTool[ChangesInput](server, b, "blog_changeset", "changeset", "创建明确的发布清单；选择文章修订、分类标签、专栏、设置或下架 ID，不能夹带其他草稿。")
	registerTool[IDInput](server, b, "blog_preview", "preview", "构建清单 ID 的私有预览。向用户展示链接、变更范围和 hash，再等待明确确认。")
	registerTool[ConfirmInput](server, b, "blog_confirm_preview", "confirm", "仅在用户明确批准此预览后建立绑定 hash 的确认记录。同 token 全权限，此步骤是操作约定。")
	registerTool[PublishInput](server, b, "blog_publish", "publish", "发布已确认预览；要求正确基准、确认 ID、清单 hash 和幂等键；失败不替换公开站点。")
	registerTool[IDInput](server, b, "blog_history", "history", "读取当前与最多三个保留的历史版本。")
	registerTool[DeleteInput](server, b, "blog_delete", "delete", "永久物理删除已下架内容。必须取得用户明确批准，先预览确认下架，再使用内容 ID 双重确认与当前数据代次。清理其所有副本、历史、独占资源和系统旧制品/备份；外部备份不受控。")
	registerLocalTool(server, "blog_rollback", "从保留历史创建回滚修订；旧编译器制品先在本地重编译，不覆盖草稿，仍须清单、预览、确认和发布。", b.rollback)
	registerLocalTool(server, "blog_store_asset", "本地 Sharp 校验 PNG/JPEG/WebP/AVIF/GIF，再上传供服务端独立校验格式、尺寸和像素上限；base64 最大 10 MiB。", b.asset)
	registerTool[IntegrationInput](server, b, "blog_integrations_update", "integrations", "保存百度推送及 SEO/GEO 服务入口。token 留空保留，clearToken 显式清除。")
	registerTool[Report](server, b, "blog_import_report", "report", "导入 version=1 的 SEO/GEO 标准 JSON 报告；保留来源、时间、指标定义与证据。")
	registerTool[ReportCSVInput](server, b, "blog_import_report_csv", "report-csv", "导入标准 CSV 报告。表头 platform,query,url,value,evidence；须另传来源、类型、时间与指标定义。")
	registerTool[struct{}](server, b, "blog_list_assets", "asset-list", "列出私有媒体资产、真实类型、尺寸与替代文本。")
	registerTool[struct{}](server, b, "blog_status", "status", "查询实例代次、当前发布版本、操作记录与推送任务。")
	registerTool[IDInput](server, b, "blog_discovery_retry", "retry", "将失败或未配置的推送任务重新加入队列；平台接收不等于收录。")
	registerTool[struct{}](server, b, "blog_backup", "backup", "生成完整敏感备份，返回需要后台鉴权的下载地址；包含第三方凭据，不包含管理密钥。")
	registerLocalTool(server, "blog_recompile_revision", "使用当前本地编译器重建指定修订的缓存；保留源文、ID、发布时间和当前草稿，不自动上线。返回修订 ID，随后通过明确清单预览、确认和发布。", b.recompile)
	registerLocalTool(server, "blog_prepare_restore", "读取指定本地 ZIP 备份，离线重编译全部修订并上传恢复候选；不激活站点。展示返回的预览和 hash，等待用户明确批准。", b.prepareRestore)
	registerTool[RestoreInput](server, b, "blog_restore_backup", "restore", "仅在用户明确批准恢复预览后，用候选 ID 与 artifactHash 激活完整恢复。替换业务数据，保留管理密钥。")
	return server.Run(ctx, &mcp.StdioTransport{})
}
