package app

const (
	kSessionCookieLifespan        = 14 // lifespan of session cookie in days
	kVerifyEmailLifespan          = 2  // lifespan of VerifyEmail request in days
	kResetPasswordLifespanMinutes = 15 // lifespan of ResetPassword request in minutes
	kMaxPaymentsToShow            = 20 // max number of payments to show in list
	kPayRequestEmailCooldown      = 1  // min number of days between pay request emails
	kAutoPayRequestEmailFrequency = 7  // automatic reminder email frequency in days
)
