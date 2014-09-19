// TODO(sadovsky): This should probably be a separate package.

package app

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"appengine/urlfetch"
)

// Stores paypal response to one "Pay" request.
// Pay API reference: http://goo.gl/D6dUR
type PayPalPayResponse struct {
	Ack           string // responseEnvelope.ack
	Build         string // responseEnvelope.build
	CorrelationId string // responseEnvelope.correlationId
	Timestamp     string // responseEnvelope.timestamp
	PayKey        string // payKey
}

// Stores the useful fields from a single paypal IPN message.
// IPN reference: http://goo.gl/bIX2Q
type PayPalIpnMessage struct {
	Status     string  // status
	PayerEmail string  // sender_email
	PayeeEmail string  // transaction[0].receiver
	Amount     float32 // extracted from transaction[0].amount
	PayKey     string  // pay_key
}

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
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", errors.New(resp.Status)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func PayPalSendPayRequest(reqCode, payeePayPalEmail, description string, amount float32, c *Context) (*PayPalPayResponse, string, error) {
	c.Aec().Debugf("PayPalSendPayRequest, payee=%q", payeePayPalEmail)

	baseUrl := fmt.Sprintf("http://%s", AppHostnameForPayPal(c))

	// NOTE(sadovsky): We could add a trackingId here, but reqCode in url seems
	// good enough.
	v := url.Values{}
	v.Set("requestEnvelope.errorLanguage", "en_US")
	v.Set("actionType", "PAY")
	v.Set("receiverList.receiver(0).email", payeePayPalEmail)
	amountStr := strconv.FormatFloat(float64(amount), 'f', 2, 32)
	v.Set("receiverList.receiver(0).amount", amountStr)
	// TODO(sadovsky): Get payment type from the PayRequest.
	v.Set("receiverList.receiver(0).paymentType", "PERSONAL")
	v.Set("currencyCode", "USD")
	v.Set("feesPayer", "SENDER")
	v.Set("memo", description)
	v.Set("cancelUrl", fmt.Sprintf("%s/pay?reqCode=%s", baseUrl, reqCode))
	v.Set("returnUrl", fmt.Sprintf("%s/pay/done?reqCode=%s", baseUrl, reqCode))
	// Note: IPN requires port 80, at least in the sandbox. This constraint is not
	// documented.
	v.Set("ipnNotificationUrl", fmt.Sprintf("%s/ipn?reqCode=%s", baseUrl, reqCode))

	// Last param (body) inferred from PostForm() implementation in
	// http://golang.org/src/pkg/net/http/client.go.
	request, err := http.NewRequest("POST", kPayEndpoint, strings.NewReader(v.Encode()))
	if err != nil {
		return nil, "", err
	}
	setHeaders(request)

	c.Aec().Debugf("Pay request: %v", request)
	respStr, err := getResponseBody(urlfetch.Client(c.Aec()).Do(request))
	if err != nil {
		return nil, "", err
	}
	values, err := url.ParseQuery(respStr)
	if err != nil {
		return nil, "", err
	}
	c.Aec().Debugf("Pay response: %v", values)

	ack := values.Get("responseEnvelope.ack")
	if ack != "Success" {
		return nil, "", errors.New(ack)
	}

	res := &PayPalPayResponse{
		Ack:           ack,
		Build:         values.Get("responseEnvelope.build"),
		CorrelationId: values.Get("responseEnvelope.correlationId"),
		Timestamp:     values.Get("responseEnvelope.timestamp"),
		PayKey:        values.Get("payKey"),
	}
	payUrl := fmt.Sprintf("%s&paykey=%s", kPayBaseUrl, res.PayKey)
	return res, payUrl, nil
}

// IPN handler references: http://goo.gl/bIX2Q and http://goo.gl/F1uej
func PayPalValidateIpn(requestBody string, c *Context) (*PayPalIpnMessage, error) {
	c.Aec().Debugf("PayPalValidateIpn")

	// Post back to IPN server to get verification.
	postBody := fmt.Sprintf("cmd=_notify-validate&%s", requestBody)
	c.Aec().Debugf("IPN post body: %v", postBody)

	respStr, err := getResponseBody(urlfetch.Client(c.Aec()).Post(
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
	c.Aec().Debugf("IPN message: %v", values)

	amountStr := values.Get("transaction[0].amount")
	currencyAndAmount := strings.Split(amountStr, " ")
	Assert(len(currencyAndAmount) == 2, "Invalid amountStr: %q", amountStr)
	// TODO(sadovsky): Support other currencies.
	Assert(currencyAndAmount[0] == "USD", "Invalid currency in amountStr: %q", amountStr)

	res := &PayPalIpnMessage{
		Status:     values.Get("status"),
		PayerEmail: ParseEmail(values.Get("sender_email")),
		PayeeEmail: ParseEmail(values.Get("transaction[0].receiver")),
		Amount:     ParseAmount(currencyAndAmount[1]),
		PayKey:     values.Get("pay_key"),
	}
	return res, nil
}
