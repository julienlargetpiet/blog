package mdtostatix

import (
    "html"
    "strings"
    "bytes"

    "github.com/yuin/goldmark"
    "github.com/yuin/goldmark/ast"
    "github.com/yuin/goldmark/extension"
    extast "github.com/yuin/goldmark/extension/ast"
    "github.com/yuin/goldmark/parser"
    "github.com/yuin/goldmark/renderer"
    gmhtml "github.com/yuin/goldmark/renderer/html"
    "github.com/yuin/goldmark/util"
)

func MarkdownToStatixHTML(md string) (string, error) {
	var buf bytes.Buffer

    mdEngine := goldmark.New(
        goldmark.WithExtensions(
            extension.GFM, // Table, checkboxes and more...
        ),
        goldmark.WithParserOptions(
            parser.WithAutoHeadingID(),
        ),
        goldmark.WithRendererOptions(
            gmhtml.WithUnsafe(),
        ),
        goldmark.WithRendererOptions(
            renderer.WithNodeRenderers(
                util.Prioritized(&statixRenderer{}, 0), // run before default HTML handlers
            ),
        ),
    )    
    
    if err := mdEngine.Convert([]byte(md), &buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

type statixRenderer struct {
}

// in fact never used, just to satisfy the interface
func (r *statixRenderer) AddOptions(...renderer.Option) {}

func (r *statixRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
  // custom functions -> intercept code blocks and tables

  reg.Register(ast.KindFencedCodeBlock, r.renderFencedCodeBlock)
  reg.Register(extast.KindTable, r.renderTable)

  // everything else falls back to base renderer
}

func (r *statixRenderer) renderFencedCodeBlock(w util.BufWriter, 
                                               source []byte, 
                                               node ast.Node, 
                                               entering bool) (ast.WalkStatus, error) {
  if !entering { // quiting the node -> do nothing
    return ast.WalkContinue, nil
  }

  // if we entering the node we do some work

  // Safely assert that this node is a *ast.FencedCodeBlock.
  // If not, exit without rendering.
  n, ok := node.(*ast.FencedCodeBlock)
  if !ok {
      return ast.WalkContinue, nil
  }

  lang := strings.TrimSpace(string(n.Language(source)))
  if lang == "" {
    lang = "text"
  }

  // Extract raw code content exactly
  var code strings.Builder
  lines := n.Lines()
  for i := 0; i < lines.Len(); i++ {
    seg := lines.At(i)
    code.Write(seg.Value(source))
  }

  // IMPORTANT: do NOT escape inside <code> if you want raw text safely displayed.
  // For HTML safety, escape. Prism still works.
  codeEsc := html.EscapeString(code.String())

  _, _ = w.WriteString(`<div class="code-block">` + "\n")
  _, _ = w.WriteString(`  <pre>` + "\n")
  _, _ = w.WriteString(`  <button class="copy-btn">Copy</button>` + "\n")
  _, _ = w.WriteString(`  <code class="language-` + html.EscapeString(lang) + `">` + "\n")
  _, _ = w.WriteString(codeEsc)
  if !strings.HasSuffix(codeEsc, "\n") {
    _, _ = w.WriteString("\n")
  }
  _, _ = w.WriteString(`  </code>` + "\n")
  _, _ = w.WriteString(`  </pre>` + "\n")
  _, _ = w.WriteString(`</div>` + "\n")

  return ast.WalkSkipChildren, nil
}

func (r *statixRenderer) renderTable(
	w util.BufWriter,
	source []byte,
	node ast.Node,
	entering bool,
) (ast.WalkStatus, error) {

	if entering {

		_, _ = w.WriteString(`<section class="form-section">` + "\n\n")

		_, _ = w.WriteString(`<div class="admin-table-scroll-top">` + "\n")
		_, _ = w.WriteString(`  <div class="admin-table-scroll-inner"></div>` + "\n")
		_, _ = w.WriteString(`</div>` + "\n\n")

		_, _ = w.WriteString(`<div class="admin-table-wrapper">` + "\n")
		_, _ = w.WriteString(`<table class="admin-table">` + "\n")

	} else {

		_, _ = w.WriteString(`</table>` + "\n")
		_, _ = w.WriteString(`</div>` + "\n")
		_, _ = w.WriteString(`</section>` + "\n")
	}

	return ast.WalkContinue, nil
}



