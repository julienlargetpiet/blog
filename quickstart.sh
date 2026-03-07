#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

########################################
# Helpers
########################################
log()  { printf "\n\033[1;32m==>\033[0m %s\n" "$*"; }
warn() { printf "\n\033[1;33m[warn]\033[0m %s\n" "$*"; }
die()  { printf "\n\033[1;31m[err]\033[0m %s\n" "$*"; exit 1; }

require_root() { [[ "$EUID" -eq 0 ]] || die "Run with sudo."; }

prompt_password() {
  local var_name="$1" label="$2"
  local p1 p2
  while true; do
    read -rsp "$label: " p1; echo
    read -rsp "Confirm $label: " p2; echo
    [[ -n "$p1" ]] || { echo "Password cannot be empty."; continue; }
    [[ "$p1" == "$p2" ]] || { echo "Passwords do not match."; continue; }
    eval "$var_name=\"$p1\""
    break
  done
}

prompt_email() {
  local var_name="$1" label="$2"
  local p1 p2
  while true; do
    read -rp "$label: " p1; echo
    read -rp "Confirm $label: " p2; echo
    [[ -n "$p1" ]] || { echo "Email cannot be empty."; continue; }
    [[ "$p1" == "$p2" ]] || { echo "Emails do not match."; continue; }
    eval "$var_name=\"$p1\""
    break
  done
}

prompt_text() {
  local var_name="$1" label="$2"
  local value
  while true; do
    read -rp "$label: " value
    [[ -n "$value" ]] || { echo "Value cannot be empty."; continue; }
    eval "$var_name=\"$value\""
    break
  done
}

yes_no_prompt() {
  local var_name="$1" label="$2" answer
  read -rp "$label [y/N]: " answer
  case "$answer" in
    y|Y|yes|YES) eval "$var_name=1" ;;
    *)           eval "$var_name=0" ;;
  esac
}

########################################
# Start
########################################
require_root

echo "======================================="
echo " Statix Quickstart (Idempotent)"
echo "======================================="
echo

read -rp "Domain (e.g. example.com): " DOMAIN
[[ -n "$DOMAIN" ]] || die "Domain required."

prompt_email ADMIN_EMAIL "Admin contact email: "
prompt_text BLOG_TITLE "Blog title (e.g. Julien's Blog)"
prompt_password DB_PASS "Database password"
prompt_password ADMIN_PASS "Admin password"

ESCAPED_DB_PASS=$(printf '%s\n' "$DB_PASS" | sed 's/[\/&]/\\&/g')
ESCAPED_ADMIN_PASS=$(printf '%s\n' "$ADMIN_PASS" | sed 's/[\/&]/\\&/g')

yes_no_prompt ENABLE_SHINY "Enable optional R Shiny log analyzer?"
yes_no_prompt ENABLE_TLS   "Enable HTTPS via Let's Encrypt?"

TLS_EMAIL=""
if [[ "$ENABLE_TLS" -eq 1 ]]; then
  read -rp "Email for Let's Encrypt: " TLS_EMAIL
  [[ -n "$TLS_EMAIL" ]] || die "Email required for certbot."
fi

########################################
# Constants
########################################
APP_USER="goblog"
APP_DIR="/var/www/go_blog"
STATIC_ROOT="$APP_DIR/dist"

DB_NAME="go_blog"
DB_USER="blog_user"

ADMIN_BIND="127.0.0.1:8080"

SHINY_DIR="/var/www/RShinyApp"
SHINY_PORT="7665"
RLIBS_DIR="/var/www/Rlibs"

CERT_DIR="/etc/letsencrypt/live/${DOMAIN}"
CERT_FULLCHAIN="${CERT_DIR}/fullchain.pem"
CERT_PRIVKEY="${CERT_DIR}/privkey.pem"

########################################
# Base packages
########################################
log "Installing base packages (safe rerun)..."
apt update
apt install -y nginx default-mysql-server git curl wget rsync openssl

########################################
# Go install (idempotent)
########################################
if ! command -v go >/dev/null 2>&1; then
  log "Installing Go..."
  wget -q https://go.dev/dl/go1.23.0.linux-amd64.tar.gz
  rm -rf /usr/local/go
  tar -C /usr/local -xzf go1.23.0.linux-amd64.tar.gz
  rm go1.23.0.linux-amd64.tar.gz
fi

########################################
# System user
########################################
if ! id -u "$APP_USER" >/dev/null 2>&1; then
  log "Creating system user $APP_USER"
  useradd -r -m -d /home/goblog -s /usr/sbin/nologin goblog
fi

mkdir -p /var/www
chown -R "$APP_USER:$APP_USER" /var/www

########################################
# Clone or update repo
########################################
if [[ -d "$APP_DIR/.git" ]]; then
  log "Repository exists. Pulling latest..."
  sudo -u "$APP_USER" git -C "$APP_DIR" pull
else
  log "Cloning repository..."
  sudo -u "$APP_USER" git clone https://github.com/julienlargetpiet/Statix "$APP_DIR"
fi

CONFIG_FILE="$APP_DIR/internal/config/config.go"

log "Injecting DB/Admin credentials into config.go"

# Replace default DB user
sed -i "s/getEnv(\"BLOG_DB_USER\", \"[^\"]*\")/getEnv(\"BLOG_DB_USER\", \"${DB_USER}\")/g" "$CONFIG_FILE"

# Replace default DB password
sed -i "s/getEnv(\"BLOG_DB_PASSWORD\", \"[^\"]*\")/getEnv(\"BLOG_DB_PASSWORD\", \"${ESCAPED_DB_PASS}\")/g" "$CONFIG_FILE"

# Replace default DB name
sed -i "s/getEnv(\"BLOG_DB_NAME\", \"[^\"]*\")/getEnv(\"BLOG_DB_NAME\", \"${DB_NAME}\")/g" "$CONFIG_FILE"

# Replace default admin password
sed -i "s/getEnv(\"BLOG_ADMIN_PASSWORD\", \"[^\"]*\")/getEnv(\"BLOG_ADMIN_PASSWORD\", \"${ESCAPED_ADMIN_PASS}\")/g" "$CONFIG_FILE"

HANDLER_FILE="$APP_DIR/internal/admin/handlers.go"

sed -i "s/PASSWORD/${ESCAPED_DB_PASS}/" "$HANDLER_FILE"

# Replace by user email

TEMPLATE_BASE="$APP_DIR/internal/templates/base.html"
TEMPLATE_ARTICLE="$APP_DIR/internal/templates/base_article.html"

log "Injecting admin contact email into templates..."

ESCAPED_EMAIL=$(printf '%s\n' "$ADMIN_EMAIL" | sed 's/[\/&]/\\&/g')

sed -i "s/username@domainname.com/${ESCAPED_EMAIL}/g" "$TEMPLATE_BASE"
sed -i "s/username@domainname.com/${ESCAPED_EMAIL}/g" "$TEMPLATE_ARTICLE"

# Replace by user title

ESCAPED_TITLE=$(printf '%s\n' "$BLOG_TITLE" | sed 's/[\/&]/\\&/g')

TEMPLATE_INDEX="$APP_DIR/internal/templates/users/index.html"

log "Injecting blog title into index template..."

sed -i "s/YourName's Blog/${ESCAPED_TITLE}/g" "$TEMPLATE_INDEX"



SHINY_GLOBAL="$APP_DIR/RShinyApp/global.R"

if [[ -f "$SHINY_GLOBAL" ]]; then
  log "Injecting Shiny admin password into global.R"

  sed -i "s/password = c(\"[^\"]*\")/password = c(\"${ADMIN_PASS}\")/g" "$SHINY_GLOBAL"
else
  warn "Shiny global.R not found. Skipping credential injection."
fi

########################################
# Build binary (always rebuild)
########################################

GO_CACHE="/var/www/go_blog/.gocache"
GO_MODCACHE="/var/www/go_blog/.gomodcache"

mkdir -p "$GO_CACHE" "$GO_MODCACHE"
chown -R "$APP_USER:$APP_USER" "$GO_CACHE" "$GO_MODCACHE"

cd "$APP_DIR"

log "Building Go binary..."

sudo -u "$APP_USER" \
  env GOCACHE="$GO_CACHE" \
      GOMODCACHE="$GO_MODCACHE" \
      HOME="/var/www/go_blog" \
      /usr/local/go/bin/go build -buildvcs=false -o go_blog_admin ./cmd/admin

log "Normalizing permissions (go_blog)..."

find "$APP_DIR" -type d -exec chmod 755 {} \;
find "$APP_DIR" -type f -exec chmod 644 {} \;

chmod 755 "$APP_DIR/go_blog_admin"

########################################
# Database (idempotent)
########################################
log "Ensuring database exists..."
mysql -u root <<EOF
CREATE DATABASE IF NOT EXISTS \`${DB_NAME}\`;
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON \`${DB_NAME}\`.* TO '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
EOF

# Check if tables exist
TABLE_COUNT=$(mysql -u "$DB_USER" -p"$DB_PASS" -Nse "
SELECT COUNT(*) FROM information_schema.tables
WHERE table_schema='${DB_NAME}';
")

if [[ "$TABLE_COUNT" -eq 0 ]]; then
  log "Importing database schema..."
  mysql -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" < "$APP_DIR/database.sql"
else
  warn "Database already contains tables. Skipping schema import."
fi

########################################
# systemd (always overwrite safely)
########################################
log "Writing systemd service..."
PUBLISH_TOKEN=$(openssl rand -hex 32)

echo "Publish token: $PUBLISH_TOKEN"

cat > /etc/systemd/system/go_blog.service <<EOF
[Unit]
Description=Go Blog Admin Server
After=network.target

[Service]
Type=simple
User=$APP_USER
WorkingDirectory=$APP_DIR
ExecStart=$APP_DIR/go_blog_admin

Environment="STATIX_PUBLISH_TOKEN=$PUBLISH_TOKEN"

Restart=on-failure
RestartSec=3
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true

[Install]
WantedBy=multi-user.target
EOF

chmod 600 /etc/systemd/system/go_blog.service

systemctl daemon-reload
systemctl enable go_blog
systemctl restart go_blog

########################################
# Optional Shiny
########################################
if [[ "$ENABLE_SHINY" -eq 1 ]]; then

  log "Preparing R Shiny directory..."

  # Ensure external Shiny directory exists
  mkdir -p "$SHINY_DIR"

  # Copy Shiny app from repo to dedicated location
  rsync -a --delete "$APP_DIR/RShinyApp/" "$SHINY_DIR/"

  chown -R "$APP_USER:$APP_USER" "$SHINY_DIR"

  ########################################
  # Inject admin password into Shiny
  ########################################
  SHINY_GLOBAL="$SHINY_DIR/global.R"

  if [[ -f "$SHINY_GLOBAL" ]]; then
    log "Injecting Shiny admin password into global.R"

    sed -i "s/password = c(\"[^\"]*\")/password = c(\"${ADMIN_PASS}\")/g" "$SHINY_GLOBAL"
  else
    warn "Shiny global.R not found in $SHINY_DIR"
  fi

  log "Normalizing permissions (RShinyApp)..."
  
  find "$SHINY_DIR" -type d -exec chmod 755 {} \;
  find "$SHINY_DIR" -type f -exec chmod 644 {} \;

  ########################################
  # Install R + dependencies
  ########################################
  log "Installing R dependencies (safe)..."

  apt install -y r-base libmaxminddb-dev libmaxminddb0 mmdb-bin \
    build-essential cmake pkg-config \
    libcurl4-openssl-dev libssl-dev libxml2-dev \
    libudunits2-dev libgdal-dev gdal-bin \
    libgeos-dev libproj-dev proj-bin

  mkdir -p "$RLIBS_DIR"
  chown -R "$APP_USER:$APP_USER" "$RLIBS_DIR"

  sudo -u "$APP_USER" R --no-save --no-restore -e "
    .libPaths('$RLIBS_DIR');
    pkgs <- c('Rcpp','s2','sf','leaflet','shiny','plotly','dplyr','lubridate','bslib',
              'readr','shinymanager','shinycssloaders','DT','stringr','purrr','shinyjs');
    missing <- pkgs[!pkgs %in% rownames(installed.packages())];
    if(length(missing)) install.packages(missing, repos='https://cloud.r-project.org');
  "

  ########################################
  # Allow Shiny to read NGINX logs
  ########################################
  if ! groups "$APP_USER" | grep -q "\badm\b"; then
    log "Adding $APP_USER to adm group"
    usermod -aG adm "$APP_USER"
  fi

  ########################################
  # Write systemd service
  ########################################
  log "Writing Shiny systemd service..."

  cat > /etc/systemd/system/shiny.service <<EOF
[Unit]
Description=Julien Shiny App
After=network.target

[Service]
Type=simple
User=$APP_USER
WorkingDirectory=$SHINY_DIR

Environment=R_LIBS_USER=$RLIBS_DIR

ExecStart=/usr/bin/R --no-save --no-restore -e "shiny::runApp('$SHINY_DIR', host='127.0.0.1', port=$SHINY_PORT)"

SupplementaryGroups=adm

Restart=always
RestartSec=5

NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
EOF

  systemctl daemon-reload
  systemctl enable shiny
  systemctl restart shiny

fi

########################################
# NGINX Configuration (Idempotent)
########################################

NGINX_SITE="/etc/nginx/sites-available/${DOMAIN}"
NGINX_LINK="/etc/nginx/sites-enabled/${DOMAIN}"

write_nginx_http() {

cat > "$NGINX_SITE" <<EOF

# --- Statix custom log format ---
log_format statix_main '\$remote_addr - \$remote_user [\$time_local] '
                        '"\$request" \$status \$body_bytes_sent '
                        '"\$http_referer" "\$http_user_agent" '
                        '"\$http_x_prefetch"';

server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};

    # --- Custom access log for Statix ---
    access_log /var/log/nginx/statix.log statix_main;

    # --- Go admin backend ---
    location /admin {
        proxy_pass http://${ADMIN_BIND};

        proxy_http_version 1.1;
        proxy_set_header Host              \$host;
        proxy_set_header X-Real-IP         \$remote_addr;
        proxy_set_header X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_redirect off;
    }

    # --- Static site ---
    root ${STATIC_ROOT};
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # --- Assets ---
    location /assets/ {
        alias ${APP_DIR}/assets/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
EOF

if [[ "$ENABLE_SHINY" -eq 1 ]]; then
cat >> "$NGINX_SITE" <<EOF

    # --- R Shiny app ---
    location /shiny/ {
        proxy_pass http://127.0.0.1:${SHINY_PORT}/;

        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_read_timeout 86400;
    }
EOF
fi

cat >> "$NGINX_SITE" <<EOF
}
EOF
}

write_nginx_https() {

cat > "$NGINX_SITE" <<EOF

# --- Statix custom log format ---
log_format statix_main '\$remote_addr - \$remote_user [\$time_local] '
                        '"\$request" \$status \$body_bytes_sent '
                        '"\$http_referer" "\$http_user_agent" '
                        '"\$http_x_prefetch"';

server {
    listen 80;
    server_name ${DOMAIN} www.${DOMAIN};
    return 301 https://\$host\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name ${DOMAIN} www.${DOMAIN};

    ssl_certificate     ${CERT_FULLCHAIN};
    ssl_certificate_key ${CERT_PRIVKEY};

    # --- Custom access log for Statix ---
    access_log /var/log/nginx/statix.log statix_main;

    # --- Go admin backend ---
    location /admin {
        proxy_pass http://${ADMIN_BIND};

        proxy_http_version 1.1;
        proxy_set_header Host              \$host;
        proxy_set_header X-Real-IP         \$remote_addr;
        proxy_set_header X-Forwarded-For   \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_redirect off;
    }

    # --- Static site ---
    root ${STATIC_ROOT};
    index index.html;

    location / {
        try_files \$uri \$uri/ /index.html;
    }

    # --- Assets ---
    location /assets/ {
        alias ${APP_DIR}/assets/;
        expires 30d;
        add_header Cache-Control "public, immutable";
    }
EOF

if [[ "$ENABLE_SHINY" -eq 1 ]]; then
cat >> "$NGINX_SITE" <<EOF

    # --- R Shiny app ---
    location /shiny/ {
        proxy_pass http://127.0.0.1:${SHINY_PORT}/;

        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection "upgrade";

        proxy_set_header Host \$host;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto \$scheme;

        proxy_read_timeout 86400;
    }
EOF
fi

cat >> "$NGINX_SITE" <<EOF
}
EOF
}

enable_nginx() {
    ln -sf "$NGINX_SITE" "$NGINX_LINK"
    nginx -t
    systemctl reload nginx
}

########################################
# Apply config
########################################

if [[ "$ENABLE_TLS" -eq 1 && -f "$CERT_FULLCHAIN" ]]; then
    write_nginx_https
else
    write_nginx_http
fi

enable_nginx

########################################
# TLS (idempotent)
########################################
if [[ "$ENABLE_TLS" -eq 1 ]]; then
  apt install -y certbot

  if [[ -f "$CERT_FULLCHAIN" && -f "$CERT_PRIVKEY" ]]; then
    warn "Certificates already exist. Skipping certbot."
  else
    mkdir -p "${STATIC_ROOT}/.well-known/acme-challenge"
    
    certbot certonly --webroot \
      -w "${STATIC_ROOT}" \
      -d "$DOMAIN" -d "www.$DOMAIN" \
      --agree-tos --email "$TLS_EMAIL" --non-interactive
  fi

  write_nginx_https
  enable_nginx
fi

echo
echo "======================================="
echo " Deployment Summary"
echo "======================================="
echo
echo "Statix blog should now be accessible at:"
echo "  http://$DOMAIN"
[[ "$ENABLE_TLS" -eq 1 ]] && echo "  https://$DOMAIN"
echo

if [[ "$ENABLE_SHINY" -eq 1 ]]; then
  echo "R Shiny dashboard is available at:"
  echo "  http://$DOMAIN/shiny/"
  [[ "$ENABLE_TLS" -eq 1 ]] && echo "  https://$DOMAIN/shiny/"
  echo
  echo "⚠ IMPORTANT — MaxMind GeoLite databases are NOT included."
  echo
  echo "You must manually download:"
  echo "  - GeoLite2-City.mmdb"
  echo "  - GeoLite2-ASN.mmdb"
  echo
  echo "Steps:"
  echo "  1) Create a free account at:"
  echo "     https://www.maxmind.com/en/geolite2/signup"
  echo
  echo "  2) Download the GeoLite2-City and GeoLite2-ASN databases (Binary format)"
  echo
  echo "  3) Place the files in:"
  echo "     $SHINY_DIR/geo/"
  echo
  echo "  4) Restart Shiny:"
  echo "     sudo systemctl restart shiny"
  echo
fi

echo
echo "Done."



