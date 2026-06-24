package password

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
)

const (
	version    = "v1"
	iterations = 120000
	saltBytes  = 16
	keyBytes   = 32
)

func Hash(plain string) (string, error) {
	var salt [saltBytes]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key := derive([]byte(plain), salt[:], iterations, keyBytes)
	return strings.Join([]string{
		version,
		strconv.Itoa(iterations),
		base64.RawURLEncoding.EncodeToString(salt[:]),
		base64.RawURLEncoding.EncodeToString(key),
	}, "$"), nil
}

func Verify(stored, plain string) bool {
	if plain == "" {
		return false
	}
	return verifyV1(stored, plain)
}

func verifyV1(stored, plain string) bool {
	parts := strings.Split(stored, "$")
	if len(parts) != 4 || parts[0] != version {
		return false
	}
	count, err := strconv.Atoi(parts[1])
	if err != nil || count < 1 {
		return false
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(salt) == 0 {
		return false
	}
	want, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil || len(want) == 0 {
		return false
	}
	got := derive([]byte(plain), salt, count, len(want))
	return hmac.Equal(got, want)
}

func derive(password, salt []byte, count, length int) []byte {
	result := make([]byte, 0, length)
	for block := 1; len(result) < length; block++ {
		u := prf(password, salt, block)
		t := append([]byte(nil), u...)
		for i := 1; i < count; i++ {
			u = prf(password, u, 0)
			for j := range t {
				t[j] ^= u[j]
			}
		}
		result = append(result, t...)
	}
	return result[:length]
}

func prf(password, data []byte, block int) []byte {
	mac := hmac.New(sha256.New, password)
	mac.Write(data)
	if block > 0 {
		mac.Write([]byte{byte(block >> 24), byte(block >> 16), byte(block >> 8), byte(block)})
	}
	return mac.Sum(nil)
}
