package blog

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var keywordKind = ast.NewNodeKind("JianxiKeyword")

type keywordInline struct {
	ast.BaseInline
	label, description, diagnostic string
	offset                         int
}

func (n *keywordInline) Kind() ast.NodeKind   { return keywordKind }
func (n *keywordInline) Dump(s []byte, l int) { ast.DumpHelper(n, s, l, nil, nil) }
func (n *keywordInline) Text([]byte) []byte   { return []byte(n.label) }

// Keywords are atomic plain text: Markdown, HTML and math inside them are never
// parsed. Only the delimiters and backslash have escape sequences.
type keywordParser struct{}

func (keywordParser) Trigger() []byte { return []byte{'['} }
func (keywordParser) Parse(parent ast.Node, r text.Reader, _ parser.Context) ast.Node {
	line, segment := r.PeekLine()
	if len(line) < 2 || line[0] != '[' || line[1] != '[' {
		return nil
	}
	// Let the standard parser consume an outer link/image bracket first.
	if len(line) > 2 && line[2] == '[' {
		return nil
	}
	separator, separatorWidth, end := -1, 1, -1
	for i := 2; i < len(line); i++ {
		if line[i] == '\\' && i+1 < len(line) && strings.ContainsRune("\\|]", rune(line[i+1])) {
			// GFM tables require escaping every pipe, including our separator.
			if separator < 0 && line[i+1] == '|' && parent.Kind() == extast.KindTableCell {
				separator, separatorWidth = i, 2
			}
			i++
			continue
		}
		if line[i] == '\n' || line[i] == '\r' {
			break
		}
		if line[i] == '|' && separator < 0 {
			separator = i
		}
		if line[i] == ']' && i+1 < len(line) && line[i+1] == ']' {
			end = i
			break
		}
	}
	// Ordinary [[text]] keeps its existing Markdown behavior.
	if separator < 0 {
		return nil
	}
	n := &keywordInline{offset: segment.Start}
	if end < 0 {
		n.diagnostic = "关键词描述必须在同一行使用 ]] 结束"
		r.Advance(len(line))
		return n
	}
	n.label = strings.TrimSpace(keywordText(string(line[2:separator])))
	n.description = strings.TrimSpace(keywordText(string(line[separator+separatorWidth : end])))
	r.Advance(end + 2)
	return n
}

func keywordText(s string) string {
	var out strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && i+1 < len(s) && strings.ContainsRune("\\|]", rune(s[i+1])) {
			i++
		}
		out.WriteByte(s[i])
	}
	return out.String()
}
