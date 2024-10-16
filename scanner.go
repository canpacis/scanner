package scanner

import (
	"encoding/json"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"

	"github.com/CanPacis/scanner/structd"
)

// Scanner interface resembles a json parser, it populates the given struct with available values based on its field tags. It should return an error when v is not a struct.
type Scanner interface {
	Scan(any) error
}

// A scanner to scan json value from an `io.Reader` to a struct
type JsonScanner struct {
	r io.Reader
}

// Scans the json onto v
func (s *JsonScanner) Scan(v any) error {
	return json.NewDecoder(s.r).Decode(v)
}

func NewJsonScanner(r io.Reader) *JsonScanner {
	return &JsonScanner{
		r: r,
	}
}

type Header struct {
	*http.Header
}

func (h *Header) Get(key string) any {
	return h.Header.Get(key)
}

// A scanner to scan header values from an `http.Header` to a struct
type HeaderScanner struct {
	h Header
}

// Scans the headers onto v
func (s *HeaderScanner) Scan(v any) error {
	return structd.New(&s.h, "header").Decode(v)
}

func NewHeaderScanner(h *http.Header) *HeaderScanner {
	return &HeaderScanner{
		h: Header{h},
	}
}

type QueryValues struct {
	*url.Values
}

func (v QueryValues) Get(key string) any {
	return v.Values.Get(key)
}

func (v QueryValues) Cast(from any, to reflect.Type) (any, error) {
	return structd.DefaultCast(from, to)
}

// A scanner to scan url query values from a `*url.Values` to a struct
type QueryScanner struct {
	q QueryValues
}

// Scans the query values onto v
func (s *QueryScanner) Scan(v any) error {
	return structd.New(s.q, "query").Decode(v)
}

func NewQueryScanner(v *url.Values) *QueryScanner {
	return &QueryScanner{
		q: QueryValues{v},
	}
}

type CookieValues struct {
	http.CookieJar
	url *url.URL
}

func (v CookieValues) Get(key string) any {
	list := v.Cookies(v.url)
	for _, cookie := range list {
		if cookie.Name == key {
			return cookie.Value
		}
	}

	return nil
}

// A scanner to scan http cookies for a url from a `http.CookieJar` to a struct
type CookieScanner struct {
	c CookieValues
}

// Scans the cookie values onto v
func (s *CookieScanner) Scan(v any) error {
	return structd.New(s.c, "cookie").Decode(v)
}

func NewCookieScanner(jar http.CookieJar, url *url.URL) *CookieScanner {
	return &CookieScanner{
		c: CookieValues{jar, url},
	}
}

// A scanner to scan form values from a `*url.Values` to a struct
type FormScanner struct {
	f QueryValues
}

// Scans the form data onto v
func (s *FormScanner) Scan(v any) error {
	return structd.New(s.f, "form").Decode(v)
}

func NewFormScanner(v *url.Values) *FormScanner {
	return &FormScanner{
		f: QueryValues{v},
	}
}

type MultipartValues struct {
	Files map[string]multipart.File
}

func (v MultipartValues) Get(key string) any {
	return v.Files[key]
}

type MultipartParser interface {
	ParseMultipartForm(int64) error
	FormFile(string) (multipart.File, *multipart.FileHeader, error)
}

// MultipartValuesFromParser takes a generic parser that is usually an `*http.Request` and
// returns `*scanner.MultipartValues` to use it with a `scanner.MultipartScanner` or `scanner.ImageScanner`
func MultipartValuesFromParser(p MultipartParser, size int64, names ...string) (*MultipartValues, error) {
	if err := p.ParseMultipartForm(size); err != nil {
		return nil, err
	}

	files := map[string]multipart.File{}

	for _, name := range names {
		file, _, err := p.FormFile(name)
		if err != nil {
			return nil, err
		}
		files[name] = file
	}

	return &MultipartValues{Files: files}, nil
}

// A scanner to scan multipart form values, files, from a `*scanner.MultipartValues` to a struct
// You can create a `*scanner.MultipartValues` instance with the `scanner.MultipartValuesFromParser` function.
type MultipartScanner struct {
	v *MultipartValues
}

// Scans the multipart form data onto v
func (s *MultipartScanner) Scan(v any) error {
	return structd.New(s.v, "file").Decode(v)
}

func NewMultipartScanner(v *MultipartValues) *MultipartScanner {
	return &MultipartScanner{
		v: v,
	}
}

type ImageValues struct {
	mv *MultipartValues
}

func (v ImageValues) Get(key string) any {
	file := v.mv.Get(key)
	if file == nil {
		return nil
	}

	mfile := file.(multipart.File)
	img, _, _ := image.Decode(mfile)

	return img
}

type ImageScanner struct {
	v *ImageValues
}

// Scans the multipart form data and turns them into image.Image and sets v
func (s *ImageScanner) Scan(v any) error {
	return structd.New(s.v, "image").Decode(v)
}

func NewImageScanner(v *MultipartValues) *ImageScanner {
	return &ImageScanner{
		v: &ImageValues{mv: v},
	}
}

type PipeScanner []Scanner

// Runs provided scanners in sequence
func (s *PipeScanner) Scan(v any) error {
	value := v

	for _, scanner := range *s {
		if err := scanner.Scan(value); err != nil {
			return err
		}
	}

	return nil
}

func NewPipeScanner(scanners ...Scanner) *PipeScanner {
	s := PipeScanner(scanners)
	return &s
}
