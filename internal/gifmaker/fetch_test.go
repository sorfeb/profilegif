package gifmaker

import (
	"context"
	"testing"
)

func TestMockStatsDeterministic(t *testing.T) {
	a := MockStats("sorfeb")
	b := MockStats("sorfeb")
	if a != b {
		t.Errorf("mock stats not deterministic: %+v vs %+v", a, b)
	}
	if a.Login != "sorfeb" {
		t.Errorf("login: got %q", a.Login)
	}
	if a.TotalCommits <= 0 || a.Followers <= 0 {
		t.Errorf("mock stats should be non-zero: %+v", a)
	}
	// Different logins should (almost always) differ.
	if MockStats("octocat") == a {
		t.Error("different logins produced identical stats")
	}
}

func TestFetchMockMode(t *testing.T) {
	t.Setenv("PROFILEGIF_MOCK", "1")
	s, err := Fetch(context.Background(), "anyone", "")
	if err != nil {
		t.Fatalf("mock fetch should not error: %v", err)
	}
	if s.Login != "anyone" || s.TotalCommits <= 0 {
		t.Errorf("unexpected mock stats: %+v", s)
	}
}

func TestFetchRequiresTokenWhenNotMock(t *testing.T) {
	t.Setenv("PROFILEGIF_MOCK", "0")
	if _, err := Fetch(context.Background(), "someone", ""); err == nil {
		t.Fatal("expected an error when no token and not in mock mode")
	}
}

func TestFetchEmptyLogin(t *testing.T) {
	if _, err := Fetch(context.Background(), "   ", "token"); err == nil {
		t.Fatal("expected an error for empty login")
	}
}
