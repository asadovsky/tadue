// Copyright 2012 Adam Sadovsky. All rights reserved.

package tadue

const (
	COOKIE_LIFESPAN                   = 7  // lifespan of cookie in days
	VERIFICATION_LIFESPAN             = 3  // lifespan of verification request in days
	PASSWORD_RESET_LIFESPAN           = 1  // lifespan of password reset request in days
	MAX_PAYMENTS_TO_SHOW              = 20 // max number of payments to show in list
	PAY_REQUEST_EMAIL_RATE_LIMIT      = 1  // min number of days between pay request emails
	AUTO_PAY_REQUEST_EMAIL_RATE_LIMIT = 7  // automatic reminder email frequency in days
)
