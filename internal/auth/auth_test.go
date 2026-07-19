package auth

import "testing"

func TestSignVerifyAndTamper(t *testing.T) {
	pub, priv, err := NewIdentity()
	if err != nil {
		t.Fatal(err)
	}
	msg := []byte("pair-request")
	sig := Sign(priv, msg)
	if !Verify(encode(pub), sig, msg) {
		t.Fatal("valid signature rejected")
	}
	if Verify(encode(pub), sig, []byte("tampered")) {
		t.Fatal("tampered message accepted")
	}
}

func TestTokenUniqueAndHashDoesNotExposeToken(t *testing.T) {
	a, _ := Token()
	b, _ := Token()
	if a == b || len(a) < 32 {
		t.Fatal("weak token")
	}
	if Hash(a) == a {
		t.Fatal("token stored in clear")
	}
}

func encode(b []byte) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	out := make([]byte, (len(b)*8+5)/6)
	var acc uint
	var bits uint
	var n int
	for _, v := range b {
		acc = acc<<8 | uint(v)
		bits += 8
		for bits >= 6 {
			bits -= 6
			out[n] = chars[(acc>>bits)&63]
			n++
		}
	}
	if bits > 0 {
		out[n] = chars[(acc<<(6-bits))&63]
	}
	return string(out)
}
