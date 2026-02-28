library(shiny)
library(plotly)
library(dplyr)
library(lubridate)
library(bslib)
library(readr)
library(shinymanager)
library(shinycssloaders)
library(DT)
library(stringr)
library(scales)
library(leaflet)
library(purrr)
library(shinyjs)

cat("GLOBAL LOADED\n")

credentials <- data.frame(
  user = c("admin"),
  password = c("adminpass"),
  admin = c(TRUE),
  stringsAsFactors = FALSE
)

Sys.setlocale("LC_TIME", "C")
options(shiny.maxRequestSize = 300 * 1024^2)

ip_exclude <- c("")

bot_keywords <- unique(c(
  # Core bot terms
  "bot","crawler","spider",

  # SEO / scraping bots
  "ahrefs","ahrefsbot","semrush","mj12","dotbot",

  # Search engines
  "googlebot","bingbot","yandex","baiduspider","slurp",

  # Monitoring / uptime
  "uptime","pingdom","monitor",

  # CLI / scripting
  "curl","wget","python","python-requests","scrapy",

  # Headless / automation frameworks
  "headless","phantomjs","selenium",
  "playwright","puppeteer",

  # Programmatic HTTP clients
  "node-fetch","axios",
  "go-http-client","libwww-perl","java/",
  "httpclient",

  # Social media fetchers
  "facebookexternalhit",

  # Additional suspicious / automation UAs
  "okhttp",
  "httpx",
  "restsharp",
  "powershell",
  "postmanruntime",
  "insomnia",
  "apache-httpclient",
  "ruby",
  "perl",
  "mechanize",
  "feedfetcher",
  "dataprovider",
  "masscan",
  "zgrab",
  "nmap",
  "gobuster",
  "sqlmap"
))

bot_regex <- paste(bot_keywords, collapse = "|")

residential_isp <- c(

# --- 🇫🇷 France ---
"Orange", # Orange S.A.
"SFR", # SFR / Numericable
"Bouygues Telecom", # Bbox
"Free", # Free / Iliad
"Iliad", # Free Mobile
"Proxad", # Free anciennement Proxad
"Nordnet", # Fournisseur secondaire
"Wanadoo", # Anciennement Orange

# --- 🇩🇪 Allemagne ---
"Deutsche Telekom",
"Vodafone Germany",
"Telefonica Germany",
"O2 Germany",
"1&1 Versatel",

# --- 🇬🇧 Royaume-Uni ---
"BT",
"Openreach",
"Sky Broadband",
"Virgin Media",
"TalkTalk",
"EE Limited",

# --- 🇪🇸 Espagne ---
"Telefonica",
"Movistar",
"Orange Spain",
"Vodafone Spain",
"MasMovil",

# --- 🇮🇹 Italie ---
"TIM",
"Telecom Italia",
"Vodafone Italia",
"Wind Tre",
"Fastweb",

# --- 🇳🇱 Pays-Bas ---
"KPN",
"Ziggo",
"T-Mobile Netherlands",
"Odido",

# --- 🇧🇪 Belgique ---
"Proximus",
"Telenet",
"Orange Belgium",

# --- 🇨🇭 Suisse ---
"Swisscom",
"Sunrise",
"Salt Mobile",

# --- 🇺🇸 États-Unis ---
"Comcast",
"Xfinity",
"AT&T",
"Verizon",
"Charter Communications",
"Spectrum",
"Cox Communications",
"Frontier Communications",
"CenturyLink",
"Windstream",

# --- 🇨🇦 Canada ---
"Bell Canada",
"Rogers Communications",
"Shaw Communications",
"Telus",
"Videotron",

# --- 🇦🇺 Australie ---
"Telstra",
"Optus",
"TPG Telecom",
"iiNet",

# --- 🇯🇵 Japon ---
"NTT",
"NTT Communications",
"SoftBank",
"KDDI",
"au by KDDI",

# --- 🇰🇷 Corée du Sud ---
"KT Corporation",
"SK Broadband",
"LG Uplus",

# --- 🇧🇷 Brésil ---
"Vivo",
"Claro Brasil",
"TIM Brasil",
"Oi"
)

residential_regex <- paste(residential_isp, collapse="|")

mult_map <- c(
  h = 3600,
  d = 24 * 3600,
  w = 7 * 24 * 3600,
  m = 30 * 24 * 3600,
  y = 365 * 24 * 3600
)

interval_map <- c(
  h = "hour",
  d = "day",
  w = "week",
  m = "month",
  y = "year"
)

cloud_asn_patterns <- c(

  # --- Hyperscalers ---
  "Amazon",
  "AWS",
  "Google",
  "Microsoft",
  "Azure",
  "Alibaba",
  "Tencent",
  "Oracle",
  "IBM Cloud",

  # --- Major Hosting / VPS ---
  "OVH",
  "OVHcloud",
  "DigitalOcean",
  "Hetzner",
  "Linode",
  "Vultr",
  "Scaleway",
  "UpCloud",
  "Contabo",
  "Leaseweb",
  "LeaseWeb",
  "Online SAS",
  "Ionos",
  "1&1",
  "GoDaddy",
  "Namecheap",
  "DreamHost",
  "Hostinger",
  "Bluehost",
  "SiteGround",
  "A2 Hosting",
  "HostGator",

  # --- CDN / Edge ---
  "Cloudflare",
  "Fastly",
  "Akamai",
  "StackPath",
  "Bunny.net",
  "CDN77",
  "Edgecast",
  "G-Core",
  "Imperva",

  # --- Cheap / Bot-heavy infra ---
  "Choopa",
  "ColoCrossing",
  "Quadranet",
  "Psychz",
  "Sharktech",
  "BuyVM",
  "M247",
  "Datacamp",
  "Hostwinds",
  "EUserv",

  # --- SEO / Crawlers infra ---
  "Babbar",
  "Ahrefs",
  "Semrush",
  "MJ12",
  "Majestic",
  "Dotbot",

  # --- Asian cloud infra ---
  "Huawei",
  "China Telecom",
  "China Unicom",
  "China Mobile",

  # --- Misc infra providers ---
  "Digital Realty",
  "Equinix",
  "CoreWeave",
  "Packet",
  "Vercel",
  "Heroku",
  "Render",
  "Fly.io"
)

cloud_asn_regex <- paste(cloud_asn_patterns, collapse = "|")

## Extract URL from a typical request string: "GET /path HTTP/1.1"
#extract_url <- function(request_col) {
#  # Works even if method differs, and avoids brittle space indexing
#  url <- str_match(request_col, '^"\\S+\\s+([^\\s]+)')[,2]
#  # fallback if malformed
#  ifelse(is.na(url), request_col, url)
#}

extract_url <- function(request_col) {
  sapply(strsplit(request_col, " "), function(x) {
    if (length(x) >= 2) x[2] else NA_character_
  })
}


geo_cache_path <- "geo_cache.rds"
asn_cache_path <- "asn_cache.rds"

load_geo_cache <- function() {
  if (file.exists(geo_cache_path)) {
    readRDS(geo_cache_path)
  } else {
    tibble(
      ip = character(),
      country = character(),
      city = character(),
      lat = double(),
      lon = double()
    )
  }
}

save_geo_cache <- function(cache) {
  saveRDS(cache, geo_cache_path)
}

load_asn_cache <- function() {
  if (file.exists(asn_cache_path)) {
    readRDS(asn_cache_path)
  } else {
    tibble(
      ip = character(),
      asn = integer(),
      asn_org = character()
    )
  }
}

save_asn_cache <- function(cache) {
  saveRDS(cache, asn_cache_path)
}

geo_db_path <- "geo/GeoLite2-City.mmdb"
asn_db_path <- "geo/GeoLite2-ASN.mmdb"

`%||%` <- function(a, b) if (!is.null(a)) a else b

lookup_ip <- function(ip, db_path) {

  safe_empty <- tibble(
    ip = ip,
    country = NA_character_,
    country_code = NA_character_,
    lat = NA_real_,
    lon = NA_real_,
    timezone = NA_character_
  )

  ##get_field <- function(...) {
  ##  res <- tryCatch(
  ##    system2(
  ##      "mmdblookup",
  ##      args = c("--file", db_path, "--ip", ip, ...),
  ##      stdout = TRUE,
  ##      stderr = FALSE
  ##    ),
  ##    error = function(e) NULL
  ##  )

  ##  #print(paste("IP:", ip))
  ##  #print(res)

  ##  if (is.null(res) || length(res) == 0) return(NA)

  ##  # Extract quoted value if present
  ##  line <- res[grepl('"', res)][1]
  ##  if (is.na(line)) return(NA)

  ##  sub('.*"([^"]+)".*', '\\1', line)
  ##}

  get_field <- function(...) {
  
    res <- tryCatch(
      system2(
        "mmdblookup",
        args = c("--file", db_path, "--ip", ip, ...),
        stdout = TRUE,
        stderr = FALSE
      ),
      error = function(e) NULL
    )
  
    if (is.null(res) || length(res) == 0) return(NA)
  
    # 1️⃣ Try quoted string (country, timezone)
    quoted_line <- grep('"', res, value = TRUE)
    if (length(quoted_line) > 0) {
      return(sub('.*"([^"]+)".*', '\\1', quoted_line[1]))
    }
  
    # 2️⃣ Try numeric double (lat/lon)
    double_line <- grep("<double>", res, value = TRUE)
    if (length(double_line) > 0) {
      # Extract numeric value before <double>
      value <- sub(" <double>.*", "", trimws(double_line[1]))
      return(value)
    }
  
    return(NA)
  }

  country <- get_field("country", "names", "en")
  country_code <- get_field("country", "iso_code")
  lat <- suppressWarnings(as.numeric(get_field("location", "latitude")))
  lon <- suppressWarnings(as.numeric(get_field("location", "longitude")))
  timezone <- get_field("location", "time_zone")

  tibble(
    ip = ip,
    country = country,
    country_code = country_code,
    lat = lat,
    lon = lon,
    timezone = timezone
  )
}

lookup_ips <- function(ips, db_path) {

  cache <- load_geo_cache()

  known_ips <- cache$ip
  new_ips <- setdiff(ips, known_ips)

  if (length(new_ips) > 0) {

    message("Looking up ", length(new_ips), " new IPs...")

    new_data <- lapply(new_ips, lookup_ip, db_path = db_path) %>%
      bind_rows()

    cache <- bind_rows(cache, new_data) %>%
      distinct(ip, .keep_all = TRUE)

    save_geo_cache(cache)
  }

  cache %>% filter(ip %in% ips)
}

lookup_asn_single <- function(ip, db_path) {

  get_field <- function(...) {
    res <- tryCatch(
      system2(
        "mmdblookup",
        args = c("--file", db_path, "--ip", ip, ...),
        stdout = TRUE,
        stderr = FALSE
      ),
      error = function(e) NULL
    )

    cat("RES: \n", res, "\n")

    if (is.null(res) || length(res) == 0) return(NA)

    uint_line <- grep("<uint32>", res, value = TRUE)
    if (length(uint_line) > 0) {
      # robust: keep only digits
      value <- gsub("[^0-9]", "", uint_line[1])
      return(value)
    }

    quoted_line <- grep('"', res, value = TRUE)
    if (length(quoted_line) > 0) {
      return(sub('.*"([^"]+)".*', '\\1', quoted_line[1]))
    }

    return(NA)
  }

  asn_number <- suppressWarnings(as.integer(get_field("autonomous_system_number")))
  asn_org    <- get_field("autonomous_system_organization")

  tibble(
    ip = ip,
    asn = asn_number,
    asn_org = asn_org
  )
}

lookup_asns <- function(ips, db_path) {

  cache <- load_asn_cache()

  known_ips <- cache$ip
  new_ips <- setdiff(ips, known_ips)

  if (length(new_ips) > 0) {

    message("Looking up ", length(new_ips), " new ASNs...")

    new_data <- lapply(new_ips, lookup_asn_single, db_path = db_path) %>%
      bind_rows()

    cache <- bind_rows(cache, new_data) %>%
      distinct(ip, .keep_all = TRUE) # keep_all are for columns

    save_asn_cache(cache)
  }

  cache %>% filter(ip %in% ips)
}


