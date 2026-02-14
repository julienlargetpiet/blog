# Statix (Go) — Production Deployment Guide  
NGINX Reverse Proxy + MariaDB/MySQL + systemd

This document describes the complete production deployment of **Statix**,  
a Go-based static website generator with an admin backend.

It covers:

- Building the Go binary
- Database setup (MySQL or MariaDB)
- Proper Linux permissions
- systemd service configuration
- Global NGINX configuration
- NGINX reverse proxy
- HTTPS (Let’s Encrypt)
- Optional Cloudflare integration
- Safe update procedure

Domain placeholder used in this guide:

    example.com

Replace it with your real domain.

---

# 1. Prerequisites

Server: Debian / Ubuntu  
Privileges: sudo access  
Go version: 1.23+  

Install required packages:

```bash
sudo apt update
sudo apt install -y nginx mysql-server
```

If using MariaDB instead:

```bash
sudo apt install -y mariadb-server
```

Install Go:

On Linux:

```
$ wget https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
$ sudo tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
$ export PATH=$PATH:/usr/local/go/bin
```

---

# 2. Build the Go Admin Binary

From the root of your project (where `go.mod` is located):

```bash
go mod tidy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o go_blog_admin
```

This produces:

```
./go_blog_admin
```

The binary is now ready for production.

---

# 3. Create System User

Create a dedicated system user for security:

```bash
sudo useradd -r -s /bin/false goblog
```

---

# 4. Deploy Project Files

Create production directory:

```bash
sudo mkdir -p /var/www/go_blog
```

Move binary:

```bash
sudo mv ./go_blog_admin /var/www/go_blog/go_blog_admin
```

Your final layout should be:

```
/var/www/go_blog/
    go_blog_admin
    dist/
    assets/
        common_files/
    _uploads/
```

---

# 5. Set Permissions (Critical Step)

Ensure everything belongs to the `goblog` user:

```bash
sudo chown -R goblog:goblog /var/www/go_blog
```

Make binary executable:

```bash
sudo chmod 0755 /var/www/go_blog/go_blog_admin
```

This ensures:
- The service can execute the binary
- The app can write to `_uploads`
- Static files are readable

---

# 6. Database Setup (MySQL or MariaDB)

Login:

```bash
sudo mysql -u root -p
```

## Create Database

```sql
CREATE DATABASE go_blog;
```

## Create Main Application User

```sql
CREATE USER 'blog_user'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON go_blog.* TO 'blog_user'@'localhost';
FLUSH PRIVILEGES;
```

## Create Restricted Backup User

```sql
CREATE USER 'goblog'@'localhost' IDENTIFIED BY 'm';
GRANT SELECT, SHOW VIEW, TRIGGER, LOCK TABLES
ON go_blog.* TO 'goblog'@'localhost';
FLUSH PRIVILEGES;
```

---

# 7. Initialize Schema

Create a file named `database.sql`:

```sql
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
```

Import schema:

```bash
mysql -u blog_user -p go_blog < database.sql
```

---

# 8. Configure Credentials

Edit:

```
internal/config/config.go
```

Example hardcoded configuration:

```go
cfg := Config{
    DB: db.Config{
        User:     "blog_user",
        Password: "your_secure_password",
        Host:     "127.0.0.1",
        Port:     3306,
        DBName:   "go_blog",
    },
    AdminAddr: ":8080",
    AdminPass: "your_admin_password",
}
```

⚠ Never commit real credentials to version control.

---

# 9. systemd Service

Create:

```
/etc/systemd/system/go_blog.service
```

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

Environment=GIN_MODE=release

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

# 10. Global NGINX Configuration

The main NGINX configuration file is:

```
/etc/nginx/nginx.conf
```

In most installations, the default configuration is sufficient.

Verify that inside the `http {}` block you have:

```nginx
include /etc/nginx/conf.d/*.conf;
include /etc/nginx/sites-enabled/*;
```

These directives are required so that your site configuration
(`/etc/nginx/sites-available/example.com`) is loaded.

After any modification:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

---

# 11. NGINX Reverse Proxy (Site Configuration)

Create:

```
/etc/nginx/sites-available/example.com
```

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

Enable site:

```bash
sudo ln -s /etc/nginx/sites-available/example.com /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

---

# 12. HTTPS (Let’s Encrypt)

Install:

```bash
sudo apt install -y certbot python3-certbot-nginx
```

Generate certificate:

```bash
sudo certbot --nginx -d example.com -d www.example.com
```

---

# 13. Optional: Cloudflare Real IP Support

Inside `/etc/nginx/nginx.conf` → `http {}` block:

```nginx
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
```

Reload NGINX after editing:

```bash
sudo nginx -t
sudo systemctl reload nginx
```

---

# 14. Updating the Application (Safe Procedure)

Build new binary:

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o go_blog_admin
```

Deploy update:

```bash
sudo systemctl stop go_blog
sudo mv go_blog_admin /var/www/go_blog/go_blog_admin
sudo chown goblog:goblog /var/www/go_blog/go_blog_admin
sudo chmod 0755 /var/www/go_blog/go_blog_admin
sudo systemctl start go_blog
```

If something fails:

```bash
journalctl -u go_blog -xe
```

---

# 15. Stylistic advises when writing articles

You absolutely do not need any custom (html-in) css when you write your articles such as `style="..."` because all is handled in the `/assets/css/style.css`

## Table

When you need to create a table use this pattern:

```
<section class="form-section">

<div class="admin-table-scroll-top">
  <div class="admin-table-scroll-inner"></div>
</div>
  
<div class="admin-table-wrapper">
<table class="admin-table">
  <tr>
    <th>column 1</th>
    <th>column 2</th>
    ...
  </tr>
  <tr><td>cell at row 1 column 1</td><td>cell at row 1 column2</td>...</tr>
  <tr><td>cell at row 2 column 1</td><td>cell at row 2 column2</td>...</tr>
</table>
</div>
</section>

```

## Code

When you need to write a code block use this pattern:

```
<div class="code-block">
<pre>
<button class="copy-btn">Copy</button>
<code class="language-cpp">#include &lt;cublas_v2.h&gt;
<!-- Code goes here -->
</code>
</pre>
</div>
```

Note that this example is for `C++` code, so when you write bash code do that:

```
<div class="code-block">
<pre>
<button class="copy-btn">Copy</button>
<code class="language-bash">#include &lt;cublas_v2.h&gt; <!-- MODIFIED -->
<!-- Code goes here -->
</code>
</pre>
</div>
```

Currently supported languages highlighting are:

- C
- C++
- haskell
- rust
- go
- R
- bash
- NGINX
- systemd

you can add new languages downloading the prism `JS` code you can find here:

<a href="https://github.com/PrismJS/prism/tree/master/components?utm_source=chatgpt.com">https://github.com/PrismJS/prism/tree/master/components</a>

you now put the js code into `/assets/prism/components/`

and now edit `/internal/templates/base.html` to add:

```
<script defer src="/assets/prism/components/prism-yourlanguage.min.js"></script>`
```

## Math 

Use the Katex synthax like so:

```
<p>
$$
\det(A) = \sum_{j=0}^{n-1} (-1)^j a_{0j} \det(M_{0j})
$$
</p>
```

## 📊 Shiny Log Analyzer (Optional Module)

The `RShiny/` directory contains a complete **R Shiny dashboard** for analyzing your NGINX access logs.

It provides:

- Bot filtering (User-Agent + rate heuristics)
- RegEx-based page filtering
- Traffic evolution over time
- Top visited pages (Top N + Other)
- Dark / Light mode
- Authentication via `shinymanager`
- Reverse proxy support
- systemd service deployment

This module is optional and intended for internal analytics.

---

### 1️⃣ Install R (Ubuntu / Debian)

```bash
sudo apt update
sudo apt install r-base
sudo apt install -y libcurl4-openssl-dev libssl-dev libxml2-dev
```

The development libraries are required for packages like `curl`, `httr`, and `plotly`.

---

### 2️⃣ Install Required R Packages

Start R:

```bash
R
```

(Optional but recommended) use a user-level library:

```r
.libPaths("~/.local/share/R/library")
```

Install all required packages:

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
  "curl",
  "httr"
))
```

Exit R:

```r
q()
```

---

### -- Mandatory Step --

Edit `Rshiny/global.R` and modify you admin password.

⚠ Never commit real credentials to version control.

---

### 3️⃣ Run the App Manually (Optional Test)

Before configuring systemd, you can test locally:

```bash
R
```

```r
shiny::runApp('/var/www/R_Shiny_NGINX', host='127.0.0.1', port=7665)
```

Then open:

```
http://127.0.0.1:7665
```

---

### 4️⃣ Configure NGINX Reverse Proxy

To expose the dashboard at:

```
https://yourdomain.com/shiny/
```

Edit:

```
/etc/nginx/sites-available/example.com
```

Add this block inside your `server` configuration:

```nginx
# --- R Shiny app ---
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

Validate configuration:

```bash
sudo nginx -t
```

Reload NGINX:

```bash
sudo systemctl reload nginx
```

---

## 5️⃣ Create a systemd Service

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
User=julien
WorkingDirectory=/var/www/R_Shiny_NGINX

ExecStart=/usr/bin/R --no-save --no-restore -e "shiny::runApp('/var/www/R_Shiny_NGINX', host='127.0.0.1', port=7665)"

Restart=always
RestartSec=5

# Basic hardening
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
```

Check status:

```bash
sudo systemctl status shiny
```

View logs:

```bash
journalctl -u shiny -f
```

---

## 6️⃣ Security Considerations

- The app listens only on `127.0.0.1`
- It is exposed externally via NGINX reverse proxy
- Authentication is handled via `shinymanager`
- Consider additional protections:
  - IP allowlists
  - NGINX rate limiting
  - Firewall restrictions
  - Dedicated system user (e.g., `shiny`)
  - Restrictive permissions on log files

---

## 7️⃣ Access the Dashboard

Once running, open:

```
https://yourdomain.com/shiny/
```

Login using your configured credentials.

---

## ✅ Result

You now have a self-hosted NGINX log analytics dashboard featuring:

- Intelligent bot filtering
- Page-based traffic analysis
- Time-window aggregation
- Interactive charts (Plotly)
- Dark mode support
- systemd-managed background service
- Secure reverse proxy exposure

Clean, reproducible, production-ready.

# Final Verification Checklist

- [ ] Binary built (`go_blog_admin`)
- [ ] Permissions applied
- [ ] Database created
- [ ] blog_user created
- [ ] goblog backup user created
- [ ] Schema imported
- [ ] systemd service running
- [ ] nginx.conf verified
- [ ] Site configuration enabled
- [ ] HTTPS enabled

Statix should now be accessible at:

https://example.com  
https://example.com/admin


