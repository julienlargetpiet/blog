function(input, output, session) {

  res_auth <- secure_server(
    check_credentials = check_credentials(credentials)
  )

  observeEvent(input$time_unit, ignoreInit = TRUE, {
    updateSelectInput(session, "time_unit_2", selected = input$time_unit)
  })
  observeEvent(input$time_unit_2, ignoreInit = TRUE, {
    updateSelectInput(session, "time_unit", selected = input$time_unit_2)
  })

  observeEvent(input$last_n, ignoreInit = TRUE, {
    updateNumericInput(session, "last_n_2", value = input$last_n)
  })
  
  observeEvent(input$last_n_2, ignoreInit = TRUE, {
    updateNumericInput(session, "last_n", value = input$last_n_2)
  })

  observe({
    session$sendCustomMessage("getTimezone", list())
  })

  #observeEvent(input$dark_mode, ignoreInit = TRUE, {
  #
  #  session$setCurrentTheme(
  #    if (isTRUE(input$dark_mode)) {
  #      bs_theme(
  #        version = 5,
  #        bootswatch = "darkly",
  #        base_font = font_google("Nunito"),
  #        code_font = font_google("Nunito")
  #      )
  #    } else {
  #      bs_theme(
  #        version = 5,
  #        bootswatch = "litera",
  #        base_font = font_google("Nunito"),
  #        code_font = font_google("Nunito")
  #      )
  #    }
  #  )
  #
  #})

  theme_reactive <- reactive({
    if (isTRUE(input$dark_mode)) {
      bs_theme(
        version = 5,
        bootswatch = "darkly",
        base_font = font_google("Nunito"),
        code_font = font_google("Nunito")
      )
    } else {
      bs_theme(
        version = 5,
        bootswatch = "litera",
        base_font = font_google("Nunito"),
        code_font = font_google("Nunito")
      )
    }
  })

  observeEvent(input$dark_mode, {
    session$setCurrentTheme(theme_reactive())
  }, ignoreInit = TRUE)

  file_path <- "/var/log/nginx/access.log"

  raw_data <- reactive({
    fp <- file_path
    req(!is.null(fp))
  
    df <- read_delim(
      fp,
      delim = " ",
      quote = '"',
      col_names = FALSE,
      trim_ws = TRUE,
      progress = FALSE,
      col_types = cols(
        .default = col_character(),
        X7 = col_double(),
        X8 = col_double()
      )
    )
  
    # Safety: ensure we have enough columns
    req(ncol(df) >= 2)
  
    parsed <- tibble(
      ip = df[[1]],
      date_raw = paste(df[[4]], df[[5]]),
      request_raw = df[[6]],
      ua = df[[ncol(df) - 1]],        # second to last
      x_prefetch = df[[ncol(df)]]     # last column
    )
  
    parsed <- parsed %>%
      mutate(
        date = as.POSIXct(
          gsub("\\[|\\]", "", date_raw),
          format = "%d/%b/%Y:%H:%M:%S %z",
          tz = "UTC"
        ),
        target = extract_url(request_raw),
        is_prefetch = x_prefetch == "1"
      ) %>%
      select(ip, date, target, ua, is_prefetch)
  
    parsed %>% 
      filter(!is.na(date), !is.na(target))
  })

  filtered_data <- reactive({

    req(input$time_unit, input$last_n)

    df <- raw_data()
    req(nrow(df) > 0)
  
    # -----------------------------
    # BOT DETECTION
    # -----------------------------
    if (!isTRUE(input$show_bots)) {
    
      # 1️⃣ Remove prefetch first
      df <- df %>%
        filter(!is_prefetch)
    
      # 2️⃣ UA detection
      df <- df %>%
        mutate(is_bot_ua = grepl(bot_regex, ua,
                                 ignore.case = TRUE,
                                 perl = TRUE))
    
      # 3️⃣ Asset heuristic
      df <- df %>%
        group_by(ip) %>%
        mutate(
          total_requests = n(),
          html_requests = sum(grepl("\\.html$|/$", target)),
          asset_ratio = html_requests / total_requests,
          is_bot_asset = asset_ratio > 0.9
        ) %>%
        ungroup()
    
      # 4️⃣ Rate heuristic
      df <- df %>%
        group_by(ip, sec = floor_date(date, "second")) %>%
        mutate(req_per_sec = n()) %>%
        ungroup() %>%
        mutate(is_bot_rate = req_per_sec > 10)
   
      # 5️⃣ Reading-time heuristic (aggressive)
      df <- df %>%
        arrange(ip, date) %>%
        group_by(ip) %>%
        mutate(
          next_date = lead(date),
          time_on_page = as.numeric(difftime(next_date, date, units = "secs")),
          is_article = grepl(
            "^/articles/[^/]*[A-Za-z][^/]*\\.html$",
            target
          ),
          is_bot_readtime = is_article &
                            !is.na(time_on_page) &
                            time_on_page < 30
        ) %>%
        ungroup()

      cat("\n OK \n")

      # 6 Final bot flag
      df <- df %>%
        mutate(is_bot = is_bot_ua | is_bot_rate | is_bot_asset | is_bot_readtime) %>%
        filter(!is_bot) %>%
        select(-is_bot_ua, -req_per_sec, -is_bot_rate,
               -is_bot_asset, -asset_ratio,
               -total_requests, -html_requests,
               -next_date,
               -is_article, -is_bot_readtime,
               -is_bot)
    }
    # -----------------------------
    # STATIC ASSET FILTER
    # -----------------------------
    if (!isTRUE(input$show_static)) {
      df <- df %>%
        filter(!grepl(
          "\\.(css|js|png|jpg|jpeg|gif|svg|ico|woff2?|ttf)(\\?|$)",
          target,
          ignore.case = TRUE
        ))
    }

    df <- df %>%
      mutate(target = sub("\\?.*$", "", target),
	     target = trimws(target))

    article_regex <- "^/articles/[^/]*[A-Za-z][^/]*\\.html$"
    
    df <- df %>%
      filter(
        grepl(article_regex, target) |
        (!input$only_articles & target == "/")
      )

    if (length(ip_exclude) > 0 && any(nzchar(ip_exclude))) {
      ip_exclude_regex <- paste(ip_exclude[nzchar(ip_exclude)], collapse="|")
      df <- df %>% filter(!grepl(ip_exclude_regex, ip))
    }

    # -----------------------------
    # TIME WINDOW FILTER
    # -----------------------------
    last <- input$last_n * mult_map[[input$time_unit]]

    if (nrow(df) == 0) return(df)
    cutoff <- max(df$date) - last
  
    # --- ASN enrichment (minimal) ---
    ips <- sort(unique(df$ip))
  
    asn_data <- lookup_asns(
      ips,
      db_path = asn_db_path
    )

    df <- df %>% left_join(asn_data, by = "ip")

    df <- df %>%
      group_by(ip) %>%
      mutate(total_requests = n()) %>%
      ungroup() %>%
      mutate(
        is_cloud_asn = grepl(cloud_asn_regex, asn_org, ignore.case = TRUE),
        is_bot_asn = is_cloud_asn & total_requests > 3,
        is_residential = grepl(residential_regex, asn_org, ignore.case = TRUE)
    )

    df <- switch(
      input$strict,
      "low" = df,
      "medium" = df %>% filter(!is_bot_asn),
      "high" = df %>% filter(!is_cloud_asn),
      "very_high" = df %>% filter(is_residential)
    )

    df %>% filter(date >= cutoff)

  })  
 
  geo_cache_reactive <- reactiveVal(NULL)
  last_ips <- reactiveVal(character())
  
  observeEvent(filtered_data(), {
  
    ips <- sort(unique(filtered_data()$ip))
  
    if (!identical(ips, last_ips())) {
  
      geo_data <- lookup_ips(
        ips,
        db_path = geo_db_path
      )
  
      geo_cache_reactive(geo_data)
      last_ips(ips)
    }
  
  }, ignoreInit = FALSE)

  geo_enriched_data <- reactive({
 
    cat("GEO_ENRICHED_DATA CALLED\n")

    df  <- filtered_data()
    geo <- geo_cache_reactive()
  
    if (!is.null(geo)) {
      df <- df %>% left_join(geo, by = "ip")
    }
  
  })

  # KPIs
  output$kpi_hits <- renderText({
    df <- filtered_data()
    format(nrow(df), big.mark = " ")
  })

  output$kpi_ips <- renderText({
    df <- filtered_data()
    format(dplyr::n_distinct(df$ip), big.mark = " ")
  })

  output$kpi_pages <- renderText({
    df <- filtered_data()
    format(dplyr::n_distinct(df$target), big.mark = " ")
  })

  article_readtime_stats <- reactive({
  
    df <- filtered_data()
    req(nrow(df) > 0)
  
    df %>%
      mutate(valid_read = !is.na(time_on_page) &
                          time_on_page > 0 &
                          time_on_page < 1500) %>%
      group_by(target) %>%
      summarise(
        median_readtime = median(time_on_page[valid_read], na.rm = TRUE),
        valid_reads = sum(valid_read),
        .groups = "drop"
      ) %>%
      filter(valid_reads > 0) %>%   
      arrange(desc(median_readtime))
  })

  output$kpi_med_readtime <- renderText({
  
    df <- filtered_data()
  
    req(nrow(df) > 0)
  
    median_time <- df %>%
      filter(
        !is.na(time_on_page),
        time_on_page > 0,
        time_on_page < 1500   # safety cap (1 hour max)
      ) %>%
      summarise(med = median(time_on_page)) %>%
      pull(med)
  
    if (is.na(median_time)) return("—")
  
    mins <- floor(median_time / 60)
    secs <- round(median_time %% 60)
  
    sprintf("%02d:%02d", mins, secs)
  })

  # Pie chart
  output$pie_chart <- renderPlotly({

    input$dark_mode

    df <- filtered_data()
    req(nrow(df) > 0)

    agg <- df %>%
      count(target, name = "hits") %>%
      arrange(desc(hits))

    topn <- 5
    top <- head(agg, topn)

    if (nrow(agg) > topn) {
      other_hits <- sum(agg$hits[(topn + 1):nrow(agg)])
      top <- bind_rows(top,
                       tibble(target = "Other",
                              hits = other_hits))
    }

    dark <- isTRUE(input$dark_mode)

    plot_ly(
      data = top,
      labels = ~target,
      values = ~hits,
      type = "pie",
      textinfo = "label+percent",
      insidetextorientation = "radial"
    ) %>%
      layout(
        template = if (dark) "plotly_dark" else "plotly_white",
        title = list(
          text = "Most visited targets (Top 5 + Other)"
        ),
        paper_bgcolor = "transparent",
        plot_bgcolor  = "transparent",
        showlegend = TRUE
      )
  })

  # ✅ FIXED REGEX GROUPING LOGIC
  output$graph <- renderPlotly({

    input$dark_mode

    df <- filtered_data()
    req(nrow(df) > 0)

    patterns <- input$webpages

    if (!is.null(patterns) && nzchar(patterns)) {

      pats <- strsplit(patterns, "--", fixed = TRUE)[[1]]

      # Create empty grouping column
      df$target_group <- NA_character_

      # FIRST MATCH WINS (no reassignment)
      for (p in pats) {
        idx <- is.na(df$target_group) & grepl(p, df$target)
        df$target_group[idx] <- p
      }

      # Remove rows that matched none
      df <- df[!is.na(df$target_group), ]

      req(nrow(df) > 0)

    } else {
      df$target_group <- df$target
    }

    interval <- interval_map[[input$time_unit]]

    df <- df %>%
      mutate(date_bucket = floor_date(date, unit = interval)) %>%
      count(target_group, date_bucket, name = "hits")

    dark <- isTRUE(input$dark_mode)
   
    text_col <- if (dark) "#F5F5F5" else "#000000"
    grid_col <- if (dark) "#333333" else "#E5E5E5"

    plot_ly(
      data = df,
      x = ~date_bucket,
      y = ~hits,
      color = ~target_group,
      type = "scatter",
      mode = "lines+markers"
    ) %>%
      layout(
        template = "none",  # 🔥 critical

        title = list(
          text = "Traffic by URL (regex buckets — first match wins)",
          font = list(color = text_col)
        ),

        paper_bgcolor = "rgba(0,0,0,0)",
        plot_bgcolor  = "rgba(0,0,0,0)",

        font = list(color = text_col),

        legend = list(
          font = list(color = text_col),
          bgcolor = "rgba(0,0,0,0)"
        ),

        xaxis = list(
          title = list(text = "Date", font = list(color = text_col)),
          tickfont = list(color = text_col),
          gridcolor = grid_col
        ),

        yaxis = list(
          title = list(text = "Number of requests", font = list(color = text_col)),
          tickfont = list(color = text_col),
          gridcolor = grid_col
        )
    )  
  })

  output$mytable <- renderDT({
    df <- geo_enriched_data()
    req(input$client_tz)
  
    df <- df %>%
      mutate(
        date = lubridate::with_tz(date, tzone = input$client_tz),
        date = format(date, "%Y-%m-%d %H:%M:%S")
      )
  
    datatable(
      df %>% 
        arrange(desc(date)) %>% 
        mutate(target = paste0(
          '<a href=\"https://domain.com', 
          target,
          '\" target=\"_blank\">',
          target,
          "</a>"
        )) %>%
        select(country, asn_org, ip, date, target, time_on_page),
      options = list(
        pageLength = 100,
        scrollX = TRUE,
        ordering = TRUE
      ),
      rownames = FALSE,
      escape = FALSE
    )
  })

  output$read_time <- renderDT({
  
    stats <- article_readtime_stats()
    req(nrow(stats) > 0)
  
    stats <- stats %>%
      mutate(
        median_seconds = median_readtime,
        median_readtime = sprintf(
          "%02d:%02d",
          floor(median_readtime / 60),
          round(median_readtime %% 60)
        )
      )
  
    datatable(
      stats %>%
        select(target, median_readtime, valid_reads),
      options = list(
        pageLength = 20
      ),
      rownames = FALSE
    )
  })

  output$map <- renderLeaflet({
  
    df <- geo_enriched_data()
    req(nrow(df) > 0)
  
    df <- df %>%
      filter(!is.na(lat), !is.na(lon))
  
    req(nrow(df) > 0)
  
    agg <- df %>%
      group_by(country) %>%
      summarise(
        hits = n(),
        unique_ips = n_distinct(ip),
        lat = mean(lat),
        lon = mean(lon),
        .groups = "drop"
      )
  
    dark <- isTRUE(input$dark_mode)
  
    leaflet(agg) %>%
      addProviderTiles(
        if (dark)
          providers$CartoDB.DarkMatter
        else
          providers$CartoDB.Positron
      ) %>%
      setView(lng = 0, lat = 20, zoom = 2) %>%
      addCircleMarkers(
        lng = ~lon,
        lat = ~lat,
        radius = ~pmin(25, pmax(5, sqrt(hits) * 3)),
        stroke = FALSE,
        fillOpacity = 0.75,
        popup = ~paste0(
          "<b>Country:</b> ", country, "<br>",
          "<b>Total hits:</b> ", hits, "<br>",
          "<b>Unique IPs:</b> ", unique_ips
        ),
        clusterOptions = if (isTRUE(input$map_cluster))
          markerClusterOptions()
        else NULL
      )
  
  })

}



