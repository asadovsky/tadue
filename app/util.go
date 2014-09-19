package app

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"runtime/debug"
	text_template "text/template"

	"appengine"
	"securecookie"
)

type PageData struct {
	FullName string
	Message  string
	Title    template.HTML
	Css      template.HTML
	Body     template.HTML
	Js       template.HTML
}

var tmpl = template.Must(template.ParseGlob("templates/*.html"))
var text_tmpl = text_template.Must(text_template.ParseGlob("templates/*.txt"))

func fillPageData(name string, data interface{}, pd *PageData) error {
	// Body is required, unlike title, css, and js.
	html, err := ExecuteTemplate(name+"-body", data)
	if err != nil {
		return err
	}
	pd.Body = html

	maybeExecuteSubTemplate := func(subTemplateName string, field *template.HTML) error {
		fullName := name + "-" + subTemplateName
		if tmpl.Lookup(fullName) == nil {
			return nil
		}
		if html, err = ExecuteTemplate(fullName, data); err != nil {
			return err
		}
		*field = html
		return nil
	}

	if err = maybeExecuteSubTemplate("title", &pd.Title); err != nil {
		return err
	}
	if err = maybeExecuteSubTemplate("css", &pd.Css); err != nil {
		return err
	}
	if err = maybeExecuteSubTemplate("js", &pd.Js); err != nil {
		return err
	}

	return nil
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

	pd := &PageData{}
	if err := fillPageData(name, data, pd); err != nil {
		ServeError(w, err)
		return
	}
	pd.FullName = fullName
	pd.Message = c.Flash()

	setContentTypeUtf8(w)
	if err := tmpl.ExecuteTemplate(w, "base.html", pd); err != nil {
		ServeError(w, err)
	}
}

func RenderTemplateOrDie(w http.ResponseWriter, name string, data interface{}) {
	setContentTypeUtf8(w)
	if err := tmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// If the returned error is not nil, it is guaranteed to have type
// template.Error.
func ExecuteTemplate(name string, data interface{}) (template.HTML, error) {
	buf := &bytes.Buffer{}
	if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func ExecuteTextTemplate(name string, data interface{}) (string, error) {
	buf := &bytes.Buffer{}
	if err := text_tmpl.ExecuteTemplate(buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
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
				c.Aec().Errorf(fmt.Sprint(data))
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

		if appengine.IsDevAppServer() {
			tmpl = template.Must(template.ParseGlob("templates/*.html"))
			text_tmpl = text_template.Must(text_template.ParseGlob("templates/*.txt"))
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

type errorWithStackTrace struct {
	stack []byte // from debug.Stack()
	err   error
}

func (err *errorWithStackTrace) Error() string {
	if err.stack != nil {
		return fmt.Sprintf("%s\n%v", err.stack, err.err)
	}
	return fmt.Sprint(err.err)
}

func debugStack() []byte {
	if appengine.IsDevAppServer() {
		return debug.Stack()
	}
	return nil
}

func CheckError(err error) {
	if err != nil {
		panic(&errorWithStackTrace{
			stack: debugStack(),
			err:   err,
		})
	}
}

func Assert(condition bool, v ...interface{}) {
	if !condition {
		panic(&errorWithStackTrace{
			stack: debugStack(),
			err:   errors.New(fmt.Sprint(v...)),
		})
	}
}

func GenerateSecureRandomString() []byte {
	return securecookie.GenerateRandomKey(32)
}

func SaltAndHash(salt []byte, password string) []byte {
	h := sha1.New()
	io.WriteString(h, string(salt))
	io.WriteString(h, password)
	return h.Sum(nil)
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
