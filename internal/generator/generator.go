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
    "encoding/xml"
    "time"
    "fmt"

	"blog/internal/model"
    "blog/internal/db"
)

var defaultSubject = model.Subject{
	Title: "Default",
	Slug:  "default",
}

func (g *Generator) buildSubjectMap() map[int32]model.Subject {
	m := make(map[int32]model.Subject, len(g.Subjects))
	for _, s := range g.Subjects {
		m[s.Id] = s
	}
	return m
}

func (g *Generator) buildArticleViews() []model.ArticleView {
	subjectMap := g.buildSubjectMap()

	views := make([]model.ArticleView, 0, len(g.Articles))

	for _, a := range g.Articles {

        subject, ok := subjectMap[a.SubjectId]
        if !ok {
        	panic(fmt.Sprintf("subject %d not found in subjectMap", a.SubjectId))
        }

		views = append(views, model.ArticleView{
			ID:        a.ID,
			Title:     a.Title,
			SubjectId: a.SubjectId,
			Slug:      subject.Slug,
			HTML:      template.HTML(a.HTML),
			CreatedAt: a.CreatedAt,
		})
	}

	return views
}

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
    Subjects []model.Subject
	OutDir   string
}

type IndexView struct {
	Articles []model.ArticleView
	Subjects []model.Subject
	ActiveSubject string
}

func (g *Generator) Build() error {
	if err := os.RemoveAll(g.OutDir); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(g.OutDir, "articles"), 0o755); err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Join(g.OutDir, "sub"), 0o755); err != nil {
		return err
	}

    sort.Slice(g.Articles, func(i, j int) bool {
    	return g.Articles[i].ID > g.Articles[j].ID
    })

	if err := g.buildIndex(); err != nil {
		return err
	}

	if err := g.buildSubjects(); err != nil {
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

	views := g.buildArticleViews()

	for _, view := range views {

		filename := filepath.Join(
			g.OutDir,
			"articles",
			strconv.FormatInt(view.ID, 10)+".html",
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

    views := g.buildArticleViews()
    
    sort.Slice(views, func(i, j int) bool {
    	return views[i].ID > views[j].ID
    })
    
    page := IndexView{
    	Articles:      views,
    	Subjects:      g.Subjects,
    	ActiveSubject: "",
    }

	filename := filepath.Join(g.OutDir, "index.html")
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.ExecuteTemplate(f, "base", page)
}

func (g *Generator) buildSubjects() error {
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

	views := g.buildArticleViews()

	// group by subject id
	grouped := make(map[int32][]model.ArticleView)
	for _, v := range views {
		grouped[v.SubjectId] = append(grouped[v.SubjectId], v)
	}

	for _, subject := range g.Subjects {

		filtered := grouped[subject.Id]

		page := IndexView{
			Articles:      filtered,
			Subjects:      g.Subjects,
			ActiveSubject: subject.Title,
		}

		filename := filepath.Join(
			g.OutDir,
			"sub",
			subject.Slug+".html",
		)

		f, err := os.Create(filename)
		if err != nil {
			return err
		}

		if err := func() error {
			defer f.Close()
			return tmpl.ExecuteTemplate(f, "base", page)
		}(); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) BuildSitemap() error {
    type URL struct {
        Loc     string `xml:"loc"`
        LastMod string `xml:"lastmod,omitempty"`
    }

    type URLSet struct {
        XMLName xml.Name `xml:"urlset"`
        Xmlns   string   `xml:"xmlns,attr"`
        URLs    []URL    `xml:"url"`
    }

    base := "https://julienlargetpiet.tech"

    urls := []URL{
        {
            Loc:     base + "/",
            LastMod: time.Now().Format("2006-01-02"),
        },
    }

    for _, a := range g.Articles {
        urls = append(urls, URL{
            Loc:     fmt.Sprintf("%s/articles/%d.html", base, a.ID),
            LastMod: a.CreatedAt.Format("2006-01-02"),
        })
    }

    for _, s := range g.Subjects {
    	urls = append(urls, URL{
    		Loc: fmt.Sprintf("%s/sub/%s.html", base, s.Slug),
    		LastMod: time.Now().Format("2006-01-02"),
    	})
    }

    sitemap := URLSet{
        Xmlns: "http://www.sitemaps.org/schemas/sitemap/0.9",
        URLs:  urls,
    }

    data, err := xml.MarshalIndent(sitemap, "", "  ")
    if err != nil {
        return err
    }

    data = append([]byte(xml.Header), data...)

    filename := filepath.Join(g.OutDir, "sitemap.xml")
    return os.WriteFile(filename, data, 0644)
}



