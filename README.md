# Digital Garden SSG

A custom Static Site Generator (SSG) written in Go, designed for a "Digital Garden" blog. It features a minimalist, split-pane layout with a Text User Interface (TUI) aesthetic, monospaced fonts, and high-contrast visuals.

## Features

*   **Dual Content Streams:**
    *   **Blog Posts (`content/`):** Standard linear blog posts, sorted by date.
    *   **Digital Garden (`notes/`):** Recursive, tree-structured notes for knowledge management.
*   **Custom Homepage:** Content managed via `notes/_index.md`.
*   **Clean URLs:** Generates clean URLs (e.g., `/notes/my-note/` instead of `/notes/my-note.html`) for better SEO and aesthetics.
*   **Asset Management:**
    *   Images stored in `notes/images/` are automatically copied and can be referenced via absolute paths.
    *   `assets/` folder for site-wide assets like favicons and social media banners.
*   **SEO & Social:**
    *   Automatic `sitemap.xml` and `robots.txt` generation.
    *   Open Graph and Twitter Card meta tags for rich social sharing.
    *   Canonical URL support.
    *   **Analytics:** Integrated support for [Umami Analytics](https://umami.is/) (Cloud or Self-Hosted).
*   **Configuration:** Simple `config.json` for site metadata and analytics.
*   **Syntax Highlighting:** Server-side highlighting using `chroma` (Monokai style).
*   **Search:** Client-side JSON search index for fast content discovery.
*   **TUI Aesthetic:** CSS styling that mimics a terminal environment.

## Project Structure

```text
.
├── config.json       # Site configuration (Title, Desc, BaseURL, Analytics)
├── content/          # Standard blog posts (flat list)
├── notes/            # Recursive tree of markdown files (Digital Garden)
│   ├── images/       # Central location for images referenced in notes
│   └── _index.md     # Source content for the Homepage "Bio" section
├── assets/           # Site-wide assets (favicon.png, banner.png)
├── static/           # CSS, JavaScript, and other static assets
├── templates/        # Go HTML templates & SEO files (sitemap, robots)
├── main.go           # SSG Logic (Build script)
└── public/           # Generated static site (git-ignored)
```

## Getting Started

### Prerequisites

*   Go 1.23 or higher.

### Building the Site

To generate the static site in the `public/` directory, run:

```bash
go run main.go
```

To preview, you can use any static file server. For example, with Python:

```bash
# Serve the public directory
python3 -m http.server 8000 --directory public
```

Then visit `http://localhost:8000`.

## Configuration

Edit `config.json` to change the site title, description, and base URL. You can also configure analytics here.

### Basic Config
```json
{
    "title": "My Digital Garden",
    "description": "Learning in public.",
    "base_url": "/"
}
```

### Analytics (Umami)
To enable analytics, add your Umami Website ID.

**Cloud Version:**
```json
{
    "umami_id": "your-umami-website-id"
}
```

**Self-Hosted Version:**
If you are hosting your own Umami instance, specify the `umami_url` (your instance domain).

```json
{
    "umami_id": "your-umami-website-id",
    "umami_url": "https://analytics.yourdomain.com"
}
```
*Note: Do not include `/script.js` in the `umami_url`.*

## Writing Content

### Frontmatter

All markdown files support YAML frontmatter.

```yaml
---
title: "My Post Title"
date: 2026-01-01
description: "A short summary for SEO and social media cards."
tags: ["go", "web", "learning"]
---
```

### Blog Posts (`content/`)

Place standard articles here. They will appear in the "Latest Posts" list on the homepage.

### Notes (`notes/`)

Place "evergreen" notes here. You can create subdirectories to organize them into a tree structure.
*   **Navigation:** The sidebar automatically builds a navigation tree from this directory.
*   **Ordering:** Folders and files are sorted alphabetically.
*   **Exclusions:** The `images` folder and files starting with `.` (dotfiles) or `_` (underscore) are hidden from the sidebar.

### Homepage (`notes/_index.md`)

This special file controls the "Bio" or introductory text on the main homepage.
*   It is **excluded** from the sidebar navigation and search index.
*   It supports standard Markdown.
*   **Tip:** Add your "Highlighted Notes" or "Quick Links" directly in this file using Markdown lists.

### Images & Assets

*   **Note Images:** Store in `notes/images/`. Reference as `![Alt](/notes/images/file.png)`.
*   **Favicon:** Replace `assets/favicon.png` with your own.
*   **Social Banner:** Replace `assets/banner.png` with your own (1200x630px recommended).

## Development

*   **Templates:** Modify `templates/*.html` to change the structure.
*   **Styling:** Edit `static/style.css` to adjust the TUI theme.
*   **Logic:** The core generator logic resides in `main.go`. It handles file walking, markdown parsing (Goldmark), and template execution.

## Deployment

### Cloudflare Pages

1.  Connect your GitHub repository to Cloudflare Pages.
2.  **Build Command:** `go run main.go`
3.  **Build Output Directory:** `public`
4.  **Environment Variables:** Ensure Go 1.23+ is selected in settings (you may need to set `GO_VERSION` environment variable).

### GitHub Pages

1.  Create a file `.github/workflows/deploy.yml`:

```yaml
name: Deploy to GitHub Pages

on:
  push:
    branches: [ main ]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.23'
          
      - name: Build
        run: go run main.go
        
      - name: Upload artifact
        uses: actions/upload-pages-artifact@v2
        with:
          path: ./public

  deploy:
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    runs-on: ubuntu-latest
    needs: build
    permissions:
      pages: write
      id-token: write
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v2
```

2.  Go to repository **Settings > Pages**.
3.  Set **Source** to "GitHub Actions".
4.  Update `config.json` `base_url` if you are deploying to a subdirectory (e.g., `/repo-name/`).
