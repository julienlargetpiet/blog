package generator

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strconv"
    "strings"
    "regexp"
    "html"

	"blog/internal/model"
)

var htmlTagRe = regexp.MustCompile(`<[^>]+>`)

func excerpt(htmlContent string, words int) string {
	// 1. Strip HTML tags
	text := htmlTagRe.ReplaceAllString(htmlContent, "")
	text = html.UnescapeString(text)

	// 2. Normalize whitespace
	fields := strings.Fields(text)
	if len(fields) <= words {
		return strings.Join(fields, " ")
	}

	// 3. Cut + ellipsis
	return strings.Join(fields[:words], " ") + "…"
}

type Generator struct {
	Articles []model.Article
	OutDir   string
}

func (g *Generator) Build() error {
	if err := os.RemoveAll(g.OutDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(g.OutDir, "articles"), 0o755); err != nil {
		return err
	}

    sort.Slice(g.Articles, func(i, j int) bool {
    	return g.Articles[i].ID > g.Articles[j].ID
    })

	if err := g.buildIndex(); err != nil {
		return err
	}

	return g.buildArticles()
}


func (g *Generator) buildArticles() error {
	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/users/article.html",
	)
	if err != nil {
		return err
	}

	for _, a := range g.Articles {
		view := model.ArticleView{
			ID:        a.ID,
			Title:     a.Title,
			HTML:      template.HTML(a.HTML), // 🔑 TRUSTED HTML
			CreatedAt: a.CreatedAt,
		}

		filename := filepath.Join(
			g.OutDir,
			"articles",
			strconv.FormatInt(a.ID, 10)+".html",
		)

		f, err := os.Create(filename)
		if err != nil {
			return err
		}

		if err := func() error {
			defer f.Close()
			return tmpl.ExecuteTemplate(f, "base", view)
		}(); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) buildIndex() error {
    funcs := template.FuncMap{
    	"mod": func(a, b int) int { return a % b },
    	"add": func(a, b int) int { return a + b },
    	"excerpt": func(html template.HTML, n int) string {
    		return excerpt(string(html), n)
    	},
    }
	
    tmpl, err := template.New("base").
		Funcs(funcs).
		ParseFiles(
			"internal/templates/base.html",
			"internal/templates/users/index.html",
		)

	if err != nil {
		return err
	}

	// 🔑 Convert DB models → view models
	views := make([]model.ArticleView, 0, len(g.Articles))
	for _, a := range g.Articles {
		views = append(views, model.ArticleView{
			ID:        a.ID,
			Title:     a.Title,
			HTML:      template.HTML(a.HTML), // trusted HTML
			CreatedAt: a.CreatedAt,
		})
	}

    sort.Slice(views, func(i, j int) bool {
		return views[i].ID > views[j].ID
	})

	filename := filepath.Join(g.OutDir, "index.html")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.ExecuteTemplate(f, "base", views)
}



