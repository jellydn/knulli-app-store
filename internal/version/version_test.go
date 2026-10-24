package version

import "testing"

func TestCompareOrdersNumbersNumericallyNotLexically(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"2.10", "2.9", 1},
		{"2.9", "2.10", -1},
		{"V1.2.3", "v1.2.3", 0},
		{"2.34", "2.34", 0},
		{"1.13.0", "1.13.0", 0},
	}
	for _, item := range cases {
		if got := Compare(item.left, item.right); got != item.want {
			t.Fatalf("Compare(%q, %q) = %d, want %d", item.left, item.right, got, item.want)
		}
	}
}

func TestCompareRanksAShorterRunListLower(t *testing.T) {
	if got := Compare("scarab", "scarab-dev-a1b2c3"); got != -1 {
		t.Fatalf("Compare(codename, suffixed codename) = %d, want -1", got)
	}
	if got := Compare("1.2", "1.2.0"); got != -1 {
		t.Fatalf("Compare(1.2, 1.2.0) = %d, want -1", got)
	}
}

// TestCompareRanksAPrereleaseHigherThanItsRelease records the lenient rule's
// one known weakness. Compare cannot see that a hyphen introduces a prerelease,
// which is exactly why CompareNumeric exists and why callers that must not
// invent an update use it instead.
func TestCompareRanksAPrereleaseHigherThanItsRelease(t *testing.T) {
	if got := Compare("1.13.0-alpha1", "1.13.0"); got != 1 {
		t.Fatalf("Compare(prerelease, release) = %d, want the lenient rule's 1", got)
	}
	if got, ok := CompareNumeric("1.13.0-alpha1", "1.13.0"); got != -1 || !ok {
		t.Fatalf("CompareNumeric(prerelease, release) = %d, %v, want -1, true", got, ok)
	}
}

func TestStartsNumericSeparatesReleaseNumbersFromCodenames(t *testing.T) {
	cases := map[string]bool{
		"2026.05": true,
		"2.34":    true,
		"scarab":  false,
		"":        false,
		"v1.0.0":  false,
	}
	for value, want := range cases {
		if got := StartsNumeric(value); got != want {
			t.Fatalf("StartsNumeric(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestNormalizeStripsTagDecoration(t *testing.T) {
	cases := map[string]string{
		" V1.0.0 ": "1.0.0",
		"v1.13.0":  "1.13.0",
		"1.0.0":    "1.0.0",
		"  ":       "",
	}
	for value, want := range cases {
		if got := Normalize(value); got != want {
			t.Fatalf("Normalize(%q) = %q, want %q", value, got, want)
		}
	}
}

func TestParseReadsNumbersAndPrerelease(t *testing.T) {
	parsed, ok := Parse("v1.13.0-alpha1")
	if !ok {
		t.Fatal("expected v1.13.0-alpha1 to parse")
	}
	if len(parsed.Numbers) != 3 || parsed.Numbers[0] != 1 || parsed.Numbers[1] != 13 || parsed.Numbers[2] != 0 {
		t.Fatalf("unexpected numbers: %v", parsed.Numbers)
	}
	if parsed.Prerelease != "alpha1" {
		t.Fatalf("unexpected prerelease: %q", parsed.Prerelease)
	}
	if parsed, ok := Parse("1.0.0-rc.1+build.9"); !ok || parsed.Prerelease != "rc.1" || len(parsed.Numbers) != 3 {
		t.Fatalf("build metadata should not affect parsing: %+v, %v", parsed, ok)
	}
}

func TestParseRejectsWhatItCannotReadCompletely(t *testing.T) {
	for _, value := range []string{"", "1..0", "1.0.", "1.0-", "-1.0", "scarab", "1.0.0-", "1.-1.0", "1.0.x"} {
		if _, ok := Parse(value); ok {
			t.Fatalf("Parse(%q) should report that it cannot read the value", value)
		}
	}
}

func TestCompareNumericPadsShorterNumberLists(t *testing.T) {
	cases := []struct {
		left, right string
		want        int
		ok          bool
	}{
		{"1.0", "1.0.0", 0, true},
		{"1.0.0", "1.0", 0, true},
		{"1.0.1", "1.0.0", 1, true},
		{"v5.2.0.0", "5.1.0.0", 1, true},
		{"5.1.0.0", "5.2.0.0", -1, true},
		{"1.0.0", "0.9.1", 1, true},
		{"1.0.0-alpha", "1.0.0", -1, true},
		{"1.0.0", "1.0.0-alpha", 1, true},
		{"1.0.0-alpha", "1.0.0-alpha", 0, true},
	}
	for _, item := range cases {
		got, ok := CompareNumeric(item.left, item.right)
		if got != item.want || ok != item.ok {
			t.Fatalf("CompareNumeric(%q, %q) = %d, %v, want %d, %v", item.left, item.right, got, ok, item.want, item.ok)
		}
	}
}

func TestCompareNumericRefusesToOrderWhatItCannot(t *testing.T) {
	cases := []struct{ left, right string }{
		{"prerelease", "0.9.1"},
		{"0.9.1", "prerelease"},
		{"1.0.0-alpha", "1.0.0-beta"},
		{"", "1.0.0"},
	}
	for _, item := range cases {
		if got, ok := CompareNumeric(item.left, item.right); ok {
			t.Fatalf("CompareNumeric(%q, %q) = %d and claimed an order it cannot support", item.left, item.right, got)
		}
	}
}
