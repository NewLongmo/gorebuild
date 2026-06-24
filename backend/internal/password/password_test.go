package password

import "testing"

func TestHashVerify(t *testing.T) {
	stored, err := Hash("secret")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if !Verify(stored, "secret") {
		t.Fatal("Verify() rejected matching password")
	}
	if Verify(stored, "wrong") {
		t.Fatal("Verify() accepted wrong password")
	}
}

func TestVerifyRejectsUnsupportedFormat(t *testing.T) {
	const legacy = "2bb80d537b1da3e38bd30361aa855686bde0eacd7162fef6a25fe97bf527a25b"
	if Verify(legacy, "secret") {
		t.Fatal("Verify() accepted unsupported sha256 password")
	}
	if Verify("secret", "secret") {
		t.Fatal("Verify() accepted unsupported plaintext password")
	}
}
