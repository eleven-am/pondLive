package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"time"
)

const (
	DefaultTokenTTL = 2 * time.Minute
	minSecretLen    = 32
	expiryLen       = 8
)

var (
	errTokenMalformed = errors.New("server: malformed attach token")
	errTokenInvalid   = errors.New("server: invalid attach token")
	errTokenExpired   = errors.New("server: expired attach token")
)

type attachTokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func newAttachTokens(secret []byte, ttl time.Duration) (*attachTokens, error) {
	if len(secret) == 0 {
		secret = make([]byte, minSecretLen)
		if _, err := rand.Read(secret); err != nil {
			return nil, err
		}
	}
	if len(secret) < minSecretLen {
		return nil, errors.New("server: attach secret must be at least 32 bytes")
	}
	if ttl <= 0 {
		ttl = DefaultTokenTTL
	}
	return &attachTokens{secret: append([]byte(nil), secret...), ttl: ttl, now: time.Now}, nil
}

func (a *attachTokens) sign(sid string, expiry []byte) []byte {
	mac := hmac.New(sha256.New, a.secret)
	mac.Write([]byte(sid))
	mac.Write(expiry)
	return mac.Sum(nil)
}

func (a *attachTokens) issue(sid string) string {
	expiry := make([]byte, expiryLen)
	binary.BigEndian.PutUint64(expiry, uint64(a.now().Add(a.ttl).Unix()))
	return base64.RawURLEncoding.EncodeToString(append(expiry, a.sign(sid, expiry)...))
}

func (a *attachTokens) verify(sid, token string) error {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil || len(raw) != expiryLen+sha256.Size {
		return errTokenMalformed
	}
	expiry, mac := raw[:expiryLen], raw[expiryLen:]
	if !hmac.Equal(mac, a.sign(sid, expiry)) {
		return errTokenInvalid
	}
	if a.now().Unix() > int64(binary.BigEndian.Uint64(expiry)) {
		return errTokenExpired
	}
	return nil
}
