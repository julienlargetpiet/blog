--CREATE DATABASE IF NOT EXISTS go_blog;

DROP TABLE articles;

CREATE TABLE subjects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    title_url VARCHAR(255) NOT NULL,
    subject_id INT NOT NULL,
    html MEDIUMTEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    INDEX idx_articles_subject (subject_id),

    CONSTRAINT fk_articles_subject
        FOREIGN KEY (subject_id)
        REFERENCES subjects(id)
        ON DELETE RESTRICT
);

INSERT INTO subjects (title, slug)
VALUES ('Default', 'default');
