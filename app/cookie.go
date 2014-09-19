package app

import (
	"net/http"
	"time"

	"securecookie"
)

const (
	flashKey string = "_flash"
)

var codecs = securecookie.CodecsFromPairs(kHashKey, kBlockKey)

// Subset of http://golang.org/pkg/net/http/#Cookie.
type CookieOptions struct {
	MaxAge int
}

func SetCookie(name string, value interface{}, options *CookieOptions, w http.ResponseWriter) error {
	encoded, err := securecookie.EncodeMulti(name, value, codecs...)
	if err != nil {
		return err
	}

	// NOTE(sadovsky): If path is not "/", Chrome will not set cookies on a 302
	// redirect.
	cookie := &http.Cookie{
		Name:     name,
		Value:    encoded,
		Path:     "/",
		MaxAge:   options.MaxAge,
		HttpOnly: true, // see http://goo.gl/n4Bui
	}
	if options.MaxAge > 0 {
		cookie.Expires = time.Now().Add(time.Duration(options.MaxAge) * time.Second)
	} else if options.MaxAge < 0 {
		cookie.Expires = time.Unix(0, 0)
	}
	http.SetCookie(w, cookie)
	return nil
}

// Reads cookie value into dst.
// Returns http.ErrNoCookie if there is no cookie with the given name.
func GetCookie(name string, r *http.Request, dst interface{}) error {
	cookie, err := r.Cookie(name)
	if err != nil {
		return err
	}
	if err := securecookie.DecodeMulti(name, cookie.Value, dst, codecs...); err != nil {
		return err
	}
	return nil
}

func DeleteCookie(name string, w http.ResponseWriter) error {
	options := &CookieOptions{
		MaxAge: -1,
	}
	return SetCookie(name, "", options, w)
}

func SetFlash(value string, w http.ResponseWriter) error {
	options := &CookieOptions{
		MaxAge: 86400,
	}
	return SetCookie(flashKey, &value, options, w)
}

// Returns http.ErrNoCookie if there is no flash message.
func ConsumeFlash(w http.ResponseWriter, r *http.Request) (string, error) {
	value := ""
	if err := GetCookie(flashKey, r, &value); err != nil {
		return "", err
	}
	if err := DeleteCookie(flashKey, w); err != nil {
		return "", err
	}
	return value, nil
}
