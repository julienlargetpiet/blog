package admin

import (
	"html/template"
	"net/http"
	"strconv"
	"strings"
    "os/exec"
    "io"
    "bytes"
    "database/sql"
    "errors"

	"blog/internal/db"
	"blog/internal/generator"
    "blog/internal/model"
)

type EditArticleView struct {
	Article  model.Article
	Subjects []model.Subject
}

func (s *Server) rebuildSite() error {
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
		Articles: articles,
		Subjects: subjects,
		OutDir:   "dist",
	}

	if err := gen.Build(); err != nil {
		return err
	}

	return gen.BuildSitemap()
}

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
    	html := r.FormValue("html")
    
    	if title == "" {
    		http.Error(w, "title is required", http.StatusBadRequest)
    		return
    	}
    
    	// 🔥 Convert subject_id to int32
    	subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 32)
    	if err != nil {
    		http.Error(w, "invalid subject id", http.StatusBadRequest)
    		return
    	}
    	subjectId := int32(subjectId64)
    
    	// 1. Update DB
    	if err := repo.Update(id, title, subjectId, html); err != nil {
    		http.Error(w, err.Error(), http.StatusInternalServerError)
    		return
    	}

        if err := s.rebuildSite(); err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError)
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

	// 1️⃣ Delete article
	if err := articleRepo.Delete(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    if err := s.rebuildSite(); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

	// 5️⃣ Redirect back to admin
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
		html := r.FormValue("html")

		if title == "" {
			http.Error(w, "title is required", http.StatusBadRequest)
			return
		}

		// 🔥 Convert subject_id
		subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 32)
		if err != nil {
			http.Error(w, "invalid subject id", http.StatusBadRequest)
			return
		}
		subjectId := int32(subjectId64)

		// 1️⃣ Insert into DB
		if _, err := articleRepo.Create(title, subjectId, html); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

        if err := s.rebuildSite(); err != nil {
        	http.Error(w, err.Error(), http.StatusInternalServerError)
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
        "mysqldump",
        "-u", "root",
        "-pm",
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

	// 1️⃣ Insert subject
	if _, err := subjectRepo.Create(name); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

    if err := s.rebuildSite(); err != nil {
    	http.Error(w, err.Error(), http.StatusInternalServerError)
    	return
    }

	// 4️⃣ Redirect
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
    	subjectId64, err := strconv.ParseInt(subjectIdStr, 10, 32)
    	if err != nil {
    		http.Error(w, "invalid subject id", http.StatusBadRequest)
    		return
    	}
    	subjectId := int32(subjectId64)

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
	id64, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil || id64 <= 0 {
		http.NotFound(w, r)
		return
	}

	id := int32(id64)
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




