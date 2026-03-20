package csrf

import (
	"crypto/subtle"
	"net/http"

	"mini-dba/internal/auth"
)

const (
	cookieName   = "minidba_csrf"
	cookieMaxAge = 86400 * 7
	formField    = "csrf_token"
)

// Cookie 返回 CSRF double-submit cookie。
func Cookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

func ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// FieldName 表单字段名。
func FieldName() string { return formField }

// Ensure 若已登录但无 CSRF Cookie，则写入新令牌（在 RequireAuth 之后对 GET 调用）。
func Ensure(w http.ResponseWriter, r *http.Request) (token string, err error) {
	c, err := r.Cookie(cookieName)
	if err == nil && c.Value != "" {
		return c.Value, nil
	}
	token, err = auth.RandomHex(32)
	if err != nil {
		return "", err
	}
	http.SetCookie(w, Cookie(token))
	return token, nil
}

// Validate POST：表单 token 必须与 Cookie 一致。
func Validate(r *http.Request) bool {
	if err := r.ParseForm(); err != nil {
		return false
	}
	c, err := r.Cookie(cookieName)
	if err != nil || c.Value == "" {
		return false
	}
	formTok := r.FormValue(formField)
	if len(formTok) != len(c.Value) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(formTok), []byte(c.Value)) == 1
}
