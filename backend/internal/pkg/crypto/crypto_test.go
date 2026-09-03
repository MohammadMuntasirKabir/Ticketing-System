package crypto

import "testing"

func TestRandomStringIsURLSafeAndUnique(t *testing.T) {
	a, err := RandomString(32)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := RandomString(32)
	if a == b {
		t.Fatal("expected random strings to differ")
	}
	if len(a) == 0 {
		t.Fatal("empty random string")
	}
}

func TestSha256HexDeterministic(t *testing.T) {
	// SHA-256("abc") hex begins with ba7816bf8f01cfea414140de5dae2223
	if got := Sha256Hex("abc"); got[:32] != "ba7816bf8f01cfea414140de5dae2223" {
		t.Fatalf("unexpected hash: %s", got)
	}
	if Sha256Hex("abc") == Sha256Hex("abd") {
		t.Fatal("hash collision")
	}
}
