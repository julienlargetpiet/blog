package generator

import (
	"html/template"
	"os"
	"path/filepath"
	"sort"
    "strings"
    "regexp"
    "html"
    "encoding/xml"
    "time"
    "fmt"

	"blog/internal/model"
	"blog/internal/utils"
    "blog/internal/db"
)

var defaultSubject = model.Subject{
	Title: "Default",
	Slug:  "default",
}

func writeFileAtomic(filename string, write func(f *os.File) error) error {
    dir := filepath.Dir(filename)

    tmp, err := os.CreateTemp(dir, filepath.Base(filename)+".tmp-*")
    if err != nil {
        return err
    }

    tmpName := tmp.Name()

    defer func() {
        if tmp != nil {
            tmp.Close()
            os.Remove(tmpName)
        }
    }()

    if err := write(tmp); err != nil {
        return err
    }

    if err := tmp.Sync(); err != nil {
        return err
    }

    // 🔥 set correct permissions BEFORE rename
    if err := tmp.Chmod(0644); err != nil {
        return err
    }

    if err := tmp.Close(); err != nil {
        return err
    }

    tmp = nil

    return os.Rename(tmpName, filename)
}

func (g *Generator) BuildSubjectMap() map[int64]model.Subject {
	m := make(map[int64]model.Subject, len(g.Subjects))
	for _, s := range g.Subjects {
		m[s.Id] = s
	}
	return m
}

func (g *Generator) BuildArticleViews() []model.ArticleView {
	subjectMap := g.BuildSubjectMap()

	views := make([]model.ArticleView, 0, len(g.Articles))

    for i := range g.Articles {
        a := &g.Articles[i]
    
        subject, ok := subjectMap[a.SubjectId]
        if !ok {
            panic(fmt.Sprintf("subject %d not found", a.SubjectId))
        }
    
        views = append(views, model.ArticleView{
            ID:        a.ID,
            Title:     a.Title,
            TitleURL:  a.TitleURL,
            SubjectId: a.SubjectId,
            Slug:      subject.Slug,
            IsPublic:  a.IsPublic,
            HTML:      template.HTML(a.HTML),
            CreatedAt: a.CreatedAt,
        })
    }

	return views
}

func (g *Generator) BuildArticleViewsForSubject(subject_id int64) []model.ArticleView {
	subjectMap := g.BuildSubjectMap()

	views := make([]model.ArticleView, 0, len(g.Articles))

    for i := range g.Articles {
        a := &g.Articles[i]
   
        if a.SubjectId != subject_id {
            continue
        }

        subject, ok := subjectMap[a.SubjectId]
        if !ok {
            panic(fmt.Sprintf("subject %d not found", a.SubjectId))
        }
    
        views = append(views, model.ArticleView{
            ID:        a.ID,
            Title:     a.Title,
            TitleURL:  a.TitleURL,
            SubjectId: a.SubjectId,
            Slug:      subject.Slug,
            IsPublic:  a.IsPublic,
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
    AuthorContent template.HTML
    ArticleRepo   db.ArticleRepo
    SubjectRepo   db.SubjectRepo
	Articles      []model.Article
    Subjects      []model.Subject
	OutDir        string
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

func (g *Generator) LocalizedBuild(title string, 
                                   subject_id int64,
                                   is_deletion bool) error {

    title_url := utils.Slugify(title)

    //if is_deletion {
    //    os.Remove("/dist/articles/" + title_url + ".html")
    //}

    subject_slug, err := g.SubjectRepo.GetSlugByID(subject_id)
    if err != nil {
        return err
    }

    sort.Slice(g.Articles, func(i, j int) bool {
    	return g.Articles[i].ID > g.Articles[j].ID
    })

	if err := g.buildIndex(); err != nil {
		return err
	}

	if err := g.buildSubject(subject_id); err != nil {
		return err
	}

    if !is_deletion {
	    return g.buildArticle(title_url, subject_slug)
    }

    return nil

}

func (g* Generator) SubjectEventBuild() error {
	
    if err := g.buildIndex(); err != nil {
		return err
	}

    return nil

}

func (g* Generator) SubjectEditBuild(subject_id int64) error {
	
    if err := g.buildIndex(); err != nil {
		return err
	}

    if err := g.buildArticlesForSubject(subject_id); err != nil {
		return err
	} 

    if err := g.buildSubjects(); err != nil {
		return err
	} 

    return nil

}

func (g *Generator) buildArticlesForSubject(subject_id int64) error {
	tmpl, err := template.ParseFiles(
		"internal/templates/base_article.html",
		"internal/templates/users/article.html",
	)
	if err != nil {
		return err
	}

	views := g.BuildArticleViewsForSubject(subject_id)

	for _, view := range views {

		filename := filepath.Join(
			g.OutDir,
			"articles",
			view.TitleURL + ".html",
		)

        if err := writeFileAtomic(filename, func(f *os.File) error {
            return tmpl.ExecuteTemplate(f, "base_article", view)
        }); err != nil {
            return err
        }
	}

	return nil
}

func (g *Generator) buildArticles() error {
	tmpl, err := template.ParseFiles(
		"internal/templates/base_article.html",
		"internal/templates/users/article.html",
	)
	if err != nil {
		return err
	}

	views := g.BuildArticleViews()

	for _, view := range views {

		filename := filepath.Join(
			g.OutDir,
			"articles",
			view.TitleURL + ".html",
		)

        if err := writeFileAtomic(filename, func(f *os.File) error {
            return tmpl.ExecuteTemplate(f, "base_article", view)
        }); err != nil {
            return err
        }
	}

	return nil
}

func (g *Generator) buildArticle(title_url, slug_val string) error {
	tmpl, err := template.ParseFiles(
		"internal/templates/base_article.html",
		"internal/templates/users/article.html",
	)
	if err != nil {
		return err
	}

    article, err := g.ArticleRepo.GetByTitleURL(title_url)

    if err != nil {
        return err
    }

	view := model.ArticleView{
            ID:        article.ID,
            Title:     article.Title,
            TitleURL:  article.TitleURL,
            SubjectId: article.SubjectId,
            Slug:      slug_val,
            IsPublic:  article.IsPublic,
            HTML:      template.HTML(article.HTML),
            CreatedAt: article.CreatedAt,
        }

	filename := filepath.Join(
		g.OutDir,
		"articles",
		view.TitleURL + ".html",
	)

    return writeFileAtomic(filename, func(f *os.File) error {
        return tmpl.ExecuteTemplate(f, "base_article", view)
    })
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

    views := g.BuildArticleViews()
    
    sort.Slice(views, func(i, j int) bool {
    	return views[i].ID > views[j].ID
    })
    
    page := IndexView{
    	Articles:      views,
    	Subjects:      g.Subjects,
    	ActiveSubject: "",
    }

	filename := filepath.Join(g.OutDir, "index.html")

    return writeFileAtomic(filename, func(f *os.File) error {
        return tmpl.ExecuteTemplate(f, "base", page)
    })

}

func (g *Generator) BuildAuthor() error {
	tmpl, err := template.New("base").
		ParseFiles(
			"internal/templates/base.html",
			"internal/templates/admin/author.html",
		)
	if err != nil {
		return err
	}

	filename := filepath.Join(g.OutDir, "author.html")

	data := struct {
		Content template.HTML
	}{
		Content: g.AuthorContent,
	}

    return writeFileAtomic(filename, func(f *os.File) error {
        return tmpl.ExecuteTemplate(f, "base", data)
    })

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

	views := g.BuildArticleViews()

	// group by subject id
	grouped := make(map[int64][]model.ArticleView)
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

        if err := writeFileAtomic(filename, func(f *os.File) error {
            return tmpl.ExecuteTemplate(f, "base", page)
        }); err != nil {
            return err
        }

	}

	return nil
}

func (g *Generator) buildSubject(subject_id int64) error {
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

	views := g.BuildArticleViews()

	// group by subject id
	grouped := make(map[int64][]model.ArticleView)
	for _, v := range views {
		grouped[v.SubjectId] = append(grouped[v.SubjectId], v)
	}

    subject, err := g.SubjectRepo.GetByID(subject_id)
    if err != nil {
        return err
    }

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

    return writeFileAtomic(filename, func(f *os.File) error {
        return tmpl.ExecuteTemplate(f, "base", page)
    })
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
            Loc:     fmt.Sprintf("%s/articles/%s.html", base, a.TitleURL),
            LastMod: a.CreatedAt.Format("2006-01-02"),
        })
    }

    for _, s := range g.Subjects {
    	urls = append(urls, URL{
    		Loc: fmt.Sprintf("%s/sub/%s.html", base, s.Slug),
    		LastMod: time.Now().Format("2006-01-02"),
    	})
    }

    urls = append(urls, URL{
    	Loc: fmt.Sprintf("%s/author.html", base),
    	LastMod: time.Now().Format("2006-01-02"),
    })

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

    return writeFileAtomic(filename, func(f *os.File) error {
        _, err := f.Write(data)
        return err
    })

}

func (g *Generator) BuildRSS() error {
    type Item struct {
        Title       string `xml:"title"`
        Link        string `xml:"link"`
        GUID        string `xml:"guid"`
        PubDate     string `xml:"pubDate"`
        Description string `xml:"description"`
    }

    type Channel struct {
        Title         string `xml:"title"`
        Link          string `xml:"link"`
        Description   string `xml:"description"`
        LastBuildDate string `xml:"lastBuildDate"`
        Items         []Item `xml:"item"`
    }

    type RSS struct {
        XMLName xml.Name `xml:"rss"`
        Version string   `xml:"version,attr"`
        Channel Channel  `xml:"channel"`
    }

    const base = "https://julienlargetpiet.tech"

    items := make([]Item, 0, len(g.Articles))

    for _, a := range g.Articles {
        if !a.IsPublic {
            continue
        }

        link := fmt.Sprintf("%s/articles/%s.html", base, a.TitleURL)

        items = append(items, Item{
            Title:   a.Title,
            Link:    link,
            GUID:    link,
            PubDate: a.CreatedAt.Format(time.RFC1123Z),
            Description: "New article published",
        })
    }

    rss := RSS{
        Version: "2.0",
        Channel: Channel{
            Title:         "Julien Larget-Piet Updates",
            Link:          base,
            Description:   "Article publication notifications.",
            LastBuildDate: time.Now().Format(time.RFC1123Z),
            Items:         items,
        },
    }

    data, err := xml.MarshalIndent(rss, "", "  ")
    if err != nil {
        return err
    }

    data = append([]byte(xml.Header), data...)

    filename := filepath.Join(g.OutDir, "rss.xml")
    return writeFileAtomic(filename, func(f *os.File) error {
    	_, err := f.Write(data)
    	return err
    })
}



