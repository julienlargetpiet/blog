package db

import (
	"database/sql"
    "errors"

	"blog/internal/model"
    "blog/internal/utils"
)

type SubjectRepo struct {
	DB *sql.DB
}


func (r *SubjectRepo) Delete(id int32) error {
	_, err := r.DB.Exec(
		`DELETE FROM subjects WHERE id = ?`,
		id,
	)
	return err
}

func (r *SubjectRepo) Create(title string) (int64, error) {
	slug := utils.Slugify(title)

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
	`, title, utils.Slugify(title), id)

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

func (r *SubjectRepo) ExistsByName(name string) (bool, error) {
	var exists int

	err := r.DB.QueryRow(`
		SELECT 1
		FROM subjects
		WHERE slug = ?
        LIMIT 1
	`, utils.Slugify(name)).Scan(&exists)

	if err != nil {
        if (errors.Is(err, sql.ErrNoRows)) {
    	    return false, nil
        }
		return false, err
	}

	return true, nil
}



