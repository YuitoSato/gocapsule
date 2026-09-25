package newdefined

// Token is a defined type constructed by a plain New()
type Token string // want Token:`&\{New\}`

// New creates a new Token
func New(s string) Token {
	return Token(s)
}
