// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"net/http"
	"time"

	"appengine/datastore"
)

const (
	cookieKeyForSessionKey string = "sk"
)

// TODO(sadovsky): Add memcached layer.

// Makes a new session, sets session key cookie, and updates context.
func MakeSession(userId int64, email, fullName string, w http.ResponseWriter, c *Context) error {
	sessionKey := NewSessionKey(c.Aec())
	session := &Session{
		UserId:    userId,
		Timestamp: time.Now(),
		Email:     email,
		FullName:  fullName,
	}
	if _, err := datastore.Put(c.Aec(), sessionKey, session); err != nil {
		return err
	}
	http.SetCookie(w, NewCookie(cookieKeyForSessionKey, sessionKey.Encode(), COOKIE_LIFESPAN))
	c.SetSession(sessionKey, session)
	return nil
}

// Reads an existing session and updates context.
func ReadSession(r *http.Request, c *Context) error {
	cookie, err := r.Cookie(cookieKeyForSessionKey)
	// TODO(sadovsky): Distinguish between "no cookie" and real errors.
	if err != nil {
		return err
	}
	sessionKey, err := datastore.DecodeKey(cookie.Value)
	if err != nil {
		return err
	}
	session := &Session{}
	if err = datastore.Get(c.Aec(), sessionKey, session); err != nil {
		return err
	}
	// TODO(sadovsky): Check whether session has expired.
	c.SetSession(sessionKey, session)
	return nil
}

// Deletes existing session (if any) and updates context.
func DeleteSession(w http.ResponseWriter, c *Context) error {
	if err := datastore.Delete(c.Aec(), c.SessionKey()); err != nil {
		return err
	}
	// Expire the session cookie.
	http.SetCookie(w, NewCookie(cookieKeyForSessionKey, "", -1))
	c.DeleteSession()
	return nil
}
