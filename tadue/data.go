// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"code.google.com/p/goauth2/oauth"
)

// Keyed by email address string.
type UserId struct {
	UserId int64
}

// Keyed by int (NewIncompleteKey).
type User struct {
	Email       string // primary email of account holder
	Salt        []byte
	PassHash    []byte // hash of salted password
	FullName    string // full name of user
	PayPalEmail string // paypal account email
	EmailOk     bool   // true if user has verified their primary email
}

// Keyed by service name (e.g. "google"), with User as parent.
type OAuthToken oauth.Token

// Payment types.
const (
	_ = iota
	PTPersonal
	PTGoods
	PTServices
)

// Keyed by int (NewIncompleteKey), with payee User as parent.
// TODO(sadovsky):
//  - Add field for currency code (same as in paypal request).
//  - Maybe add a PaymentStatus struct.
type PayRequest struct {
	PayeeEmail       string // primary email of payee
	PayerEmail       string // email of payer
	Amount           float32
	PaymentType      int // PTPersonal, PTGoods, or PTServices
	Description      string
	CreationDate     time.Time
	IsPaid           bool      // needed for datastore queries
	PaymentDate      time.Time // unix epoch if not yet paid in full
	DeletionDate     time.Time // unix epoch if not deleted
	ReminderSentDate time.Time // most recent reminder send date, or unix epoch
}

// Keyed by secure random number (NewEphemeralKey).
type VerifyEmail struct {
	UserId    int64     // user to verify
	Timestamp time.Time // when this request was made
}

// Keyed by secure random number (NewEphemeralKey).
type ResetPassword struct {
	UserId    int64     // user for which to reset password
	Timestamp time.Time // when this request was made
}

////////////////////////////////////////
// Key factories

func NewEphemeralKey(c appengine.Context, kind string) *datastore.Key {
	key := fmt.Sprintf("%v-%x", time.Now().Unix(), GenerateSecureRandomString())
	return datastore.NewKey(c, kind, key, 0, nil)
}

func ToUserKey(c appengine.Context, userId int64) *datastore.Key {
	return datastore.NewKey(c, "User", "", userId, nil)
}

func ToUserIdKey(c appengine.Context, email string) *datastore.Key {
	return datastore.NewKey(c, "UserId", email, 0, nil)
}

func ToOAuthTokenKey(c appengine.Context, userId int64, service string) *datastore.Key {
	userKey := ToUserKey(c, userId)
	return datastore.NewKey(c, "OAuthToken", service, 0, userKey)
}

////////////////////////////////////////
// Simple data getters

func GetUserOrDie(key *datastore.Key, c *Context) *User {
	user := &User{}
	CheckError(datastore.Get(c.Aec(), key, user))
	return user
}

func GetUserId(email string, c *Context) (int64, error) {
	userIdKey := ToUserIdKey(c.Aec(), email)
	userId := &UserId{}
	if err := datastore.Get(c.Aec(), userIdKey, userId); err != nil {
		return 0, err
	}
	return userId.UserId, nil
}

func GetUserIdOrDie(email string, c *Context) int64 {
	userId, err := GetUserId(email, c)
	CheckError(err)
	return userId
}

func GetUserFromUserId(userId int64, c *Context) (*User, error) {
	userKey := ToUserKey(c.Aec(), userId)
	user := &User{}
	if err := datastore.Get(c.Aec(), userKey, user); err != nil {
		return nil, err
	}
	return user, nil
}

func GetUserFromUserIdOrDie(userId int64, c *Context) *User {
	user, err := GetUserFromUserId(userId, c)
	CheckError(err)
	return user
}

func GetUserFromSessionOrDie(c *Context) *User {
	c.AssertLoggedIn()
	return GetUserFromUserIdOrDie(c.Session().UserId, c)
}

func GetUserFromEmail(email string, c *Context) (int64, *User, error) {
	userId, err := GetUserId(email, c)
	if err != nil {
		return 0, nil, err
	}
	user, err := GetUserFromUserId(userId, c)
	if err != nil {
		return 0, nil, err
	}
	return userId, user, nil
}

func GetUserFromEmailOrDie(email string, c *Context) (int64, *User) {
	userId, user, err := GetUserFromEmail(email, c)
	CheckError(err)
	return userId, user
}

func GetOAuthTokenFromUserId(userId int64, service string, c *Context) (*OAuthToken, error) {
	tokenKey := ToOAuthTokenKey(c.Aec(), userId, service)
	token := &OAuthToken{}
	if err := datastore.Get(c.Aec(), tokenKey, token); err != nil {
		return nil, err
	}
	return token, nil
}

//////////////////////////////
// Other util functions

func GetPayeeUserKey(reqCode string) *datastore.Key {
	reqKey, err := datastore.DecodeKey(reqCode)
	CheckError(err)
	return reqKey.Parent()
}

//////////////////////////////
// String parsing functions

// Typically used for parsing form values.
// All "Parse" functions assert on error.

var paymentTypeMap = map[string]int{
	"personal": PTPersonal,
	"goods":    PTGoods,
	"services": PTServices,
}

func ParsePaymentType(paymentTypeStr string) int {
	res := paymentTypeMap[paymentTypeStr]
	Assert(res != 0, fmt.Sprintf("Invalid paymentTypeStr: %q", paymentTypeStr))
	return res
}

var emailRegexp = regexp.MustCompile(`^\S+@\S+\.\S+$`)

func ParseEmail(email string) string {
	Assert(emailRegexp.MatchString(email), "Invalid email: %q", email)
	// Canonicalize the email address.
	return strings.ToLower(email)
}

func ParseAmount(amount string) float32 {
	amount64, err := strconv.ParseFloat(strings.TrimLeft(amount, "$"), 32)
	CheckError(err)
	return float32(amount64)
}

var fullNameRegexp = regexp.MustCompile(`^(?:\S+ )+\S+$`)

func ParseFullName(fullName string) string {
	Assert(fullNameRegexp.MatchString(fullName), "Invalid fullName: %q", fullName)
	return fullName
}
