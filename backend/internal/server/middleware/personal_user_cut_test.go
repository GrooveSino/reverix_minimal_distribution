package middleware

import "testing"

func TestPersonalUserFeatureBlocked(t *testing.T) {
	t.Parallel()

	cases := []struct {
		method string
		path   string
		want   bool
	}{
		{"GET", "/api/v1/user/profile", false},
		{"PUT", "/api/v1/user/password", false},
		{"GET", "/api/v1/keys", false},
		{"GET", "/api/v1/usage", false},
		{"GET", "/api/v1/channel-monitors", false},
		{"PUT", "/api/v1/user", true},
		{"GET", "/api/v1/redeem", true},
		{"POST", "/api/v1/redeem", true},
		{"GET", "/api/v1/subscriptions", true},
		{"GET", "/api/v1/user/aff", true},
		{"GET", "/api/v1/user/totp/status", true},
		{"GET", "/api/v1/user/passkeys", true},
		{"POST", "/api/v1/payment/orders", true},
		{"GET", "/api/v1/channels/available", true},
	}

	for _, tc := range cases {
		got := personalUserFeatureBlocked(tc.path, tc.method)
		if got != tc.want {
			t.Fatalf("%s %s: got %v want %v", tc.method, tc.path, got, tc.want)
		}
	}
}
