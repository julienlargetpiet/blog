ui <- fluidPage(

  #theme = bs_theme(
  #  version = 5,
  #  bootswatch = "flatly",   # light theme
  #  base_font = font_google("Lexend"),
  #  code_font = font_google("Lexend")
  #),

  theme = bs_theme(
     version = 5,
     bootswatch = "litera",   # same as your light mode default
     base_font = font_google("Nunito"),
     code_font = font_google("Nunito")
  ),

  checkboxInput("dark_mode", "Dark mode", value = FALSE),

  navset_tab(
    nav_panel(
      title = "Most Visited Pages",
      page_sidebar(
        title = "Input Log File",
        sidebar = tagList(
          selectInput(
            inputId = "time_unit",
            label = "Time Unit",
            choices = c("h", "d", "w", "m", "y"),
            selected = "h"
          ),
          numericInput(
            inputId = "last_n",
            label = "Last n units",
            value = 15,
            min = 1,
            step = 1
          )
        ),

        # KPI row + pie
        layout_column_wrap(
          width = 1/3,
          value_box(title = "Total requests", value = textOutput("kpi_hits")),
          value_box(title = "Unique IPs", value = textOutput("kpi_ips")),
          value_box(title = "Unique pages", value = textOutput("kpi_pages"))
        ),

        value_box(
          title = NULL,
          value = withSpinner(plotlyOutput("pie_chart"), type = 5, size = 1.3)
        )
      )
    ),

    nav_panel(
      title = "WebPages",
      page_sidebar(
        title = "Specific WebPages",
        sidebar = tagList(
          textInput(
            inputId = "webpages",
            label = "RegEx expression(s) (separate with --)",
            value = "articles",
            placeholder = "Example: ^/$--^/blog"
          ),
          selectInput(
            inputId = "time_unit_2",
            label = "Time Unit",
            choices = c("h", "d", "w", "m", "y"),
            selected = "h"
          ),
          numericInput(
            inputId = "last_n_2",
            label = "Last n units",
            value = 15,
            min = 1,
            step = 1
          )
        ),
        value_box(
          title = NULL,
          value = withSpinner(plotlyOutput("graph"), type = 5, size = 1.3)
        )
      )
    ),

    nav_panel(
      title = "Data Page",
      page_sidebar(
        title = "Raw data (filtered)",
        card(
          withSpinner(DTOutput("mytable"), type = 5, size = 1.0)
        )
      )
    )
  )
)

ui <- secure_app(ui)



