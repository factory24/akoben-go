package connectivity

import "testing"

func TestEnvPrefersTheNewNameAndStillReadsTheOld(t *testing.T) {
	t.Setenv("CONNECTIVITY.BASE_URL", "https://old.example/v1")
	if got := env("BASE_URL"); got != "https://old.example/v1" {
		t.Fatalf("old prefix not read: %q", got)
	}
	t.Setenv("AKOBEN_BASE_URL", "https://underscored.example/v1")
	if got := env("BASE_URL"); got != "https://underscored.example/v1" {
		t.Fatalf("new underscored name does not win over the old prefix: %q", got)
	}
	t.Setenv("AKOBEN.BASE_URL", "https://new.example/v1")
	if got := env("BASE_URL"); got != "https://new.example/v1" {
		t.Fatalf("new dotted name does not win: %q", got)
	}
	t.Setenv("AKOBEN_CELLULAR_API_KEY", "k")
	if got := env("CELLULAR.API_KEY"); got != "k" {
		t.Fatalf("nested key not underscored: %q", got)
	}
}
