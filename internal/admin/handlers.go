package admin

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
    "os/exec"
    "os"
    "path/filepath"
    "sort"
    "io"
    "bytes"
    "database/sql"
    "errors"
    "fmt"

	"blog/internal/db"
	"blog/internal/generator"
    "blog/internal/model"
)

type EditArticleView struct {
	Article  model.Article
	Subjects []model.Subject
}

func (s *Server) listThemes() ([]string, error) {
	base := "/var/www/go_blog/assets/css/themes"

	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}

	var themes []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".css") {
			name := strings.TrimSuffix(e.Name(), ".css")
			themes = append(themes, name)
		}
	}

	sort.Strings(themes)
	return themes, nil
}

func (s *Server) currentTheme() string {
	link := "/var/www/go_blog/assets/css/theme.css"

	target, err := os.Readlink(link)
	if err != nil {
		return ""
	}

	base := filepath.Base(target)
	return strings.TrimSuffix(base, ".css")
}

func (s *Server) applyTheme(name string) error {
	baseCSS := "/var/www/go_blog/assets/css"
	baseFavicon := "/var/www/go_blog/assets"

	// ---- CSS ----
	cssTarget := filepath.Join(baseCSS, "themes", name+".css")
	cssLink := filepath.Join(baseCSS, "theme.css")
	cssTmp := cssLink + ".tmp"

	// ---- Favicon ----
	favTarget := filepath.Join(baseFavicon, "favicons", name+".svg")
	favLink := filepath.Join(baseFavicon, "favicon.svg")
	favTmp := favLink + ".tmp"

	// Validate existence
	if _, err := os.Stat(cssTarget); err != nil {
		return err
	}
	if _, err := os.Stat(favTarget); err != nil {
		return err
	}

	// --- Swap CSS symlink ---
	os.Remove(cssTmp)
	if err := os.Symlink(cssTarget, cssTmp); err != nil {
		return err
	}
	if err := os.Rename(cssTmp, cssLink); err != nil {
		return err
	}

	// --- Swap favicon symlink ---
	os.Remove(favTmp)
	if err := os.Symlink(favTarget, favTmp); err != nil {
		return err
	}
	if err := os.Rename(favTmp, favLink); err != nil {
		return err
	}

	return nil
}

func (s *Server) rebuildSite() error {
	articleRepo := db.ArticleRepo{DB: s.DB}
	subjectRepo := db.SubjectRepo{DB: s.DB}
	authorRepo  := db.AuthorRepo{DB: s.DB}

	articles, err := articleRepo.ListAll()
	if err != nil {
		return err
	}

	subjects, err := subjectRepo.ListAll()
	if err != nil {
		return err
	}

	content, err := authorRepo.GetContent()
	if err != nil {
		return err
	}

	gen := generator.Generator{
        AuthorContent: template.HTML(content),
        ArticleRepo: articleRepo,
        SubjectRepo: subjectRepo,
		Articles: articles,
		Subjects: subjects,
		OutDir:   "dist",
	}

	if err := gen.Build(); err != nil {
		return err
	}

    err = gen.BuildRSS()
    if err != nil {
        return err
    }

    err = gen.BuildAuthor()
    if err != nil {
        return err
    }

	return gen.BuildSitemap()
}

func (s *Server) rebuildSiteLocalize(title string, 
                                     subject_id int64,
                                     sitemap_build bool,
                                     is_deletion bool) error {
	articleRepo := db.ArticleRepo{DB: s.DB}
	subjectRepo := db.SubjectRepo{DB: s.DB}

	articles, err := articleRepo.ListAll()
	if err != nil {
		return err
	}

	subjects, err := subjectRepo.ListAll()
	if err != nil {
		return err
	}

	gen := generator.Generator{
        AuthorContent: template.HTML(""),
        ArticleRepo: articleRepo,
        SubjectRepo: subjectRepo,
		Articles: articles,
		Subjects: subjects,
		OutDir:   "dist",
	}

	if err := gen.LocalizedBuild(title, subject_id, is_deletion); err != nil {
		return err
	}

    if sitemap_build {

        err = gen.BuildRSS()
        if err != nil {
            return err
        }

	    return gen.BuildSitemap()
    }

    return nil

}

func (s *Server) rebuildAuthor() error {

	articleRepo := db.ArticleRepo{DB: s.DB}
	subjectRepo := db.SubjectRepo{DB: s.DB}
	authorRepo  := db.AuthorRepo{DB: s.DB}

	content, err := authorRepo.GetContent()
	if err != nil {
		return err
	}

	gen := generator.Generator{
        AuthorContent: template.HTML(content),
        ArticleRepo:   articleRepo,
        SubjectRepo:   subjectRepo,
		Articles:      []model.Article{},
		Subjects:      []model.Subject{},
		OutDir:        "dist",
	}

	if err := gen.BuildAuthor(); err != nil {
		return err
	}

    return nil

}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodGet {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
    }

    articleRepo := db.ArticleRepo{DB: s.DB}

	articles, err := articleRepo.ListAll()
	if err != nil {
    	http.Error(w, "error occured in articleRepo.ListAll()", http.StatusBadRequest)
		return
	}

    tmpl, err := template.New("base").
        ParseFiles(
            "internal/templates/base.html",
            "internal/templates/admin/index.html",
        )

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", articles); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleAuthor(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodGet {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
    }

    authorRepo := db.AuthorRepo{DB: s.DB}

	content, err := authorRepo.GetContent()
	if err != nil {
    	http.Error(w, "error occured in authorRepo.GetContent()", http.StatusBadRequest)
		return
	}

    data := struct {
		Content string
	}{
		Content: content,
	}

    tmpl, err := template.New("base").
        ParseFiles(
            "internal/templates/base.html",
            "internal/templates/admin/edit_author.html",
        )

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleUpdateAuthor(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodPost {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
        return
    }

    repo := db.AuthorRepo{DB: s.DB}

    if err := r.ParseForm(); err != nil {
    	http.Error(w, err.Error(), http.StatusBadRequest)
    	return
    }
    
    html := r.FormValue("html")
    
    if err := repo.Update(html); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

    if err := s.rebuildAuthor(); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }
    
    http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleEditArticle(w http.ResponseWriter, r *http.Request) {
	idStr := strings.TrimPrefix(r.URL.Path, "/admin/articles/")
	if idStr == "" {
		http.NotFound(w, r)
		return
	}

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	repo := db.ArticleRepo{DB: s.DB}

	// -------- POST: save + build --------

    if r.Method == http.MethodPost {
    	if err := r.ParseForm(); err != nil {
    		http.Error(w, err.Error(), http.StatusBadRequest)
    		return
    	}
    
    	title := r.FormValue("title")
    	subjectIdStr := r.FormValue("subject_id")
    	isPublicStr := r.FormValue("is_public")
    	html := r.FormValue("html")
    
    	if title == "" {
    		http.Error(w, "title is required", http.StatusBadRequest)
    		return
    	}
    
    	subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 64)
    	if err != nil {
    		http.Error(w, "invalid subject id", http.StatusBadRequest)
    		return
    	}
    	subjectId := int64(subjectId64)
   
        isPublic, err := strconv.ParseBool(isPublicStr)
        if err != nil {
			http.Error(w, "invalid visibility", http.StatusBadRequest)
            return
        }

        exists, err := repo.ExistsByTitle(title, id)
        if err != nil {
	    	http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        if exists {
	    	http.Error(w, "Article Title already exists", http.StatusConflict)
            return
        }

    	// 1. Update DB
    	if err := repo.Update(id, title, subjectId, isPublic, html); err != nil {
    		http.Error(w, err.Error(), http.StatusInternalServerError)
    		return
    	}

        if err := s.rebuildSiteLocalize(title, subjectId, false, false); err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError)
        	return
        }
   
        if r.Header.Get("X-Statix-Token") != "" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("article edited\n"))
            return
        }

    	http.Redirect(w, r, "/admin", http.StatusSeeOther)
    	return
    }

	// -------- GET: render edit page --------
	article, err := repo.GetByID(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/edit.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    subjectRepo := db.SubjectRepo{DB: s.DB}
    subjects, err := subjectRepo.ListAll()
    if err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }
    
    data := EditArticleView{
    	Article:  article,
    	Subjects: subjects,
    }
    
    if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

}

func (s *Server) handleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/admin/delete/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	articleRepo := db.ArticleRepo{DB: s.DB}

    article, err := articleRepo.GetByID(id)
    if err != nil {
        if errors.Is(err, sql.ErrNoRows) {
            http.NotFound(w, r)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if err := articleRepo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    if err := s.rebuildSiteLocalize(article.Title, article.SubjectId, true, true); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleNewArticle(w http.ResponseWriter, r *http.Request) {
	articleRepo := db.ArticleRepo{DB: s.DB}
	subjectRepo := db.SubjectRepo{DB: s.DB}

	// -------- POST: create + rebuild --------
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		subjectIdStr := r.FormValue("subject_id")
		isPublicStr := r.FormValue("is_public")
		html := r.FormValue("html")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// 🔥 Convert subject_id
		subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 64)
		if err != nil {
			http.Error(w, "invalid subject id", http.StatusBadRequest)
			return
		}
		subjectId := int64(subjectId64)

        isPublic, err := strconv.ParseBool(isPublicStr)
        if err != nil {
			http.Error(w, "invalid visibility", http.StatusBadRequest)
            return
        }

        exists, err := articleRepo.ExistsByTitleRaw(title)
        if err != nil {
	    	http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        if exists {
	    	http.Error(w, "Article Title already exists", http.StatusConflict)
            return
        }

		// 1️⃣ Insert into DB
		if _, err := articleRepo.Create(title, subjectId, isPublic, html); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

        if err := s.rebuildSiteLocalize(title, subjectId, true, false); err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError)
        	return
        }

        if r.Header.Get("X-Statix-Token") != "" {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("article published\n"))
            return
        }

		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	// -------- GET: render form --------
	subjects, err := subjectRepo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/new.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// You may want to pass subjects to template for the <select>
	data := struct {
		Subjects []model.Subject
	}{
		Subjects: subjects,
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDumpDb(w http.ResponseWriter, r *http.Request) {

    cmd := exec.Command(
        "/usr/bin/mysqldump",
        "-u", "goblog",
        "-pm",
        "go_blog",
    )
    
    stderr, err := cmd.StderrPipe()
    if err != nil {
        http.Error(w, "failed to create stderr pipe", 500)
        return
    }
    
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        http.Error(w, "failed to create stdout pipe", 500)
        return
    }
    
    if err := cmd.Start(); err != nil {
        http.Error(w, "failed to start dump", 500)
        return
    }
    
    w.Header().Set("Content-Type", "application/sql")
    w.Header().Set("Content-Disposition", "attachment; filename=\"go_blog.sql\"")
    
    go io.Copy(io.Discard, stderr) // consume stderr to avoid blocking
    
    io.Copy(w, stdout)
    
    cmd.Wait()

}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		if r.ParseForm() != nil {
			http.Error(w, "bad request", 400)
			return
		}

		if r.FormValue("password") == s.AdminPass {
			setSession(w, s.AdminPass)
			http.Redirect(w, r, "/admin", http.StatusSeeOther)
			return
		}

		http.Error(w, "invalid password", http.StatusUnauthorized)
		return
	}

	tmpl, _ := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/login.html",
	)
	tmpl.ExecuteTemplate(w, "base", nil)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	clearSession(w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

func (s *Server) handleNewSubject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	name := r.FormValue("subject")
	if name == "" {
		http.Error(w, "subject is required", http.StatusBadRequest)
		return
	}

	subjectRepo := db.SubjectRepo{DB: s.DB}

    exists, err := subjectRepo.ExistsByNameRaw(name)
    if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    if exists {
		http.Error(w, "Subject Name already exists", http.StatusConflict)
        return
    }

	if _, err := subjectRepo.Create(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/subjects", http.StatusSeeOther)
}

func (s *Server) handleEditSubject(w http.ResponseWriter, r *http.Request) {

    subjectRepo := db.SubjectRepo{DB: s.DB}

	if r.Method == http.MethodPost {

	    if err := r.ParseForm(); err != nil {
	    	http.Error(w, err.Error(), http.StatusBadRequest)
	    	return
	    }

	    name := r.FormValue("subject")
	    if name == "" {
	    	http.Error(w, "subject is required", http.StatusBadRequest)
	    	return
	    }

        subjectIdStr := r.FormValue("subject_id")
    	subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 64)
    	if err != nil {
    		http.Error(w, "invalid subject id", http.StatusBadRequest)
    		return
    	}
    	subjectId := int64(subjectId64)

        exists, err := subjectRepo.ExistsByName(name, subjectId)
        if err != nil {
	    	http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        if exists {
	    	http.Error(w, "Subject Name already exists", http.StatusConflict)
            return
        }

	    // 1️⃣ Insert subject
	    if err := subjectRepo.Update(subjectId, name); err != nil {
	    	http.Error(w, err.Error(), http.StatusInternalServerError)
	    	return
	    }

        if err := s.rebuildSite(); err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError)
        	return
        }

	    // 4️⃣ Redirect
	    http.Redirect(w, r, "/admin/subjects", http.StatusSeeOther)

        return
    }

    subjects, err := subjectRepo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    data := struct {
		Subjects []model.Subject
	}{
		Subjects: subjects,
	}

	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/edit_subject.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDeleteSubject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/admin/subjects/delete/")
	id64, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id64 <= 0 {
		http.NotFound(w, r)
		return
	}

	id := int64(id64)
	subjectRepo := db.SubjectRepo{DB: s.DB}

	subject, err := subjectRepo.GetByID(id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	if subject.Slug == "default" {
		http.Error(w, "cannot delete default subject", http.StatusBadRequest)
		return
	}

	if err := subjectRepo.Delete(id); err != nil {
		http.Error(w, "cannot delete subject with existing articles", http.StatusConflict)
		return
	}

	if err := s.rebuildSite(); err != nil {
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/subjects", http.StatusSeeOther)
}

func (s *Server) handleSubject(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodGet {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
    }

	repo := db.SubjectRepo{DB: s.DB}

	subjects, err := repo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/subjects.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    var buf bytes.Buffer
    
    if err := tmpl.ExecuteTemplate(&buf, "base", subjects); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Write(buf.Bytes())

}

func (s *Server) handleReSlugAll(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodPost {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
    }

	articleRepo := db.ArticleRepo{DB: s.DB}
    
    err := articleRepo.ReslugAll()
	if err != nil {
        http.Error(w, "internal server error: ReslugAll", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin", http.StatusSeeOther)

}

func (s *Server) handleBuildAll(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

    if err := s.rebuildSite(); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

	// 5️⃣ Redirect back to admin
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleCustomTheme(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "bad request", 400)
			return
		}

		selected := r.FormValue("theme")

		themes, err := s.listThemes()
		if err != nil {
			http.Error(w, "failed to list themes", 500)
			return
		}

		valid := false
		for _, t := range themes {
			if t == selected {
				valid = true
				break
			}
		}

		if !valid {
			http.Error(w, "invalid theme", 400)
			return
		}

		if err := s.applyTheme(selected); err != nil {
			http.Error(w, "failed to apply theme", 500)
			return
		}

		http.Redirect(w, r, "/admin/theme", http.StatusSeeOther)
		return
	}

	themes, err := s.listThemes()
	if err != nil {
		http.Error(w, "failed to load themes", 500)
		return
	}

	data := map[string]interface{}{
		"Themes":       themes,
		"CurrentTheme": s.currentTheme(),
	}

	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/custom_theme.html",
	)
	if err != nil {
		http.Error(w, "template error", 500)
		return
	}

	tmpl.ExecuteTemplate(w, "base", data)
}

func (s *Server) handleRequestArticles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repo := db.ArticleRepo{DB: s.DB}

	articles, err := repo.ListIDAndTitle()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	for _, a := range articles {
		fmt.Fprintf(w, "%d\t%s\n", a.ID, a.Title)
	}
}



