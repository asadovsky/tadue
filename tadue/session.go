// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"net/http"
	"time"

	"appengine/datastore"
)

const (
	cookieSessionKey string = "sk"
)

// TODO(sadovsky): Add memcached layer.

// Makes a new session, sets session key cookie, and updates context.
func MakeSession(email, fullName string, w http.ResponseWriter, c *Context) error {
	sessionKey := NewSessionKey(c.Aec())
	session := &Session{
		Email:     email,
		Timestamp: time.Now(),
		FullName:  fullName,
	}
	if _, err := datastore.Put(c.Aec(), sessionKey, session); err != nil {
		return err
	}
	// TODO(sadovsky): Maybe save sessionKey.StringID() instead of
	// sessionKey.Encode(). Can sessionKey.StringID() contain invalid chars?
	http.SetCookie(w, NewCookie(cookieSessionKey, sessionKey.Encode(), COOKIE_LIFESPAN))
	c.SetSession(sessionKey.StringID(), session)
	return nil
}

// Reads an existing session and updates context.
// TODO(sadovsky): Handle session expiration.
func ReadSession(r *http.Request, c *Context) error {
	cookie, err := r.Cookie(cookieSessionKey)
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
	c.SetSession(sessionKey.StringID(), session)
	return nil
}

// Deletes existing session (if any) and updates context.
func DeleteSession(w http.ResponseWriter, c *Context) error {
	key := ToSessionKey(c.Aec(), c.SessionKey())
	if err := datastore.Delete(c.Aec(), key); err != nil {
		return err
	}
	// Expire the session cookie.
	http.SetCookie(w, NewCookie(cookieSessionKey, "", -1))
	c.DeleteSession()
	return nil
}
