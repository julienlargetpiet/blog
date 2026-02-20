package db

import (
	"database/sql"
    "strings"
    "regexp"

	"blog/internal/model"
)

type SubjectRepo struct {
	DB *sql.DB
}

func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func (r *SubjectRepo) Delete(id int32) error {
	_, err := r.DB.Exec(
		`DELETE FROM subjects WHERE id = ?`,
		id,
	)
	return err
}

func (r *SubjectRepo) Create(title string) (int64, error) {
	slug := Slugify(title)

	res, err := r.DB.Exec(`
		INSERT INTO subjects (title, slug)
		VALUES (?, ?)
	`, title, slug)

	if err != nil {
		return 0, err
	}

	return res.LastInsertId()
}

func (r *SubjectRepo) ListAll() ([]model.Subject, error) {
	rows, err := r.DB.Query(`
		SELECT id, title, slug
		FROM subjects
		ORDER BY title ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []model.Subject

	for rows.Next() {
		var s model.Subject
		if err := rows.Scan(&s.Id, &s.Title, &s.Slug); err != nil {
			return nil, err
		}
		subjects = append(subjects, s)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subjects, nil
}

func (r *SubjectRepo) Update(id int32, title string) error {
	_, err := r.DB.Exec(`
		UPDATE subjects
		SET title = ?, slug = ?
		WHERE id = ?
	`, title, Slugify(title), id)

	return err
}

func (r *SubjectRepo) GetByID(id int32) (model.Subject, error) {
	var s model.Subject

	err := r.DB.QueryRow(`
		SELECT id, title, slug
		FROM subjects
		WHERE id = ?
	`, id).Scan(&s.Id, &s.Title, &s.Slug)

	if err != nil {
		return model.Subject{}, err
	}

	return s, nil
}

