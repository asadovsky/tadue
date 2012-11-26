// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

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

func prependHost(url string, c *Context) string {
	Assert(strings.Index(url, "/") == 0, url)
	return fmt.Sprintf("http://%s%s", AppHostname(c), url)
}

func makePayUrl(reqCode string) string {
	return fmt.Sprintf("/pay?reqCode=%s", reqCode)
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

// Steers user through login page if they aren't already logged in.
// Returns true if request has been handled, false otherwise.
// If user is not logged in, request must be a GET request.
func steerThroughLogin(w http.ResponseWriter, r *http.Request, c *Context) bool {
	if c.LoggedIn() {
		return false
	}
	if r.Method != "GET" {
		Serve404(w)
		return true
	}
	escapedTarget := url.QueryEscape(r.URL.String())
	http.Redirect(
		w, r, fmt.Sprintf("/login?target=%s", escapedTarget), http.StatusSeeOther)
	return true
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

	var updatedReqCodes []string
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		updatedReqCodes = []string{} // ensure transaction is idempotent
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
	if time.Now().After(v.Timestamp.Add(time.Minute * RESET_PASSWORD_LIFESPAN_MINUTES)) {
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
	email := ParseEmail(r.FormValue("login-email"))
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
	salt := GenerateSecureRandomString()
	newUser := &User{
		Email:    ParseEmail(r.FormValue("signup-email")),
		Salt:     salt,
		PassHash: SaltAndHash(salt, r.FormValue("signup-password")),
		FullName: ParseFullName(r.FormValue("signup-name")),
	}
	if r.FormValue("signup-copy-email") == "on" {
		newUser.PayPalEmail = newUser.Email
	} else {
		newUser.PayPalEmail = ParseEmail(r.FormValue("signup-paypal-email"))
	}
	// TODO(sadovsky): Check that the PayPal account is valid and confirmed.

	// Check whether user already exists. If so, report error; if not, create new
	// account.
	var userId int64
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		userId = 0 // ensure transaction is idempotent
		userIdKey := ToUserIdKey(c.Aec(), newUser.Email)
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
	// Check that it's a known user email.
	userId, user := GetUserFromEmailOrDie(email, c)

	// Create the ResetPassword record.
	v := &ResetPassword{
		UserId:    userId,
		Timestamp: time.Now(),
	}
	key := NewEphemeralKey(c.Aec(), "ResetPassword")
	key, err := datastore.Put(c.Aec(), key, v)
	if err != nil {
		return err
	}

	// Send the email.
	resetUrl := prependHost(fmt.Sprintf("/account/change-password?key=%s", key.Encode()), c)
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
		Sender:   "Tadue <noreply@tadue.com>",
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
	verifUrl := prependHost(fmt.Sprintf("/account/verif?key=%s", key.Encode()), c)
	data := map[string]interface{}{
		"fullName": c.Session().FullName,
		"verifUrl": verifUrl,
	}
	body, err := ExecuteTemplate("email-verif.html", data)
	if err != nil {
		return err
	}

	msg := &mail.Message{
		Sender:   "Tadue <noreply@tadue.com>",
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

func doSetEmailOk(userId int64, c *Context) (email string, sentPayRequestEmails bool, err error) {
	var user *User
	alreadyVerified := false
	updateFn := func(userToUpdate *User) bool {
		user = userToUpdate
		if user.EmailOk {
			alreadyVerified = true
			return false
		}
		user.EmailOk = true
		return true
	}
	if err := updateUser(userId, nil, updateFn, c); err != nil {
		return "", false, err
	} else if alreadyVerified {
		return user.Email, false, nil
	}
	c.Aec().Infof("Verified email: %q", user.Email)

	// Enqueue pay request emails.
	userKey := ToUserKey(c.Aec(), userId)
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

	// Note: If we call ParseForm() before ReadAll(), the IPN dance fails because
	// ParseForm() mutates the r.Body (io.ReadCloser).
	CheckError(r.ParseForm())
	reqCode := r.FormValue("reqCode")
	Assert(reqCode != "", "No reqCode")
	reqKey, err := datastore.DecodeKey(reqCode)
	CheckError(err)

	msg, err := PayPalValidateIpn(string(requestBytes), c)
	CheckError(err)
	c.Aec().Infof("%+v", msg) // plus flag (%+v) adds field names

	// If the transaction is not completed, we don't care.
	// TODO(sadovsky): Should we care? Probably.
	if msg.Status != "COMPLETED" {
		return
	}

	var shouldSendEmail bool
	err = datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		shouldSendEmail = true // ensure transaction is idempotent
		req := &PayRequest{}
		err := datastore.Get(aec, reqKey, req)
		if err != nil {
			return err
		}

		// Check payee email and amount.
		// TODO(sadovsky): Maybe store payer's PayPal email, since we get it here.
		if msg.PayeeEmail != req.PayeeEmail {
			return errors.New(fmt.Sprintf("Wrong payee: %q != %q", msg.PayeeEmail, req.PayeeEmail))
		}
		if msg.Amount != req.Amount {
			return errors.New(fmt.Sprintf("Wrong amount: %v != %v", msg.Amount, req.Amount))
		}

		// If already marked as paid, return without sending an email.
		// It's important not to send an email here because PayPal sometimes sends
		// multiple IPNs for a successful payment. In at least one such case, the
		// only difference between the two IPNs was that the second included
		// "reason_code:CLEARED".
		if req.PaymentDate != time.Unix(0, 0) {
			shouldSendEmail = false
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

	if shouldSendEmail {
		CheckError(doEnqueueGotPaidEmail(reqCode, c))
	}
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
		RedirectWithMessage(w, r, "/", "Already paid.")
		return
	}

	// Get payee's User object so we can get their name and paypal email address.
	payee := GetUserOrDie(GetPayeeUserKey(reqCode), c)

	if method == "" {
		data := map[string]interface{}{
			"payerEmail":       req.PayerEmail,
			"payeeEmail":       payee.PayPalEmail,
			"payeeFullName":    payee.FullName,
			"amount":           renderAmount(req.Amount),
			"description":      req.Description,
			"payWithPayPalUrl": fmt.Sprintf("%s&method=paypal", r.URL.String()),
		}
		RenderPageOrDie(w, c, "pay", data)
	} else { // method == "paypal"
		// According to the PayPal documentation, the pay key is only valid for
		// three hours, so we must request it when the payer arrives.
		_, payUrl, err := PayPalSendPayRequest(
			reqCode, payee.PayPalEmail, req.Description, req.Amount, c)
		CheckError(err)
		// TODO(sadovsky): Maybe store the PayPalPayResponse inside the PayRequest.
		http.Redirect(w, r, payUrl, http.StatusSeeOther)
	}
}

func handlePayDone(w http.ResponseWriter, r *http.Request, c *Context) {
	RedirectWithMessage(w, r, "/", "Payment processed successfully. Thanks for using Tadue!")
}

func handleRequestPayment(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "request-payment", nil)
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}

	var user *User
	var err error
	isNewUser := false
	if c.LoggedIn() {
		user = GetUserFromSessionOrDie(c)
	} else {
		doSignupValue := r.FormValue("do-signup")
		isNewUser = doSignupValue == "true"
		if isNewUser {
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

	paymentType := ParsePaymentType(r.FormValue("payment-type"))
	// Make it so all requests have the same creation date.
	creationDate := time.Now()

	reqs := []*PayRequest{}
	for k, v := range r.Form {
		if strings.HasPrefix(k, "payer-email-") {
			id := k[len("payer-email-"):]
			req := &PayRequest{
				PayeeEmail:       c.Session().Email,
				PayerEmail:       ParseEmail(v[0]),
				Amount:           ParseAmount(r.FormValue("amount-" + id)),
				PaymentType:      paymentType,
				Description:      r.FormValue("description"),
				CreationDate:     creationDate,
				PaymentDate:      time.Unix(0, 0),
				DeletionDate:     time.Unix(0, 0),
				ReminderSentDate: time.Unix(0, 0),
			}
			reqs = append(reqs, req)
		}
	}
	Assert(len(reqs) > 0, "No requests")
	Assert(len(reqs) < 50, "Too many requests")

	var reqCodes []string
	err = datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		reqCodes = []string{} // ensure transaction is idempotent
		for _, req := range reqs {
			incompleteReqKey := datastore.NewIncompleteKey(
				c.Aec(), "PayRequest", ToUserKey(c.Aec(), c.Session().UserId))
			reqKey, err := datastore.Put(c.Aec(), incompleteReqKey, req)
			if err != nil {
				return err
			}
			reqCodes = append(reqCodes, reqKey.Encode())
		}
		return nil
	}, nil)
	CheckError(err)

	// If payee's email is already verified, enqueue the pay request emails.
	if user.EmailOk {
		CheckError(doEnqueuePayRequestEmails(reqCodes, c))
	}

	target := "/payments"
	if isNewUser {
		target = "/payments?new"
	}
	RedirectWithMessage(w, r, target, "Payment request made.")
}

func handleLogin(w http.ResponseWriter, r *http.Request, c *Context) {
	escapedTarget := r.FormValue("target") // may be empty
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "login", map[string]interface{}{"target": escapedTarget})
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertNotLoggedIn()
	// TODO(sadovsky): Show nice error page on failed login.
	_, err := doLogin(w, r, c)
	CheckError(err)
	target := "/payments"
	if escapedTarget != "" {
		target, err = url.QueryUnescape(escapedTarget)
		CheckError(err)
	}
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func handleLogout(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	CheckError(DeleteSession(w, c))
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func handleSignup(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "signup", nil)
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertNotLoggedIn()
	_, err := doSignup(w, r, c)
	CheckError(err)
	http.Redirect(w, r, "/payments?new", http.StatusSeeOther)
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
	// return date.Format("Jan 2 15:04:05")
	return date.Format("Jan 2")
}

func renderAmount(amount float32) string {
	return fmt.Sprintf("$%.2f", amount)
}

func getRecentPayRequestsOrDie(
	userId int64, emailOk bool, sentReminderReqCodes []string, c *Context) []RenderablePayRequest {
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
		rpr.PayUrl = makePayUrl(rpr.ReqCode)
		rpr.PayerEmail = pr.PayerEmail
		rpr.Amount = renderAmount(pr.Amount)
		rpr.Description = pr.Description
		rpr.IsPaid = pr.IsPaid
		// TODO(sadovsky): Get user's time zone during signup.
		// https://bitbucket.org/pellepim/jstimezonedetect/wiki/Home
		// http://arshaw.com/xdate/
		if pr.PaymentDate != time.Unix(0, 0) {
			rpr.Status = "Paid on " + renderDate(pr.PaymentDate)
		} else if pr.ReminderSentDate != time.Unix(0, 0) {
			// If this function was called via handleSendReminder, the reminder emails
			// have been enqueued, but may not have been sent yet. Optimistically show
			// them as sent.
			// TODO(sadovsky): Handle the case where reminder email was blocked by
			// cooldown.
			reminderSentDate := pr.ReminderSentDate
			if ContainsString(sentReminderReqCodes, rpr.ReqCode) {
				reminderSentDate = time.Now()
			}
			rpr.Status = "Emailed on " + renderDate(reminderSentDate)
		} else if emailOk {
			// This function was called by handlePayments. User is verified, so emails
			// must have been enqueued, but apparently they have not been sent yet.
			// Optimistically show them as sent.
			rpr.Status = "Emailed on " + renderDate(time.Now())
		} else {
			rpr.Status = "Pending verification"
		}
		rpr.CreationDate = renderDate(pr.CreationDate)
	}
	return rendReqs
}

func handlePayments(w http.ResponseWriter, r *http.Request, c *Context) {
	if steerThroughLogin(w, r, c) {
		return
	}
	user := GetUserFromSessionOrDie(c)
	rendReqs := getRecentPayRequestsOrDie(c.Session().UserId, user.EmailOk, []string{}, c)
	data := map[string]interface{}{
		"user":              user,
		"isNew":             r.Form["new"] != nil,
		"rendReqs":          rendReqs,
		"undoableReqCodes":  "",
		"reminderFrequency": AUTO_PAY_REQUEST_EMAIL_FREQUENCY,
	}
	RenderPageOrDie(w, c, "payments", data)
}

func renderRecentRequests(
	w http.ResponseWriter, undoableReqCodes, sentReminderReqCodes []string, c *Context) {
	c.AssertLoggedIn()
	rendReqs := getRecentPayRequestsOrDie(c.Session().UserId, false, sentReminderReqCodes, c)
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
	renderRecentRequests(w, undoableReqCodes, []string{}, c)
}

func handleSendReminder(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertLoggedIn()
	reqCodes := strings.Split(r.FormValue("reqCodes"), ",")
	// TODO(sadovsky): Show error if user is not verified.
	// TODO(sadovsky): Show error if user exceeds email rate limit.
	CheckError(doEnqueuePayRequestEmails(reqCodes, c))
	renderRecentRequests(w, []string{}, reqCodes, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request, c *Context) {
	// Note similarity to handleMarkAsPaid().
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	c.AssertLoggedIn()
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
	renderRecentRequests(w, undoableReqCodes, []string{}, c)
}

func handleSettings(w http.ResponseWriter, r *http.Request, c *Context) {
	if steerThroughLogin(w, r, c) {
		return
	}
	if r.Method == "GET" {
		user := GetUserFromSessionOrDie(c)
		data := map[string]interface{}{
			"email":       user.Email,
			"fullName":    user.FullName,
			"payPalEmail": user.PayPalEmail,
		}
		RenderPageOrDie(w, c, "settings", data)
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}

	// For now, we don't allow a user to change his email, because then we'd need
	// to verify the new email before actually making the change.
	fullName := ParseFullName(r.FormValue("name"))
	payPalEmail := ParseEmail(r.FormValue("paypal-email"))

	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		userKey := ToUserKey(c.Aec(), c.Session().UserId)
		user := &User{}
		if err := datastore.Get(aec, userKey, user); err != nil {
			return err
		}
		if user.FullName == fullName && user.PayPalEmail == payPalEmail {
			// Nothing changed, so just return.
			return nil
		}
		if user.FullName != fullName {
			// Update Session record.
			session := c.Session()
			session.FullName = fullName // mutates c.Session()
			CheckError(UpdateSession(session, w, c))
		}
		// Update User record.
		user.FullName = fullName
		user.PayPalEmail = payPalEmail
		if _, err := datastore.Put(aec, userKey, user); err != nil {
			return err
		}
		return nil
	}, makeXG())

	CheckError(err)
	// http://en.wikipedia.org/wiki/Post/Redirect/Get
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

// Handles both changes and resets.
func handleChangePassword(w http.ResponseWriter, r *http.Request, c *Context) {
	encodedKey := r.FormValue("key")
	isPasswordResetRequest := encodedKey != ""
	if !isPasswordResetRequest {
		if steerThroughLogin(w, r, c) {
			return
		}
	}

	if r.Method == "GET" {
		if !isPasswordResetRequest {
			RenderPageOrDie(w, c, "change-password", map[string]interface{}{"key": nil})
		} else { // password reset request
			_, err := useResetPassword(encodedKey, c)
			if err != nil {
				RedirectWithMessage(w, r, "/", err.Error())
				return
			}
			RenderPageOrDie(w, c, "change-password", map[string]interface{}{"key": encodedKey})
		}
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}

	var err error = nil
	updateFn := func(user *User) bool {
		salt := GenerateSecureRandomString()
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
	RedirectWithMessage(w, r, "/", "Password changed successfully.")
}

func handleResetPassword(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "reset-password", nil)
		return
	} else if r.Method != "POST" {
		Serve404(w)
		return
	}
	email := ParseEmail(r.FormValue("email"))
	CheckError(doInitiateResetPassword(email, c))
	RedirectWithMessage(w, r, "/", makeSentLinkMessage("Password reset", email))
}

func handleSendVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	CheckError(doInitiateVerifyEmail(c))
	RedirectWithMessage(w, r, "/", makeSentLinkMessage("Email verification", c.Session().Email))
}

func doRenderVerifMsg(
	email string, sentPayRequestEmails bool, w http.ResponseWriter, r *http.Request, c *Context) {
	msg := fmt.Sprintf("Email address %s has been verified.", email)
	if sentPayRequestEmails {
		msg += " All pending payment requests have been sent."
	}
	RedirectWithMessage(w, r, "/", msg)
}

func handleDebugVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	c.AssertLoggedIn()
	email, sentPayRequestEmails, err := doSetEmailOk(c.Session().UserId, c)
	Assert(email == c.Session().Email, "")
	CheckError(err)
	doRenderVerifMsg(email, sentPayRequestEmails, w, r, c)
}

func handleVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	encodedKey := r.FormValue("key")
	Assert(encodedKey != "", "No key")
	userId, err := useVerifyEmail(encodedKey, c)
	if err != nil {
		RedirectWithMessage(w, r, "/", err.Error())
	}
	email, sentPayRequestEmails, err := doSetEmailOk(userId, c)
	CheckError(err)
	doRenderVerifMsg(email, sentPayRequestEmails, w, r, c)
}

func handleSendPayRequestEmails(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
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
		} else if req.ReminderSentDate.After(time.Now().AddDate(0, 0, -PAY_REQUEST_EMAIL_COOLDOWN)) {
			return false
		}

		isReminder := req.ReminderSentDate != time.Unix(0, 0)
		data := map[string]interface{}{
			"payerEmail":    req.PayerEmail,
			"payeeEmail":    payee.PayPalEmail,
			"payeeFullName": payee.FullName,
			"amount":        renderAmount(req.Amount),
			"description":   req.Description,
			"payUrl":        prependHost(makePayUrl(reqCode), c),
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
			Sender:   "Tadue <noreply@tadue.com>",
			To:       []string{req.PayerEmail},
			Cc:       []string{req.PayeeEmail},
			Subject:  subject,
			HTMLBody: string(body),
		}
		CheckError(mail.Send(c.Aec(), msg))
		c.Aec().Infof("Sent PayRequest email: payee=%q, payer=%q, amount=%q",
			req.PayeeEmail, req.PayerEmail, renderAmount(req.Amount))

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
		Filter("ReminderSentDate <", time.Now().AddDate(0, 0, -AUTO_PAY_REQUEST_EMAIL_FREQUENCY)).
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
		Sender:   "Tadue <noreply@tadue.com>",
		To:       []string{req.PayeeEmail},
		Subject:  "You've been paid!",
		HTMLBody: string(body),
	}
	CheckError(mail.Send(c.Aec(), msg))
	c.Aec().Infof("Sent GotPaid email: payee=%q, payer=%q, amount=%q",
		req.PayeeEmail, req.PayerEmail, renderAmount(req.Amount))
}

// Uses reflection to print records from datastore.
// Reference: http://golang.org/doc/articles/laws_of_reflection.html
func handleDump(w http.ResponseWriter, r *http.Request, c *Context) {
	typeName := r.FormValue("t")
	var res interface{}
	if typeName == "PayRequest" {
		res = &PayRequest{}
	} else if typeName == "Session" {
		res = &Session{}
	} else if typeName == "User" {
		res = &User{}
	} else if typeName == "UserId" {
		res = &UserId{}
	} else {
		Assert(false, "Cannot handle typeName: %q", typeName)
	}

	renderValue := func(value interface{}) string {
		res := strconv.QuoteToASCII(fmt.Sprintf("%v", value))
		return res[1 : len(res)-1] // strip quotes
	}

	// Make headers.
	headers := []string{"Key", "EncodedKey"}
	s := reflect.ValueOf(res).Elem()
	t := s.Type()
	for i := 0; i < s.NumField(); i++ {
		headers = append(headers, t.Field(i).Name)
	}

	// Make rows.
	rows := [][]string{}
	q := datastore.NewQuery(typeName)
	for it := q.Run(c.Aec()); ; {
		key, err := it.Next(res)
		if err == datastore.Done {
			break
		}
		CheckError(err)
		s := reflect.ValueOf(res).Elem()
		row := []string{renderValue(key.String()), renderValue(key.Encode())}
		for i := 0; i < s.NumField(); i++ {
			row = append(row, renderValue(s.Field(i).Interface()))
		}
		rows = append(rows, row)
	}
	data := map[string]interface{}{
		"headers": headers,
		"rows":    rows,
	}
	RenderTemplateOrDie(w, "dump.html", data)
}

func handleWipe(w http.ResponseWriter, r *http.Request, c *Context) {
	typeNames := [...]string{
		"PayRequest", "ResetPassword", "User", "UserId", "VerifyEmail"}
	for _, typeName := range typeNames {
		q := datastore.NewQuery(typeName).KeysOnly()
		keys, err := q.GetAll(c.Aec(), nil)
		CheckError(err)
		CheckError(datastore.DeleteMulti(c.Aec(), keys))
	}
	c.DeleteSession()
	RedirectWithMessage(w, r, "/", "Datastore has been wiped.")
}

func init() {
	http.HandleFunc("/", WrapHandler(handleHome))
	http.HandleFunc("/ipn", WrapHandlerNoParseForm(handleIpn))
	// Account-related handlers.
	http.HandleFunc("/settings", WrapHandler(handleSettings))
	http.HandleFunc("/account/change-password", WrapHandler(handleChangePassword))
	http.HandleFunc("/account/reset-password", WrapHandler(handleResetPassword))
	http.HandleFunc("/account/sendverif", WrapHandler(handleSendVerif))
	http.HandleFunc("/account/verif", WrapHandler(handleVerif))
	// Payments page and its ajax handlers.
	http.HandleFunc("/payments", WrapHandler(handlePayments))
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
	http.HandleFunc("/dev/dv", WrapHandler(handleDebugVerif))
	//http.HandleFunc("/dev/wipe", WrapHandler(handleWipe))
	//http.HandleFunc("/dev/fix", WrapHandler(handleFix))
}
