// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

// TODO(sadovsky):
//  - Protect against CSRF.
//  - Check that all transactions are idempotent.

import (
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"appengine"
	"appengine/datastore"
	"appengine/mail"
	"appengine/taskqueue"
)

func makePayUrl(reqCode string, c *Context) string {
	// TODO(sadovsky): Use https?
	return fmt.Sprintf("http://%s/pay?reqCode=%s",
		appengine.DefaultVersionHostname(c.Aec()), reqCode)
}

func makeWrongPasswordError(email string) error {
	return errors.New(fmt.Sprintf("Wrong password for user: %q", email))
}

func makePayRequestQuery(userKey *datastore.Key, isPaid bool) *datastore.Query {
	q := datastore.NewQuery("PayRequest")
	if userKey != nil {
		q = q.Ancestor(userKey)
	}
	return q.Filter("DeletionDate =", time.Unix(0, 0)).Filter("IsPaid =", isPaid)
}

func makeSentLinkMessage(linkType, email string) string {
	return fmt.Sprintf("%s link sent to %s.", linkType, email)
}

func makeExpiredLinkError(linkType string) error {
	return errors.New(fmt.Sprintf("%s link has expired. Please request another.", linkType))
}

func makeXG() *datastore.TransactionOptions {
	return &datastore.TransactionOptions{
		XG: true,
	}
}

// Applies updateFn to each PayRequest specified in reqCodes.
// If checkUser is true, aborts the transaction if any PayRequest does not
// belong to the current user.
func updatePayRequests(
	reqCodes []string, updateFn func(reqCode string, req *PayRequest) bool, checkUser bool,
	c *Context) ([]string, error) {
	Assert(len(reqCodes) > 0, "No reqCodes")
	if checkUser {
		c.AssertLoggedIn()
	}

	// NOTE(sadovsky): Multi-row, single entity group transaction.
	updatedReqCodes := []string{}
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		for _, reqCode := range reqCodes {
			reqKey, err := datastore.DecodeKey(reqCode)
			CheckError(err)
			req := &PayRequest{}
			if err := datastore.Get(aec, reqKey, req); err != nil {
				return err
			}
			// Check that this PayRequest belongs to the current user; if not, abort.
			if checkUser && (c.Session().Email != req.PayeeEmail) {
				return errors.New(
					fmt.Sprintf("Unauthorized user: %q != %q", c.Session().Email, req.PayeeEmail))
			}
			if updateFn(reqCode, req) {
				updatedReqCodes = append(updatedReqCodes, reqCode)
				if _, err := datastore.Put(aec, reqKey, req); err != nil {
					return err
				}
			}
		}
		return nil
	}, nil)

	if err != nil {
		return []string{}, err
	}
	return updatedReqCodes, nil
}

// If password is nil, performs update without checking password.
// Use with caution!
func updateUser(userId int64, password *string, updateFn func(user *User) bool, c *Context) error {
	// Sanity check.
	if password != nil {
		c.AssertLoggedIn()
		Assert(c.Session().UserId == userId, "")
	}
	return datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		userKey := ToUserKey(c.Aec(), userId)
		user := &User{}
		if err := datastore.Get(aec, userKey, user); err != nil {
			return err
		}
		// Check password.
		if password != nil && SaltAndHash(user.Salt, *password) != user.PassHash {
			return makeWrongPasswordError(user.Email)
		}
		if updateFn(user) {
			if _, err := datastore.Put(aec, userKey, user); err != nil {
				return err
			}
		}
		return nil
	}, nil)
}

// TODO(sadovsky): Delete ResetPassword record.
func useResetPassword(encodedKey string, c *Context) (int64, error) {
	key, err := datastore.DecodeKey(encodedKey)
	CheckError(err)
	v := &ResetPassword{}
	CheckError(datastore.Get(c.Aec(), key, v))
	if time.Now().After(v.Timestamp.AddDate(0, 0, RESET_PASSWORD_LIFESPAN)) {
		return 0, makeExpiredLinkError("Password reset")
	}
	return v.UserId, nil
}

// TODO(sadovsky): Delete VerifyEmail record.
func useVerifyEmail(encodedKey string, c *Context) (int64, error) {
	key, err := datastore.DecodeKey(encodedKey)
	CheckError(err)
	v := &VerifyEmail{}
	CheckError(datastore.Get(c.Aec(), key, v))
	if time.Now().After(v.Timestamp.AddDate(0, 0, VERIFY_EMAIL_LIFESPAN)) {
		return 0, makeExpiredLinkError("Email verification")
	}
	return v.UserId, nil
}

// FIXME(sadovsky): Differentiate between user error and app error.
func doLogin(w http.ResponseWriter, r *http.Request, c *Context) (*User, error) {
	// TODO(sadovsky): Form validation.
	email := r.FormValue("login-email")
	password := r.FormValue("login-password")
	Assert(email != "", "")

	userId, user, err := GetUserFromEmail(email, c)
	if err == datastore.ErrNoSuchEntity {
		return nil, errors.New(fmt.Sprintf("No such user: %q", email))
	}
	CheckError(err)

	if SaltAndHash(user.Salt, password) != user.PassHash {
		return nil, makeWrongPasswordError(user.Email)
	}

	CheckError(MakeSession(userId, user.Email, user.FullName, w, c))
	c.Aec().Infof("Logged in user: %q", user.Email)
	return user, nil
}

// FIXME(sadovsky): Differentiate between user error and app error.
func doSignup(w http.ResponseWriter, r *http.Request, c *Context) (*User, error) {
	// TODO(sadovsky): Form validation.
	salt := NewSalt()
	newUser := &User{
		Email:       r.FormValue("signup-email"),
		Salt:        salt,
		PassHash:    SaltAndHash(salt, r.FormValue("signup-password")),
		FullName:    r.FormValue("signup-name"),
		PayPalEmail: r.FormValue("signup-paypal-email"),
	}
	if r.FormValue("signup-copy-email") == "on" {
		newUser.PayPalEmail = r.FormValue("signup-email")
	}
	// TODO(sadovsky): Check that the PayPal account is valid and confirmed.

	// Check whether user already exists. If so, report error; if not, create new
	// account.
	userIdKey := ToUserIdKey(c.Aec(), newUser.Email)
	var userId int64 = 0
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		userIdStruct := &UserId{}
		err := datastore.Get(aec, userIdKey, userIdStruct)
		if err != nil && err != datastore.ErrNoSuchEntity {
			return err
		}
		if err == nil { // entity already exists
			return errors.New(fmt.Sprintf("User already exists: %q", newUser.Email))
		}

		incompleteUserKey := datastore.NewIncompleteKey(c.Aec(), "User", nil)
		userKey, err := datastore.Put(c.Aec(), incompleteUserKey, newUser)
		CheckError(err)

		userId = userKey.IntID()
		userIdStruct = &UserId{
			UserId: userId,
		}
		if _, err := datastore.Put(aec, userIdKey, userIdStruct); err != nil {
			return err
		}
		aec.Infof("Signed up user: %q", newUser.Email)
		return nil
	}, makeXG())

	if err != nil {
		return nil, err
	}

	if err = MakeSession(userId, newUser.Email, newUser.FullName, w, c); err != nil {
		return nil, err
	}
	if err = doInitiateVerifyEmail(c); err != nil {
		return nil, err
	}
	return newUser, nil
}

func doInitiateResetPassword(email string, c *Context) error {
	// First, check that it's a known user email.
	userId, user := GetUserFromEmailOrDie(email, c)

	// Next, create the ResetPassword record.
	v := &ResetPassword{
		UserId:    userId,
		Timestamp: time.Now(),
	}
	key := NewEphemeralKey(c.Aec(), "ResetPassword")
	key, err := datastore.Put(c.Aec(), key, v)
	if err != nil {
		return err
	}

	// Finally, send the email.
	resetUrl := fmt.Sprintf("http://%s/account/change-password?key=%s",
		appengine.DefaultVersionHostname(c.Aec()), key.Encode())
	data := map[string]interface{}{
		"fullName": user.FullName,
		"email":    user.Email,
		"resetUrl": resetUrl,
	}
	body, err := ExecuteTemplate("email-reset-password.html", data)
	if err != nil {
		return err
	}

	msg := &mail.Message{
		Sender:   "noreply@tadue.com",
		To:       []string{user.Email},
		Subject:  "Reset your Tadue password",
		HTMLBody: string(body),
	}
	return mail.Send(c.Aec(), msg)
}

func doInitiateVerifyEmail(c *Context) error {
	// Create the VerifyEmail record.
	v := &VerifyEmail{
		UserId:    c.Session().UserId,
		Timestamp: time.Now(),
	}
	key := NewEphemeralKey(c.Aec(), "VerifyEmail")
	key, err := datastore.Put(c.Aec(), key, v)
	if err != nil {
		return err
	}

	// Send the email.
	verifUrl := fmt.Sprintf("http://%s/account/verif?key=%s",
		appengine.DefaultVersionHostname(c.Aec()), key.Encode())
	data := map[string]interface{}{
		"fullname": c.Session().FullName,
		"verifUrl": verifUrl,
	}
	body, err := ExecuteTemplate("email-verif.html", data)
	if err != nil {
		return err
	}

	msg := &mail.Message{
		Sender:   "noreply@tadue.com",
		To:       []string{c.Session().Email},
		Subject:  "Welcome to Tadue",
		HTMLBody: string(body),
	}
	return mail.Send(c.Aec(), msg)
}

func doEnqueuePayRequestEmails(reqCodes []string, c *Context) error {
	c.Aec().Infof("Enqueuing pay request emails for reqCodes: %v", reqCodes)
	if len(reqCodes) == 0 {
		return nil
	}
	v := url.Values{}
	v.Set("reqCodes", strings.Join(reqCodes, ","))
	t := taskqueue.NewPOSTTask("/tasks/send-pay-request-emails", v)
	_, err := taskqueue.Add(c.Aec(), t, "")
	return err
}

func doEnqueueGotPaidEmail(reqCode string, c *Context) error {
	c.Aec().Infof("Enqueuing got paid email for reqCode: %v", reqCode)
	v := url.Values{}
	v.Set("reqCode", reqCode)
	t := taskqueue.NewPOSTTask("/tasks/send-got-paid-email", v)
	_, err := taskqueue.Add(c.Aec(), t, "")
	return err
}

// TODO(sadovsky): Maybe use updateUser() here.
func doSetEmailOk(userId int64, c *Context) (email string, sentPayRequestEmails bool, err error) {
	userKey := ToUserKey(c.Aec(), userId)
	user := &User{}
	alreadyVerified := false

	// Update user.EmailOk.
	err = datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		err := datastore.Get(aec, userKey, user)
		if err != nil {
			return err
		}
		// If already verified, do nothing.
		if user.EmailOk {
			alreadyVerified = true
			return nil
		}
		user.EmailOk = true
		_, err = datastore.Put(aec, userKey, user)
		return err
	}, nil)

	if err != nil {
		return "", false, err
	} else if alreadyVerified {
		return user.Email, false, nil
	}
	c.Aec().Infof("Verified email: %q", user.Email)

	// Enqueue pay request emails.
	q := makePayRequestQuery(userKey, false).KeysOnly()
	reqKeys, err := q.GetAll(c.Aec(), nil)
	CheckError(err)
	reqCodes := make([]string, len(reqKeys))
	for i, reqKey := range reqKeys {
		reqCodes[i] = reqKey.Encode()
	}
	CheckError(doEnqueuePayRequestEmails(reqCodes, c))
	return user.Email, len(reqCodes) > 0, nil
}

//////////////////////////////
// Handlers

func handleHome(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.URL.Path != "/" {
		Serve404(w)
		return
	}
	RenderPageOrDie(w, c, "home", nil)
}

func handleAbout(w http.ResponseWriter, r *http.Request, c *Context) {
	RenderPageOrDie(w, c, "about", nil)
}

func handleTerms(w http.ResponseWriter, r *http.Request, c *Context) {
	RenderPageOrDie(w, c, "terms", nil)
}

func handlePrivacy(w http.ResponseWriter, r *http.Request, c *Context) {
	RenderPageOrDie(w, c, "privacy", nil)
}

func handleHelp(w http.ResponseWriter, r *http.Request, c *Context) {
	RenderPageOrDie(w, c, "help", nil)
}

func handleIpn(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	requestBytes, err := ioutil.ReadAll(r.Body)
	CheckError(err)

	// Note: If we call ParseForm() before ReadAll(), the IPN dance fails.
	// ParseForm() must be mutating the request somehow.
	CheckError(r.ParseForm())
	reqCode := r.FormValue("reqCode")
	Assert(reqCode != "", "No reqCode")
	reqKey, err := datastore.DecodeKey(reqCode)
	CheckError(err)

	msg, err := PayPalValidateIpn(string(requestBytes), c.Aec())
	CheckError(err)
	c.Aec().Infof("%+v", msg) // plus flag (%+v) adds field names

	// If the transaction is not completed, we don't care.
	// TODO(sadovsky): Should we care? Probably.
	if msg.Status != "COMPLETED" {
		ServeEmpty200(w)
		return
	}

	currencyAndAmount := strings.Split(msg.Amount, " ")
	Assert(len(currencyAndAmount) == 2, "Unexpected msg.Amount: %q", msg.Amount)
	// TODO(sadovsky): Support other currencies.
	Assert(currencyAndAmount[0] == "USD", "Unexpected currency in msg.Amount: %q", msg.Amount)

	amount64, err := strconv.ParseFloat(currencyAndAmount[1], 32)
	CheckError(err)

	err = datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		req := &PayRequest{}
		err := datastore.Get(aec, reqKey, req)
		if err != nil {
			return err
		}

		// Check payee email and amount.
		// We don't check payer email because their paypal email may differ from
		// their personal email.
		if msg.PayeeEmail != req.PayeeEmail {
			return errors.New(fmt.Sprintf("Wrong payee: %q != %q", msg.PayeeEmail, req.PayeeEmail))
		}
		if float32(amount64) != req.Amount {
			return errors.New(fmt.Sprintf("Wrong amount: %v != %v", amount64, req.Amount))
		}

		// If already marked as paid, do nothing.
		if req.PaymentDate != time.Unix(0, 0) {
			return nil
		}
		req.IsPaid = true
		req.PaymentDate = time.Now()
		if _, err := datastore.Put(aec, reqKey, req); err != nil {
			return err
		}
		return nil
	}, nil)
	CheckError(err)

	CheckError(doEnqueueGotPaidEmail(reqCode, c))
	ServeEmpty200(w)
}

func handlePay(w http.ResponseWriter, r *http.Request, c *Context) {
	reqCode := r.FormValue("reqCode")
	Assert(reqCode != "", "No reqCode")
	reqKey, err := datastore.DecodeKey(reqCode)
	CheckError(err)

	method := r.FormValue("method")
	Assert(method == "" || method == "paypal", "Invalid method: %q", method)

	// TODO(sadovsky): Cache PayRequest and User lookups so that multiple loads of
	// this page (e.g. first with method="", then with method="paypal") don't all
	// hit the datastore.
	req := &PayRequest{}
	CheckError(datastore.Get(c.Aec(), reqKey, req))

	// If request has already been paid, show an error.
	// TODO(sadovsky): Make error message more friendly.
	if req.PaymentDate != time.Unix(0, 0) {
		RenderPageOrDie(w, c, "text", "Already paid.")
		return
	}

	// Get payee's User object so we can get their name and paypal email address.
	payee := GetUserOrDie(GetPayeeUserKey(reqCode), c)

	if method == "" {
		data := map[string]interface{}{
			"url":   r.URL.String(),
			"req":   req,
			"payee": payee,
		}
		RenderPageOrDie(w, c, "pay", data)
	} else { // method == "paypal"
		// Note: According to the documentation, the pay key is only valid for three
		// hours. As such, we cannot request it before the payer arrives.
		response, err := PayPalSendPayRequest(
			reqCode, payee.PayPalEmail, req.Description, req.Amount, c.Aec())
		CheckError(err)
		// TODO(sadovsky): Store the response inside the PayRequest via transaction.
		if response.Ack != "Success" {
			ServeError(w, response)
			return
		}
		http.Redirect(w, r, PayPalMakePayUrl(response.PayKey), http.StatusFound)
	}
}

func handlePayDone(w http.ResponseWriter, r *http.Request, c *Context) {
	// TODO(sadovsky): Invite user to sign up for Tadue.
	RenderPageOrDie(w, c, "text", "Payment processed successfully. Thanks for using Tadue!")
}

func handleRequestPayment(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "request-payment", nil)
	} else if r.Method == "POST" {
		CheckError(r.ParseForm())

		var user *User
		var err error

		if c.LoggedIn() {
			user = GetUserFromSessionOrDie(c)
		} else {
			doSignupValue := r.FormValue("do-signup")
			if doSignupValue == "true" {
				user, err = doSignup(w, r, c)
				CheckError(err)
			} else {
				Assert(doSignupValue == "false", "Invalid doSignupValue: %q", doSignupValue)
				// TODO(sadovsky): Show nice error page on failed login.
				user, err = doLogin(w, r, c)
				CheckError(err)
			}
		}

		// At this point the user must be logged in, and we must have their User
		// struct.
		c.AssertLoggedIn()
		Assert(user != nil, "User is nil")

		amount, err := strconv.ParseFloat(strings.TrimLeft(r.FormValue("amount"), "$"), 32)
		CheckError(err)

		// TODO(sadovsky): Form validation.
		req := &PayRequest{
			PayeeEmail:       c.Session().Email,
			PayerEmail:       r.FormValue("payer-email"),
			Amount:           float32(amount),
			PaymentType:      r.FormValue("payment-type"),
			Description:      r.FormValue("description"),
			CreationDate:     time.Now(),
			PaymentDate:      time.Unix(0, 0),
			DeletionDate:     time.Unix(0, 0),
			ReminderSentDate: time.Unix(0, 0),
		}

		incompleteReqKey := datastore.NewIncompleteKey(
			c.Aec(), "PayRequest", ToUserKey(c.Aec(), c.Session().UserId))
		reqKey, err := datastore.Put(c.Aec(), incompleteReqKey, req)
		CheckError(err)
		reqCode := reqKey.Encode()

		// If payee's email is already verified, enqueue the payment request email.
		if user.EmailOk {
			CheckError(doEnqueuePayRequestEmails([]string{reqCode}, c))
		}

		payUrl := makePayUrl(reqCode, c)
		data := map[string]interface{}{
			"user":       user,
			"payerEmail": req.PayerEmail,
			"payUrl":     payUrl,
		}
		RenderPageOrDie(w, c, "sent-request", data)
	} else {
		Serve404(w)
	}
}

func handleLogin(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "login", nil)
	} else if r.Method == "POST" {
		c.AssertNotLoggedIn()
		CheckError(r.ParseForm())
		// TODO(sadovsky): Show nice error page on failed login.
		_, err := doLogin(w, r, c)
		CheckError(err)
		http.Redirect(w, r, "/payments", http.StatusFound)
	} else {
		Serve404(w)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	CheckError(DeleteSession(w, c))
	RenderPageOrDie(w, c, "home", nil)
}

func handleSignup(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "signup", nil)
	} else if r.Method == "POST" {
		c.AssertNotLoggedIn()
		CheckError(r.ParseForm())
		_, err := doSignup(w, r, c)
		CheckError(err)
		http.Redirect(w, r, "/payments?new", http.StatusFound)
	} else {
		Serve404(w)
	}
}

type RenderablePayRequest struct {
	ReqCode      string
	PayUrl       string
	PayerEmail   string
	Amount       string
	Description  string
	IsPaid       bool
	Status       string
	CreationDate string
}

func renderDate(date time.Time) string {
	return date.Format("Jan 2")
}

func renderAmount(amount float32) string {
	return strconv.FormatFloat(float64(amount), 'f', 2, 32)
}

func getRecentPayRequestsOrDie(userId int64, c *Context) []RenderablePayRequest {
	userKey := ToUserKey(c.Aec(), userId)
	reqs := []PayRequest{}

	// Get unpaid requests. Then, if there's still space, append paid requests.
	// Note the similarity to email inbox rendering.
	q := makePayRequestQuery(userKey, false).
		Order("-CreationDate").Limit(MAX_PAYMENTS_TO_SHOW)
	reqKeys, err := q.GetAll(c.Aec(), &reqs)
	CheckError(err)

	// NOTE(sadovsky): Because GAE does not allow Filter("x !="), and also does
	// not allow Filter("x >") with Order("y") for x != y, we must filter on
	// IsPaid rather than PaymentDate.
	if len(reqs) < MAX_PAYMENTS_TO_SHOW {
		q = makePayRequestQuery(userKey, true).
			Order("-CreationDate").Limit(MAX_PAYMENTS_TO_SHOW - len(reqs))
		extraReqKeys, err := q.GetAll(c.Aec(), &reqs)
		CheckError(err)
		reqKeys = append(reqKeys, extraReqKeys...)
	}
	Assert(len(reqs) <= MAX_PAYMENTS_TO_SHOW, "")
	Assert(len(reqs) == len(reqKeys), "")

	// Convert PayRequests to RenderablePayRequests.
	rendReqs := make([]RenderablePayRequest, len(reqs))
	for i, pr := range reqs {
		rpr := &rendReqs[i]
		rpr.ReqCode = reqKeys[i].Encode()
		rpr.PayUrl = makePayUrl(rpr.ReqCode, c)
		rpr.PayerEmail = pr.PayerEmail
		rpr.Amount = fmt.Sprintf("$%.2f", pr.Amount)
		rpr.Description = pr.Description
		rpr.IsPaid = pr.IsPaid
		// TODO(sadovsky): Get user's time zone during signup.
		// https://bitbucket.org/pellepim/jstimezonedetect/wiki/Home
		// http://arshaw.com/xdate/
		if pr.PaymentDate != time.Unix(0, 0) {
			rpr.Status = "Paid on " + renderDate(pr.PaymentDate)
		} else if pr.ReminderSentDate != time.Unix(0, 0) {
			rpr.Status = "Emailed on " + renderDate(pr.ReminderSentDate)
		} else {
			// FIXME(sadovsky): Show different status if user is verified but email
			// hasn't been sent.
			rpr.Status = "Pending verification"
		}
		rpr.CreationDate = renderDate(pr.CreationDate)
	}
	return rendReqs
}

func handleAccount(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	CheckError(r.ParseForm())
	isNew := r.Form["new"] != nil

	user := GetUserFromSessionOrDie(c)
	rendReqs := getRecentPayRequestsOrDie(c.Session().UserId, c)
	data := map[string]interface{}{
		"user":             user,
		"isNew":            isNew,
		"rendReqs":         rendReqs,
		"undoableReqCodes": "",
	}
	RenderPageOrDie(w, c, "payments", data)
}

func renderRecentRequests(w http.ResponseWriter, undoableReqCodes []string, c *Context) {
	c.AssertLoggedIn()
	rendReqs := getRecentPayRequestsOrDie(c.Session().UserId, c)
	data := map[string]interface{}{
		"rendReqs":         rendReqs,
		"undoableReqCodes": strings.Join(undoableReqCodes, ","),
	}
	RenderTemplateOrDie(w, "payments-data", data)
}

func handleMarkAsPaid(w http.ResponseWriter, r *http.Request, c *Context) {
	// Note similarity to handleDelete().
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertLoggedIn()
	CheckError(r.ParseForm())
	reqCodes := strings.Split(r.FormValue("reqCodes"), ",")
	undo := r.Form["undo"] != nil
	updateFn := func(reqCode string, req *PayRequest) bool {
		if undo {
			Assert(req.IsPaid, reqCode)
			req.IsPaid = false
			req.PaymentDate = time.Unix(0, 0)
		} else {
			if req.IsPaid {
				return false
			}
			req.IsPaid = true
			req.PaymentDate = time.Now()
		}
		return true
	}
	undoableReqCodes, err := updatePayRequests(reqCodes, updateFn, true, c)
	CheckError(err)
	renderRecentRequests(w, undoableReqCodes, c)
}

func handleSendReminder(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertLoggedIn()
	CheckError(r.ParseForm())
	reqCodes := strings.Split(r.FormValue("reqCodes"), ",")
	// TODO(sadovsky): Show error if user is not verified.
	// TODO(sadovsky): Show error if user exceeds email rate limit.
	// TODO(sadovsky): Optimistically show requests as sent.
	CheckError(doEnqueuePayRequestEmails(reqCodes, c))
	renderRecentRequests(w, []string{}, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request, c *Context) {
	// Note similarity to handleMarkAsPaid().
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertLoggedIn()
	CheckError(r.ParseForm())
	reqCodes := strings.Split(r.FormValue("reqCodes"), ",")
	undo := r.Form["undo"] != nil
	updateFn := func(reqCode string, req *PayRequest) bool {
		if undo {
			req.DeletionDate = time.Unix(0, 0)
		} else {
			req.DeletionDate = time.Now()
		}
		return true
	}
	undoableReqCodes, err := updatePayRequests(reqCodes, updateFn, true, c)
	CheckError(err)
	renderRecentRequests(w, undoableReqCodes, c)
}

func handleSettings(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	RenderPageOrDie(w, c, "settings", nil)
}

// Handles both changes and resets.
func handleChangePassword(w http.ResponseWriter, r *http.Request, c *Context) {
	CheckError(r.ParseForm())
	encodedKey := r.FormValue("key")
	isPasswordResetRequest := encodedKey != ""
	if !isPasswordResetRequest {
		// TODO(sadovsky): Instead of asserting, redirect to login page.
		c.AssertLoggedIn()
	}
	if r.Method == "GET" {
		if !isPasswordResetRequest {
			RenderPageOrDie(w, c, "change-password", map[string]interface{}{"key": nil})
		} else { // password reset request
			_, err := useResetPassword(encodedKey, c)
			if err != nil {
				RenderPageOrDie(w, c, "text", err)
				return
			}
			RenderPageOrDie(w, c, "change-password", map[string]interface{}{"key": encodedKey})
		}
	} else if r.Method == "POST" {
		var err error = nil
		updateFn := func(user *User) bool {
			salt := NewSalt()
			user.Salt = salt
			user.PassHash = SaltAndHash(salt, r.FormValue("new-password"))
			return true
		}
		if !isPasswordResetRequest {
			currentPassword := r.FormValue("current-password")
			err = updateUser(c.Session().UserId, &currentPassword, updateFn, c)
		} else { // password reset request
			userId, err := useResetPassword(encodedKey, c)
			CheckError(err)
			err = updateUser(userId, nil, updateFn, c)
		}
		// FIXME(sadovsky): Differentiate between user error and app error.
		CheckError(err)
		RenderPageOrDie(w, c, "text", "Password changed successfully.")
	} else {
		Serve404(w)
	}
}

func handleResetPassword(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "reset-password", nil)
	} else if r.Method == "POST" {
		CheckError(r.ParseForm())
		email := r.FormValue("email")
		CheckError(doInitiateResetPassword(email, c))
		RenderPageOrDie(w, c, "text", makeSentLinkMessage("Password reset", email))
	} else {
		Serve404(w)
	}
}

func handleSendVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	CheckError(doInitiateVerifyEmail(c))
	RenderPageOrDie(w, c, "text", makeSentLinkMessage("Email verification", c.Session().Email))
}

func doRenderVerifMsg(email string, sentPayRequestEmails bool, w http.ResponseWriter, c *Context) {
	msg := fmt.Sprintf("Email address %s has been verified.", email)
	if sentPayRequestEmails {
		msg += " All pending payment requests have been sent."
	}
	RenderPageOrDie(w, c, "text", msg)
}

func handleDebugVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	email, sentPayRequestEmails, err := doSetEmailOk(c.Session().UserId, c)
	Assert(email == c.Session().Email, "")
	CheckError(err)
	doRenderVerifMsg(email, sentPayRequestEmails, w, c)
}

func handleVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	CheckError(r.ParseForm())
	encodedKey := r.FormValue("key")
	Assert(encodedKey != "", "No key")
	userId, err := useVerifyEmail(encodedKey, c)
	if err != nil {
		RenderPageOrDie(w, c, "text", err)
	}
	email, sentPayRequestEmails, err := doSetEmailOk(userId, c)
	CheckError(err)
	doRenderVerifMsg(email, sentPayRequestEmails, w, c)
}

func handleSendPayRequestEmails(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	CheckError(r.ParseForm())
	reqCodes := strings.Split(r.FormValue("reqCodes"), ",")
	Assert(len(reqCodes) > 0, "No reqCodes")

	payeeUserKey := GetPayeeUserKey(reqCodes[0])
	// Verify that all reqCodes have the same parent.
	for _, reqCode := range reqCodes {
		Assert(GetPayeeUserKey(reqCode).Equal(payeeUserKey), "")
	}

	payee := &User{}
	// TODO(sadovsky): Add write-through memcached layer, at least for User.
	CheckError(datastore.Get(c.Aec(), payeeUserKey, payee))
	if !payee.EmailOk {
		// Payee's email has not been verified, so do not send any emails.
		return
	}

	// Sends payment request email and updates ReminderSentDate in PayRequest.
	updateFn := func(reqCode string, req *PayRequest) bool {
		if req.IsPaid {
			return false
		} else if req.ReminderSentDate.After(time.Now().AddDate(0, 0, -PAY_REQUEST_EMAIL_RATE_LIMIT)) {
			return false
		}

		isReminder := req.ReminderSentDate != time.Unix(0, 0)
		data := map[string]interface{}{
			"payerEmail":    req.PayerEmail,
			"payeeEmail":    payee.PayPalEmail,
			"payeeFullName": payee.FullName,
			"amount":        renderAmount(req.Amount),
			"description":   req.Description,
			"payUrl":        makePayUrl(reqCode, c),
			"isReminder":    isReminder,
			"creationDate":  renderDate(req.CreationDate),
		}
		body, err := ExecuteTemplate("email-pay-request.html", data)
		CheckError(err)

		var subject string
		if isReminder {
			subject = "Reminder of payment"
		} else {
			subject = "Payment"
		}
		subject += fmt.Sprintf(" request from %s", template.HTMLEscapeString(payee.FullName))

		msg := &mail.Message{
			Sender:   "noreply@tadue.com",
			To:       []string{req.PayerEmail},
			Subject:  subject,
			HTMLBody: string(body),
		}
		CheckError(mail.Send(c.Aec(), msg))

		req.ReminderSentDate = time.Now()
		return true
	}

	// Process each reqCode separately to avoid sending extra emails on failure.
	for _, reqCode := range reqCodes {
		_, err := updatePayRequests([]string{reqCode}, updateFn, false, c)
		CheckError(err)
	}
}

func handleEnqueueReminderEmails(w http.ResponseWriter, r *http.Request, c *Context) {
	q := makePayRequestQuery(nil, false).
		Filter("ReminderSentDate <", time.Now().AddDate(0, 0, -AUTO_PAY_REQUEST_EMAIL_RATE_LIMIT)).
		KeysOnly()
	count := 0
	for it := q.Run(c.Aec()); ; {
		reqKey, err := it.Next(nil)
		if err == datastore.Done {
			break
		}
		CheckError(err)
		CheckError(doEnqueuePayRequestEmails([]string{reqKey.Encode()}, c))
		count++
	}
	c.Aec().Infof("Enqueued %d reminder emails", count)
}

func handleSendGotPaidEmail(w http.ResponseWriter, r *http.Request, c *Context) {
	reqCode := r.FormValue("reqCode")
	Assert(reqCode != "", "No reqCode")
	reqKey, err := datastore.DecodeKey(reqCode)
	CheckError(err)
	payeeUserKey := reqKey.Parent()

	// TODO(sadovsky): Parallelize lookups using goroutines.
	req := &PayRequest{}
	CheckError(datastore.Get(c.Aec(), reqKey, req))
	payee := &User{}
	CheckError(datastore.Get(c.Aec(), payeeUserKey, payee))

	data := map[string]interface{}{
		"payeeFullName": payee.FullName,
		"payerEmail":    req.PayerEmail,
		"amount":        renderAmount(req.Amount),
		"description":   req.Description,
	}
	body, err := ExecuteTemplate("email-got-paid.html", data)
	CheckError(err)

	msg := &mail.Message{
		Sender:   "noreply@tadue.com",
		To:       []string{req.PayeeEmail},
		Subject:  "You've been paid!",
		HTMLBody: string(body),
	}
	CheckError(mail.Send(c.Aec(), msg))
}

func handleLogo(w http.ResponseWriter, r *http.Request, c *Context) {
	RenderTemplateOrDie(w, "logo.html", nil)
}

func handleDump(w http.ResponseWriter, r *http.Request, c *Context) {
	typeName := r.FormValue("t")
	var res interface{}
	if typeName == "PayRequest" {
		res = &PayRequest{}
	} else if typeName == "User" {
		res = &User{}
	} else if typeName == "VerifyEmail" {
		res = &VerifyEmail{}
	} else {
		Assert(false, "Cannot handle typeName: %q", typeName)
	}
	q := datastore.NewQuery(typeName)
	results := []template.HTML{}
	for it := q.Run(c.Aec()); ; {
		key, err := it.Next(res)
		if err == datastore.Done {
			break
		}
		CheckError(err)
		// Reference: http://golang.org/doc/articles/laws_of_reflection.html
		s := reflect.ValueOf(res).Elem()
		t := s.Type()
		// TODO(sadovsky): Use templates for everything.
		out := "<table>"
		out += fmt.Sprintf("<tr><td>Key<td>%v</tr>", key)
		out += fmt.Sprintf("<tr><td>EncodedKey<td>%v</tr>", key.Encode())
		for i := 0; i < s.NumField(); i++ {
			out += fmt.Sprintf("<tr><td>%s<td>%v</tr>", t.Field(i).Name, s.Field(i).Interface())
		}
		out += "</table>"
		results = append(results, template.HTML(out))
	}
	RenderTemplateOrDie(w, "dump.html", results)
}

func handleWipe(w http.ResponseWriter, r *http.Request, c *Context) {
	typeNames := [...]string{"PayRequest", "Session", "User", "VerifyEmail"}
	for _, typeName := range typeNames {
		q := datastore.NewQuery(typeName).KeysOnly()
		keys, err := q.GetAll(c.Aec(), nil)
		CheckError(err)
		CheckError(datastore.DeleteMulti(c.Aec(), keys))
	}
	c.DeleteSession()
	RenderPageOrDie(w, c, "text", "Datastore has been wiped.")
}

func init() {
	http.HandleFunc("/", WrapHandler(handleHome))
	http.HandleFunc("/ipn", WrapHandler(handleIpn))
	// Account stuff.
	http.HandleFunc("/settings", WrapHandler(handleSettings))
	http.HandleFunc("/account/change-password", WrapHandler(handleChangePassword))
	http.HandleFunc("/account/reset-password", WrapHandler(handleResetPassword))
	http.HandleFunc("/account/sendverif", WrapHandler(handleSendVerif))
	http.HandleFunc("/account/verif", WrapHandler(handleVerif))
	// Payments page and its ajax handlers.
	http.HandleFunc("/payments", WrapHandler(handleAccount))
	http.HandleFunc("/payments/mark-as-paid", WrapHandler(handleMarkAsPaid))
	http.HandleFunc("/payments/send-reminder", WrapHandler(handleSendReminder))
	http.HandleFunc("/payments/delete", WrapHandler(handleDelete))
	// Request and pay.
	http.HandleFunc("/pay", WrapHandler(handlePay))
	http.HandleFunc("/pay/done", WrapHandler(handlePayDone))
	http.HandleFunc("/request-payment", WrapHandler(handleRequestPayment))
	// Signup, login, logout.
	http.HandleFunc("/login", WrapHandler(handleLogin))
	http.HandleFunc("/logout", WrapHandler(handleLogout))
	http.HandleFunc("/signup", WrapHandler(handleSignup))
	// Tasks.
	http.HandleFunc("/tasks/send-pay-request-emails", WrapHandler(handleSendPayRequestEmails))
	http.HandleFunc("/tasks/enqueue-reminder-emails", WrapHandler(handleEnqueueReminderEmails))
	http.HandleFunc("/tasks/send-got-paid-email", WrapHandler(handleSendGotPaidEmail))
	// Bottom links.
	http.HandleFunc("/about", WrapHandler(handleAbout))
	http.HandleFunc("/privacy", WrapHandler(handlePrivacy))
	http.HandleFunc("/terms", WrapHandler(handleTerms))
	http.HandleFunc("/help", WrapHandler(handleHelp))
	// Admin links.
	http.HandleFunc("/admin/dump", WrapHandler(handleDump))
	// Development links.
	// TODO(sadovsky): Disable in prod.
	http.HandleFunc("/dev/dv", WrapHandler(handleDebugVerif))
	http.HandleFunc("/dev/logo", WrapHandler(handleLogo))
	http.HandleFunc("/dev/wipe", WrapHandler(handleWipe))
}
