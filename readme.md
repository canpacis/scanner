# Scanner

Scanner is a utility package to extract certain values and cast them to usable values in the context of http servers. It can extract request bodies, form values, url queries, cookies and headers. Scanner defines a `Scanner` interface for you to extend it to your own needs.

```go
type Scanner interface {
  Scan(any) error
}
```

## Example

An example that extracts header values from an `http.Header` object.

```go
type Params struct {
  // provide the header name in the field tag
  Language string `header:"accept-language"`
}

// ...

// Create your instance
p := &Params{}
// Provide the request headers
s := scanner.NewHeaderScanner(r.Headers())
if err := s.Scan(p); err != nil {
  // handle error
}
// p.Language -> r.Headers().Get("Accept-Language")
```

You can compose your scanners. There are a handful of pre-built scanners for the most common of use cases.

- `scanner.JsonScanner`: Scans json data from an `io.Reader`
- `scanner.HeaderScanner`: Scans header data from an `*http.Header`
- `scanner.QueryScanner`: Scans url query values from a `*url.Values`
- `scanner.FormScanner`: Scans form data from a `*url.Values`
- `scanner.CookieScanner`: Scans cookies from a `http.CookieJar`
- `scanner.MultipartScanner`: Scans multipart form data from a `scanner.MultipartValues`
- `scanner.ImageScanner`: Scans multipart images from a `scanner.MultipartValues`

eg.:

```go
type Params struct {
  IDs      []string `query:"ids"`
  Language string   `header:"accept-language"`
  Token    string   `cookie:"token"`
}

// ...

// Create your instance
p := &Params{}
var s scanner.Scanner

s = scanner.NewQueryScanner(/* url.Values */)
s.Scan(p) // Don't forget to handle errors

s = scanner.NewHeaderScanner(/* http.Header */)
s.Scan(p) // Don't forget to handle errors

s = scanner.NewCookieScanner(/* http.CookieJar */)
s.Scan(p) // Don't forget to handle errors
```

Or, alternatively, you can use a `scanner.PipeScanner` to streamline the process.

```go
type Params struct {
  IDs      []string `query:"ids"`
  Language string   `header:"accept-language"`
  Token    string   `cookie:"token"`
}

// ...

// Create your instance
p := &Params{}
s := scanner.NewPipeScanner(
  scanner.NewQueryScanner(/* url.Values */),
  scanner.NewHeaderScanner(/* http.Header */),
  scanner.NewCookieScanner(/* http.CookieJar */),
)
s.Scan(p) // Don't forget to handle errors
```

This will populate your struct's fields with available values.