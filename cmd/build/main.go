package main

import (
	"log"

	"blog/internal/config"
	"blog/internal/db"
	"blog/internal/generator"
)

func main() {
	cfg := config.Load()

	conn, err := db.Open(cfg.DB)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	repo := db.ArticleRepo{DB: conn}

	articles, err := repo.ListAll()
	if err != nil {
		log.Fatal(err)
	}

	gen := generator.Generator{
		Articles: articles,
		OutDir:   "dist",
	}

	if err := gen.Build(); err != nil {
		log.Fatal(err)
	}
}
