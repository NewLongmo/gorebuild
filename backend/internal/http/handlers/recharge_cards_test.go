package handlers

import "testing"

func TestValidateRechargeCardCreatePayload(t *testing.T) {
	tests := []struct {
		name    string
		req     rechargeCardCreatePayload
		wantErr bool
	}{
		{name: "valid", req: rechargeCardCreatePayload{Count: 10, Amount: 50}},
		{name: "custom codes ignore count", req: rechargeCardCreatePayload{Amount: 50, Codes: []string{" CARD-1 ", "CARD-2"}}},
		{name: "zero count", req: rechargeCardCreatePayload{Count: 0, Amount: 50}, wantErr: true},
		{name: "too many", req: rechargeCardCreatePayload{Count: 101, Amount: 50}, wantErr: true},
		{name: "duplicate custom code", req: rechargeCardCreatePayload{Amount: 50, Codes: []string{"CARD-1", "card-1"}}, wantErr: true},
		{name: "zero amount", req: rechargeCardCreatePayload{Count: 1, Amount: 0}, wantErr: true},
		{name: "negative amount", req: rechargeCardCreatePayload{Count: 1, Amount: -1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRechargeCardCreatePayload(tt.req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateRechargeCardCreatePayload() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNormalizeRechargeCardCodes(t *testing.T) {
	codes := normalizeRechargeCardCodes([]string{" A ", "", "B", "  "})
	if len(codes) != 2 {
		t.Fatalf("len(codes) = %d, want 2", len(codes))
	}
	if codes[0] != "A" || codes[1] != "B" {
		t.Fatalf("codes = %#v, want [A B]", codes)
	}
}

func TestRandomRechargeCardCode(t *testing.T) {
	code, err := randomRechargeCardCode(16)
	if err != nil {
		t.Fatalf("randomRechargeCardCode() error = %v", err)
	}
	if len(code) != 16 {
		t.Fatalf("code length = %d, want 16", len(code))
	}
	for _, ch := range code {
		if !stringsContainsRune(rechargeCardAlphabet, ch) {
			t.Fatalf("code contains unexpected character %q", ch)
		}
	}
}

func stringsContainsRune(value string, target rune) bool {
	for _, ch := range value {
		if ch == target {
			return true
		}
	}
	return false
}
