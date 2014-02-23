package app

import (
	"net/http"
	"time"
)

const (
	sessionKey string = "_session"
)

type Session struct {
	UserId    int64
	Timestamp time.Time // when this session was created
	Email     string    // email of user, stored here for convenience
	FullName  string    // full name of user, stored here for convenience
}

// Sets session cookie and updates context.
func MakeSession(userId int64, email, fullName string, w http.ResponseWriter, c *Context) error {
	s := &Session{
		UserId:    userId,
		Timestamp: time.Now(),
		Email:     email,
		FullName:  fullName,
	}
	options := &CookieOptions{
		MaxAge: 86400 * SESSION_COOKIE_LIFESPAN,
	}
	if err := SetCookie(sessionKey, s, options, w); err != nil {
		return err
	}
	c.SetSession(s)
	return nil
}

// Updates session cookie and context.
func UpdateSession(s *Session, w http.ResponseWriter, c *Context) error {
	options := &CookieOptions{
		MaxAge: 86400*SESSION_COOKIE_LIFESPAN - int(time.Now().Sub(s.Timestamp).Seconds()),
	}
	if err := SetCookie(sessionKey, s, options, w); err != nil {
		return err
	}
	c.SetSession(s)
	return nil
}

// Reads session cookie and updates context.
func ReadSession(r *http.Request, c *Context) error {
	s := &Session{}
	err := GetCookie(sessionKey, r, s)
	if err == http.ErrNoCookie {
		return nil
	} else if err != nil {
		return err
	}
	if time.Now().Before(s.Timestamp.AddDate(0, 0, SESSION_COOKIE_LIFESPAN)) {
		c.SetSession(s)
	}
	return nil
}

// Deletes session cookie (if any) and updates context.
func DeleteSession(w http.ResponseWriter, c *Context) error {
	if err := DeleteCookie(sessionKey, w); err != nil {
		return err
	}
	c.DeleteSession()
	return nil
}
