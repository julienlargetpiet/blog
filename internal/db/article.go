package db

import (
	"database/sql"

	"blog/internal/model"
	"blog/internal/utils"
)

type ArticleRepo struct {
	DB *sql.DB
}

func (r *ArticleRepo) ListAll() ([]model.Article, error) {
	rows, err := r.DB.Query(`
		SELECT id, title, title_url, subject_id, html, created_at
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
            &a.TitleURL,
			&a.SubjectId,
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
		SELECT id, title, title_url, subject_id, html, created_at
		FROM articles
		WHERE id = ?
	`, id).Scan(
		&a.ID,
		&a.Title,
        &a.TitleURL,
		&a.SubjectId,
		&a.HTML,
		&a.CreatedAt,
	)

	return a, err
}

func (r *ArticleRepo) Update(id int64, title string, subjectId int32, html string) error {
	_, err := r.DB.Exec(`
		UPDATE articles
		SET title = ?, title_url = ?, subject_id = ?, html = ?
		WHERE id = ?
	`, title, utils.Slugify(title), subjectId, html, id)
	return err
}

func (r *ArticleRepo) Delete(id int64) error {
	_, err := r.DB.Exec(
		`DELETE FROM articles WHERE id = ?`,
		id,
	)
	return err
}

func (r *ArticleRepo) Create(title string, subjectId int32, html string) (int64, error) {
	res, err := r.DB.Exec(`
		INSERT INTO articles (title, title_url, subject_id, html, created_at)
		VALUES (?, ?, ?, ?, NOW())
	`, title, utils.Slugify(title), subjectId, html)
	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *ArticleRepo) GetDefaultSubjectID() (int32, error) {
	var id int32
    const DefaultSubjectSlug = "default"

	err := r.DB.QueryRow(`
		SELECT id
		FROM subjects
		WHERE slug = ?
	`, DefaultSubjectSlug).Scan(&id)

    if err != nil {
        return 0, err
    }

	return id, nil
}

func (r *ArticleRepo) ReslugAll() error {
	rows, err := r.DB.Query(`
		SELECT id, title
		FROM articles
	`)
	if err != nil {
		return err
	}
	defer rows.Close()

	tx, err := r.DB.Begin()
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(`
		UPDATE articles
		SET title_url = ?
		WHERE id = ?
	`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for rows.Next() {
		var id int64
		var title string

		if err := rows.Scan(&id, &title); err != nil {
			tx.Rollback()
			return err
		}

		slug := utils.Slugify(title)

		if _, err := stmt.Exec(slug, id); err != nil {
			tx.Rollback()
			return err
		}
	}

	if err := rows.Err(); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}


