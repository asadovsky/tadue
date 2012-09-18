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

func makePayUrl(reqId string, c *Context) string {
	// TODO(sadovsky): Use https?
	return fmt.Sprintf("http://%s/pay?reqId=%s",
		appengine.DefaultVersionHostname(c.Aec()), reqId)
}

func makePayRequestQuery(userKey *datastore.Key, isPaid bool) *datastore.Query {
	q := datastore.NewQuery("PayRequest")
	if userKey != nil {
		q = q.Ancestor(userKey)
	}
	return q.Filter("DeletionDate =", time.Unix(0, 0)).Filter("IsPaid =", isPaid)
}

// Applies updateFn to each PayRequest specified in reqIds.
// If checkUser is true, aborts the transaction if any PayRequest does not
// belong to the current user.
func updatePayRequests(
	reqIds []string, updateFn func(reqId string, req *PayRequest) bool, checkUser bool,
	c *Context) ([]string, error) {
	Assert(len(reqIds) > 0, "No reqIds")
	if checkUser {
		AssertLoggedIn(c)
	}

	// NOTE(sadovsky): Multi-row, single entity group transaction.
	updatedReqIds := []string{}
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		for _, reqId := range reqIds {
			reqKey, err := datastore.DecodeKey(reqId)
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
			if updateFn(reqId, req) {
				updatedReqIds = append(updatedReqIds, reqId)
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
	return updatedReqIds, nil
}

func doLogin(w http.ResponseWriter, r *http.Request, c *Context) (*User, error) {
	// TODO(sadovsky): Form validation.
	email := r.FormValue("login-email")
	password := r.FormValue("login-password")

	var err error
	user := &User{}
	if email != "" {
		key := ToUserKey(c.Aec(), email)
		err = datastore.Get(c.Aec(), key, user)
	}
	if email == "" || err == datastore.ErrNoSuchEntity {
		// TODO(sadovsky): Handle "user does not exist" case better.
		return nil, errors.New(fmt.Sprintf("No such user: %q", email))
	}
	CheckError(err)

	if SaltAndHash(user.Salt, password) == user.PassHash {
		c.Aec().Infof("Logged in user: %q", user.Email)
	} else {
		// TODO(sadovsky): Handle "wrong password" case better.
		return nil, errors.New(fmt.Sprintf("Wrong password for user: %q", email))
	}

	CheckError(MakeSession(user.Email, user.FullName, w, c))
	return user, nil
}

func doSignup(w http.ResponseWriter, r *http.Request, c *Context) (*User, error) {
	// TODO(sadovsky): Form validation.
	salt := NewSalt()
	newUser := &User{
		Email:       r.FormValue("signup-email"),
		Salt:        salt,
		PassHash:    SaltAndHash(salt, r.FormValue("signup-password")),
		FullName:    r.FormValue("signup-name"),
		PaypalEmail: r.FormValue("signup-paypal"),
	}
	// TODO(sadovsky): Check that this Paypal account is valid and confirmed.
	if r.FormValue("signup-copy-email") == "on" {
		newUser.PaypalEmail = r.FormValue("signup-email")
	}

	// Check whether user already exists. If so, report error; if not, create new
	// account.
	key := ToUserKey(c.Aec(), newUser.Email)
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
		user := &User{}
		err := datastore.Get(aec, key, user)
		if err != nil && err != datastore.ErrNoSuchEntity {
			return err
		}
		if err == nil { // entity already exists
			// TODO(sadovsky): Handle "user already exists" case better.
			return errors.New(fmt.Sprintf("User already exists: %q", user.Email))
		}
		if _, err := datastore.Put(aec, key, newUser); err != nil {
			return err
		}
		aec.Infof("New user: %q", newUser.Email)
		return nil
	}, nil)

	if err != nil {
		return nil, err
	}

	if err = MakeSession(newUser.Email, newUser.FullName, w, c); err != nil {
		return nil, err
	}
	if err = doSendVerif(c); err != nil {
		return nil, err
	}
	return newUser, nil
}

func doSendVerif(c *Context) error {
	v := &VerifyEmail{
		Email:     c.Session().Email,
		Timestamp: time.Now(),
	}
	key := NewEphemeralKey(c.Aec(), "VerifyEmail")
	key, err := datastore.Put(c.Aec(), key, v)
	if err != nil {
		return err
	}

	verifUrl := fmt.Sprintf("http://%s/account/verif?key=%s",
		appengine.DefaultVersionHostname(c.Aec()), key.Encode())
	data := map[string]interface{}{
		"session":  c.Session(),
		"verifUrl": verifUrl,
	}
	body, err := ExecuteTemplate("email-verif.html", data)
	if err != nil {
		return err
	}

	msg := &mail.Message{
		Sender:   "noreply@tadue.com",
		To:       []string{v.Email},
		Subject:  "Welcome to Tadue",
		HTMLBody: string(body),
	}
	return mail.Send(c.Aec(), msg)
}

func doEnqueuePayRequestEmails(reqIds []string, c *Context) error {
	c.Aec().Infof("Enqueuing pay request emails for reqIds: %v", reqIds)
	if len(reqIds) == 0 {
		return nil
	}
	v := url.Values{}
	v.Set("reqIds", strings.Join(reqIds, ","))
	t := taskqueue.NewPOSTTask("/tasks/send-pay-request-emails", v)
	_, err := taskqueue.Add(c.Aec(), t, "")
	return err
}

func doEnqueueGotPaidEmail(reqId string, c *Context) error {
	c.Aec().Infof("Enqueuing got paid email for reqId: %v", reqId)
	v := url.Values{}
	v.Set("reqId", reqId)
	t := taskqueue.NewPOSTTask("/tasks/send-got-paid-email", v)
	_, err := taskqueue.Add(c.Aec(), t, "")
	return err
}

func doSetEmailOk(email string, c *Context) (bool, error) {
	userKey := ToUserKey(c.Aec(), email)
	user := &User{}
	alreadyVerified := false

	// Update user.EmailOk.
	err := datastore.RunInTransaction(c.Aec(), func(aec appengine.Context) error {
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
		return false, err
	} else if alreadyVerified {
		return false, nil
	}
	c.Aec().Infof("Verified email: %q", email)

	// Enqueue pay request emails.
	q := makePayRequestQuery(userKey, false).KeysOnly()
	reqKeys, err := q.GetAll(c.Aec(), nil)
	CheckError(err)
	reqIds := make([]string, len(reqKeys))
	for i, reqKey := range reqKeys {
		reqIds[i] = reqKey.Encode()
	}
	CheckError(doEnqueuePayRequestEmails(reqIds, c))
	return len(reqIds) > 0, nil
}

func getUserOrDie(email string, c *Context) *User {
	key := ToUserKey(c.Aec(), email)
	user := &User{}
	CheckError(datastore.Get(c.Aec(), key, user))
	return user
}

func getUserFromSessionOrDie(c *Context) *User {
	AssertLoggedIn(c)
	return getUserOrDie(c.Session().Email, c)
}

func getPayRequestOrDie(c *Context, reqKey *datastore.Key) *PayRequest {
	req := &PayRequest{}
	CheckError(datastore.Get(c.Aec(), reqKey, req))
	return req
}

////////////////////////////////////////////////////////////////////////////////
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
	reqId := r.FormValue("reqId")
	Assert(reqId != "", "No reqId")
	reqKey, err := datastore.DecodeKey(reqId)
	CheckError(err)

	msg, err := ValidateIpn(string(requestBytes), c.Aec())
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

	CheckError(doEnqueueGotPaidEmail(reqId, c))
	ServeEmpty200(w)
}

func handlePay(w http.ResponseWriter, r *http.Request, c *Context) {
	reqId := r.FormValue("reqId")
	Assert(reqId != "", "No reqId")
	reqKey, err := datastore.DecodeKey(reqId)
	CheckError(err)

	method := r.FormValue("method")
	Assert(method == "" || method == "paypal", "Invalid method: %q", method)

	// TODO(sadovsky): Cache PayRequest and User lookups so that multiple loads of
	// this page (e.g. first with method="", then with method="paypal") don't all
	// hit the datastore.
	req := &PayRequest{}
	CheckError(datastore.Get(c.Aec(), reqKey, req))

	// If request has already been paid, show an error.
	// TODO(sadovsky): Make this more elegant.
	if req.PaymentDate != time.Unix(0, 0) {
		RenderPageOrDie(w, c, "text", fmt.Sprintf("Already paid."))
		return
	}

	// Get payee's User object so we can get their name and paypal email address.
	payee := getUserOrDie(req.PayeeEmail, c)

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
		response, err := SendPaypalPayRequest(
			reqId, payee.PaypalEmail, req.Description, req.Amount, c.Aec())
		CheckError(err)
		// TODO(sadovsky): Store the response inside the PayRequest via transaction.
		if response.Ack != "Success" {
			ServeError(w, response)
			return
		}
		http.Redirect(w, r, MakePaypalPayUrl(response.PayKey), http.StatusFound)
	}
}

func handleRequestPayment(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "request-payment", nil)
	} else if r.Method == "POST" {
		CheckError(r.ParseForm())

		var user *User
		var err error

		// Note: The following logic elegantly handles the case where user signs up
		// during payment request, then re-posts the form.
		if c.Session() == nil {
			doSignupValue := r.FormValue("do-signup")
			if doSignupValue == "true" {
				user, err = doSignup(w, r, c)
				CheckError(err)
			} else {
				Assert(doSignupValue == "false", "Invalid doSignupValue: %q", doSignupValue)
				user, err = doLogin(w, r, c)
				CheckError(err)
			}
		} else {
			user = getUserFromSessionOrDie(c)
		}

		// At this point the user must be logged in, and we must have their User
		// struct.
		AssertLoggedIn(c)
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

		reqKey := datastore.NewIncompleteKey(
			c.Aec(), "PayRequest", ToUserKey(c.Aec(), c.Session().Email))
		reqKey, err = datastore.Put(c.Aec(), reqKey, req) // overwrite incomplete key
		CheckError(err)
		reqId := reqKey.Encode()

		// If payee's email is already verified, enqueue the payment request email.
		if user.EmailOk {
			CheckError(doEnqueuePayRequestEmails([]string{reqId}, c))
		}

		payUrl := makePayUrl(reqId, c)
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
		Assert(c.Session() == nil, "Already logged in")
		CheckError(r.ParseForm())
		_, err := doLogin(w, r, c)
		CheckError(err)
		http.Redirect(w, r, "/account", http.StatusFound)
	} else {
		Serve404(w)
	}
}

func handleLogout(w http.ResponseWriter, r *http.Request, c *Context) {
	AssertLoggedIn(c)
	CheckError(DeleteSession(w, c))
	RenderPageOrDie(w, c, "home", nil)
}

func handleSignup(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method == "GET" {
		RenderPageOrDie(w, c, "signup", nil)
	} else if r.Method == "POST" {
		Assert(c.Session() == nil, "Already logged in")
		CheckError(r.ParseForm())
		_, err := doSignup(w, r, c)
		CheckError(err)
		http.Redirect(w, r, "/account?new", http.StatusFound)
	} else {
		Serve404(w)
	}
}

type RenderablePayRequest struct {
	ReqId        string
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

func getRecentPayRequestsOrDie(email string, c *Context) []RenderablePayRequest {
	userKey := ToUserKey(c.Aec(), email)
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
		rpr.ReqId = reqKeys[i].Encode()
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
	AssertLoggedIn(c)
	CheckError(r.ParseForm())
	isNew := r.Form["new"] != nil

	user := getUserFromSessionOrDie(c)
	rendReqs := getRecentPayRequestsOrDie(c.Session().Email, c)
	data := map[string]interface{}{
		"user":           user,
		"isNew":          isNew,
		"rendReqs":       rendReqs,
		"undoableReqIds": "",
	}
	RenderPageOrDie(w, c, "payments", data)
}

func renderRecentRequests(w http.ResponseWriter, undoableReqIds []string, c *Context) {
	AssertLoggedIn(c)
	rendReqs := getRecentPayRequestsOrDie(c.Session().Email, c)
	data := map[string]interface{}{
		"rendReqs":       rendReqs,
		"undoableReqIds": strings.Join(undoableReqIds, ","),
	}
	RenderTemplateOrDie(w, "payments-data", data)
}

func handleMarkAsPaid(w http.ResponseWriter, r *http.Request, c *Context) {
	// Note similarity to handleDelete().
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	AssertLoggedIn(c)
	CheckError(r.ParseForm())
	reqIds := strings.Split(r.FormValue("reqIds"), ",")
	undo := r.Form["undo"] != nil
	updateFn := func(reqId string, req *PayRequest) bool {
		if undo {
			Assert(req.IsPaid, reqId)
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
	undoableReqIds, err := updatePayRequests(reqIds, updateFn, true, c)
	CheckError(err)
	renderRecentRequests(w, undoableReqIds, c)
}

func handleSendReminder(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	AssertLoggedIn(c)
	CheckError(r.ParseForm())
	reqIds := strings.Split(r.FormValue("reqIds"), ",")
	// TODO(sadovsky): Show error if user is not verified.
	// TODO(sadovsky): Show error if user exceeds email rate limit.
	// TODO(sadovsky): Optimistically show requests as sent.
	CheckError(doEnqueuePayRequestEmails(reqIds, c))
	renderRecentRequests(w, []string{}, c)
}

func handleDelete(w http.ResponseWriter, r *http.Request, c *Context) {
	// Note similarity to handleMarkAsPaid().
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	AssertLoggedIn(c)
	CheckError(r.ParseForm())
	reqIds := strings.Split(r.FormValue("reqIds"), ",")
	undo := r.Form["undo"] != nil
	updateFn := func(reqId string, req *PayRequest) bool {
		if undo {
			req.DeletionDate = time.Unix(0, 0)
		} else {
			req.DeletionDate = time.Now()
		}
		return true
	}
	undoableReqIds, err := updatePayRequests(reqIds, updateFn, true, c)
	CheckError(err)
	renderRecentRequests(w, undoableReqIds, c)
}

func handleSettings(w http.ResponseWriter, r *http.Request, c *Context) {
	AssertLoggedIn(c)
	RenderPageOrDie(w, c, "settings", nil)
}

func handleSendVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	AssertLoggedIn(c)
	CheckError(doSendVerif(c))
	RenderPageOrDie(w, c, "text",
		fmt.Sprintf("A new verification link has been sent to %s.", c.Session().Email))
}

func doRenderVerifMsg(sentEmail bool, w http.ResponseWriter, c *Context) {
	msg := fmt.Sprintf("Email address %s has been verified.", c.Session().Email)
	if sentEmail {
		msg += " All pending payment requests have been sent."
	}
	RenderPageOrDie(w, c, "text", msg)
}

func handleDebugVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	AssertLoggedIn(c)
	sentEmail, err := doSetEmailOk(c.Session().Email, c)
	CheckError(err)
	doRenderVerifMsg(sentEmail, w, c)
}

func handleVerif(w http.ResponseWriter, r *http.Request, c *Context) {
	CheckError(r.ParseForm())
	encodedKey := r.FormValue("key")
	Assert(encodedKey != "", "No key")
	key, err := datastore.DecodeKey(encodedKey)
	CheckError(err)

	v := &VerifyEmail{}
	CheckError(datastore.Get(c.Aec(), key, v))

	// Check whether verification request has expired.
	// TODO(sadovsky): Make error message more friendly.
	if time.Now().After(v.Timestamp.AddDate(0, 0, VERIFICATION_LIFESPAN)) {
		RenderPageOrDie(w, c, "text",
			fmt.Sprintf("Verification link has expired. Please request another."))
		return
	}

	// TODO(sadovsky): Check that email matches?
	sentEmail, err := doSetEmailOk(v.Email, c)
	CheckError(err)

	// Note: We do not delete the VerifyEmail record; it will eventually expire
	// and be deleted by a batch deletion process.
	doRenderVerifMsg(sentEmail, w, c)
}

func handleSendPayRequestEmails(w http.ResponseWriter, r *http.Request, c *Context) {
	if r.Method != "POST" {
		Serve404(w)
		return
	}
	CheckError(r.ParseForm())
	reqIds := strings.Split(r.FormValue("reqIds"), ",")
	Assert(len(reqIds) > 0, "No reqIds")

	getPayeeUserKey := func(reqId string) *datastore.Key {
		reqKey, err := datastore.DecodeKey(reqIds[0])
		CheckError(err)
		return reqKey.Parent()
	}
	payeeUserKey := getPayeeUserKey(reqIds[0])
	// Verify that all reqIds have the same parent.
	for _, reqId := range reqIds {
		Assert(getPayeeUserKey(reqId).Equal(payeeUserKey), "")
	}

	payee := &User{}
	// TODO(sadovsky): Add write-through memcached layer, at least for User.
	CheckError(datastore.Get(c.Aec(), payeeUserKey, payee))
	if !payee.EmailOk {
		// Payee's email has not been verified, so do not send any emails.
		return
	}

	// Sends payment request email and updates ReminderSentDate in PayRequest.
	updateFn := func(reqId string, req *PayRequest) bool {
		if req.IsPaid {
			return false
		} else if req.ReminderSentDate.After(time.Now().AddDate(0, 0, -PAY_REQUEST_EMAIL_RATE_LIMIT)) {
			return false
		}

		isReminder := req.ReminderSentDate != time.Unix(0, 0)
		data := map[string]interface{}{
			"payerEmail":    req.PayerEmail,
			"payeeEmail":    payee.PaypalEmail,
			"payeeFullName": payee.FullName,
			"amount":        renderAmount(req.Amount),
			"description":   req.Description,
			"payUrl":        makePayUrl(reqId, c),
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

	// Process each reqId separately to avoid sending extra emails on failure.
	for _, reqId := range reqIds {
		_, err := updatePayRequests([]string{reqId}, updateFn, false, c)
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
	reqId := r.FormValue("reqId")
	Assert(reqId != "", "No reqId")
	reqKey, err := datastore.DecodeKey(reqId)
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
	// Account.
	http.HandleFunc("/account", WrapHandler(handleAccount))
	http.HandleFunc("/account/settings", WrapHandler(handleSettings))
	http.HandleFunc("/account/sendverif", WrapHandler(handleSendVerif))
	http.HandleFunc("/account/verif", WrapHandler(handleVerif))
	// Account ajax handlers.
	http.HandleFunc("/account/mark-as-paid", WrapHandler(handleMarkAsPaid))
	http.HandleFunc("/account/send-reminder", WrapHandler(handleSendReminder))
	http.HandleFunc("/account/delete", WrapHandler(handleDelete))
	// Request and pay.
	http.HandleFunc("/pay", WrapHandler(handlePay))
	http.HandleFunc("/pay/cancel", PlaceholderHandler("pay/cancel"))
	http.HandleFunc("/pay/done", PlaceholderHandler("pay/done"))
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
