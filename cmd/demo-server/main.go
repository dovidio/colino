package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

func main() {
	port := flag.Int("port", 8080, "Port to run the demo server on")
	host := flag.String("host", "localhost", "Host to bind the demo server to")
	flag.Parse()

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", *host, *port),
		Handler: createHandler(),
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Demo server starting on http://%s:%d", *host, *port)
		log.Printf("RSS feed available at: http://%s:%d/rss", *host, *port)
		log.Printf("Articles available at: http://%s:%d/articles/[1-4]", *host, *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down demo server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Demo server stopped")
}

// createHandler creates the HTTP handler for the demo server
func createHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/rss", rssHandler)
	mux.HandleFunc("/articles/", articlesHandler)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			homeHandler(w, r)
		} else {
			http.NotFound(w, r)
		}
	})
	return mux
}

// homeHandler serves a simple home page explaining the demo server
func homeHandler(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := strings.TrimSuffix(fmt.Sprintf("%s://%s", scheme, r.Host), "/")

	html := `<!DOCTYPE html>
<html>
<head>
    <title>Colino Demo Server</title>
    <style>
        body { font-family: system-ui, sans-serif; max-width: 800px; margin: 0 auto; padding: 20px; }
        .header { border-bottom: 2px solid #10b981; padding-bottom: 10px; margin-bottom: 20px; }
        .endpoint { background: #f3f4f6; padding: 10px; border-radius: 5px; margin: 10px 0; }
        .url { color: #059669; font-family: monospace; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🌱 Colino Demo Server</h1>
        <p>This server provides mock RSS feeds and articles for demonstrating Colino's functionality.</p>
    </div>

    <h2>Available Endpoints</h2>
    <div class="endpoint">
        <strong>RSS Feed:</strong> <a href="%[1]s/rss" class="url">%[1]s/rss</a>
        <p>Contains 4 sample articles about technology and seasons.</p>
    </div>

    <div class="endpoint">
        <strong>Articles:</strong> <a href="%[1]s/articles/1" class="url">%[1]s/articles/[1-4]</a>
        <p>Individual article content pages.</p>
    </div>

    <h2>Usage with Colino</h2>
    <p>Add this RSS feed to Colino:</p>
    <pre><code>%[1]s/rss</code></pre>

    <p>Then run the ingest command to fetch all articles.</p>
</body>
</html>`

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, html, baseURL)
}

// rssHandler returns a list of articles in RSS format
func rssHandler(w http.ResponseWriter, r *http.Request) {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	baseURL := strings.TrimSuffix(fmt.Sprintf("%s://%s", scheme, r.Host), "/")

	rssTemplate := `<?xml version="1.0" encoding="utf-8" standalone="yes"?>
<rss version="2.0" xmlns:atom="http://www.w3.org/2005/Atom">
	<channel>
		<title>Tech Insights Blog</title>
		<link>%[1]s/</link>
		<description>Thoughts on technology, productivity, and the changing seasons of development</description>
		<lastBuildDate>%[2]s</lastBuildDate>
		<atom:link href="%[1]s/rss" rel="self" type="application/rss+xml"/>
		<item>
			<title>The Four Seasons of Software Development</title>
			<link>%[1]s/articles/1</link>
			<pubDate>Mon, 25 Aug 2025 10:30:00 +0000</pubDate>
			<guid>seasons-of-dev</guid>
			<description>Why software development really only has two seasons: shipping and maintaining.</description>
		</item>
		<item>
			<title>Beyond Autumn: Rethinking Seasonal Metaphors in Tech</title>
			<link>%[1]s/articles/2</link>
			<pubDate>Sun, 24 Aug 2025 14:15:00 +0000</pubDate>
			<guid>beyond-autumn</guid>
			<description>Our industry loves seasonal metaphors, but what if we've been thinking about it all wrong?</description>
		</item>
		<item>
			<title>Summer to Winter: The Abrupt Transitions of Product Cycles</title>
			<link>%[1]s/articles/3</link>
			<pubDate>Sat, 23 Aug 2025 09:45:00 +0000</pubDate>
			<guid>product-cycles</guid>
			<description>How product development jumps between intense growth and steady maintenance without the gentle transitions of nature.</description>
		</item>
		<item>
			<title>Embracing Binary Seasons: Finding Peace in Development's Rhythms</title>
			<link>%[1]s/articles/4</link>
			<pubDate>Fri, 22 Aug 2025 16:20:00 +0000</pubDate>
			<guid>binary-seasons</guid>
			<description>Learning to work with, rather than against, the natural rhythm of building and maintaining software.</description>
		</item>
	</channel>
</rss>`

	currentTime := time.Now().Format(time.RFC1123Z)
	content := fmt.Sprintf(rssTemplate, baseURL, currentTime)

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.Write([]byte(content))
}

// articlesHandler serves individual article content
func articlesHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/articles/")
	if path == "" {
		http.Error(w, "Article ID required", http.StatusBadRequest)
		return
	}

	articleID, err := strconv.Atoi(path)
	if err != nil || articleID < 1 || articleID > 4 {
		http.Error(w, "Invalid article ID (use 1-4)", http.StatusBadRequest)
		return
	}

	articles := map[int]struct {
		title   string
		content string
	}{
		1: {
			title: "The Four Seasons of Software Development",
			content: `<h1>The Four Seasons of Software Development</h1>
<p>It occurred to me the other day that in software development, we really only have two seasons, not four. While nature gracefully transitions between spring's renewal, summer's growth, autumn's harvest, and winter's rest, our industry tends to jump abruptly between intense periods of creation and long stretches of maintenance.</p>
<p>We talk about "spring cleaning" our codebases and "summer projects" and "autumn releases," but these are just metaphors. The reality is more binary: we're either building something new or keeping something running. There's no gentle transition—just the sharp shift from feature development to bug fixes, from innovation to optimization.</p>
<p>Maybe this isn't a bug but a feature. Perhaps the binary nature of our work teaches us to be more adaptable, to pivot quickly between creation and preservation. But I can't help but wonder what we lose when we skip the gentle transitions that nature provides.</p>
<p>Part 1 of 4.</p>`,
		},
		2: {
			title: "Beyond Autumn: Rethinking Seasonal Metaphors in Tech",
			content: `<h1>Beyond Autumn: Rethinking Seasonal Metaphors in Tech</h1>
<p>The tech industry loves seasonal metaphors. We talk about "winter is coming" for layoffs, "summer hiring sprees," "spring cleaning" for technical debt, and "harvest time" for product launches. But these metaphors break down under scrutiny because, unlike nature's cycles, our transitions are anything but gradual.</p>
<p>When a project moves from development to maintenance, it's not like leaves slowly changing color. It's more like a light switch flipping. One day you're debating architecture and implementing features; the next you're fixing bugs and answering support tickets. The change is immediate and jarring.</p>
<p>Perhaps we need new metaphors that better capture the binary reality of our work. Maybe we're like lighthouses—periodically sending out brilliant beams of innovation against a backdrop of steady, reliable operation. Or perhaps we're like volcanoes, with periods of dormant stability punctuated by eruptions of creative activity.</p>
<p>Part 2 of 4.</p>`,
		},
		3: {
			title: "Summer to Winter: The Abrupt Transitions of Product Cycles",
			content: `<h1>Summer to Winter: The Abrupt Transitions of Product Cycles</h1>
<p>Product development cycles are perhaps the clearest example of our binary seasons. During the "summer" of development, teams move fast, break things, and push boundaries. Resources flow freely, experimentation is encouraged, and the pace is frantic. Then comes the launch, and suddenly it's "winter."</p>
<p>The transition isn't gradual. One day you're celebrating a successful launch; the next you're in emergency mode fixing production issues. The same codebase that was praised for its innovative features is now criticized for its technical debt. The same team that was empowered to take risks is now constrained by the need to maintain stability.</p>
<p>This whiplash effect takes its toll on developers and organizations alike. The skills that make someone great at product development aren't necessarily the same skills that make someone great at production maintenance. Yet we expect individuals and teams to excel at both, often switching between these modes with little warning or preparation.</p>
<p>Part 3 of 4.</p>`,
		},
		4: {
			title: "Embracing Binary Seasons: Finding Peace in Development's Rhythms",
			content: `<h1>Embracing Binary Seasons: Finding Peace in Development's Rhythms</h1>
<p>After years of fighting against the binary nature of development work, I'm starting to wonder if the solution isn't to resist it, but to embrace it. Instead of trying to force gradual transitions where none naturally exist, maybe we should design our workflows, teams, and expectations around these binary seasons.</p>
<p>What if we accepted that development and maintenance require different mindsets, skills, and even different people? What if we built organizations that could pivot between creation and preservation without the trauma that typically accompanies these transitions? What if we learned to find beauty not in gentle gradients, but in bold contrasts?</p>
<p>The two-season model of software development isn't a flaw to be fixed—it's a reality to be understood. By working with this rhythm instead of against it, we might just find ourselves more productive, more satisfied, and more sane in the process.</p>
<p>Part 4 of 4.</p>`,
		},
	}

	article, exists := articles[articleID]
	if !exists {
		http.NotFound(w, r)
		return
	}

	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <style>
        body {
            font-family: Georgia, serif;
            line-height: 1.6;
            max-width: 700px;
            margin: 0 auto;
            padding: 40px 20px;
            color: #333;
        }
        h1 {
            color: #1a1a1a;
            border-bottom: 2px solid #10b981;
            padding-bottom: 10px;
        }
        p {
            margin: 1.5em 0;
            font-size: 1.1em;
        }
    </style>
</head>
<body>
    %s
</body>
</html>`

	finalHTML := fmt.Sprintf(htmlTemplate, article.title, article.content)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(finalHTML))
}