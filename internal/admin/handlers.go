package admin

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
    "os/exec"
    "io"

	"blog/internal/db"
	"blog/internal/generator"
)

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {

    if r.Method != http.MethodGet {
    	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
    	return
    }

	repo := db.ArticleRepo{DB: s.DB}

	articles, err := repo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(
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
		html := r.FormValue("html")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// 1. Update DB
		if err := repo.Update(id, title, html); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. Reload all articles (single source of truth)
		articles, err := repo.ListAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Build site
		gen := generator.Generator{
			Articles: articles,
			OutDir:   "dist",
		}

		if err := gen.Build(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := gen.BuildSitemap(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 4. Redirect back to admin
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

	if err := tmpl.ExecuteTemplate(w, "base", article); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDeleteArticle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, 
                                "/admin/delete/")

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	repo := db.ArticleRepo{DB: s.DB}

	// 1. Delete from DB
	if err := repo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 2. Reload articles
	articles, err := repo.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 3. Build site
	gen := generator.Generator{
		Articles: articles,
		OutDir:   "dist",
	}

	if err := gen.Build(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := gen.BuildSitemap(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// 4. Redirect back to admin
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func (s *Server) handleNewArticle(w http.ResponseWriter, r *http.Request) {
	repo := db.ArticleRepo{DB: s.DB}

	// -------- POST: create + build --------
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		title := r.FormValue("title")
		html := r.FormValue("html")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// 1. Insert into DB
		_, err := repo.Create(title, html)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. Reload all articles
		articles, err := repo.ListAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. Build site
		gen := generator.Generator{
			Articles: articles,
			OutDir:   "dist",
		}

		if err := gen.Build(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := gen.BuildSitemap(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// 4. Redirect back to admin list
		http.Redirect(w, r, "/admin", http.StatusSeeOther)
		return
	}

	// -------- GET: render new article page --------
	tmpl, err := template.ParseFiles(
		"internal/templates/base.html",
		"internal/templates/admin/new.html",
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "base", nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handleDumpDb(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodGet {
        http.Error(w, "bad request", http.StatusBadRequest)
        return
    }

    cmd := exec.Command(
        "mysqldump",
        "-u", "root",
        "-p m",
        "go_blog",
    )

    stdout, err := cmd.StdoutPipe()
    if err != nil {
        http.Error(w, "failed to create pipe", 500)
        return
    }

    if err := cmd.Start(); err != nil {
        http.Error(w, "failed to start dump", 500)
        return
    }

    w.Header().Set("Content-Type", "application/sql")
    w.Header().Set("Content-Disposition", "attachment; filename=\"go_blog.sql\"")

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



