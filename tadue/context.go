// Copyright 2012 Adam Sadovsky. All rights reserved.

// Specialized version of Gorilla context object:
// http://gorilla-web.appspot.com/src/pkg/gorilla/context/context.go

package tadue

import (
	"appengine"
)

type Context struct {
	m       map[interface{}]interface{}
	aec     appengine.Context
	session *Session
	flash   string
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

func (c *Context) SetSession(s *Session) {
	c.session = s
}

func (c *Context) DeleteSession() {
	c.session = nil
}

func (c *Context) Flash() string {
	return c.flash
}

func (c *Context) SetFlash(flash string) {
	c.flash = flash
}
