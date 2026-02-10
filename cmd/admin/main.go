package main

import (
	"log"
	"net/http"

	"blog/internal/admin"
	"blog/internal/config"
	"blog/internal/db"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	router := admin.NewRouter(conn, cfg.AdminPass)

	log.Printf("admin listening on %s\n", cfg.AdminAddr)
	if err := http.ListenAndServe(cfg.AdminAddr, router); err != nil {
		log.Fatal(err)
	}
}
