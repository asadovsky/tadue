// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"fmt"
	"time"

	"appengine"
	"appengine/datastore"
)

// Data associated with session key.
// TODO(sadovsky):
//  - Maybe just store a key-value map?
type Session struct {
	Email     string    // key into user table
	Timestamp time.Time // when this session was created
	FullName  string    // full name of user, stored here for convenience
}

// TODO(sadovsky):
//  - Maybe add bool specifying whether user accepts paypal.
type User struct {
	Email       string // primary email of account holder
	Salt        string
	PassHash    string // hash of salted password
	FullName    string // full name of user
	PaypalEmail string // paypal account email
	EmailOk     bool   // true if user has verified their primary email
}

// Keyed by secure random number.
type VerifyEmail struct {
	Email     string    // email account to verify
	Timestamp time.Time // when this verification request was sent
}

// Keyed by secure random number.
type ResetPassword struct {
	Email string // email account for which to reset password
}

// Stores paypal response to one "Pay" request.
// Pay API reference: http://goo.gl/D6dUR
// TODO(sadovsky):
//  - Convert timestamp to time.Time?
type PaypalPayResponse struct {
	Ack           string // responseEnvelope.ack
	Build         string // responseEnvelope.build
	CorrelationId string // responseEnvelope.correlationId
	Timestamp     string // responseEnvelope.timestamp
	PayKey        string // payKey
}

// Stores the useful fields from a single paypal IPN message.
// IPN reference: http://goo.gl/bIX2Q
// TODO(sadovsky):
//  - Maybe store more fields, e.g. status_for_sender_txn.
//  - Store array of txns so we can support chained payments?
type PaypalIpnMessage struct {
	Status     string // status
	PayerEmail string // sender_email
	PayeeEmail string // transaction[0].receiver
	Amount     string // transaction[0].amount
	PayKey     string // pay_key
}

// Key in datastore is numeric id.
// TODO(sadovsky):
//  - Use enum for PaymentType.
//  - Add field for currency code (same as in paypal request).
//  - Add a PaymentStatus struct. We'll need to keep track of paypal
//    transactions, in-person payments, confirmations, etc.
type PayRequest struct {
	PayeeEmail       string // primary email of payee
	PayerEmail       string // email of payer
	Amount           float32
	PaymentType      string // "personal", "goods", or "services"
	Description      string
	CreationDate     time.Time
	IsPaid           bool      // needed for datastore queries
	PaymentDate      time.Time // unix epoch if not yet paid in full
	DeletionDate     time.Time // unix epoch if not deleted
	ReminderSentDate time.Time // most recent reminder send date, or unix epoch
}

func NewEphemeralKey(c appengine.Context, kind string) *datastore.Key {
	expirationDate := time.Now().AddDate(0, 0, 2) // expires in 2 days
	key := fmt.Sprintf("%v-%s", expirationDate.Unix(), string(SecureRandom(32)))
	return datastore.NewKey(c, kind, key, 0, nil)
}

func NewSessionKey(c appengine.Context) *datastore.Key {
	return datastore.NewKey(c, "Session", string(SecureRandom(32)), 0, nil)
}

func ToSessionKey(c appengine.Context, randomKey string) *datastore.Key {
	return datastore.NewKey(c, "Session", randomKey, 0, nil)
}

func ToUserKey(c appengine.Context, email string) *datastore.Key {
	return datastore.NewKey(c, "User", email, 0, nil)
}
