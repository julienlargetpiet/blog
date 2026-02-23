# Statix (Go) — Production Deployment Guide  
NGINX + MySQL/MariaDB + systemd

This document describes the complete production deployment of **Statix**, a Go-based static website generator with an integrated admin backend.

Architecture:

- Go admin backend → 127.0.0.1:8080
- NGINX reverse proxy
- MySQL or MariaDB database
- Static files served from /var/www/go_blog/dist
- systemd-managed service
- Dedicated non-root system user (goblog)

Domain placeholder used in this guide:

    example.com

Replace it with your real domain.

---

# 1️⃣ Prerequisites

Server: Debian / Ubuntu  
Privileges: sudo  

Install required packages:

```bash
sudo apt update
sudo apt install -y nginx mysql-server
```

If using MariaDB:

```bash
sudo apt install -y mariadb-server
```

Install Go (manual):

```bash
wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
/usr/local/go/bin/go version
```

---

# 2️⃣ Create Dedicated System User

```bash
sudo useradd -r -s /bin/false goblog
```

---

# 3️⃣ Clone Project Directly to Production Path

```bash
sudo mkdir -p /var/www
sudo chown -R goblog:goblog /var/www

sudo -u goblog git clone https://github.com/julienlargetpiet/blog /var/www/go_blog
cd /var/www/go_blog
```

The project must live directly inside:

    /var/www/go_blog

---

# 4️⃣ Build Production Binary

From the repo root:

```bash
cd /var/www/go_blog
sudo -u goblog /usr/local/go/bin/go build -buildvcs=false -o go_blog_admin ./cmd/admin
```

Binary location:

    /var/www/go_blog/go_blog_admin

---

# 5️⃣ Set Correct Linux Permissions

Ownership:

```bash
sudo chown -R goblog:goblog /var/www/go_blog
```

Directories must be traversable by nginx:

```bash
sudo find /var/www/go_blog -type d -exec chmod 755 {} \;
```

Files must be readable by nginx:

```bash
sudo find /var/www/go_blog -type f -exec chmod 644 {} \;
```

Binary must remain executable:

```bash
sudo chmod 755 /var/www/go_blog/go_blog_admin
```

Final permission model:

- Owner: goblog
- nginx: read-only
- No 777 anywhere

---

# 6️⃣ Database Setup (MySQL or MariaDB)

Login:

```bash
sudo mysql -u root -p
```

Create database:

```sql
CREATE DATABASE go_blog;
```

Create application user:

```sql
CREATE USER 'blog_user'@'localhost' IDENTIFIED BY 'secure_password';
GRANT ALL PRIVILEGES ON go_blog.* TO 'blog_user'@'localhost';
FLUSH PRIVILEGES;
```

(Optional) restricted backup user:

```sql
CREATE USER 'goblog_backup'@'localhost' IDENTIFIED BY 'strong_password';
GRANT SELECT, SHOW VIEW, TRIGGER, LOCK TABLES
ON go_blog.* TO 'goblog_backup'@'localhost';
FLUSH PRIVILEGES;
```

---

# 7️⃣ Import Schema (from your repo)

From the repo root (the file is assumed to exist in the repository):

```bash
cd /var/www/go_blog
mysql -u blog_user -p go_blog < database.sql
```

That’s it.

---

# 8️⃣ Configure Application Credentials

Edit:

    internal/config/config.go

Example:

```go
cfg := Config{
    DB: db.Config{
        User:     "blog_user",
        Password: "secure_password",
        Host:     "127.0.0.1",
        Port:     3306,
        DBName:   "go_blog",
    },
    AdminAddr: ":8080",
    AdminPass: "your_admin_password",
}
```

⚠ Never commit real credentials.

---

# 9️⃣ systemd Service

Create:

    /etc/systemd/system/go_blog.service

```ini
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

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable go_blog
sudo systemctl start go_blog
sudo systemctl status go_blog
```

View logs:

```bash
journalctl -u go_blog -f
```

---

# 🔟 NGINX Reverse Proxy + Static Serving

Create:

    /etc/nginx/sites-available/example.com

```nginx
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
```

Enable:

```bash
sudo ln -s /etc/nginx/sites-available/example.com /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

# 1️⃣1️⃣ HTTPS (Let’s Encrypt)

```bash
sudo apt install -y certbot python3-certbot-nginx
sudo certbot --nginx -d example.com -d www.example.com
```

---

# 1️⃣2️⃣ Updating the Application

Rebuild:

```bash
cd /var/www/go_blog
sudo -u goblog /usr/local/go/bin/go build -buildvcs=false -o go_blog_admin ./cmd/admin
```

Restart:

```bash
sudo systemctl restart go_blog
```

Logs:

```bash
journalctl -u go_blog -xe
```

---

# ✅ Final Verification Checklist

- [ ] Repository cloned to /var/www/go_blog
- [ ] Binary built (go_blog_admin)
- [ ] Permissions applied
- [ ] Database created
- [ ] blog_user created + privileges granted
- [ ] database.sql imported
- [ ] systemd service running
- [ ] NGINX site enabled
- [ ] HTTPS enabled

Statix should now be accessible at:

    https://example.com
    https://example.com/admin


# 📊 Optional Module — R Shiny Log Analyzer

The `RShinyApp/` directory contains a complete **R Shiny dashboard** for analyzing NGINX access logs.

Features:

- Bot filtering (User-Agent + rate heuristics)
- RegEx-based page filtering
- Traffic evolution over time
- Top visited pages (Top N + Other)
- Dark / Light mode
- Authentication via `shinymanager`
- Reverse proxy support
- systemd-managed service

This module is optional and intended for internal analytics.

---

# 1️⃣ Install R (Debian / Ubuntu)

```bash
sudo apt update
sudo apt install -y r-base
```

---

# 2️⃣ Install Required R Packages

Start R:

```bash
R
```

(Optional — use a user-level library):

```r
.libPaths("~/.local/share/R/library")
```

Install required packages:

```r
install.packages(c(
  "shiny",
  "plotly",
  "dplyr",
  "lubridate",
  "bslib",
  "readr",
  "shinymanager",
  "shinycssloaders",
  "DT",
  "stringr",
  "jsonlite"
))
```

Exit R:

```r
q()
```

---

# 3️⃣ Deploy the Shiny App

Place the Shiny project in:

```
/var/www/RShinyApp
```

Set ownership and permissions:

```bash
sudo chown -R goblog:goblog /var/www/RShinyApp
sudo find /var/www/RShinyApp -type d -exec chmod 755 {} \;
sudo find /var/www/RShinyApp -type f -exec chmod 644 {} \;
```

⚠ Edit `RShinyApp/global.R` and configure your admin credentials.

Never commit real credentials.

---

# 4️⃣ Manual Test (Optional)

```bash
R
```

```r
shiny::runApp('/var/www/RShinyApp', host='127.0.0.1', port=7665)
```

Open:

```
http://127.0.0.1:7665
```

Stop with:

```
Ctrl+C
```

---

# 5️⃣ NGINX Reverse Proxy Configuration

Edit:

```
/etc/nginx/sites-available/example.com
```

Add inside the HTTPS server block:

```nginx
location /shiny/ {
    proxy_pass http://127.0.0.1:7665/;

    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";

    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
    proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    proxy_set_header X-Forwarded-Proto $scheme;

    proxy_read_timeout 86400;
}
```

Reload NGINX:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

Dashboard URL:

```
https://example.com/shiny/
```

---

# 6️⃣ systemd Service for Shiny

Create:

```
/etc/systemd/system/shiny.service
```

```ini
[Unit]
Description=Shiny Log Analyzer
After=network.target

[Service]
Type=simple
User=goblog
Group=goblog
WorkingDirectory=/var/www/RShinyApp

ExecStart=/usr/bin/R --no-save --no-restore -e "shiny::runApp('/var/www/RShinyApp', host='127.0.0.1', port=7665)"

Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable shiny
sudo systemctl start shiny
sudo systemctl status shiny
```

Logs:

```bash
journalctl -u shiny -f
```

---

# 🔐 Security Notes

- Shiny listens only on `127.0.0.1`
- It is exposed via NGINX reverse proxy
- Authentication handled via `shinymanager`
- Consider adding:
  - Firewall restrictions
  - NGINX rate limiting
  - IP allowlist (if internal-only)

---

# ✅ Result

You now have a self-hosted NGINX log analytics dashboard:

- Intelligent bot filtering
- Page-based traffic analysis
- Interactive Plotly charts
- Dark mode support
- systemd-managed background service
- Secure reverse proxy exposure



