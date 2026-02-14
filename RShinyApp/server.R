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

  observeEvent(input$dark_mode, ignoreInit = TRUE, {
  
    session$setCurrentTheme(
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
    )
  
  })

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

    ua_col <- if (ncol(df) >= 10) df[[10]] else rep("", nrow(df))

    parsed <- tibble(
      ip = df[[1]],
      date_raw = paste(df[[4]], df[[5]]),
      request_raw = df[[6]],
      ua = df[[(ncol(df))]]  # safer: last column
    )
    
    parsed <- parsed %>%
      mutate(
        date = as.POSIXct(
          gsub("\\[|\\]", "", date_raw),
          format = "%d/%b/%Y:%H:%M:%S %z",
          tz = "UTC"
        ),
        target = extract_url(request_raw)
      ) %>%
      select(ip, date, target, ua)

    parsed %>% filter(!is.na(date), !is.na(target))
  })

  filtered_data <- reactive({
    df <- raw_data()
    req(nrow(df) > 0)
  
    # -----------------------------
    # BOT DETECTION
    # -----------------------------
    if (!isTRUE(input$show_bots)) {
  
      bot_regex <- paste(
        c(
          "bot","crawler","spider",
          "ahrefs","semrush","mj12","dotbot",
          "googlebot","bingbot","yandex","baiduspider",
          "headless","phantomjs","selenium",
          "playwright","puppeteer",
          "node-fetch","axios",
          "go-http-client","libwww-perl","java/"
        ),
        collapse = "|"
      )
  
      df <- df %>%
        mutate(is_bot_ua = grepl(bot_regex, ua,
                                 ignore.case = TRUE,
                                 perl = TRUE))
  
      #df <- df %>%
      #  group_by(ip, sec = floor_date(date, "second")) %>%
      #  mutate(req_per_sec = n()) %>%
      #  ungroup() %>%
      #  mutate(is_bot_rate = req_per_sec > 10)
  
      #df <- df %>%
      #  mutate(is_bot = is_bot_ua | is_bot_rate) %>%
      #  filter(!is_bot) %>%
      #  select(-is_bot_ua, -req_per_sec, -is_bot_rate, -is_bot)
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
	     target = trimws(target)) %>%
      filter(
        target == "/" |
        target == "/articles/" |
        grepl("^/articles/.*\\.html$", target)
      )

    # -----------------------------
    # TIME WINDOW FILTER
    # -----------------------------
    last <- input$last_n * mult_map[[input$time_unit]]
    cutoff <- max(df$date) - last
  
    df %>% filter(date >= cutoff)

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
    df <- filtered_data()
  
    datatable(
      df %>% arrange(desc(date)) %>% select(ip, date, target),
      options = list(
        pageLength = 100,
        scrollX = TRUE,
        ordering = TRUE
      ),
      rownames = FALSE
    )
  })

}

