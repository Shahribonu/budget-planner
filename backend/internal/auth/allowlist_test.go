package auth

import "testing"

func TestAllowlist(t *testing.T) {
	a, err := ParseAllowlist(" 5174914961, 42 ,")
	if err != nil {
		t.Fatal(err)
	}
	if !a.Allows(5174914961) || !a.Allows(42) || a.Allows(7) {
		t.Fatalf("allowlist %v", a)
	}

	empty, err := ParseAllowlist("")
	if err != nil || !empty.Allows(7) {
		t.Fatalf("empty list must allow everyone: %v %v", empty, err)
	}

	for _, bad := range []string{"abc", "-1", "0", "12,x"} {
		if _, err := ParseAllowlist(bad); err == nil {
			t.Errorf("ParseAllowlist(%q) accepted", bad)
		}
	}
}
