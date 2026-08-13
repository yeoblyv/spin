package phpversion

import (
	"fmt"
	"strings"
)

// clause is one comparator, e.g. ">=" 8.2.0.
type clause struct {
	op      string
	version Version
	// precision is how many of version's components were actually given
	// in the source string - a bare "8.2" constraint matches any patch,
	// but ">=8.2" does not need this, since >= already means "or higher".
	precision int
}

// Constraint is a parsed Composer-style version constraint: an OR of
// AND-groups, matching the subset of Composer's syntax PHP version
// requirements actually use in practice (comparators, caret, tilde, and
// comma/space/"||"-separated combinations of them).
type Constraint struct {
	orGroups [][]clause
	source   string
}

// String returns the original constraint text.
func (c Constraint) String() string {
	return c.source
}

// ParseConstraint parses a Composer-style version constraint such as
// ">=8.2", "^8.2", "~8.2.0", or "^8.1 || ^8.2".
func ParseConstraint(s string) (Constraint, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return Constraint{}, fmt.Errorf("empty version constraint")
	}

	var orGroups [][]clause
	for _, orPart := range strings.Split(trimmed, "||") {
		var clauses []clause
		for _, andPart := range strings.FieldsFunc(orPart, func(r rune) bool { return r == ',' || r == ' ' }) {
			c, err := parseClause(andPart)
			if err != nil {
				return Constraint{}, fmt.Errorf("parsing constraint %q: %w", s, err)
			}
			clauses = append(clauses, c)
		}
		if len(clauses) == 0 {
			return Constraint{}, fmt.Errorf("parsing constraint %q: empty clause", s)
		}
		orGroups = append(orGroups, clauses)
	}

	return Constraint{orGroups: orGroups, source: trimmed}, nil
}

// parseClause parses a single comparator token: a bare version, an
// explicit operator (>=, >, <=, <, =), a caret (^X.Y[.Z] - locks the
// major version), or a tilde (~X.Y[.Z] - locks everything but the last
// given component).
func parseClause(token string) (clause, error) {
	token = strings.TrimSpace(strings.TrimSuffix(token, ".*"))

	for _, op := range []string{">=", "<=", "==", "!=", ">", "<", "="} {
		if rest, ok := strings.CutPrefix(token, op); ok {
			v, err := ParseVersion(strings.TrimSpace(rest))
			if err != nil {
				return clause{}, err
			}
			normalized := op
			if normalized == "==" {
				normalized = "="
			}
			return clause{op: normalized, version: v}, nil
		}
	}

	if rest, ok := strings.CutPrefix(token, "^"); ok {
		v, precision, err := parsePartialVersion(rest)
		if err != nil {
			return clause{}, err
		}
		return clause{op: "^", version: v, precision: precision}, nil
	}

	if rest, ok := strings.CutPrefix(token, "~"); ok {
		v, precision, err := parsePartialVersion(rest)
		if err != nil {
			return clause{}, err
		}
		return clause{op: "~", version: v, precision: precision}, nil
	}

	v, precision, err := parsePartialVersion(token)
	if err != nil {
		return clause{}, err
	}
	return clause{op: "=", version: v, precision: precision}, nil
}

// parsePartialVersion parses a version that may omit trailing components
// (e.g. "8" or "8.2"), returning how many components were actually given.
func parsePartialVersion(s string) (Version, int, error) {
	v, err := ParseVersion(s)
	if err != nil {
		return Version{}, 0, err
	}
	return v, len(strings.Split(s, ".")), nil
}

// Satisfies reports whether v meets the constraint.
func (c Constraint) Satisfies(v Version) bool {
	for _, group := range c.orGroups {
		allMatch := true
		for _, cl := range group {
			if !cl.matches(v) {
				allMatch = false
				break
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// matches evaluates a single clause. Comparators with a partial version
// (e.g. "<=8.2") compare against that version zero-filled to X.Y.0, not
// Composer's own "treat as an inclusive range" expansion for < and <= -
// PHP version constraints in practice specify the full precision they
// mean, so the simpler reading is used here.
func (cl clause) matches(v Version) bool {
	switch cl.op {
	case ">=":
		return v.Compare(cl.version) >= 0
	case ">":
		return v.Compare(cl.version) > 0
	case "<=":
		return v.Compare(cl.version) <= 0
	case "<":
		return v.Compare(cl.version) < 0
	case "=":
		return matchesPrefix(v, cl.version, cl.precision)
	case "^":
		upper := Version{Major: cl.version.Major + 1}
		return v.Compare(cl.version) >= 0 && v.Compare(upper) < 0
	case "~":
		var upper Version
		if cl.precision <= 2 {
			upper = Version{Major: cl.version.Major + 1}
		} else {
			upper = Version{Major: cl.version.Major, Minor: cl.version.Minor + 1}
		}
		return v.Compare(cl.version) >= 0 && v.Compare(upper) < 0
	default:
		return false
	}
}

// matchesPrefix reports whether v agrees with want on its first precision
// components, treating the rest as wildcards.
func matchesPrefix(v, want Version, precision int) bool {
	if precision >= 1 && v.Major != want.Major {
		return false
	}
	if precision >= 2 && v.Minor != want.Minor {
		return false
	}
	if precision >= 3 && v.Patch != want.Patch {
		return false
	}
	return true
}
