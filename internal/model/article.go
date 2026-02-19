package model

import (
    "time"
    "html/template"
)

type Article struct {
	ID        int64
	Title     string
    SubjectId int32
	HTML      string
	CreatedAt time.Time
}

type ArticleView struct {
	ID          int64
	Title       string
    SubjectId   int32
    Slug        string
	HTML        template.HTML
	CreatedAt   time.Time
}
