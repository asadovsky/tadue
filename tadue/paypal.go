// Copyright 2012 Adam Sadovsky. All rights reserved.

// Style note: We always write Paypal with a lowercase second 'p'.
// TODO(sadovsky): This should probably be a separate package.

package tadue

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"appengine"
	"appengine/urlfetch"
)

// Sandbox values.
const (
	kSandboxHost     = "http://98.248.42.191"
	kUserId          = "adam_1335650859_biz_api1.gmail.com"
	kPassword        = "1335650898"
	kSignature       = "A9gqVQB8-mmb1uodjDQ9XZGG8wdlAohgwF511kB0uyIlOlahuHK9YiQb"
	kAppId           = "APP-80W284485P519543T"
	kPayEndpoint     = "https://svcs.sandbox.paypal.com/AdaptivePayments/Pay"
	kDetailsEndpoint = "https://svcs.sandbox.paypal.com/AdaptivePayments/PaymentDetails"
	kPayBaseUrl      = "https://www.sandbox.paypal.com/cgi-bin/webscr?cmd=_ap-payment"
	kValidateIpnUrl  = "https://www.sandbox.paypal.com/cgi-bin/webscr"
)

// TODO(sadovsky): Add prod values.

var headers = map[string]string{
	"X-PAYPAL-SECURITY-USERID":      kUserId,
	"X-PAYPAL-SECURITY-PASSWORD":    kPassword,
	"X-PAYPAL-SECURITY-SIGNATURE":   kSignature,
	"X-PAYPAL-REQUEST-DATA-FORMAT":  "NV",
	"X-PAYPAL-RESPONSE-DATA-FORMAT": "NV",
	"X-PAYPAL-APPLICATION-ID":       kAppId,
}

func setHeaders(r *http.Request) {
	for k, v := range headers {
		r.Header.Set(k, v)
	}
}

// Wrapper around urlfetch.Client to extract response body string. Returns error
// if response status is not 200.
func getResponseBody(resp *http.Response, err error) (string, error) {
	if resp != nil {
		// TODO(sadovsky): Is this right? See http://goo.gl/2zs4n for discussion.
		defer resp.Body.Close()
	}
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func SendPaypalPayRequest(reqId, payeePaypalEmail, description string, amount float32,
	c appengine.Context) (*PaypalPayResponse, error) {
	c.Debugf("SendPayRequest, payee=%q", payeePaypalEmail)

	baseUrl := fmt.Sprintf("http://%s", appengine.DefaultVersionHostname(c))
	if kSandboxHost != "" {
		baseUrl = kSandboxHost
	}

	// NOTE(sadovsky): We could add a trackingId here, but reqId in url seems good
	// enough.
	v := url.Values{}
	v.Set("requestEnvelope.errorLanguage", "en_US")
	v.Set("actionType", "PAY")
	v.Set("receiverList.receiver(0).email", payeePaypalEmail)
	amountStr := strconv.FormatFloat(float64(amount), 'f', 2, 32)
	v.Set("receiverList.receiver(0).amount", amountStr)
	v.Set("receiverList.receiver(0).paymentType", "PERSONAL")
	v.Set("currencyCode", "USD")
	v.Set("feesPayer", "EACHRECEIVER")
	v.Set("memo", description)
	v.Set("cancelUrl", fmt.Sprintf("%s/pay/cancel?reqId=%s", baseUrl, reqId))
	v.Set("returnUrl", fmt.Sprintf("%s/pay/done?reqId=%s", baseUrl, reqId))
	// Note: IPN requires port 80, at least in the sandbox. This constraint is not
	// documented.
	v.Set("ipnNotificationUrl", fmt.Sprintf("%s/ipn?reqId=%s", baseUrl, reqId))

	// Last param (body) inferred from PostForm() implementation in
	// http://golang.org/src/pkg/net/http/client.go.
	request, err := http.NewRequest("POST", kPayEndpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, err
	}
	setHeaders(request)

	c.Debugf("Pay request: %v", request)
	respStr, err := getResponseBody(urlfetch.Client(c).Do(request))
	if err != nil {
		return nil, err
	}
	values, err := url.ParseQuery(respStr)
	if err != nil {
		return nil, err
	}
	c.Infof("Pay response: %v", values)

	res := &PaypalPayResponse{
		Ack:           values.Get("responseEnvelope.ack"),
		Build:         values.Get("responseEnvelope.build"),
		CorrelationId: values.Get("responseEnvelope.correlationId"),
		Timestamp:     values.Get("responseEnvelope.timestamp"),
		PayKey:        values.Get("payKey"),
	}
	return res, nil
}

// IPN handler references: http://goo.gl/bIX2Q and http://goo.gl/F1uej
func ValidateIpn(requestBody string, c appengine.Context) (*PaypalIpnMessage, error) {
	c.Debugf("ValidateIpn")

	// Post back to IPN server to get verification.
	postBody := fmt.Sprintf("cmd=_notify-validate&%s", requestBody)
	c.Debugf("IPN post body: %v", postBody)

	respStr, err := getResponseBody(urlfetch.Client(c).Post(
		kValidateIpnUrl, "application/x-www-form-urlencoded", strings.NewReader(postBody)))
	if err != nil {
		return nil, err
	}
	if respStr != "VERIFIED" {
		return nil, errors.New(respStr) // INVALID
	}

	// Parse original IPN request and extract the useful values.
	values, err := url.ParseQuery(requestBody)
	if err != nil {
		return nil, err
	}
	c.Infof("IPN message: %v", values)

	res := &PaypalIpnMessage{
		Status:     values.Get("status"),
		PayerEmail: values.Get("sender_email"),
		PayeeEmail: values.Get("transaction[0].receiver"),
		Amount:     values.Get("transaction[0].amount"),
		PayKey:     values.Get("pay_key"),
	}
	return res, nil
}

func MakePaypalPayUrl(payKey string) string {
	// TODO(sadovsky): Make this more robust.
	return fmt.Sprintf("%s&paykey=%s", kPayBaseUrl, payKey)
}
