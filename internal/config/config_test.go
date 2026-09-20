package config

import "testing"

func TestValidateAdminToken(t *testing.T) {
	t.Parallel()
	ok := "abcdefghijklmnopqrstuvwx" // 24
	cases := []struct {
		token   string
		wantErr bool
	}{
		{ok, false},
		{"short", true},
		{"change-me", true},
		{"nexa", true},
		{"replace-with-a-long-random-token", true},
		{"replace-with-something-longer-ok", true},
		{"abcdefghijklmnopqrstuvw", true}, // 23
	}
	for _, tc := range cases {
		err := validateAdminToken(tc.token)
		if (err != nil) != tc.wantErr {
			t.Fatalf("%q: err=%v wantErr=%v", tc.token, err, tc.wantErr)
		}
	}
}

func TestValidateRequiresDatabase(t *testing.T) {
	t.Parallel()
	cfg := Config{AdminToken: "abcdefghijklmnopqrstuvwx"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected database error")
	}
	cfg.DatabaseURL = "postgres://x"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
