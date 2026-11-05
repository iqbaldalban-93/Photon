package authx

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"app/libweb/configx"
)

const MinPasswordLen = 8

const (
	cookieName   = "photon_session"
	sessionDays  = 30
	loginWindow  = 10
	loginLockSec = 30
)

type sessionClaims struct {
	U string `json:"u"`
	E int64  `json:"e"`
}

func HashPassword(pw string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(b), err
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func SessionCookie(username string) (*http.Cookie, error) {
	secret := configx.Get().Auth.SessionSecret
	if secret == "" {
		return nil, ErrNotConfigured
	}
	claims, _ := json.Marshal(sessionClaims{U: username, E: time.Now().Add(sessionDays * 24 * time.Hour).Unix()})
	payload := base64.RawURLEncoding.EncodeToString(claims)
	mac := sign(payload, secret)
	return &http.Cookie{
		Name:     cookieName,
		Value:    payload + "." + mac,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   sessionDays * 24 * 3600,
	}, nil
}

func ClearCookie() *http.Cookie {
	return &http.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   -1,
	}
}

func Validate(r *http.Request) (string, bool) {
	secret := configx.Get().Auth.SessionSecret
	if secret == "" {
		return "", false
	}
	c, err := r.Cookie(cookieName)
	if err != nil {
		return "", false
	}
	payload, mac, ok := strings.Cut(c.Value, ".")
	if !ok {
		return "", false
	}
	expect := sign(payload, secret)
	if subtle.ConstantTimeCompare([]byte(mac), []byte(expect)) != 1 {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return "", false
	}
	var claims sessionClaims
	if json.Unmarshal(raw, &claims) != nil || claims.E < time.Now().Unix() {
		return "", false
	}
	return claims.U, true
}

func Authorized(r *http.Request) bool {
	_, ok := Validate(r)
	return ok
}

func sign(payload, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

var ErrNotConfigured = errAuth("auth is not configured")

type errAuth string

func (e errAuth) Error() string { return string(e) }

type failState struct {
	count int
	last  int64
}

var (
	failMu sync.Mutex
	fails  = map[string]*failState{}
)

const maxTrackedClients = 1024

func LoginThrottled(client string) bool {
	failMu.Lock()
	defer failMu.Unlock()
	f := fails[client]
	if f == nil || f.count < loginWindow {
		return false
	}
	return time.Now().Unix()-f.last < loginLockSec
}

func RecordLoginResult(client string, ok bool) {
	failMu.Lock()
	defer failMu.Unlock()
	if ok {
		delete(fails, client)
		return
	}
	now := time.Now().Unix()
	if len(fails) >= maxTrackedClients {
		for k, v := range fails {
			if now-v.last >= loginLockSec {
				delete(fails, k)
			}
		}
	}
	f := fails[client]
	if f == nil {
		f = &failState{}
		fails[client] = f
	}
	f.count++
	f.last = now
}

func NewSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func NeedsSetup() bool { return configx.Get().Auth.PasswordHash == "" }

func SetupUser(username, password string) error {
	if username == "" || len(password) < MinPasswordLen {
		return errAuth("username required and password must be at least 8 characters")
	}
	hash, err := HashPassword(password)
	if err != nil {
		return err
	}
	taken := false
	err = configx.Update(func(c *configx.Config) {
		if c.Auth.PasswordHash != "" {
			taken = true
			return
		}
		c.Auth.Username = username
		c.Auth.PasswordHash = hash
		c.Auth.SessionSecret = NewSecret()
	})
	if err == nil && taken {
		return errAuth("already configured")
	}
	return err
}

func Login(username, password string) bool {
	a := configx.Get().Auth
	userOK := subtle.ConstantTimeCompare([]byte(a.Username), []byte(username)) == 1
	passOK := a.PasswordHash != "" && CheckPassword(a.PasswordHash, password)
	return userOK && passOK
}
