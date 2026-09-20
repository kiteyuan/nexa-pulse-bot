package kernel

import "testing"

func TestAbsoluteHTTPURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		wantErr bool
	}{
		{"https://example.com/a", false},
		{"http://example.com", false},
		{"https://user:pass@example.com/", true},
		{"ftp://example.com", true},
		{"", true},
		{"not-a-url", true},
	}
	for _, tc := range cases {
		_, err := AbsoluteHTTPURL(tc.in)
		if (err != nil) != tc.wantErr {
			t.Fatalf("%q: err=%v wantErr=%v", tc.in, err, tc.wantErr)
		}
	}
}

func TestSafeHTTPURLRejectsPrivate(t *testing.T) {
	t.Parallel()
	blocked := []string{
		"http://127.0.0.1/",
		"http://127.0.0.1:8081/api",
		"http://localhost/x",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://172.16.5.5/",
		"http://169.254.169.254/latest/meta-data/",
		"http://0.0.0.0/",
		"http://[::1]/",
		"http://[fd00:ec2::254]/",
		"https://user:pass@example.com/",
	}
	for _, raw := range blocked {
		if _, err := SafeHTTPURL(raw); err == nil {
			t.Fatalf("expected block for %s", raw)
		}
	}
}

func TestSafeHTTPURLAllowsPublicIP(t *testing.T) {
	t.Parallel()
	got, err := SafeHTTPURL("https://8.8.8.8/resolve")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://8.8.8.8/resolve" {
		t.Fatal(got)
	}
}
