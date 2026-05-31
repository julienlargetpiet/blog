<h2 align="center">“Express your ideas without friction.”</h2>


<p align="center">
  <img src="logo.png" alt="Statix Logo" width="220">
</p>

**[Quickstart](#quickstart)** — Deploy Statix in seconds

**[CLI](#cli)** — Command Line Interface

**[Example](https://julienlargetpiet.tech)** — Running Statix Blog Example

## Presentation

<details>
  <summary><b>Blog</b> — screenshots (Dark Theme / Default)</summary>
  <br/>

  <p align="center">
    <img src="presentation_pics/blog/cap0.webp" width="260" />
    <img src="presentation_pics/blog/cap1.webp" width="260" />
    <img src="presentation_pics/blog/cap2.webp" width="260" />
  </p>
  <p align="center">
    <img src="presentation_pics/blog/cap3.webp" width="260" />
    <img src="presentation_pics/blog/cap4.webp" width="260" />
    <img src="presentation_pics/blog/cap5.webp" width="260" />
  </p>
  <p align="center">
    <img src="presentation_pics/blog/cap6.webp" width="260" />
    <img src="presentation_pics/blog/cap7.webp" width="260" />
    <img src="presentation_pics/blog/cap8.webp" width="260" />
  </p>
  <p align="center">
    <img src="presentation_pics/blog/cap9.webp" width="260" />
  </p>
</details>

<details>
  <summary><b>Some Default Themes & Selection</b> — screenshots</summary>
  <br/>

  <p align="center">
    <img src="presentation_pics/themes/theme0.webp" width="260" />
    <img src="presentation_pics/themes/theme1.webp" width="260" />
  </p>
  <p align="center">
    <img src="presentation_pics/themes/theme2.webp" width="260" />
    <img src="presentation_pics/themes/theme3.webp" width="260" />
  </p>
</details>

<details>
  <summary><b>Shiny</b> — screenshots</summary>
  <br/>

  <p align="center">
    <img src="presentation_pics/shiny/caps0.webp" width="260" />
    <img src="presentation_pics/shiny/caps1.webp" width="260" />
  </p>
  <p align="center">
    <img src="presentation_pics/shiny/caps3.webp" width="260" />
    <img src="presentation_pics/shiny/caps4.webp" width="260" />
  </p>
</details>

# Quickstart

```

> git clone https://github.com/julienlargetpiet/Statix

> cd Statix

> bash quickstart.sh

```

Statix should now be accessible at:

```

    https://example.com
    https://example.com/admin

```

## What Statix Is

- Deterministic static publishing engine  
- Production-first deployment model  
- Infrastructure-aware analytics system  
- Self-hosted and transparent  

## What Statix Is Not

- A dynamic CMS  
- A SaaS blogging platform  
- A JavaScript-based tracking system  
- A marketing analytics suite  


## Philosophy

Statix is built around a simple idea:

> Publishing should be deterministic, atomic, and observable.

Modern blog platforms often mix:
- runtime rendering
- partial deployments
- third-party tracking
- opaque analytics pipelines

Statix deliberately avoids that.

### Deterministic Builds

Every build produces a fully isolated, immutable output.

A build either succeeds and is promoted, or it does not exist.

Production never sees intermediate artifacts.

---

### Clear Separation of Concerns

Statix separates responsibilities cleanly:

- **Go admin backend** -> content orchestration & build control
- **NGINX** -> static file serving
- **MySQL/MariaDB** -> structured content storage
- **R Shiny module (optional)** -> infrastructure-level analytics

Each component has a single responsibility.

---

### Analytics

The optional analytics module is:

- Log-based
- Server-side
- JS-free

Instead of tracking users, Statix analyzes:

- Request behavior
- ASN infrastructure
- Bot patterns
- Median read-time estimation (log-derived)

Analytics are derived from server logs, not client-side surveillance.

Statix does not treat all traffic equally.

It can distinguish:

- Residential ISP traffic
- Cloud / hosting providers
- Data center infrastructure
- Suspicious behavioral patterns

This allows infrastructure-level filtering and realistic engagement analysis.

### CLI

You can write your article in Markdown in your favorite text editor and directly push articles via a command, such as (Neovim):

```

vim.api.nvim_create_user_command("Publish", function()
  vim.cmd("write")

  local file = vim.api.nvim_buf_get_name(0)

  vim.cmd("!" .. "stx publish --file " .. file)
end, {})

```

The command above is jst a wrapper arround the `publish` command of Statix.

Its whole CLI is:

```

stx - Statix Publishing CLI

Commands:
  set-credentials --url URL --password TOKEN --server_username SERVERUSERNAME --internal_location BLOGPATHONSERVER
  publish --file FILE -m MESSAGE
  nickname create --title TITLE --subject_id ID --is_public true|false NAME
  nickname import ARTICLE_ID NAME
  nickname import-content [--markdown] ARTICLE_ID NAME
  nickname edit [--title TITLE] [--subject_id ID] [--is_public true|false] NAME
  nickname remove [--sync] [-m MESSAGE] NAME
  nickname list
  nickname rename OLD_NAME NEW_NAME
  file upload -m MESSAGE FILE...
  file delete [-m MESSAGE] FILE
  file list
  articles
  subjects
  subject add NAME
  subject delete NAME
  subject rename OLD_NAME NEW_NAME
  dumpdb
  rsync [-m MESSAGE] FOLDER
  build
  completion [bash|zsh]

```

## Note on `code blocks`

Of course yo can write normal code blocks with \`\`\` ... \`\`\` synthax.

But if you need to compare code -> have different code tabs on one code block, you can do it via this synthax:

```

<div class="code-tabs">
  <div class="code-tabs-header">
    <button class="code-tab active" data-tab="rust">Rust</button>
    <button class="code-tab" data-tab="cpp">C++</button>
  </div>

  <div class="code-tab-panel active" data-panel="rust">

\`\`\`rust
fn main() {
    println!("Hello");
}
\`\`\`

  </div>

  <div class="code-tab-panel" data-panel="cpp">

\`\`\`cpp
#include &lt;iostream&gt;

int main() {
    std::cout &lt;&lt; "Hello\n";
}
\`\`\`

  </div>
</div>

```

### Personalization — Without Compromise

Statix allows visual customization.

Themes and Fonts are prebuilt, curated, and fully versioned.  
Switching a theme is an atomic state transition, not a file mutation ( underneath it is done via sylinks ) 

## Architecture Overview

### Publishing Pipeline

```
Editor / Admin
        |
        V
Go Admin Backend (127.0.0.1:8080)
        |
        V
Atomic Build Engine
        |
        V
Isolated Immutable Output (dist/)
        |
        V
Promotion to Production
        |
        V
NGINX Static Serving
        |
        V
End Users
```


### Analytics Pipeline (Optional Module)

```
NGINX access.log
        |
        V
R Shiny Log Analyzer
        |
        V
GeoLite2 (ASN + City)
        |
        V
Infrastructure Classification
        |
        V
Behavioral Heuristics
        |
        V
Engagement Metrics (Median Read Time)
        |
        V
Interactive Dashboard
```

<p align="center">
  <a href="https://star-history.com/#julienlargetpiet/statix&Date">
    <img src="https://api.star-history.com/svg?repos=julienlargetpiet/statix&type=Date" />
  </a>
</p>


