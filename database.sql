--CREATE DATABASE IF NOT EXISTS go_blog;

DROP TABLE articles;

CREATE TABLE articles (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    title TEXT NOT NULL,
    html MEDIUMTEXT NOT NULL,
    created_at DATETIME NOT NULL
);

INSERT INTO articles (title, html, created_at)
VALUES (
  'First post',
  '<p>Hello from the database.</p>',
  NOW()
);

