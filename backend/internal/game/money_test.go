package game

import "testing"

func TestCashoutRoundsDownToCents(t *testing.T) {
	multiplier, err := multiplierFor(25, 3, 1)
	if err != nil {
		t.Fatalf("calculate multiplier: %v", err)
	}
	if got := multiplier.RatString(); got != "12/11" {
		t.Fatalf("multiplier = %s, want %s", got, "12/11")
	}

	payout, err := payoutFor(1000, multiplier)
	if err != nil {
		t.Fatalf("calculate payout: %v", err)
	}
	if payout != 1090 {
		t.Fatalf("payout = %d, want 1090", payout)
	}
	if formatted := formatDollars(payout); formatted != "10.90" {
		t.Fatalf("formatted payout = %q, want %q", formatted, "10.90")
	}
}
