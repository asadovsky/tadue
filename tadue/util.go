// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"runtime"
	"time"

	"appengine"
)

// Data passed to Execute() for any template.
// For "base.html", Data will be a *PageData. In all other cases, it will be
// handler-supplied data.
type RenderData struct {
	FullName string // if non-empty, user is logged in
	Data     interface{}
}

type PageData struct {
	Note  string
	Title template.HTML
	Css   template.HTML
	Body  template.HTML
	Js    template.HTML
}

type ErrorWithInfo struct {
	File string // from runtime.Caller
	Line int    // from runtime.Caller
	Err  error  // underlying error
}

func (e *ErrorWithInfo) Error() string {
	return fmt.Sprintf("%s %d: %v", e.File, e.Line, e.Err)
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func makePageData(name string, data *RenderData) (*PageData, error) {
	pd := &PageData{}

	// Unlike title, css, and js, body is required.
	html, err := ExecuteTemplate(name+"-body", data)
	if err != nil {
		return nil, err
	}
	pd.Body = html

	// TODO(sadovsky): Check if 'field' can be a reference.
	maybeExecuteSubTemplate := func(subTemplateName string, field *template.HTML) error {
		fullName := name + "-" + subTemplateName
		if tmpl.Lookup(fullName) == nil {
			return nil
		}
		html, err = ExecuteTemplate(fullName, data)
		if err != nil {
			return err
		}
		*field = html
		return nil
	}

	err = maybeExecuteSubTemplate("title", &pd.Title)
	if err != nil {
		return nil, err
	}
	err = maybeExecuteSubTemplate("css", &pd.Css)
	if err != nil {
		return nil, err
	}
	err = maybeExecuteSubTemplate("js", &pd.Js)
	if err != nil {
		return nil, err
	}

	return pd, nil
}

func setContentTypeUtf8(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}

func RenderNoteOrDie(w http.ResponseWriter, c *Context, note string) {
	renderPageOrDieInternal(w, c, "home", note, nil)
}

func RenderPageOrDie(w http.ResponseWriter, c *Context, name string, data interface{}) {
	renderPageOrDieInternal(w, c, name, "", data)
}

func renderPageOrDieInternal(
	w http.ResponseWriter, c *Context, name, note string, data interface{}) {
	fullName := ""
	if c.LoggedIn() {
		fullName = c.Session().FullName
	}

	rd := &RenderData{FullName: fullName, Data: data}
	pd, err := makePageData(name, rd)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	pd.Note = note
	rd.Data = pd

	setContentTypeUtf8(w)
	if err = tmpl.ExecuteTemplate(w, "base.html", rd); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func RenderTemplateOrDie(w http.ResponseWriter, name string, data interface{}) {
	setContentTypeUtf8(w)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// If returned error is not nil, it is guaranteed to have type template.Error.
func ExecuteTemplate(name string, data interface{}) (template.HTML, error) {
	buf := &bytes.Buffer{}
	err := tmpl.ExecuteTemplate(buf, name, data)
	if err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func ServeInfo(w http.ResponseWriter, info string) {
	setContentTypeUtf8(w)
	w.Write([]byte(info))
}

func Serve404(w http.ResponseWriter) {
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
}

func ServeError(w http.ResponseWriter, data interface{}) {
	http.Error(w, fmt.Sprint(data), http.StatusInternalServerError)
}

type AppHandlerFunc func(http.ResponseWriter, *http.Request, *Context)

// Wraps other http handlers. Creates context object, recovers from panics, etc.
func WrapHandlerImpl(fn AppHandlerFunc, parseForm bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := &Context{}
		c.SetAec(appengine.NewContext(r))

		// See http://blog.golang.org/2010/08/defer-panic-and-recover.html.
		defer func() {
			if data := recover(); data != nil {
				c.Aec().Errorf(fmt.Sprintf("%v", data))
				ServeError(w, data)
			}
		}()

		ReadSession(r, c)
		if parseForm {
			CheckError(r.ParseForm())
		}
		fn(w, r, c)
	}
}

func WrapHandler(fn AppHandlerFunc) http.HandlerFunc {
	return WrapHandlerImpl(fn, true)
}

func WrapHandlerNoParseForm(fn AppHandlerFunc) http.HandlerFunc {
	return WrapHandlerImpl(fn, false)
}

func PlaceholderHandler(name string) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request, c *Context) {
		if r.Method != "GET" {
			Serve404(w)
			return
		}
		RenderPageOrDie(w, c, "text", name)
	}
	return WrapHandler(handler)
}

func DefaultHandler(name string) http.HandlerFunc {
	handler := func(w http.ResponseWriter, r *http.Request, c *Context) {
		if r.Method != "GET" {
			Serve404(w)
			return
		}
		RenderPageOrDie(w, c, name, nil)
	}
	return WrapHandler(handler)
}

func CheckError(err error) {
	if err != nil {
		e := &ErrorWithInfo{}
		_, e.File, e.Line, _ = runtime.Caller(1)
		e.Err = err
		panic(e)
	}
}

func Assert(condition bool, format string, v ...interface{}) {
	if !condition {
		e := &ErrorWithInfo{}
		_, e.File, e.Line, _ = runtime.Caller(1)
		e.Err = errors.New(fmt.Sprintf(format, v...))
		panic(e)
	}
}

func NewSalt() string {
	return string(SecureRandom(32))
}

func SaltAndHash(salt, password string) string {
	h := sha1.New()
	io.WriteString(h, salt)
	io.WriteString(h, password)
	return string(h.Sum(nil))
}

// Taken from Gorilla GenerateRandomKey().
func SecureRandom(length int) []byte {
	k := make([]byte, length)
	if _, err := rand.Read(k); err != nil {
		return nil
	}
	return k
}

func NewCookie(name, value string, expirationInDays int) *http.Cookie {
	cookie := &http.Cookie{
		Name:    name,
		Value:   value,
		Expires: time.Now().AddDate(0, 0, expirationInDays),
	}
	return cookie
}

func AppHostname(c *Context) string {
	if kAppHostname != "" {
		return kAppHostname
	}
	return appengine.DefaultVersionHostname(c.Aec())
}

func AppHostnameForPayPal(c *Context) string {
	if kAppHostnameForPayPal != "" {
		return kAppHostnameForPayPal
	}
	return AppHostname(c)
}

func ContainsString(slice []string, str string) bool {
	for _, v := range slice {
		if str == v {
			return true
		}
	}
	return false
}
