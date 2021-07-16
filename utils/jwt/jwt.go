package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ray1422/YAM-api/utils"
)

// InvalidSignatureError invalid signature error
type InvalidSignatureError error

const (
	// JWTMaxLength jwt max length
	JWTMaxLength = 2048
)

var (
	// JWTSecret JWT_SECRET from env
	JWTSecret = utils.GetEnv("JWT_SECRET", "JWT_SECRET_EXAMPLE")
)

// JWT JWT
type JWT struct {
	header    map[string]string
	payload   map[string]string
	signature string
}

func (j *JWT) genUnencryptedPartStr() string {
	headerJSON, _ := json.Marshal(j.header)
	payloadJSON, _ := json.Marshal(j.payload)
	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	return headerB64 + "." + payloadB64
}

// GenSignature generate the correct signature
func (j *JWT) GenSignature() string {
	if j.signature == "" {
		j.signature = hmacSha256Str(j.genUnencryptedPartStr(), []byte(JWTSecret))
	}
	return j.signature
}

// TokenString Token to String
func (j *JWT) TokenString() string {
	return j.genUnencryptedPartStr() + j.GenSignature()
}

// FromString init JWT from string with verifying signature
func FromString(s string) (*JWT, error) {
	if len(s) > JWTMaxLength {
		return nil, InvalidSignatureError(errors.New("token too long"))
	}
	return parseToken(s)
}

// Check check if a jwt token is not expire_time
func (j *JWT) Check() bool {
	// signature is verified in parse.
	// check time
	if ts, ok := j.payload["expire_time"]; ok {
		if t, err := strconv.ParseInt(ts, 10, 64); err == nil {
			if t < time.Now().Unix() { // seconds
				return false
			}
		} else {
			return false
		}
	} else {
		return false
	}
	return true
}

// parse token into JWT instance and verify signature
func parseToken(s string) (*JWT, error) {
	arr := strings.Split(s, ".")
	if len(arr) != 3 {
		return nil, errors.New("invalid format")
	}
	headerStr, payloadStr, sig := func() (string, string, string) { return arr[0], arr[1], arr[2] }()
	header := map[string]string{}
	payload := map[string]string{}
	if hmacSha256Str(headerStr+"."+payloadStr, []byte(JWTSecret)) != sig {
		return nil, errors.New("")
	}
	if headerJSON, err := base64.RawURLEncoding.DecodeString(headerStr); err == nil {
		if err := json.Unmarshal(headerJSON, &header); err != nil {
			return nil, err
		}
	}
	if payloadJSON, err := base64.RawURLEncoding.DecodeString(payloadStr); err == nil {
		if err := json.Unmarshal(payloadJSON, &payload); err != nil {
			return nil, err
		}
	}

	jwt := JWT{}
	jwt.header = header
	jwt.payload = payload
	return &jwt, nil
}
