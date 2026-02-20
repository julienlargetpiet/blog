package admin

import (
	"database/sql"
	"net/http"
)

type Server struct {
	DB *sql.DB
    AdminPass string
}

func NewRouter(db *sql.DB, adminPass string) http.Handler {
	s := &Server{DB: db,
                 AdminPass: adminPass}

	mux := http.NewServeMux()
	
    mux.HandleFunc("/admin/login",  s.handleLogin)
	mux.HandleFunc("/admin/logout", s.handleLogout)

    mux.HandleFunc("/admin",           s.requireAuth(s.handleIndex))
    mux.HandleFunc("/admin/new",       s.requireAuth(s.handleNewArticle))
    mux.HandleFunc("/admin/articles/", s.requireAuth(s.handleEditArticle))
    mux.HandleFunc("/admin/delete/",   s.requireAuth(s.handleDeleteArticle))
    
    mux.HandleFunc("/admin/subjects",         s.requireAuth(s.handleSubject))
    mux.HandleFunc("/admin/subjects/delete/", s.requireAuth(s.handleDeleteSubject))
    mux.HandleFunc("/admin/subjects/edit",    s.requireAuth(s.handleEditSubject))
    mux.HandleFunc("/admin/subjects/add",     s.requireAuth(s.handleNewSubject))

    mux.HandleFunc("/admin/files",         s.requireAuth(s.handleFiles))
	mux.HandleFunc("/admin/files/delete/", s.requireAuth(s.handleDeleteFile))

    mux.HandleFunc("/admin/dump",          s.requireAuth(s.handleDumpDb))

    mux.HandleFunc("/admin/reslug",        s.requireAuth(s.handleReSlugAll))

    mux.Handle(
    	"/assets/",
    	http.StripPrefix(
    		"/assets/",
    		http.FileServer(http.Dir("assets")),
    	),
    )

	return mux
}





