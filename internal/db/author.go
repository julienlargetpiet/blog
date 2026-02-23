package db

import (
	"database/sql"
)

type AuthorRepo struct {
	DB *sql.DB
}

func (r *AuthorRepo) Update(content string) error {

	_, err := r.DB.Exec(`
		UPDATE author
		SET html = ?
        WHERE id = 0
	`, content)

    return err
}

func (r *AuthorRepo) GetContent() (string, error) {

    var content string

	err := r.DB.QueryRow(`
		SELECT html 
        FROM author
        WHERE id = 0
	`).Scan(&content)

    if err != nil {
        return "", err
    }

    return content, nil
}





