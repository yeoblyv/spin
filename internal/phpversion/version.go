// Package phpversion reads a Spider project's required PHP version from its
// composer.json, detects the PHP interpreter available on the machine, and
// reports whether the two are compatible.
package phpversion

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a parsed X.Y.Z version. A component absent from the source
// string is zero.
type Version struct {
	Major, Minor, Patch int
}

// String renders v in X.Y.Z form.
func (v Version) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare returns -1, 0, or 1 as v is less than, equal to, or greater than
// other.
func (v Version) Compare(other Version) int {
	for _, pair := range [][2]int{{v.Major, other.Major}, {v.Minor, other.Minor}, {v.Patch, other.Patch}} {
		if pair[0] != pair[1] {
			if pair[0] < pair[1] {
				return -1
			}
			return 1
		}
	}
	return 0
}

// ParseVersion parses an X, X.Y, or X.Y.Z version string. A trailing
// pre-release/build suffix (e.g. "8.3.0-dev") is ignored.
func ParseVersion(s string) (Version, error) {
	s = strings.SplitN(s, "-", 2)[0]
	s = strings.SplitN(s, "+", 2)[0]
	parts := strings.Split(s, ".")
	if len(parts) == 0 || len(parts) > 3 {
		return Version{}, fmt.Errorf("invalid version %q", s)
	}

	nums := [3]int{}
	for i, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return Version{}, fmt.Errorf("invalid version %q: %w", s, err)
		}
		nums[i] = n
	}
	return Version{Major: nums[0], Minor: nums[1], Patch: nums[2]}, nil
}
