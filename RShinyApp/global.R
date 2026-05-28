library(shiny)
library(plotly)
library(dplyr)
library(lubridate)
library(bslib)
library(shinymanager)
library(shinycssloaders)
library(DT)
library(stringr)
library(scales)
library(leaflet)
library(purrr)
library(shinyjs)
library(data.table)

cat("GLOBAL LOADED\n")

log_step <- function(name, start, df = NULL) {
  elapsed <- as.numeric(difftime(Sys.time(), start, units = "secs"))

  if (!is.null(df)) {
    cat(sprintf("[filtered_data] %-25s %.4f sec | rows: %s\n",
                name,
                elapsed,
                format(nrow(df), big.mark = " ")))
  } else {
    cat(sprintf("[filtered_data] %-25s %.4f sec\n",
                name,
                elapsed))
  }
}

credentials <- data.frame(
  user = "admin",
  password = "PASSWORD",
  admin = TRUE,
  stringsAsFactors = FALSE
)

Sys.setlocale("LC_TIME", "C")
options(shiny.maxRequestSize = 300 * 1024^2)

ip_exclude <- c("86.242.190.96")

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
  "Vercel",
  "Heroku",
  "Render",
  "Fly.io",
  "CoreWeave"
)

#cloud_asn_patterns <- c(
#
#  # --- Hyperscalers ---
#  "Amazon",
#  "AWS",
#  "Google",
#  "Microsoft",
#  "Azure",
#  "Alibaba",
#  "Tencent",
#  "Oracle",
#  "IBM Cloud",
#
#  # --- Major Hosting / VPS ---
#  "OVH",
#  "OVHcloud",
#  "DigitalOcean",
#  "Hetzner",
#  "Linode",
#  "Vultr",
#  "Scaleway",
#  "UpCloud",
#  "Contabo",
#  "Leaseweb",
#  "LeaseWeb",
#  "Online SAS",
#  "Ionos",
#  "1&1",
#  "GoDaddy",
#  "Namecheap",
#  "DreamHost",
#  "Hostinger",
#  "Bluehost",
#  "SiteGround",
#  "A2 Hosting",
#  "HostGator",
#
#  # --- CDN / Edge ---
#  "Cloudflare",
#  "Fastly",
#  "Akamai",
#  "StackPath",
#  "Bunny.net",
#  "CDN77",
#  "Edgecast",
#  "G-Core",
#  "Imperva",
#
#  # --- Cheap / Bot-heavy infra ---
#  "Choopa",
#  "ColoCrossing",
#  "Quadranet",
#  "Psychz",
#  "Sharktech",
#  "BuyVM",
#  "M247",
#  "Datacamp",
#  "Hostwinds",
#  "EUserv",
#
#  # --- SEO / Crawlers infra ---
#  "Babbar",
#  "Ahrefs",
#  "Semrush",
#  "MJ12",
#  "Majestic",
#  "Dotbot",
#
#  # --- Asian cloud infra ---
#  "Huawei",
#  "China Telecom",
#  "China Unicom",
#  "China Mobile",
#
#  # --- Misc infra providers ---
#  "Digital Realty",
#  "Equinix",
#  "CoreWeave",
#  "Packet",
#  "Vercel",
#  "Heroku",
#  "Render",
#  "Fly.io"
#)

cloud_asn_regex <- paste(cloud_asn_patterns, collapse = "|")

## Extract URL from a typical request string: "GET /path HTTP/1.1"

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

lookup_ip <- function(ip, db_path) {

  safe_empty <- tibble(
    ip = ip,
    country = NA_character_,
    country_code = NA_character_,
    lat = NA_real_,
    lon = NA_real_,
    timezone = NA_character_
  )

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
  
    quoted_line <- grep('"', res, value = TRUE)
    if (length(quoted_line) > 0) {
      return(sub('.*"([^"]+)".*', '\\1', quoted_line[1]))
    }
  
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

clear_ip_caches <- function() {
  if (file.exists(geo_cache_path)) file.remove(geo_cache_path)
  if (file.exists(asn_cache_path)) file.remove(asn_cache_path)
}

file_path <- "/var/log/nginx/statix.log"

load_raw_data <- function(file_path) {
   
  #readr::read_tsv(
  #  file_path,
  #  col_names = c("ip", "ts", "target", "status", "ua"),
  #  col_types = readr::cols(
  #    ip = readr::col_character(),
  #    ts = readr::col_double(),
  #    target = readr::col_character(),
  #    status = readr::col_integer(),
  #    ua = readr::col_character()
  #  ),
  #  progress = FALSE
  #) %>%
  #  mutate(
  #    date = as.POSIXct(ts, origin = "1970-01-01", tz = "UTC")
  #  ) %>%
  #  select(ip, date, target, status, ua) %>%
  #  filter(
  #    !is.na(date),
  #    !is.na(target),
  #    !is.na(status),
  #    status == 200
  #  ) %>%
  #  select(-status)

    data.table::fread(input = file_path,
                      sep="\t",
                      col.names = c("ip", "ts", "target", "status", "ua"),
                      header = FALSE,
                      colClasses = list(
                                        character = c(1, 3, 5),
                                        integer = 2,
                                        double = 4
                                       ),
                      showProgress = FALSE
                ) %>%
                 mutate(
                   date = as.POSIXct(ts, origin = "1970-01-01", tz = "UTC")
                 ) %>%
                 select(ip, date, target, status, ua) %>%
                 filter(
                   !is.na(date),
                   !is.na(target),
                   !is.na(status),
                   status == 200
                 ) %>%
                 select(-status)

}

honey_pots <- c(
    "/articles/initialize-a-model-and-tokenizer.html",
    "/articles/ai-agents-in-2026-revolutionizing-industries.html",
    "/articles/the-future-of-autonomous-systems.html",
    "/articles/limits-of-artificial-intelligence.html",
    "/articles/use-of-for-subheaders.html",
    "/articles/llms-and-the-future-of-software-engineering.html",
    "/articles/autonomous-systems-the-future-of-transportation-and-beyond.html",
    "/articles/how-llms-change-software-engineering.html",
    "/articles/bold-and-italic-supported.html",
    "/articles/image-links-are-not-allowed-use-image-description-instead.html",
    "/articles/no-in-the-text.html",
    "/articles/use-of-emojis-to-enhance-readability.html",
    "/articles/threading-and-performance-in-ai-inference-engines.html",
    "/articles/a-minimum-of-3-subheadings.html",
    "/articles/title-must-be-the-death-of-traditional-blogging.html",
    "/articles/understanding-kv-cache-and-memory-bottlenecks.html",
    "/articles/dfdfdf.html",
    "/articles/can-ai-systems-be-exploited.html",
    "/articles/and-header-will-be.html",
    "/articles/this-is-a-valid-markdown-header.html",
    "/articles/and-for-all-other-headers.html",
    "/articles/journal-april-2026.html",
    "/articles/clap-de-fin.html",
    "/articles/header.html"
)

country_coords <- tibble::tribble(
  ~country, ~country_lat, ~country_lon,

  # Europe
  "France", 48.8566, 2.3522,
  "Germany", 52.5200, 13.4050,
  "United Kingdom", 51.5074, -0.1278,
  "Netherlands", 52.3676, 4.9041,
  "Belgium", 50.8503, 4.3517,
  "Switzerland", 46.9480, 7.4474,
  "Spain", 40.4168, -3.7038,
  "Italy", 41.9028, 12.4964,
  "Portugal", 38.7223, -9.1393,
  "Ireland", 53.3498, -6.2603,
  "Austria", 48.2082, 16.3738,
  "Poland", 52.2297, 21.0122,
  "Czechia", 50.0755, 14.4378,
  "Sweden", 59.3293, 18.0686,
  "Norway", 59.9139, 10.7522,
  "Denmark", 55.6761, 12.5683,
  "Finland", 60.1699, 24.9384,
  "Romania", 44.4268, 26.1025,
  "Bulgaria", 42.6977, 23.3219,
  "Greece", 37.9838, 23.7275,
  "Hungary", 47.4979, 19.0402,
  "Ukraine", 50.4501, 30.5234,
  "Russia", 55.7558, 37.6173,
  "Turkey", 39.9334, 32.8597,

  # North America
  "United States", 38.9072, -77.0369,
  "Canada", 45.4215, -75.6972,
  "Mexico", 19.4326, -99.1332,

  # South America
  "Brazil", -15.7939, -47.8828,
  "Argentina", -34.6037, -58.3816,
  "Chile", -33.4489, -70.6693,
  "Colombia", 4.7110, -74.0721,
  "Peru", -12.0464, -77.0428,
  "Venezuela", 10.4806, -66.9036,
  "Uruguay", -34.9011, -56.1645,
  "Trinidad and Tobago", 10.6549, -61.5019,

  # Asia
  "Singapore", 1.3521, 103.8198,
  "Japan", 35.6762, 139.6503,
  "India", 28.6139, 77.2090,
  "China", 39.9042, 116.4074,
  "Hong Kong", 22.3193, 114.1694,
  "Taiwan", 25.0330, 121.5654,
  "South Korea", 37.5665, 126.9780,
  "Indonesia", -6.2088, 106.8456,
  "Malaysia", 3.1390, 101.6869,
  "Thailand", 13.7563, 100.5018,
  "Vietnam", 21.0278, 105.8342,
  "Philippines", 14.5995, 120.9842,
  "Pakistan", 33.6844, 73.0479,
  "Bangladesh", 23.8103, 90.4125,
  "United Arab Emirates", 24.4539, 54.3773,
  "Saudi Arabia", 24.7136, 46.6753,
  "Israel", 31.7683, 35.2137,
  "Iran", 35.6892, 51.3890,

  # Oceania
  "Australia", -35.2809, 149.1300,
  "New Zealand", -41.2865, 174.7762,

  # Africa
  "South Africa", -25.7479, 28.2293,
  "Egypt", 30.0444, 31.2357,
  "Morocco", 34.0209, -6.8416,
  "Algeria", 36.7538, 3.0588,
  "Tunisia", 36.8065, 10.1815,
  "Nigeria", 9.0765, 7.3986,
  "Kenya", -1.2921, 36.8219,
  "Ethiopia", 9.0300, 38.7400,
  "Ghana", 5.6037, -0.1870,
  "Ivory Coast", 5.3600, -4.0083,

  "Angola", -8.8390, 13.2894,
  "Benin", 6.4969, 2.6289,
  "Botswana", -24.6282, 25.9231,
  "Burkina Faso", 12.3714, -1.5197,
  "Burundi", -3.3614, 29.3599,
  "Cameroon", 3.8480, 11.5021,
  "Cape Verde", 14.9330, -23.5133,
  "Central African Republic", 4.3947, 18.5582,
  "Chad", 12.1348, 15.0557,
  "Comoros", -11.7172, 43.2473,
  "Democratic Republic of the Congo", -4.4419, 15.2663,
  "Republic of the Congo", -4.2634, 15.2429,
  "Djibouti", 11.5721, 43.1456,
  "Equatorial Guinea", 3.7500, 8.7833,
  "Eritrea", 15.3229, 38.9251,
  "Eswatini", -26.3054, 31.1367,
  "Gabon", 0.4162, 9.4673,
  "Gambia", 13.4549, -16.5790,
  "Guinea", 9.6412, -13.5784,
  "Guinea-Bissau", 11.8817, -15.6170,
  "Lesotho", -29.3158, 27.4869,
  "Liberia", 6.3156, -10.8074,
  "Libya", 32.8872, 13.1913,
  "Madagascar", -18.8792, 47.5079,
  "Malawi", -13.9626, 33.7741,
  "Mali", 12.6392, -8.0029,
  "Mauritania", 18.0735, -15.9582,
  "Mauritius", -20.1609, 57.5012,
  "Mozambique", -25.9692, 32.5732,
  "Namibia", -22.5609, 17.0658,
  "Niger", 13.5116, 2.1254,
  "Rwanda", -1.9441, 30.0619,
  "São Tomé and Príncipe", 0.3365, 6.7273,
  "Senegal", 14.7167, -17.4677,
  "Seychelles", -4.6191, 55.4513,
  "Sierra Leone", 8.4657, -13.2317,
  "Somalia", 2.0469, 45.3182,
  "South Sudan", 4.8594, 31.5713,
  "Sudan", 15.5007, 32.5599,
  "Tanzania", -6.1630, 35.7516,
  "Togo", 6.1725, 1.2314,
  "Uganda", 0.3476, 32.5825,
  "Zambia", -15.3875, 28.3228,
  "Zimbabwe", -17.8292, 31.0522
)





