library(shiny)
library(plotly)
library(lubridate)
library(bslib)
library(shinymanager)
library(shinycssloaders)
library(DT)
library(leaflet)
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
    data.table::as.data.table(
                        readRDS(geo_cache_path)
    )
  } else {
    data.table::data.table(
      ip = character(),
      country = character(),
      city = character(),
      lat = double(),
      lon = double()
    )
  }
}

save_geo_cache <- function(cache) {
  saveRDS(data.table::as.data.table(cache), 
          geo_cache_path)
}

load_asn_cache <- function() {
  if (file.exists(asn_cache_path)) {
    data.table::as.data.table(
                        readRDS(asn_cache_path)
    )
  } else {
    data.table::data.table(
      ip = character(),
      asn = integer(),
      asn_org = character()
    )
  }
}

save_asn_cache <- function(cache) {
  saveRDS(data.table::as.data.table(cache), 
          asn_cache_path)
}

geo_db_path <- "geo/GeoLite2-City.mmdb"
asn_db_path <- "geo/GeoLite2-ASN.mmdb"

lookup_ip <- function(ip, db_path) {

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

  data.table::data.table(
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

    new_data <- data.table::rbindlist(
        lapply(new_ips, lookup_ip, db_path = db_path), # or laply(new_ips, function(x) { lookup_ip(ip = x, db_path = db_path) })
        use.names = TRUE,
        fill = TRUE
    )

    cache <- data.table::rbindlist(
                            list(
                                 cache, 
                                 new_data
                                ),
                            use.names = TRUE, 
                            fill = TRUE
    )

    save_geo_cache(cache)
  }

  cache[ip %in% ips]
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

  data.table::data.table(
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

    new_data <- data.table::rbindlist(
        lapply(new_ips, lookup_asn_single, db_path = db_path),
        use.names = TRUE,
        fill = TRUE
    )

    cache <- data.table::rbindlist(list(
                                        cache, 
                                        new_data
                                        ), 
                                   use.names = TRUE, 
                                   fill = TRUE
    )

    save_asn_cache(cache)
  }

  cache[ip %in% ips]
}

clear_ip_caches <- function() {
  if (file.exists(geo_cache_path)) file.remove(geo_cache_path)
  if (file.exists(asn_cache_path)) file.remove(asn_cache_path)
}

file_path <- "/var/log/nginx/statix.log"

load_raw_data <- function(file_path) {
   
    df <- data.table::fread(input = file_path,
                      sep="\t",
                      quote = "\"",
                      col.names = c("ip", "ts", "target", "status", "ua"),
                      header = FALSE,
                      colClasses = list(
                                        character = c(1, 3, 5),
                                        double = 2,
                                        integer = 4
                                       ),
                      showProgress = FALSE
          ) 

    df[, date := as.POSIXct(ts, origin = "1970-01-01", tz = "UTC")]
    df <- df[, .(ip, date, target, status, ua)]
    df <- df[!is.na(date) & 
             !is.na(target) & 
             !is.na(status) & 
             status == 200]
    df[, status := NULL]
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

country_coords <- data.table::data.table(
  country = c(
    # Europe
    "France",
    "Germany",
    "United Kingdom",
    "Netherlands",
    "Belgium",
    "Switzerland",
    "Spain",
    "Italy",
    "Portugal",
    "Ireland",
    "Austria",
    "Poland",
    "Czechia",
    "Sweden",
    "Norway",
    "Denmark",
    "Finland",
    "Romania",
    "Bulgaria",
    "Greece",
    "Hungary",
    "Ukraine",
    "Russia",
    "Turkey",

    # North America
    "United States",
    "Canada",
    "Mexico",

    # South America
    "Brazil",
    "Argentina",
    "Chile",
    "Colombia",
    "Peru",
    "Venezuela",
    "Uruguay",
    "Trinidad and Tobago",

    # Asia
    "Singapore",
    "Japan",
    "India",
    "China",
    "Hong Kong",
    "Taiwan",
    "South Korea",
    "Indonesia",
    "Malaysia",
    "Thailand",
    "Vietnam",
    "Philippines",
    "Pakistan",
    "Bangladesh",
    "United Arab Emirates",
    "Saudi Arabia",
    "Israel",
    "Iran",

    # Oceania
    "Australia",
    "New Zealand",

    # Africa
    "South Africa",
    "Egypt",
    "Morocco",
    "Algeria",
    "Tunisia",
    "Nigeria",
    "Kenya",
    "Ethiopia",
    "Ghana",
    "Ivory Coast",
    "Angola",
    "Benin",
    "Botswana",
    "Burkina Faso",
    "Burundi",
    "Cameroon",
    "Cape Verde",
    "Central African Republic",
    "Chad",
    "Comoros",
    "Democratic Republic of the Congo",
    "Republic of the Congo",
    "Djibouti",
    "Equatorial Guinea",
    "Eritrea",
    "Eswatini",
    "Gabon",
    "Gambia",
    "Guinea",
    "Guinea-Bissau",
    "Lesotho",
    "Liberia",
    "Libya",
    "Madagascar",
    "Malawi",
    "Mali",
    "Mauritania",
    "Mauritius",
    "Mozambique",
    "Namibia",
    "Niger",
    "Rwanda",
    "São Tomé and Príncipe",
    "Senegal",
    "Seychelles",
    "Sierra Leone",
    "Somalia",
    "South Sudan",
    "Sudan",
    "Tanzania",
    "Togo",
    "Uganda",
    "Zambia",
    "Zimbabwe"
  ),

  country_lat = c(
    # Europe
    48.8566,
    52.5200,
    51.5074,
    52.3676,
    50.8503,
    46.9480,
    40.4168,
    41.9028,
    38.7223,
    53.3498,
    48.2082,
    52.2297,
    50.0755,
    59.3293,
    59.9139,
    55.6761,
    60.1699,
    44.4268,
    42.6977,
    37.9838,
    47.4979,
    50.4501,
    55.7558,
    39.9334,

    # North America
    38.9072,
    45.4215,
    19.4326,

    # South America
    -15.7939,
    -34.6037,
    -33.4489,
    4.7110,
    -12.0464,
    10.4806,
    -34.9011,
    10.6549,

    # Asia
    1.3521,
    35.6762,
    28.6139,
    39.9042,
    22.3193,
    25.0330,
    37.5665,
    -6.2088,
    3.1390,
    13.7563,
    21.0278,
    14.5995,
    33.6844,
    23.8103,
    24.4539,
    24.7136,
    31.7683,
    35.6892,

    # Oceania
    -35.2809,
    -41.2865,

    # Africa
    -25.7479,
    30.0444,
    34.0209,
    36.7538,
    36.8065,
    9.0765,
    -1.2921,
    9.0300,
    5.6037,
    5.3600,
    -8.8390,
    6.4969,
    -24.6282,
    12.3714,
    -3.3614,
    3.8480,
    14.9330,
    4.3947,
    12.1348,
    -11.7172,
    -4.4419,
    -4.2634,
    11.5721,
    3.7500,
    15.3229,
    -26.3054,
    0.4162,
    13.4549,
    9.6412,
    11.8817,
    -29.3158,
    6.3156,
    32.8872,
    -18.8792,
    -13.9626,
    12.6392,
    18.0735,
    -20.1609,
    -25.9692,
    -22.5609,
    13.5116,
    -1.9441,
    0.3365,
    14.7167,
    -4.6191,
    8.4657,
    2.0469,
    4.8594,
    15.5007,
    -6.1630,
    6.1725,
    0.3476,
    -15.3875,
    -17.8292
  ),

  country_lon = c(
    # Europe
    2.3522,
    13.4050,
    -0.1278,
    4.9041,
    4.3517,
    7.4474,
    -3.7038,
    12.4964,
    -9.1393,
    -6.2603,
    16.3738,
    21.0122,
    14.4378,
    18.0686,
    10.7522,
    12.5683,
    24.9384,
    26.1025,
    23.3219,
    23.7275,
    19.0402,
    30.5234,
    37.6173,
    32.8597,

    # North America
    -77.0369,
    -75.6972,
    -99.1332,

    # South America
    -47.8828,
    -58.3816,
    -70.6693,
    -74.0721,
    -77.0428,
    -66.9036,
    -56.1645,
    -61.5019,

    # Asia
    103.8198,
    139.6503,
    77.2090,
    116.4074,
    114.1694,
    121.5654,
    126.9780,
    106.8456,
    101.6869,
    100.5018,
    105.8342,
    120.9842,
    73.0479,
    90.4125,
    54.3773,
    46.6753,
    35.2137,
    51.3890,

    # Oceania
    149.1300,
    174.7762,

    # Africa
    28.2293,
    31.2357,
    -6.8416,
    3.0588,
    10.1815,
    7.3986,
    36.8219,
    38.7400,
    -0.1870,
    -4.0083,
    13.2894,
    2.6289,
    25.9231,
    -1.5197,
    29.3599,
    11.5021,
    -23.5133,
    18.5582,
    15.0557,
    43.2473,
    15.2663,
    15.2429,
    43.1456,
    8.7833,
    38.9251,
    31.1367,
    9.4673,
    -16.5790,
    -13.5784,
    -15.6170,
    27.4869,
    -10.8074,
    13.1913,
    47.5079,
    33.7741,
    -8.0029,
    -15.9582,
    57.5012,
    32.5732,
    17.0658,
    2.1254,
    30.0619,
    6.7273,
    -17.4677,
    55.4513,
    -13.2317,
    45.3182,
    31.5713,
    32.5599,
    35.7516,
    1.2314,
    32.5825,
    28.3228,
    31.0522
  )
)



