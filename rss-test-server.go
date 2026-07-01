package main

import (
	"encoding/xml"
	"flag"
	"fmt"
	"html/template"
	"log"
	mathrand "math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Version string   `xml:"version,attr"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	Language    string `xml:"language,omitempty"`
	TTL         int    `xml:"ttl,omitempty"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	GUID        GUID   `xml:"guid"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

type GUID struct {
	IsPermaLink string `xml:"isPermaLink,attr"`
	Value       string `xml:",chardata"`
}

type Server struct {
	publicBaseURL string
	minDelay      time.Duration
	maxDelay      time.Duration
}

func main() {
	addr := flag.String("addr", ":8080", "listen address")
	publicBaseURL := flag.String("base-url", "", "public base URL, e.g. http://localhost:8080; defaults to request Host")
	flag.Parse()

	s := &Server{
		publicBaseURL: strings.TrimRight(*publicBaseURL, "/"),
		minDelay:      400 * time.Millisecond,
		maxDelay:      4 * time.Second,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.index)
	mux.HandleFunc("GET /feed.xml", s.feed)
	mux.HandleFunc("GET /item/{unix}", s.item)

	log.Printf("RSS test server listening on %s", *addr)
	log.Printf("Feed URL: http://%s/feed.xml", normalizeAddrForLog(*addr))

	if err := http.ListenAndServe(*addr, mux); err != nil {
		log.Fatal(err)
	}
}

func normalizeAddrForLog(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "localhost" + addr
	}
	return addr
}

func (s *Server) baseURL(r *http.Request) string {
	if s.publicBaseURL != "" {
		return s.publicBaseURL
	}

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}

	if forwardedProto := r.Header.Get("X-Forwarded-Proto"); forwardedProto != "" {
		scheme = forwardedProto
	}

	host := r.Host
	if forwardedHost := r.Header.Get("X-Forwarded-Host"); forwardedHost != "" {
		host = forwardedHost
	}

	return scheme + "://" + host
}

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html>
<head>
  <meta charset="utf-8">
  <title>RSS Client Test Feed</title>
</head>
<body>
  <h1>RSS Client Test Feed</h1>
  <p>This server provides a synthetic RSS feed for client testing.</p>

  <ul>
    <li><a href="{{ .Base }}/feed.xml">RSS feed</a></li>
    <li>15 items</li>
    <li>Each item represents one ten-second bucket</li>
    <li>Item URLs sleep randomly between 400ms and 4s before responding</li>
    <li>Item timestamps are Unix times</li>
  </ul>
</body>
</html>`))

func (s *Server) index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	base := s.baseURL(r)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := indexTemplate.Execute(w, map[string]string{
		"Base": base,
	}); err != nil {
		log.Printf("execute index template: %v", err)
	}
}

func (s *Server) feed(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	base := s.baseURL(r)

	items := make([]Item, 0, 15)

	for i := 0; i < 15; i++ {
		offsetSeconds := i * 10
		t := now.Add(-time.Duration(offsetSeconds) * time.Second)

		// Round down to the nearest 10-second bucket so feed readers see stable-ish
		// IDs inside the same 10-second window.
		bucketUnix := t.Unix() / 10 * 10

		itemURL := fmt.Sprintf("%s/item/%d?offset=%d", base, bucketUnix, offsetSeconds)

		items = append(items, Item{
			Title: fmt.Sprintf("RSS test item unix=%d offset=%ds", bucketUnix, offsetSeconds),
			Link:  itemURL,
			GUID: GUID{
				IsPermaLink: "true",
				Value:       itemURL,
			},
			Description: fmt.Sprintf(
				"Test RSS item for Unix time %d. This item represents now minus %d seconds.",
				bucketUnix,
				offsetSeconds,
			),
			PubDate: time.Unix(bucketUnix, 0).UTC().Format(time.RFC1123Z),
		})
	}

	rss := RSS{
		Version: "2.0",
		Channel: Channel{
			Title:       "RSS Client Delay Test Feed",
			Link:        base + "/",
			Description: "Synthetic RSS feed containing 15 ten-second Unix-time items. Item pages respond after a random delay.",
			Language:    "en",
			TTL:         1,
			Items:       items,
		},
	}

	w.Header().Set("Content-Type", "application/rss+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte(xml.Header)); err != nil {
		log.Printf("write xml header: %v", err)
		return
	}

	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	if err := enc.Encode(rss); err != nil {
		log.Printf("encode rss: %v", err)
	}
}

func (s *Server) item(w http.ResponseWriter, r *http.Request) {
	raw := r.PathValue("unix")

	unixTime, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		http.Error(w, "invalid unix timestamp", http.StatusBadRequest)
		return
	}

	offset := r.URL.Query().Get("offset")
	if offset == "" {
		offset = "unknown"
	}

	wait := s.randomDelay(s.minDelay, s.maxDelay)
	time.Sleep(wait)

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	itemURL := s.baseURL(r) + "/item/" + url.PathEscape(raw)

	if _, err := fmt.Fprintf(w, "RSS client test item\n\nunix_time: %d\nutc_time: %s\noffset_seconds: %s\nrandom_wait_ms: %d\nurl: %s\n",
		unixTime,
		time.Unix(unixTime, 0).UTC().Format(time.RFC3339),
		offset,
		wait.Milliseconds(),
		itemURL,
	); err != nil {
		log.Printf("write item: %v", err)
	}
}

func (s *Server) randomDelay(min, max time.Duration) time.Duration {
	if max <= min {
		return min
	}

	span := int64(max - min)

	return min + time.Duration(mathrand.Int63n(span+1))
}
