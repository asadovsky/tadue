package app

const (
	kAppHostname          = ""
	kAppHostnameForPayPal = "http://98.248.42.191"
	kUserId               = "adam_1335650859_biz_api1.gmail.com"
	kPassword             = "1335650898"
	kSignature            = "A9gqVQB8-mmb1uodjDQ9XZGG8wdlAohgwF511kB0uyIlOlahuHK9YiQb"
	kAppId                = "APP-80W284485P519543T"
	kPayEndpoint          = "https://svcs.sandbox.paypal.com/AdaptivePayments/Pay"
	kDetailsEndpoint      = "https://svcs.sandbox.paypal.com/AdaptivePayments/PaymentDetails"
	kPayBaseUrl           = "https://www.sandbox.paypal.com/cgi-bin/webscr?cmd=_ap-payment"
	kValidateIpnUrl       = "https://www.sandbox.paypal.com/cgi-bin/webscr"
)

// Credentials from: https://code.google.com/apis/console/
const (
	kGoogleClientId     = "71909377510-8k8ncu2rj698g4h9pl8gjdc1hc89n2ih.apps.googleusercontent.com"
	kGoogleClientSecret = "H58q7A1ZOenZylcZARDp-kW3"
	kGoogleRedirectURL  = "http://localhost:8080/oauth2callback"
)

// Generated by: tools/genkeys.go
var kHashKey = []byte{87, 110, 138, 177, 111, 112, 147, 211, 44, 114, 16, 228, 82, 91, 38, 233, 115, 95, 112, 94, 240, 171, 27, 81, 156, 32, 198, 178, 124, 81, 46, 111, 192, 219, 27, 116, 1, 110, 7, 25, 254, 112, 29, 97, 112, 100, 147, 252, 81, 182, 96, 223, 210, 86, 110, 135, 211, 151, 44, 122, 111, 120, 243, 158}
var kBlockKey = []byte{135, 231, 64, 212, 104, 10, 143, 241, 8, 250, 71, 19, 147, 182, 45, 1, 180, 215, 203, 252, 125, 11, 180, 134, 191, 184, 36, 206, 149, 72, 210, 99}
