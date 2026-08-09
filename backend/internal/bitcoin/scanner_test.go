package bitcoin

import "testing"

func TestStatusFor(t *testing.T) {
	cases := []struct {
		name          string
		confirmations int64
		minConf       int64
		want          string
	}{
		{"zero confirmations", 0, 2, "pending"},
		{"below threshold", 1, 2, "pending"},
		{"exactly threshold", 2, 2, "confirmed"},
		{"above threshold", 10, 2, "confirmed"},
		{"threshold zero always confirmed", 0, 0, "confirmed"},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := statusFor(c.confirmations, c.minConf)
			if got != c.want {
				t.Errorf("statusFor(%d, %d) = %q, want %q", c.confirmations, c.minConf, got, c.want)
			}
		})
	}
}
