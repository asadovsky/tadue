// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

const (
	COOKIE_LIFESPAN                   = 7  // lifespan of cookie in days
	VERIFY_EMAIL_LIFESPAN             = 3  // lifespan of VerifyEmail request in days
	RESET_PASSWORD_LIFESPAN           = 1  // lifespan of ResetPassword request in days
	MAX_PAYMENTS_TO_SHOW              = 20 // max number of payments to show in list
	PAY_REQUEST_EMAIL_RATE_LIMIT      = 1  // min number of days between pay request emails
	AUTO_PAY_REQUEST_EMAIL_RATE_LIMIT = 7  // automatic reminder email frequency in days
)
