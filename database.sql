--CREATE DATABASE IF NOT EXISTS go_blog;

DROP TABLE IF EXISTS articles;

CREATE TABLE subjects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE articles (
    id INT AUTO_INCREMENT PRIMARY KEY,
    title VARCHAR(255) NOT NULL UNIQUE,
    title_url VARCHAR(255) NOT NULL UNIQUE,
    subject_id INT NOT NULL,
    is_public BOOLEAN NOT NULL,
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

CREATE TABLE author (
  id INT,
  html MEDIUMTEXT
);

INSERT INTO author (id, html)
VALUES (0, '<h2 id="author_talk">About me</h2><p>default author description</p><h2 id="contact">Contact</h2><p>contacts</p>');




