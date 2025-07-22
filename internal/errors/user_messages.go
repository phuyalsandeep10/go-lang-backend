package errors

// User-friendly error messages
const (
	MsgInvalidAddress     = "The provided address is incomplete or incorrectly formatted. Please include street, city, state, and zip code."
	MsgPropertyNotFound   = "Property not found. Please try a different address."
	MsgServiceUnavailable = "We're unable to retrieve property information right now. Please try again in a few minutes."
	MsgRateLimited        = "You're searching too quickly! Please wait a moment and try again."
	MsgInvalidParameters  = "The provided parameters are invalid. Please check your input and try again."
	MsgInternalError      = "Something went wrong on our end. Please try again later."
)
