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

credentials <- data.frame(
  user = c("admin"),
  password = c("adminpass"),
  admin = c(TRUE),
  stringsAsFactors = FALSE
)

Sys.setlocale("LC_TIME", "C")
options(shiny.maxRequestSize = 300 * 1024^2)

bot_keywords <- c(
  "bot","spider","crawler","curl","wget","python","scrapy",
  "ahrefs","ahrefsbot","semrush","mj12","dotbot",
  "googlebot","bingbot","yandex","uptime","pingdom","monitor",
  "facebookexternalhit","slurp","baiduspider"
)
bot_pat <- paste(bot_keywords, collapse = "|")

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

geo_db_path <- "geo/GeoLite2-City.mmdb"

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


