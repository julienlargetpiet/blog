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
$ wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
$ sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
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


