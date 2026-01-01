package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	chroma "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/util"
)

type HeadingRenderer struct {
	html.Config
}

func NewHeadingRenderer(opts ...html.Option) renderer.NodeRenderer {
	r := &HeadingRenderer{
		Config: html.NewConfig(),
	}
	for _, opt := range opts {
		opt.SetHTMLOption(&r.Config)
	}
	return r
}

func (r *HeadingRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindHeading, r.renderHeading)
}

func (r *HeadingRenderer) renderHeading(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	n := node.(*ast.Heading)
	if entering {
		_, _ = w.WriteString("<h")
		_ = w.WriteByte("0123456"[n.Level])
		if n.Attributes() != nil {
			html.RenderAttributes(w, n, html.HeadingAttributeFilter)
		}
		_ = w.WriteByte('>')
	} else {
		// Get the ID to link to
		if idAttr, ok := n.Attribute([]byte("id")); ok {
			var id string
			switch v := idAttr.(type) {
			case []byte:
				id = string(v)
			case string:
				id = v
			}

			// Add copy button only for H2 and H3, and only if ID exists
			if id != "" && (n.Level == 2 || n.Level == 3) {
				link := "#" + id
				btnHTML := fmt.Sprintf(` <button class="copy-link-btn" aria-label="Copy link to this section" onclick="copyToClipboard('%s', this)"><i class="fa-solid fa-link"></i></button>`, link)
				_, _ = w.WriteString(btnHTML)
			}
		}

		_, _ = w.WriteString("</h")
		_ = w.WriteByte("0123456"[n.Level])
		_ = w.WriteByte('>')
	}
	return ast.WalkContinue, nil
}

type Config struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	UmamiID     string `json:"umami_id"`
	UmamiURL    string `json:"umami_url"`
	BaseURL     string `json:"base_url"`
	HomeContent template.HTML
	NotesTree   []*NoteNode `json:"-"`
}

// NoteNode represents a file or folder in the notes tree
type NoteNode struct {
	Name     string
	URL      string
	IsDir    bool
	Children []*NoteNode
	Title    string
}

// Post represents a blog post
type Post struct {
	Title       string
	Description string
	Date        time.Time
	Tags        []string
	Content     template.HTML
	URL         string
	Slug        string
}

// Meta parsing helper
func parseFrontmatter(content []byte) (map[string]string, []byte, error) {
	if !bytes.HasPrefix(content, []byte("---\n")) {
		return nil, content, nil
	}

	parts := bytes.SplitN(content, []byte("---\n"), 3)
	if len(parts) < 3 {
		return nil, content, fmt.Errorf("invalid frontmatter format")
	}

	meta := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(parts[1]))
	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, ":"); idx > 0 {
			key := strings.TrimSpace(line[:idx])
			val := strings.TrimSpace(line[idx+1:])
			meta[key] = val
		}
	}

	return meta, parts[2], nil
}

// buildNotesTree recursively builds the notes structure
func buildNotesTree(root string, baseUrl string) ([]*NoteNode, error) {
	var nodes []*NoteNode

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		path := filepath.Join(root, entry.Name())
		name := entry.Name()

		// Skip hidden files, the images folder, and the special index file
		if strings.HasPrefix(name, ".") || name == "images" || name == "_index.md" {
			continue
		}

		if entry.IsDir() {
			children, err := buildNotesTree(path, baseUrl)
			if err != nil {
				return nil, err
			}
			nodes = append(nodes, &NoteNode{
				Name:     name,
				IsDir:    true,
				Children: children,
			})
		} else if strings.HasSuffix(name, ".md") {
			// Parse the note to get the title
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			meta, _, err := parseFrontmatter(data)
			title := name
			if err == nil && meta["title"] != "" {
				title = meta["title"]
			}

			// Generate a web-friendly URL from the file path relative to the notes dir
			relPath, _ := filepath.Rel("notes", path)
			urlPath := strings.TrimSuffix(relPath, ".md")

			nodes = append(nodes, &NoteNode{
				Name:  name,
				Title: title,
				URL:   baseUrl + "notes/" + urlPath,
				IsDir: false,
			})
		}
	}
	return nodes, nil
}

// Helper to process markdown files
func processMarkdownFile(md goldmark.Markdown, path string, outputRelPath string, baseURL string) (Post, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Post{}, err
	}

	meta, body, err := parseFrontmatter(data)
	if err != nil {
		return Post{}, err
	}

	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return Post{}, err
	}

	date, _ := time.Parse("2006-01-02", meta["date"])

	slug := strings.TrimSuffix(filepath.Base(path), ".md")

	var tags []string
	if meta["tags"] != "" {
		tagStr := strings.Trim(meta["tags"], "[]")
		tagStr = strings.ReplaceAll(tagStr, "\"", "")
		tagStr = strings.ReplaceAll(tagStr, "'", "")
		parts := strings.Split(tagStr, ",")
		for _, t := range parts {
			t = strings.TrimSpace(t)
			if t != "" {
				tags = append(tags, t)
			}
		}
	}

	return Post{
		Title:       meta["title"],
		Description: meta["description"],
		Date:        date,
		Tags:        tags,
		Content:     template.HTML(buf.String()),
		Slug:        slug,
		URL:         baseURL + outputRelPath,
	}, nil
}

func main() {
	// Markdown setup
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,
			highlighting.NewHighlighting(
				highlighting.WithStyle("monokai"),
				highlighting.WithFormatOptions(
					chroma.WithLineNumbers(true),
				),
			),
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
			renderer.WithNodeRenderers(
				util.Prioritized(NewHeadingRenderer(), 1000),
			),
		),
	)

	// Configuration
	var cfg Config
	configFile, err := os.ReadFile("config.json")
	if err == nil {
		if err := json.Unmarshal(configFile, &cfg); err != nil {
			log.Fatalf("Error parsing config.json: %v", err)
		}
	} else {
		// Default configuration if file is missing
		cfg = Config{
			Title:       "Minimal Go Blog",
			Description: "A static blog generated with Go.",
			BaseURL:     "/",
		}
		log.Println("Warning: config.json not found, using default configuration")
	}

	// Build notes tree
	notesTree, err := buildNotesTree("notes", cfg.BaseURL)
	if err != nil {
		// Just log warning if notes dir doesn't exist yet
		log.Printf("Warning: could not build notes tree: %v", err)
	}
	cfg.NotesTree = notesTree

	// Clean/Create public directory
	os.RemoveAll("public")
	if err := os.MkdirAll("public", 0755); err != nil {
		log.Fatal(err)
	}

	// Read templates
	funcMap := template.FuncMap{
		"lower": strings.ToLower,
	}
	tmpl, err := template.New("").Funcs(funcMap).ParseGlob("templates/*")
	if err != nil {
		log.Fatal(err)
	}

	// Process posts
	var posts []Post

	// Read Homepage Content from notes/_index.md
	homeData, err := os.ReadFile("notes/_index.md")
	if err == nil {
		_, homeBody, err := parseFrontmatter(homeData)
		if err == nil {
			var buf bytes.Buffer
			if err := md.Convert(homeBody, &buf); err == nil {
				cfg.HomeContent = template.HTML(buf.String())
			}
		}
	}
	if cfg.HomeContent == "" {
		cfg.HomeContent = template.HTML("<p>Welcome to my digital garden.</p>")
	}

	for _, file := range []string{} {
		slug := strings.TrimSuffix(filepath.Base(file), ".md")
		post, err := processMarkdownFile(md, file, slug, cfg.BaseURL)
		if err != nil {
			log.Printf("Skipping %s: %v", file, err)
			continue
		}
		posts = append(posts, post)

		// Create directory for clean URL (e.g., public/post-slug/index.html)
		os.MkdirAll(filepath.Join("public", slug), 0755)
		f, err := os.Create(filepath.Join("public", slug, "index.html"))
		if err != nil {
			log.Fatal(err)
		}

		err = tmpl.ExecuteTemplate(f, "post.html", map[string]interface{}{
			"Site": cfg,
			"Post": post,
		})
		f.Close()
		if err != nil {
			log.Fatal(err)
		}
	}

	// Process notes
	filepath.Walk("notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel("notes", path)

		if strings.HasSuffix(path, ".md") {
			// Skip the homepage source file if it exists in the tree traversal
			if filepath.Base(path) == "_index.md" {
				return nil
			}

			// Create clean URL structure: notes/folder/file/index.html
			cleanRelPath := strings.TrimSuffix(relPath, ".md")
			outPath := filepath.Join("public", "notes", cleanRelPath, "index.html")

			os.MkdirAll(filepath.Dir(outPath), 0755)

			post, err := processMarkdownFile(md, path, "notes/"+cleanRelPath, cfg.BaseURL)
			if err != nil {
				log.Printf("Error processing note %s: %v", path, err)
				return nil
			}

			f, err := os.Create(outPath)
			if err != nil {
				return err
			}

			// Reuse post template for now, or make a specific note template
			err = tmpl.ExecuteTemplate(f, "post.html", map[string]interface{}{
				"Site": cfg,
				"Post": post,
			})
			f.Close()
		} else {
			// Copy non-markdown files (images, assets) directly
			outPath := filepath.Join("public", "notes", relPath)
			os.MkdirAll(filepath.Dir(outPath), 0755)

			input, err := os.ReadFile(path)
			if err != nil {
				log.Printf("Error reading asset %s: %v", path, err)
				return nil
			}
			err = os.WriteFile(outPath, input, 0644)
			if err != nil {
				log.Printf("Error writing asset %s: %v", outPath, err)
			}
		}
		return nil
	})

	// Sort posts (newest first)
	sort.Slice(posts, func(i, j int) bool {
		return posts[i].Date.After(posts[j].Date)
	})

	// Write index
	f, err := os.Create(filepath.Join("public", "index.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(f, "index.html", map[string]interface{}{
		"Site":  cfg,
		"Posts": posts,
	})
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Write notes index
	f, err = os.Create(filepath.Join("public", "notes.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(f, "notes.html", map[string]interface{}{
		"Site": cfg,
	})
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Generate tags pages
	tagsMap := make(map[string][]Post)

	// Collect tags from posts
	for _, p := range posts {
		for _, t := range p.Tags {
			tagsMap[strings.ToLower(t)] = append(tagsMap[strings.ToLower(t)], p)
		}
	}

	// Collect tags from notes
	filepath.Walk("notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			if filepath.Base(path) == "_index.md" {
				return nil
			}

			data, _ := os.ReadFile(path)
			meta, _, _ := parseFrontmatter(data)

			dateStr := meta["date"]
			date, _ := time.Parse("2006-01-02", dateStr)

			relPath, _ := filepath.Rel("notes", path)
			url := cfg.BaseURL + "notes/" + strings.TrimSuffix(relPath, ".md")
			title := meta["title"]
			if title == "" {
				title = info.Name()
			}

			p := Post{
				Title:       title,
				Description: meta["description"],
				Date:        date,
				URL:         url,
			}

			if meta["tags"] != "" {
				tagStr := strings.Trim(meta["tags"], "[]")
				tagStr = strings.ReplaceAll(tagStr, "\"", "")
				tagStr = strings.ReplaceAll(tagStr, "'", "")
				parts := strings.Split(tagStr, ",")
				for _, t := range parts {
					t = strings.TrimSpace(t)
					if t != "" {
						tagsMap[strings.ToLower(t)] = append(tagsMap[strings.ToLower(t)], p)
					}
				}
			}
		}
		return nil
	})

	os.MkdirAll("public/tags", 0755)
	for tag, taggedPosts := range tagsMap {
		// Sort by date
		sort.Slice(taggedPosts, func(i, j int) bool {
			return taggedPosts[i].Date.After(taggedPosts[j].Date)
		})

		f, err := os.Create(filepath.Join("public", "tags", tag+".html"))
		if err != nil {
			continue
		}
		err = tmpl.ExecuteTemplate(f, "tag.html", map[string]interface{}{
			"Site":  cfg,
			"Tag":   tag,
			"Posts": taggedPosts,
		})
		f.Close()
	}

	// Generate search index
	var allSearchItems []map[string]string
	var allNotes []Post // Keep track of all notes for sitemap

	// Add posts
	for _, p := range posts {
		contentStr := string(p.Content)
		allSearchItems = append(allSearchItems, map[string]string{
			"title":   p.Title,
			"url":     p.URL,
			"date":    p.Date.Format("Jan 02, 2006"),
			"content": contentStr,
			"type":    "post",
			"tags":    strings.Join(p.Tags, ","),
		})
	}

	// Add notes
	filepath.Walk("notes", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".md") {
			if filepath.Base(path) == "_index.md" {
				return nil
			}

			data, _ := os.ReadFile(path)
			meta, body, _ := parseFrontmatter(data)

			var buf bytes.Buffer
			md.Convert(body, &buf)

			relPath, _ := filepath.Rel("notes", path)
			url := cfg.BaseURL + "notes/" + strings.TrimSuffix(relPath, ".md")
			title := meta["title"]
			if title == "" {
				title = info.Name()
			}

			dateStr := meta["date"]
			formattedDate := ""
			var date time.Time
			if dateStr != "" {
				t, err := time.Parse("2006-01-02", dateStr)
				if err == nil {
					formattedDate = t.Format("Jan 02, 2006")
					date = t
				} else {
					formattedDate = dateStr
				}
			}

			// Add to note list for sitemap
			allNotes = append(allNotes, Post{
				Title:       title,
				URL:         url,
				Date:        date,
				Description: meta["description"],
			})

			var tags []string
			if meta["tags"] != "" {
				tagStr := strings.Trim(meta["tags"], "[]")
				tagStr = strings.ReplaceAll(tagStr, "\"", "")
				tagStr = strings.ReplaceAll(tagStr, "'", "")
				parts := strings.Split(tagStr, ",")
				for _, t := range parts {
					t = strings.TrimSpace(t)
					if t != "" {
						tags = append(tags, t)
					}
				}
			}

			allSearchItems = append(allSearchItems, map[string]string{
				"title":   title,
				"url":     url,
				"date":    formattedDate,
				"content": buf.String(),
				"type":    "note",
				"tags":    strings.Join(tags, ","),
			})
		}
		return nil
	})

	f, err = os.Create(filepath.Join("public", "search.json"))
	if err != nil {
		log.Fatal(err)
	}
	enc := json.NewEncoder(f)
	if err := enc.Encode(allSearchItems); err != nil {
		log.Fatal(err)
	}
	f.Close()

	// Generate Sitemap
	f, err = os.Create(filepath.Join("public", "sitemap.xml"))
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")
	err = tmpl.ExecuteTemplate(f, "sitemap.xml", map[string]interface{}{
		"Site":  cfg,
		"Posts": posts,
		"Notes": allNotes,
	})
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	// Generate Robots.txt
	f, err = os.Create(filepath.Join("public", "robots.txt"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(f, "robots.txt", map[string]interface{}{
		"Site": cfg,
	})
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Generate RSS Feed
	f, err = os.Create(filepath.Join("public", "rss.xml"))
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n")

	// Combine posts and notes for RSS
	allContent := append(posts, allNotes...)
	sort.Slice(allContent, func(i, j int) bool {
		return allContent[i].Date.After(allContent[j].Date)
	})

	err = tmpl.ExecuteTemplate(f, "rss.xml", map[string]interface{}{
		"Site":      cfg,
		"Posts":     allContent,
		"BuildDate": time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"),
	})
	if err != nil {
		log.Fatal(err)
	}
	f.Close()

	// Generate 404.html
	f, err = os.Create(filepath.Join("public", "404.html"))
	if err != nil {
		log.Fatal(err)
	}
	err = tmpl.ExecuteTemplate(f, "404.html", map[string]interface{}{
		"Site": cfg,
	})
	f.Close()
	if err != nil {
		log.Fatal(err)
	}

	// Generate _redirects for Netlify/Cloudflare Pages
	f, err = os.Create(filepath.Join("public", "_redirects"))
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString("/* /404.html 404\n")
	f.Close()

	// Copy static assets
	os.MkdirAll("public/static", 0755)
	filepath.Walk("static", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		destPath := filepath.Join("public", path)
		if info.IsDir() {
			os.MkdirAll(destPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(destPath), 0755)
			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = os.WriteFile(destPath, input, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	})

	// Copy assets folder (for social media banners etc)
	os.MkdirAll("public/assets", 0755)
	filepath.Walk("assets", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// It's okay if assets folder doesn't exist
			return nil
		}
		destPath := filepath.Join("public", path)
		if info.IsDir() {
			os.MkdirAll(destPath, 0755)
		} else {
			os.MkdirAll(filepath.Dir(destPath), 0755)
			input, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = os.WriteFile(destPath, input, 0644)
			if err != nil {
				return err
			}
		}
		return nil
	})

	fmt.Println("Build complete! Output in ./public")
}
