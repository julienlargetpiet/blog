package db

import (
	"database/sql"

	"blog/internal/model"
)

type ArticleRepo struct {
	DB *sql.DB
}

func (r *ArticleRepo) ListAll() ([]model.Article, error) {
	rows, err := r.DB.Query(`
		SELECT id, title, html, created_at
		FROM articles
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var articles []model.Article

	for rows.Next() {
		var a model.Article
		if err := rows.Scan(
			&a.ID,
			&a.Title,
			&a.HTML,
			&a.CreatedAt,
		); err != nil {
			return nil, err
		}
		articles = append(articles, a)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return articles, nil
}

func (r *ArticleRepo) GetByID(id int64) (model.Article, error) {
	var a model.Article

	err := r.DB.QueryRow(`
		SELECT id, title, html, created_at
		FROM articles
		WHERE id = ?
	`, id).Scan(
		&a.ID,
		&a.Title,
		&a.HTML,
		&a.CreatedAt,
	)

	return a, err
}


func (r *ArticleRepo) Update(id int64, title, html string) error {
	_, err := r.DB.Exec(
		`UPDATE articles
		 SET title = ?, html = ?
		 WHERE id = ?`,
		title,
		html,
		id,
	)
	return err
}

func (r *ArticleRepo) Delete(id int64) error {
	_, err := r.DB.Exec(
		`DELETE FROM articles WHERE id = ?`,
		id,
	)
	return err
}

func (r *ArticleRepo) Create(title, html string) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO articles (title, html, created_at)
		VALUES (?, ?, NOW())
	`, title, html)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}



