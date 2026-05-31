<h2 align="center">“Express your ideas without friction.”</h2>


<p align="center">
  <img src="logo.png" alt="Statix Logo" width="220">
</p>

**[Quickstart](#quickstart)** — Deploy Statix in seconds

**[CLI](#cli)** — Command Line Interface

**[NeoVim](#neovim-integration)** — WorkFlow with NeoVim

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

# Statix — Deterministic Static Publishing with Infrastructure-Aware Analytics

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

### First-Class Writing & Reading Experience

Statix ships with **prebuilt authoring support** designed for frictionless content creation.

Writers are not forced to fight tooling.

Included out of the box:

- **CodeMirror 6** -> modern, extensible in-browser editor  
- **KaTeX** -> fast, deterministic LaTeX math rendering  
- **Prism.js** -> zero-runtime syntax highlighting  

This enables:

- Structured article writing  
- Code Language awareness  
- Mathematical typesetting without client-side heavy engines  
- Consistent, static-safe rendering  

You can also **preview** your articles in the editing window.



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

# Production Deployment Guide  

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

<p align="center">
  <a href="https://star-history.com/#julienlargetpiet/statix&Date">
    <img src="https://api.star-history.com/svg?repos=julienlargetpiet/statix&type=Date" />
  </a>
</p>


