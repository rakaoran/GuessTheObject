package configs

var JWTCookie = struct {
	Name     string
	MaxAge   int
	Path     string
	Domain   string
	Secure   bool
	HttpOnly bool
}{
	Name:     "token",
	MaxAge:   3600 * 24 * 365,
	Path:     "/",
	Domain:   "",
	Secure:   true,
	HttpOnly: true,
}
