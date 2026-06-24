package http

import (
	nethttp "net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestLoginAccountFromBodyNormalizesAccount(t *testing.T) {
	body := []byte(`{"account":" Admin ","password":"secret"}`)

	if got := loginAccountFromBody(body); got != "admin" {
		t.Fatalf("loginAccountFromBody() = %q, want admin", got)
	}
}

func TestLoginRateLimitExceededAllowsFiveAttemptsPerWindow(t *testing.T) {
	resetLoginAttempts()
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)

	for i := 0; i < loginAccountRateLimitMax; i++ {
		if loginRateLimited("127.0.0.1", "admin", now) {
			t.Fatalf("attempt %d was limited, want allowed", i+1)
		}
	}
	if !loginRateLimited("127.0.0.1", "admin", now) {
		t.Fatal("sixth attempt was allowed, want limited")
	}
	if loginRateLimited("127.0.0.1", "admin", now.Add(loginRateLimitWindow)) {
		t.Fatal("attempt after reset window was limited, want allowed")
	}
}

func TestLoginRateLimitAppliesIPCapAcrossAccounts(t *testing.T) {
	resetLoginAttempts()
	now := time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC)

	for i := 0; i < loginIPRateLimitMax; i++ {
		if loginRateLimited("127.0.0.1", "user"+strconv.Itoa(i), now) {
			t.Fatalf("attempt %d was limited, want allowed", i+1)
		}
	}
	if !loginRateLimited("127.0.0.1", "another-user", now) {
		t.Fatal("attempt above IP cap was allowed, want limited")
	}
}

func TestSecurityHeadersAndTraceRejection(t *testing.T) {
	app := fiber.New()
	app.Use(rejectTraceMethods)
	app.Use(securityHeaders)
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusNoContent)
	})

	resp, err := app.Test(nethttptestRequest(t, fiber.MethodGet, "/"))
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.Header.Get("X-Frame-Options") != "DENY" {
		t.Fatalf("X-Frame-Options = %q, want DENY", resp.Header.Get("X-Frame-Options"))
	}
	if resp.Header.Get("Content-Security-Policy") == "" {
		t.Fatal("Content-Security-Policy header is missing")
	}

	traceResp, err := app.Test(nethttptestRequest(t, fiber.MethodTrace, "/"))
	if err != nil {
		t.Fatalf("TRACE / failed: %v", err)
	}
	defer traceResp.Body.Close()
	if traceResp.StatusCode != fiber.StatusMethodNotAllowed {
		t.Fatalf("TRACE status = %d, want %d", traceResp.StatusCode, fiber.StatusMethodNotAllowed)
	}
}

func resetLoginAttempts() {
	loginAttempts.Lock()
	defer loginAttempts.Unlock()
	loginAttempts.buckets = map[string]loginAttemptBucket{}
}

func nethttptestRequest(t *testing.T, method, target string) *nethttp.Request {
	t.Helper()
	req, err := nethttp.NewRequest(method, target, strings.NewReader(""))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	return req
}
