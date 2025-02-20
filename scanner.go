package scanner

import (
	"bytes"
	"encoding/json"
	"errors"
	"image"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"reflect"

	"github.com/canpacis/scanner/structd"
)

// Scanner interface resembles a json parser, it populates the given struct with available values based on its field tags. It should return an error when v is not a struct.
type Scanner interface {
	Scan(any) error
}

// A scanner to scan json value from an `io.Reader` to a struct
type JSON struct {
	r io.Reader
}

// Scans the json onto v
func (s *JSON) Scan(v any) error {
	return json.NewDecoder(s.r).Decode(v)
}

func NewJSON(r io.Reader) *JSON {
	return &JSON{
		r: r,
	}
}

func NewJSONBytes(b []byte) *JSON {
	return &JSON{
		r: bytes.NewBuffer(b),
	}
}

// A scanner to scan os file's content to a struct
type Directory struct {
	files map[string]io.Reader
}

func (s *Directory) Get(key string) any {
	file, ok := s.files[key]
	if !ok {
		return []byte{}
	}

	b, _ := io.ReadAll(file)
	return b
}

func (s *Directory) Cast(from any, to reflect.Type) (any, error) {
	if to.Kind() == reflect.String {
		rt := reflect.TypeOf(from)
		if rt.Kind() == reflect.Slice && rt.Elem().Kind() == reflect.Uint8 {
			b := from.([]byte)
			return string(b), nil
		}
	}

	return nil, errors.ErrUnsupported
}

func (s *Directory) Scan(v any) error {
	return structd.New(s, "file").Decode(v)
}

func NewDirectory(fsys fs.FS) (*Directory, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}

	files := map[string]io.Reader{}
	for _, entry := range entries {
		file, err := fsys.Open(filepath.Join(".", entry.Name()))
		if err != nil {
			return nil, err
		}
		files[entry.Name()] = file
	}

	return &Directory{files: files}, nil
}

// A scanner to scan header values from an `http.Header` to a struct
type Header struct {
	*http.Header
}

func (h *Header) Get(key string) any {
	return h.Header.Get(key)
}

// Scans the headers onto v
func (s *Header) Scan(v any) error {
	return structd.New(s, "header").Decode(v)
}

func NewHeader(h *http.Header) *Header {
	return &Header{
		Header: h,
	}
}

// A scanner to scan url query values from a `*url.Values` to a struct
type Query struct {
	*url.Values
}

func (v Query) Get(key string) any {
	return v.Values.Get(key)
}

func (v Query) Cast(from any, to reflect.Type) (any, error) {
	return structd.DefaultCast(from, to)
}

// Scans the query values onto v
func (s *Query) Scan(v any) error {
	return structd.New(s, "query").Decode(v)
}

func NewQuery(v *url.Values) *Query {
	return &Query{
		Values: v,
	}
}

// A scanner to scan http cookies for a url from a `http.CookieJar` to a struct
type Cookie struct {
	http.CookieJar
	url *url.URL
}

func (v Cookie) Get(key string) any {
	list := v.Cookies(v.url)
	for _, cookie := range list {
		if cookie.Name == key {
			return cookie.Value
		}
	}

	return nil
}

// Scans the cookie values onto v
func (s *Cookie) Scan(v any) error {
	return structd.New(s, "cookie").Decode(v)
}

func NewCookie(jar http.CookieJar, url *url.URL) *Cookie {
	return &Cookie{
		CookieJar: jar,
		url:       url,
	}
}

// A scanner to scan form values from a `*url.Values` to a struct
type Form struct {
	*url.Values
}

func (v Form) Get(key string) any {
	return v.Values.Get(key)
}

func (v Form) Cast(from any, to reflect.Type) (any, error) {
	return structd.DefaultCast(from, to)
}

// Scans the form data onto v
func (s *Form) Scan(v any) error {
	return structd.New(s, "form").Decode(v)
}

func NewForm(v *url.Values) *Form {
	return &Form{
		Values: v,
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
type Multipart struct {
	v *MultipartValues
}

// Scans the multipart form data onto v
func (s *Multipart) Scan(v any) error {
	return structd.New(s.v, "multipart").Decode(v)
}

func NewMultipart(v *MultipartValues) *Multipart {
	return &Multipart{
		v: v,
	}
}

type Image struct {
	Files map[string]multipart.File
}

func (v Image) Get(key string) any {
	file, ok := v.Files[key]
	if !ok {
		return nil
	}

	img, _, _ := image.Decode(file)
	return img
}

// Scans the multipart form data and turns them into image.Image and sets v
func (s *Image) Scan(v any) error {
	return structd.New(s, "image").Decode(v)
}

func NewImage(v *MultipartValues) *Image {
	return &Image{
		Files: v.Files,
	}
}

type Pipe []Scanner

// Runs given scanners in sequence
func (s *Pipe) Scan(v any) error {
	value := v

	for _, scanner := range *s {
		if err := scanner.Scan(value); err != nil {
			return err
		}
	}

	return nil
}

func NewPipe(scanners ...Scanner) *Pipe {
	s := Pipe(scanners)
	return &s
}
