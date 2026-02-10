package model

import (
    "time"
    "html/template"
)

type Article struct {
	ID        int64
	Title     string
	HTML      string
	CreatedAt time.Time
}

type ArticleView struct {
	ID        int64
	Title     string
	HTML      template.HTML
	CreatedAt time.Time
}
