package model

import (
    "time"
    "html/template"
)

type Article struct {
	ID        int64
	Title     string
    TitleURL  string
    SubjectId int64
	HTML      string
	CreatedAt time.Time
}

type ArticleView struct {
	ID          int64
	Title       string
    TitleURL    string
    SubjectId   int64
    Slug        string
	HTML        template.HTML
	CreatedAt   time.Time
}
