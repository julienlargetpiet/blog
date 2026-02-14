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


