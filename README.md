# rss-test-server

A single-file Go RSS test server designed to synthesize RSS feeds for client testing.

## Features

It exposes the following endpoints:

*   `/feed.xml`: Returns an RSS 2.0 feed containing 15 items. Each RSS item represents one of the last 15 ten-second buckets (`now, now-10s, now-20s, ... now-140s`).
*   `/item/{unix}`: An item endpoint. Sleeps randomly between 400ms and 4s to simulate network delay, then returns a plain text page showing the Unix timestamp, UTC time, and offset.

## Running

Run the server directly:

```bash
go run rss-test-server.go
```

Then you can fetch the feed or an item:

```bash
curl http://localhost:8080/feed.xml
curl http://localhost:8080/item/1760000000
```

## Building

Build the binary:

```bash
go build -o rss-test-server rss-test-server.go
```

Run the built binary:

```bash
./rss-test-server -addr :8080
```

## Configuration Options

*   `-addr`: Listen address (default: `:8080`).
*   `-base-url`: Public base URL. Useful when behind a reverse proxy or tunnel.

**Example with externally visible URL:**

```bash
./rss-test-server -addr :8080 -base-url https://example.test
```
