package auth

import "testing"

func TestGenerateSessionToken(t *testing.T) {
	token, hash, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token is empty")
	}
	if len(hash) != 64 {
		t.Fatalf("hash length = %d, want 64", len(hash))
	}
	if HashSessionToken(token) != hash {
		t.Fatal("returned hash does not match token hash")
	}
	other, _, err := GenerateSessionToken()
	if err != nil {
		t.Fatal(err)
	}
	if other == token {
		t.Fatal("tokens are not unique")
	}
}

func TestHashSessionToken(t *testing.T) {
	// Known SHA-256 of "abc"; guards the hashing scheme against accidental change.
	const wantABC = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got := HashSessionToken("abc"); got != wantABC {
		t.Fatalf("HashSessionToken(abc) = %q, want %q", got, wantABC)
	}
	if HashSessionToken("abc") == HashSessionToken("abd") {
		t.Fatal("distinct tokens share a hash")
	}
}
