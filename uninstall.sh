#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

log()  { printf "\n\033[1;31m==>\033[0m %s\n" "$*"; }
warn() { printf "\n\033[1;33m[warn]\033[0m %s\n" "$*"; }
die()  { printf "\n\033[1;31m[err]\033[0m %s\n" "$*"; exit 1; }

[[ "$EUID" -eq 0 ]] || die "Run with sudo."

echo "======================================="
echo " Statix Full Uninstall"
echo "======================================="
echo
read -rp "Domain to remove (e.g. example.com): " DOMAIN
[[ -n "$DOMAIN" ]] || die "Domain required."

read -rp "This will DELETE EVERYTHING related to Statix. Continue? [y/N]: " CONFIRM
[[ "$CONFIRM" =~ ^[Yy]$ ]] || { echo "Aborted."; exit 0; }

APP_DIR="/var/www/go_blog"
SHINY_DIR="/var/www/RShinyApp"
RLIBS_DIR="/var/www/Rlibs"

DB_NAME="go_blog"
DB_USER="blog_user"

NGINX_SITE="/etc/nginx/sites-available/${DOMAIN}"
NGINX_LINK="/etc/nginx/sites-enabled/${DOMAIN}"

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"

########################################
# Stop services
########################################
log "Stopping services..."

systemctl stop go_blog 2>/dev/null || true
systemctl disable go_blog 2>/dev/null || true

systemctl stop shiny 2>/dev/null || true
systemctl disable shiny 2>/dev/null || true

########################################
# Remove systemd units
########################################
log "Removing systemd services..."

rm -f /etc/systemd/system/go_blog.service
rm -f /etc/systemd/system/shiny.service

systemctl daemon-reload

########################################
# Remove NGINX config
########################################
log "Removing NGINX configuration..."

rm -f "$NGINX_LINK"
rm -f "$NGINX_SITE"

nginx -t && systemctl reload nginx || warn "NGINX reload skipped."

########################################
# Remove Let's Encrypt certs
########################################
if [[ -d "$CERT_DIR" ]]; then
  log "Removing Let's Encrypt certificates..."
  rm -rf "/etc/letsencrypt/live/${DOMAIN}"
  rm -rf "/etc/letsencrypt/archive/${DOMAIN}"
  rm -f  "/etc/letsencrypt/renewal/${DOMAIN}.conf"
fi

########################################
# Drop database
########################################
log "Dropping database and user..."

mysql -u root <<EOF
DROP DATABASE IF EXISTS \`${DB_NAME}\`;
DROP USER IF EXISTS '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
EOF

########################################
# Remove application directories
########################################
log "Removing application directories..."

rm -rf "$APP_DIR"
rm -rf "$SHINY_DIR"
rm -rf "$RLIBS_DIR"

########################################
# Final
########################################
log "Statix completely removed."

echo
echo "You may now:"
echo "- Remove Go manually if unused"
echo "- Remove nginx/mysql if unused"
echo
echo "Uninstall complete."


