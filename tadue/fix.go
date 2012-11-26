// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"errors"
	"fmt"
	"net/http"

	"appengine"
	"appengine/datastore"
)

func updateAll(typeName string, makeFn func() interface{}, updateFn func(value interface{}) bool,
	c *Context) error {
	q := datastore.NewQuery(typeName).KeysOnly()
	keys, err := q.GetAll(c.Aec(), nil)
	CheckError(err)

	for _, key := range keys {
		err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
			value := makeFn()
			if err := datastore.Get(aec, key, value); err != nil {
				return err
			}
			if updateFn(value) {
				if _, err := datastore.Put(aec, key, value); err != nil {
					return err
				}
			}
			return nil
		}, nil)

		if err != nil {
			return err
		}
	}
	return nil
}

func fixUserRecordsOrDie(c *Context) {
	makeFn := func() interface{} {
		return &User{}
	}
	updateFn := func(value interface{}) bool {
		user, ok := value.(*User)
		Assert(ok, "%v", value)
		user.Email = ParseEmail(user.Email)
		user.PayPalEmail = ParseEmail(user.PayPalEmail)
		return true
	}
	CheckError(updateAll("User", makeFn, updateFn, c))
}

func fixPayRequestRecordsOrDie(c *Context) {
	makeFn := func() interface{} {
		return &PayRequest{}
	}
	updateFn := func(value interface{}) bool {
		req, ok := value.(*PayRequest)
		Assert(ok, "%v", value)
		req.PayeeEmail = ParseEmail(req.PayeeEmail)
		req.PayerEmail = ParseEmail(req.PayerEmail)
		return true
	}
	CheckError(updateAll("PayRequest", makeFn, updateFn, c))
}

func fixSessionRecordsOrDie(c *Context) {
	makeFn := func() interface{} {
		return &Session{}
	}
	updateFn := func(value interface{}) bool {
		session, ok := value.(*Session)
		Assert(ok, "%v", value)
		session.Email = ParseEmail(session.Email)
		return true
	}
	CheckError(updateAll("Session", makeFn, updateFn, c))
}

func fixUserIdRecordsOrDie(c *Context) {
	q := datastore.NewQuery("UserId").KeysOnly()
	keys, err := q.GetAll(c.Aec(), nil)
	CheckError(err)

	for _, key := range keys {
		oldEmail := key.StringID()
		newEmail := ParseEmail(oldEmail)
		if oldEmail == newEmail {
			continue
		}

		oldUserIdKey := ToUserIdKey(c.Aec(), oldEmail)
		newUserIdKey := ToUserIdKey(c.Aec(), newEmail)

		err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
			userIdStruct := &UserId{}

			// Check that there's no UserId record for newEmail.
			if err := datastore.Get(aec, newUserIdKey, userIdStruct); err != datastore.ErrNoSuchEntity {
				if err == nil {
					return errors.New(
						fmt.Sprintf("UserId already exists: %q, %q", oldEmail, newEmail))
				}
				return err
			}

			// Get the UserId record for oldEmail, and write one for newEmail.
			if err := datastore.Get(aec, oldUserIdKey, userIdStruct); err != nil {
				return err
			}
			if _, err := datastore.Put(aec, newUserIdKey, userIdStruct); err != nil {
				return err
			}

			// Delete the UserId record for oldEmail.
			if err := datastore.Delete(aec, oldUserIdKey); err != nil {
				return err
			}
			return nil
		}, makeXG())
		CheckError(err)
	}
}

func wipeSessionRecords(c *Context) {
	q := datastore.NewQuery("Session").KeysOnly()
	keys, err := q.GetAll(c.Aec(), nil)
	CheckError(err)
	CheckError(datastore.DeleteMulti(c.Aec(), keys))
}

func handleFix(w http.ResponseWriter, r *http.Request, c *Context) {
	if false {
		fixPayRequestRecordsOrDie(c)
		fixSessionRecordsOrDie(c)
		fixUserRecordsOrDie(c)
		fixUserIdRecordsOrDie(c)
	}
	wipeSessionRecords(c)
	ServeInfo(w, "Done")
}
