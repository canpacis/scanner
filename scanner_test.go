package scanner_test

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/canpacis/scanner"
	"github.com/stretchr/testify/assert"
)

type Role struct {
	Name string
}

func (r *Role) UnmarshalString(s string) error {
	r.Name = s
	return nil
}

type Params struct {
	Email string `json:"email"`
	Name  string `json:"name"`

	Language string `header:"accept-language"`

	Page  uint32 `query:"page" form:"page"`
	Done  bool   `query:"done"`
	Role  Role   `query:"role"`
	Roles []Role `query:"roles"`

	Filters []string `form:"filters"`
	Numbers []int    `form:"numbers"`

	Token string `cookie:"token"`

	Document multipart.File `multipart:"document"`

	Avatar image.Image `image:"avatar"`

	LocalFile string `file:"local.txt"`

	ID   string `path:"id"`
	Slug string `path:"slug"`
}

type Expectation struct {
	Expected any
	Actual   any
}

type Case struct {
	Scanner      scanner.Scanner
	Expectations func(p *Params) []Expectation
}

func (c Case) Run(t *testing.T) {
	assert := assert.New(t)
	p := &Params{}

	err := c.Scanner.Scan(p)
	assert.NoError(err)

	for _, e := range c.Expectations(p) {
		assert.Equal(e.Expected, e.Actual)
	}
}

func TestJsonScanner(t *testing.T) {
	body := bytes.NewBuffer([]byte(`{ "email": "test@example.com", "name": "John Doe" }`))

	c := Case{
		Scanner: scanner.NewJSON(body),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{"test@example.com", p.Email},
				{"John Doe", p.Name},
			}
		},
	}
	c.Run(t)
}

func TestHeaderScanner(t *testing.T) {
	header := &http.Header{}
	header.Set("Accept-Language", "en")

	c := Case{
		Scanner: scanner.NewHeader(header),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{"en", p.Language},
			}
		},
	}
	c.Run(t)
}

func TestQueryScanner(t *testing.T) {
	values := &url.Values{}
	values.Set("page", "2")
	values.Set("done", "true")
	values.Set("role", "admin")
	values.Set("roles", "admin,user")

	c := Case{
		Scanner: scanner.NewQuery(values),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{uint32(2), p.Page},
				{true, p.Done},
				{"admin", p.Role.Name},
				{2, len(p.Roles)},
				{"admin", p.Roles[0].Name},
				{"user", p.Roles[1].Name},
			}
		},
	}
	c.Run(t)
}

func TestFormScanner(t *testing.T) {
	form := &url.Values{}
	form.Set("filters", "sepia,monochrome")
	form.Set("numbers", "6,7,8")

	c := Case{
		Scanner: scanner.NewForm(form),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{2, len(p.Filters)},
				{3, len(p.Numbers)},
				{"sepia", p.Filters[0]},
				{"monochrome", p.Filters[1]},
				{6, p.Numbers[0]},
				{7, p.Numbers[1]},
				{8, p.Numbers[2]},
			}
		},
	}
	c.Run(t)
}

func TestPathScanner(t *testing.T) {
	req := &http.Request{}
	req.SetPathValue("id", "this_is_id")
	req.SetPathValue("slug", "this-is-slug")

	c := Case{
		Scanner: scanner.NewPath(req),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{"this_is_id", p.ID},
				{"this-is-slug", p.Slug},
			}
		},
	}
	c.Run(t)
}

func TestCookieScanner(t *testing.T) {
	jar, _ := cookiejar.New(nil)
	url, _ := url.Parse("http://url.net")
	jar.SetCookies(url, []*http.Cookie{
		{
			Name:  "token",
			Value: "cookie-token",
		},
	})

	c := Case{
		Scanner: scanner.NewCookie(jar, url),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{"cookie-token", p.Token},
			}
		},
	}
	c.Run(t)
}

type file struct {
	io.Reader
	io.ReaderAt
	io.Seeker
	io.Closer
}

func TestMultipartScanner(t *testing.T) {
	multipart := &scanner.MultipartValues{
		Files: map[string]multipart.File{
			"document": file{
				Reader: bytes.NewBuffer([]byte("text document")),
			},
		},
	}

	c := Case{
		Scanner: scanner.NewMultipart(multipart),
		Expectations: func(p *Params) []Expectation {
			file, err := io.ReadAll(p.Document)

			return []Expectation{
				{nil, err},
				{"text document", string(file)},
			}
		},
	}
	c.Run(t)
}

func hash(img image.Image) string {
	var rgba *image.RGBA
	var ok bool

	rgba, ok = img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(img.Bounds())
		draw.Draw(rgba, img.Bounds(), img, image.Pt(0, 0), draw.Over)
	}

	return fmt.Sprintf("%x", md5.Sum(rgba.Pix))
}

func TestImageScanner(t *testing.T) {
	buf := bytes.NewBuffer([]byte{})
	img := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	png.Encode(buf, img)

	multipart := &scanner.MultipartValues{
		Files: map[string]multipart.File{
			"avatar": file{
				Reader: buf,
			},
		},
	}

	c := Case{
		Scanner: scanner.NewImage(multipart),
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{hash(img), hash(p.Avatar)},
			}
		},
	}
	c.Run(t)
}

func TestDirectoryScanner(t *testing.T) {
	fsys := FS{
		Files: map[string]*File{
			"local.txt": NewFile("local.txt", []byte("mock file")),
		},
	}

	s, err := scanner.NewDirectory(fsys)
	assert.NoError(t, err)

	c := Case{
		Scanner: s,
		Expectations: func(p *Params) []Expectation {
			return []Expectation{
				{"mock file", p.LocalFile},
			}
		},
	}
	c.Run(t)
}
