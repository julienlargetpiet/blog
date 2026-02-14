# Statix (Go) — Production Deployment Guide  
NGINX Reverse Proxy + MySQL + systemd

This guide explains how to deploy the Statix static site generator and its Go admin backend behind NGINX, with MySQL and a hardened systemd service.

Domain placeholder used in this guide:

    example.com

Replace it with your real domain.

---

# Architecture

Client → (Cloudflare optional) → NGINX (HTTPS)
                           ├── /            → static site (/dist)
                           ├── /assets/     → static assets
                           └── /admin       → Go backend (127.0.0.1:8080)
                                                     ↓
                                                   MySQL

---

# 1) Install Dependencies (Debian/Ubuntu)

sudo apt update
sudo apt install -y nginx mariadb-server

Install Go 1.23+ from:
https://go.dev/dl/

Verify:

go version

---

# 2) Create System User

sudo useradd -r -s /bin/false goblog

---

# 3) Build the Go Admin Binary (IMPORTANT)

From the root of your project (where go.mod is located):

go mod tidy
go build -o go_blog_admin

This produces the binary:

    ./go_blog_admin

For a clean production build (recommended):

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o go_blog_admin

You should now have:

    go_blog_admin

---

# 4) Deploy Project Files

Recommended production layout:

/var/www/go_blog/
    go_blog_admin
    dist/
    assets/
        common_files/
    _uploads/

Create directory:

sudo mkdir -p /var/www/go_blog

Move binary:

sudo mv ./go_blog_admin /var/www/go_blog/go_blog_admin

Make executable:

sudo chmod 0755 /var/www/go_blog/go_blog_admin

---

# 5) Permissions (CRITICAL)

Run:

sudo chown -R goblog:goblog /var/www/go_blog

The goblog user must own:
- binary
- dist/
- assets/common_files/
- _uploads/

---

# 6) MySQL Setup

Login:

sudo mariadb -u root -p

Create database:

CREATE DATABASE go_blog;

Create main application user (used in config.go):

CREATE USER 'blog_user'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON go_blog.* TO 'blog_user'@'localhost';
FLUSH PRIVILEGES;

Create restricted mysqldump user:

CREATE USER 'goblog'@'localhost' IDENTIFIED BY 'm';
GRANT SELECT, SHOW VIEW, TRIGGER, LOCK TABLES
ON go_blog.* TO 'goblog'@'localhost';
FLUSH PRIVILEGES;

---

# 7) Create Initial Schema

Create file database.sql:

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

Import it:

mariadb -p go_blog -u blog_user < database.sql

---

# 8) Configure Credentials (Hardcoded Preferred)

Edit:

internal/config/config.go

Example hardcoded configuration:

cfg := Config{
    DB: db.Config{
        User:     "blog_user",
        Password: "your_secure_password",
        Host:     "127.0.0.1",
        Port:     3306,
        DBName:   "go_blog",
    },
    AdminAddr: ":8080",
    AdminPass: "your_admin_password",  // for admin panel connection on the website
}

If using environment variables instead, ensure:

BLOG_DB_PASSWORD
BLOG_ADMIN_PASSWORD

are set, or the app will exit intentionally.

---

# 9) systemd Service

Create:

/etc/systemd/system/go_blog.service

[Unit]
Description=Go Blog Admin Server
After=network.target

[Service]
Type=simple
User=goblog
Group=goblog
WorkingDirectory=/var/www/go_blog
ExecStart=/var/www/go_blog/go_blog_admin

Restart=on-failure
RestartSec=3

NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target

Enable and start:

sudo systemctl daemon-reload
sudo systemctl enable go_blog
sudo systemctl start go_blog
sudo systemctl status go_blog

View logs:

journalctl -u go_blog -f

---

# 10) NGINX Configuration

Create:

/etc/nginx/sites-available/example.com

server {
    listen 80;
    server_name example.com www.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name example.com www.example.com;

    ssl_certificate     /etc/letsencrypt/live/example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/example.com/privkey.pem;

    location /admin {
        proxy_pass http://127.0.0.1:8080;

        proxy_http_version 1.1;
        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        proxy_redirect off;
    }

    root /var/www/go_blog/dist;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /assets/ {
        alias /var/www/go_blog/assets/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
}

Enable site:

sudo ln -s /etc/nginx/sites-available/example.com /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx

---

# 11) HTTPS with Let’s Encrypt

sudo apt install -y certbot python3-certbot-nginx

sudo certbot --nginx -d example.com -d www.example.com

---

# 12) Optional: Cloudflare Real IP Support

Add inside nginx.conf → http {} block:

set_real_ip_from 173.245.48.0/20;
set_real_ip_from 103.21.244.0/22;
set_real_ip_from 103.22.200.0/22;
set_real_ip_from 103.31.4.0/22;
set_real_ip_from 141.101.64.0/18;
set_real_ip_from 108.162.192.0/18;
set_real_ip_from 190.93.240.0/20;
set_real_ip_from 188.114.96.0/20;
set_real_ip_from 197.234.240.0/22;
set_real_ip_from 198.41.128.0/17;
set_real_ip_from 162.158.0.0/15;
set_real_ip_from 104.16.0.0/13;
set_real_ip_from 104.24.0.0/14;
set_real_ip_from 172.64.0.0/13;
set_real_ip_from 131.0.72.0/22;

real_ip_header CF-Connecting-IP;
real_ip_recursive on;

---

# Final Checklist

[ ] Go binary built (go_blog_admin exists)  
[ ] Binary moved to /var/www/go_blog  
[ ] Correct ownership applied  
[ ] MySQL database created  
[ ] blog_user created  
[ ] goblog backup user created  
[ ] Schema imported  
[ ] systemd service running  
[ ] NGINX configuration valid  
[ ] HTTPS enabled  

Your site should now be accessible at:

https://example.com  
https://example.com/admin



