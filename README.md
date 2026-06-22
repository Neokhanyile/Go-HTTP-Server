# Raw HTTP Server From Scratch (Go)

A minimal HTTP/1.1 server built directly on top of raw TCP sockets — no `net/http`, no frameworks. The goal of this project was to understand what actually happens "under the hood" of every web server: opening a socket, reading raw bytes, parsing the HTTP wire format by hand, and writing a valid response back byte-for-byte.

## Why this project exists

Every web framework (Express, Flask, Gin, `net/http`, etc.) hides this layer from you. This project strips that away so the HTTP protocol stops being "magic" and becomes just text over a socket, parsed manually.

## What it does

- Listens for TCP connections on port `8080`
- Handles each connection concurrently using a goroutine
- Reads the raw request bytes off the socket
- Parses the request line (method, path, HTTP version)
- Parses headers into a map (case-insensitive keys)
- Reads the request body using `Content-Length`, correctly accounting for body bytes that may have already been captured in the initial read
- Applies a basic deadline so a slow/stalled client can't block a connection forever
- Routes requests to different responses based on path
- Constructs and sends back a raw, hand-built HTTP response

## Routes

| Method | Path      | Behavior                                  |
|--------|-----------|--------------------------------------------|
| GET    | `/`       | Returns `200 OK` with `"Home page"`        |
| GET    | `/hello`  | Returns `200 OK` with `"Hello, world!"`    |
| POST   | `/echo`   | Echoes the request body back as the response |
| any    | anything else | Returns `404 Not Found`               |

Query strings (e.g. `/hello?name=foo`) are stripped before routing, so they don't affect path matching.

## Running it

Requires Go installed (`go version` to check).

```bash
go build main.go
./main
```

You should see:
```
Listening on port 8080...
```

## Testing it

In a separate terminal:

```bash
curl http://localhost:8080/
curl http://localhost:8080/hello?name=foo
curl -X POST -d "echo this" http://localhost:8080/echo
curl http://localhost:8080/unknown
```

You can also see the exact raw bytes going over the wire using netcat:

```bash
echo -e "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n" | nc localhost 8080
```

## How it's built (request lifecycle)

1. **`main()`** opens a TCP listener and loops forever calling `Accept()`. Each accepted connection is handed off to `go handleConnection(conn)` so the main loop can immediately go back to accepting the next client — this is what makes the server concurrent.
2. **`handleConnection()`** does everything for a single connection:
   - Sets a read deadline so a stalled client can't hang the goroutine forever.
   - Reads up to 4096 bytes off the socket.
   - Splits the raw text on `\r\n` to find the request line, then splits that on spaces to get method/path/version.
   - Walks the remaining lines until it hits the blank line (`""`) that marks the end of headers, splitting each on the first `:` to build a header map (keys lowercased for case-insensitive lookup).
   - Finds the exact byte offset where the body starts (`\r\n\r\n`), figures out how much of the body was already captured in the first read, and reads any remaining bytes from the connection using `io.ReadFull` based on `Content-Length`.
   - Routes on the (query-stripped) path via a `switch` statement to decide the status code and response body.
   - Builds the response string by hand, in the exact HTTP wire format, and writes it back with `conn.Write`.
   - `defer conn.Close()` guarantees the connection is cleaned up no matter which return path is taken.

## Known limitations (intentional, for learning purposes)

This is a learning project, not a production server. Known gaps:

- **No keep-alive** — every connection is closed after a single request/response. Real HTTP/1.1 servers reuse connections for multiple requests.
- **No chunked transfer encoding** — only `Content-Length`-based bodies are supported.
- **Fixed 4096-byte initial read** — a request with headers larger than 4096 bytes would not be parsed correctly.
- **Minimal logging** — verbose by design, to make the protocol visible; not structured for production use.
- **No TLS/HTTPS** — plain HTTP only.

## What this taught me

- HTTP is plain text over TCP — there's no magic, just a well-defined format (`METHOD PATH VERSION\r\n`, headers, blank line, body).
- Reading from a socket is reading a byte stream — a single `Read` call is not guaranteed to return a complete message, which is why production parsers use buffered/looped reads rather than assuming one `Read` = one full request.
- Goroutines make concurrent connection handling almost trivial compared to manually managing threads.
- Small details — `\r\n` vs `\n`, case-insensitive headers, exact `Content-Length` accounting — are the actual hard part of implementing a protocol correctly, not the "happy path" logic.
