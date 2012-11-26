// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"runtime/debug"

	"appengine"
	"securecookie"
)

// Data passed to Execute() for any template.
// For "base.html", Data will be a *PageData. In all other cases, it will be
// handler-supplied data.
type RenderData struct {
	FullName string // if non-empty, user is logged in
	Data     interface{}
}

type PageData struct {
	Message string
	Title   template.HTML
	Css     template.HTML
	Body    template.HTML
	Js      template.HTML
}

type ErrorWithStackTrace struct {
	Stack []byte // from debug.Stack()
	Err   error
}

func (e *ErrorWithStackTrace) Error() string {
	return fmt.Sprintf("%s\n%v", e.Stack, e.Err)
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))

func makePageData(name string, data *RenderData) (*PageData, error) {
	pd := &PageData{}

	// Body is required, unlike title, css, and js.
	html, err := ExecuteTemplate(name+"-body", data)
	if err != nil {
		return nil, err
	}
	pd.Body = html

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

func RedirectWithMessage(w http.ResponseWriter, r *http.Request, url, msg string) {
	SetFlash(msg, w)
	http.Redirect(w, r, url, http.StatusSeeOther)
}

func RenderPageOrDie(w http.ResponseWriter, c *Context, name string, data interface{}) {
	fullName := ""
	if c.LoggedIn() {
		fullName = c.Session().FullName
	}

	rd := &RenderData{FullName: fullName, Data: data}
	pd, err := makePageData(name, rd)
	if err != nil {
		ServeError(w, err)
		return
	}
	pd.Message = c.Flash()
	rd.Data = pd

	setContentTypeUtf8(w)
	if err = tmpl.ExecuteTemplate(w, "base.html", rd); err != nil {
		ServeError(w, err)
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

		// See http://blog.golang.org/2010/08/defer-panic-and-recover.html.
		defer func() {
			if data := recover(); data != nil {
				c.Aec().Errorf(fmt.Sprintf("%v", data))
				ServeError(w, data)
			}
		}()

		// Initialize the request context object.
		c.SetAec(appengine.NewContext(r))
		CheckError(ReadSession(r, c))
		if msg, err := ConsumeFlash(w, r); err != nil && err != http.ErrNoCookie {
			ServeError(w, err)
			return
		} else {
			c.SetFlash(msg)
		}

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
		e := &ErrorWithStackTrace{
			Stack: debug.Stack(),
			Err:   err,
		}
		panic(e)
	}
}

func Assert(condition bool, format string, v ...interface{}) {
	if !condition {
		e := &ErrorWithStackTrace{
			Stack: debug.Stack(),
			Err:   errors.New(fmt.Sprintf(format, v...)),
		}
		panic(e)
	}
}

func GenerateSecureRandomString() string {
	return string(securecookie.GenerateRandomKey(32))
}

func SaltAndHash(salt, password string) string {
	h := sha1.New()
	io.WriteString(h, salt)
	io.WriteString(h, password)
	return string(h.Sum(nil))
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
