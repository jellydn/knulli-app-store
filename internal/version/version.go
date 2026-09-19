// Package version owns every version comparison this project performs: Knulli
// firmware releases, glibc versions, and upstream package tags. It exists
// because two callers used to carry their own copy of the ordering rules, so a
// fix in one could silently diverge from the other and a package could be
// compatible while being reported as out of date, or the reverse.
//
// Two callers need different strictness, so ordering is split rather than
// forced into one rule. Compare is lenient: it always returns an order, which
// suits a gate that is asked to order release strings it did not author.
// CompareNumeric refuses to guess, which suits a caller that must not invent an
// update when it cannot read the version.
package version

import (
	"regexp"
	"strconv"
	"strings"
)

// part matches one run of digits or letters. Splitting on runs rather than on
// separators is what lets the lenient rule order "scarab-dev-a1b2c3" and
// "2.34" without knowing which shape it was handed.
var part = regexp.MustCompile(`[0-9]+|[a-zA-Z]+`)

// numericStart matches a value that begins with a digit, which is how a caller
// tells a release number from a release codename.
var numericStart = regexp.MustCompile(`^[0-9]`)

// Normalize lowercases a version, trims surrounding space, and drops the "v"
// prefix that release tags carry and manifests do not.
func Normalize(value string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "v")
}

// StartsNumeric reports whether a value begins with a digit. A caller that
// needs two ordered values uses it to decide whether the comparison is
// meaningful at all.
func StartsNumeric(value string) bool {
	return numericStart.MatchString(value)
}

// Compare orders two values by their alternating numeric and alphabetic runs,
// comparing numbers numerically and anything else lexically. A shorter run list
// sorts lower. It always returns -1, 0, or 1 and never reports that it could
// not read a value, so a caller that must not guess has to gate the input
// itself.
func Compare(left, right string) int {
	a := part.FindAllString(strings.ToLower(left), -1)
	b := part.FindAllString(strings.ToLower(right), -1)
	for index := 0; index < len(a) || index < len(b); index++ {
		if index >= len(a) {
			return -1
		}
		if index >= len(b) {
			return 1
		}
		leftNumber, leftErr := strconv.Atoi(a[index])
		rightNumber, rightErr := strconv.Atoi(b[index])
		var result int
		if leftErr == nil && rightErr == nil {
			result = leftNumber - rightNumber
		} else {
			result = strings.Compare(a[index], b[index])
		}
		if result < 0 {
			return -1
		}
		if result > 0 {
			return 1
		}
	}
	return 0
}

// Version is one parsed numeric release: its dot-separated numbers and the
// prerelease that followed a hyphen, if any.
type Version struct {
	Numbers    []int
	Prerelease string
}

// Parse reads a dotted numeric version with an optional prerelease suffix. It
// reports false for anything it cannot read completely, including an empty
// component, a non-numeric component, a negative number, and a trailing hyphen.
// Build metadata after "+" is ignored, because it does not order releases.
func Parse(value string) (Version, bool) {
	value = Normalize(value)
	if index := strings.IndexByte(value, '+'); index >= 0 {
		value = value[:index]
	}
	prerelease := ""
	if index := strings.IndexByte(value, '-'); index >= 0 {
		prerelease = value[index+1:]
		value = value[:index]
		if prerelease == "" {
			return Version{}, false
		}
	}
	parts := strings.Split(value, ".")
	numbers := make([]int, len(parts))
	for index, item := range parts {
		if item == "" {
			return Version{}, false
		}
		number, err := strconv.Atoi(item)
		if err != nil || number < 0 {
			return Version{}, false
		}
		numbers[index] = number
	}
	return Version{Numbers: numbers, Prerelease: prerelease}, true
}

// CompareNumeric orders two versions, padding the shorter number list with
// zeroes so "1.0" and "1.0.0" are equal. A final release outranks a prerelease
// of the same number. It reports false when either value cannot be parsed, or
// when two different prereleases must be ordered, so the caller can ask for a
// manual review instead of reporting a finding it cannot support.
func CompareNumeric(discovered, current string) (int, bool) {
	left, ok := Parse(discovered)
	if !ok {
		return 0, false
	}
	right, ok := Parse(current)
	if !ok {
		return 0, false
	}
	length := len(left.Numbers)
	if len(right.Numbers) > length {
		length = len(right.Numbers)
	}
	for index := 0; index < length; index++ {
		var leftNumber, rightNumber int
		if index < len(left.Numbers) {
			leftNumber = left.Numbers[index]
		}
		if index < len(right.Numbers) {
			rightNumber = right.Numbers[index]
		}
		if leftNumber > rightNumber {
			return 1, true
		}
		if leftNumber < rightNumber {
			return -1, true
		}
	}
	if left.Prerelease == right.Prerelease {
		return 0, true
	}
	if left.Prerelease == "" {
		return 1, true
	}
	if right.Prerelease == "" {
		return -1, true
	}
	return 0, false
}
