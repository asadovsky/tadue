// Copyright 2012 Adam Sadovsky. All rights reserved.

// Specialized version of Gorilla context object:
// http://gorilla-web.appspot.com/src/pkg/gorilla/context/context.go

package tadue

import (
	"appengine"
	"appengine/datastore"
)

type Context struct {
	m          map[interface{}]interface{}
	aec        appengine.Context
	sessionKey *datastore.Key
	session    *Session
}

func (c *Context) Get(key interface{}) interface{} {
	if c.m != nil {
		return c.m[key]
	}
	return nil
}

func (c *Context) Set(key interface{}, value interface{}) {
	if c.m == nil {
		c.m = make(map[interface{}]interface{})
	}
	c.m[key] = value
}

func (c *Context) Delete(key interface{}) {
	delete(c.m, key)
}

func (c *Context) Aec() appengine.Context {
	return c.aec
}

func (c *Context) SetAec(aec appengine.Context) {
	c.aec = aec
}

func (c *Context) LoggedIn() bool {
	return c.session != nil
}

func (c *Context) AssertLoggedIn() {
	Assert(c.session != nil, "Not logged in")
}

func (c *Context) AssertNotLoggedIn() {
	Assert(c.session == nil, "Logged in")
}

func (c *Context) Session() *Session {
	Assert(c.session != nil, "Session is nil") // catch common mistake
	return c.session
}

func (c *Context) SessionKey() *datastore.Key {
	return c.sessionKey
}

func (c *Context) SetSession(sessionKey *datastore.Key, s *Session) {
	c.sessionKey = sessionKey
	c.session = s
}

func (c *Context) DeleteSession() {
	c.sessionKey = nil
	c.session = nil
}
