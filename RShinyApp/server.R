function(input, output, session) {

  res_auth <- secure_server(
    check_credentials = check_credentials(credentials),
    keep_token=TRUE
  )

  observeEvent(input$time_unit, ignoreInit = TRUE, {
    updateSelectInput(session, "time_unit", selected = input$time_unit)
  })

  observeEvent(input$last_n, ignoreInit = TRUE, {
    updateNumericInput(session, "last_n", value = input$last_n)
  })

  mmdb_bump <- reactiveVal(0)

  observeEvent(input$upload_asn_mmdb, {
    req(input$upload_asn_mmdb)
  
    src <- input$upload_asn_mmdb$datapath
    dst <- asn_db_path
  
    ok <- file.copy(src, dst, overwrite = TRUE)
    if (!ok) {
      showNotification(paste("Failed to write:", dst), type = "error")
      return()
    }

    clear_ip_caches()
    geo_cache_reactive(NULL)
    last_ips(character())

    mmdb_bump(mmdb_bump() + 1)
    
    showNotification("ASN DB uploaded and installed.", type = "message")

  })
  
  observeEvent(input$upload_city_mmdb, {
    req(input$upload_city_mmdb)
  
    src <- input$upload_city_mmdb$datapath
    dst <- geo_db_path
  
    ok <- file.copy(src, dst, overwrite = TRUE)
    if (!ok) {
      showNotification(paste("Failed to write:", dst), type = "error")
      return()
    }
  
    clear_ip_caches()
    geo_cache_reactive(NULL)
    last_ips(character())

    mmdb_bump(mmdb_bump() + 1)

    showNotification("City DB uploaded and installed.", type = "message")

  })

  output$mmdb_status <- renderUI({

    mmdb_bump()
    asn_ok  <- file.exists(asn_db_path)
    city_ok <- file.exists(geo_db_path)
  
    tags$div(
      tags$small(
        HTML(paste0(
          "<b>ASN DB:</b> ", if (asn_ok) "✅ present" else "❌ missing",
          "<br><b>City DB:</b> ", if (city_ok) "✅ present" else "❌ missing"
        ))
      )
    )
  })

  observe({
    session$sendCustomMessage("getTimezone", list())
  })
  
  filtered_data <- reactive({

    mmdb_bump()

    req(input$time_unit, input$last_n)

    df <- raw_data_static
    req(nrow(df) > 0)

    t <- Sys.time()

    # -----------------------------
    # TIME WINDOW FILTER
    # -----------------------------
    last <- input$last_n * mult_map[[input$time_unit]]
    cutoff <- max(df$date) - last
  
    df <- df %>% filter(date >= cutoff)

    log_step("Time Window", t, df)
    t <- Sys.time()

    ua_unique <- unique(df$ua)
    
    ua_is_bot <- setNames(
      grepl(
        bot_regex,
        ua_unique,
        ignore.case = TRUE,
        perl = TRUE
      ),
      ua_unique
    )
    
    df <- df %>%
      filter(!ua_is_bot[ua])
    
    log_step("UA AGENT", t, df)
    t <- Sys.time()

    # Asset heuristic

    css_clients <- df %>% 
            filter(endsWith(tolower(target), ".css")) %>%
            distinct(ip) %>%
            pull(ip)

    df <- df %>% filter(ip %in% css_clients)

    log_step("Asset heuristic", t, df)
    t <- Sys.time()

    if (nrow(df) == 0) return(df)

    df <- df %>%
      filter(grepl("^/articles/.*\\.html$", target, ignore.case=TRUE))

    log_step("Aticle filtering", t, df)
    t <- Sys.time()

    if (nrow(df) == 0) return(df)

    # Rate heuristic
    df <- df %>%
      group_by(ip, sec = floor_date(date, "second")) %>%
      mutate(req_per_sec = n()) %>%
      filter(req_per_sec < 10) %>%
      ungroup() %>%
      select(-req_per_sec)

    log_step("Rate heuristic", t, df)
    t <- Sys.time()

    if (nrow(df) == 0) return(df)

    # Reading-time heuristic
    df <- df %>%
      arrange(ip, date) %>%
      group_by(ip) %>%
      mutate(
        next_date = lead(date),
        time_on_page = as.numeric(difftime(next_date, date, units = "secs")),
        time_on_page = coalesce(time_on_page, -1)
      ) %>%
      ungroup() %>%
      filter(time_on_page == -1 | time_on_page > 5 & time_on_page < 3600) %>%
      select(-next_date)

    log_step("Read time heuristic", t, df)
    t <- Sys.time()

    if (nrow(df) == 0) return(df)

    #--- ASN enrichment (minimal)
    ips <- sort(unique(df$ip))
  
    asn_data <- lookup_asns(ips, 
                            db_path = asn_db_path
    )

    df <- df %>% left_join(asn_data, by = "ip")

    log_step("ASN Enrichment", t, df)
    t <- Sys.time()

    #df <- df %>%
    #  group_by(ip) %>%
    #  mutate(total_requests = n()) %>%
    #  ungroup() %>%
    #  mutate(
    #    is_cloud_asn = grepl(cloud_asn_regex, asn_org, ignore.case = TRUE),
    #    is_residential = grepl(residential_regex, asn_org, ignore.case = TRUE)
    #  ) %>%
    #  filter(!(is_cloud_asn & total_requests > 3)) %>%
    #  select(-is_cloud_asn, -is_residential)

    # cloud ASN repeated range burst

    df <- df %>%
      arrange(date) %>%
      mutate(
        is_cloud_asn = grepl(cloud_asn_regex, asn_org, ignore.case = TRUE),
        asn_org_clean = coalesce(asn_org, "UNKNOWN_ASN"),
        ip_16 = sub("\\.[0-9]+\\.[0-9]+$", "", ip),
        asn_changed = asn_org_clean != lag(asn_org_clean, default = first(asn_org_clean)),
        asn_bucket = cumsum(asn_changed) + 1
      ) %>%
      group_by(asn_bucket, ip_16) %>%
      mutate(ip_16_occ = n()) %>%
      ungroup() %>%
      filter(ip_16_occ == 1 | !is_cloud_asn) %>%
      select(-asn_org_clean, 
             -ip_16, -asn_changed, 
             -asn_bucket, 
             -ip_16_occ,
             -is_cloud_asn
      )

    log_step("ASN filtering", t, df)
    t <- Sys.time()

    df <- df %>% filter(!grepl(ip_exclude, ip))

    log_step("IP Exclusion", t, df)
    t <- Sys.time()

    good_ip <- df %>%
               filter(!(target %in% honey_pots)) %>%
               pull(ip)

    df <- df %>% filter(ip %in% good_ip)

    log_step("HONEY POTS", t, df)

    df

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
 
    t <- Sys.time()

    df  <- filtered_data()
    geo <- geo_cache_reactive()
  
    if (!is.null(geo)) {
      df <- df %>% left_join(geo, by = "ip")
    }
 
    log_step("GEO Enrichment", t, df)

    df

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

    t <- Sys.time()

    req(nrow(df) > 0)
  
    df <- df %>%
             filter(
                   time_on_page > 3 & time_on_page < 3600
                   ) %>%
             group_by(target) %>%
             summarise(median_readtime = median(time_on_page),
                       valid_reads = n(),
                       .groups = "drop") %>%
             arrange(desc(median_readtime))

    log_step("READTIME STATS", t, df)

    print(df)

    df

  })

  output$kpi_med_readtime <- renderText({
  
    df <- filtered_data()
 
    t <- Sys.time()

    req(nrow(df) > 0)

    # computes one median over the whole filtered table, assuming df was not already grouped with group_by(...).

    # So after summarise, you get a tibble like:
    # 
    # # A tibble: 1 × 1
    #     med
    #   <dbl>
    # 1  42.5
    # 
    # Then:
    # 
    # 42.5

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
 
    log_step("KPI MEDIAN READTIME", t, df)

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
          '<a href=\"https://julienlargetpiet.tech', 
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



