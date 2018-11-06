package runes

func Create(rs ...[]rune) (x []rune) {
	for _, r := range rs {
		x = append(x, r...)
	}
	return x
}

func IndexRune(s []rune, r rune) int {
	for i, sr := range s {
		if r == sr {
			return i
		}
	}
	return -1
}

func LastIndexRune(s []rune, r rune) int {
	for i := len(s)-1; i >= 0; i-- {
		sr := s[i]
		if r == sr {
			return i
		}
	}
	return -1
}

func Equals(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func Count(s []rune, r rune) (n int) {
	for _, sr := range s {
		if sr == r {
			n++
		}
	}
	return
}

func HasPrefix(s, prefix []rune) bool {
	return len(s) >= len(prefix) && Equals(s[0:len(prefix)], prefix)
}

func HasSuffix(s, suffix []rune) bool {
	return len(s) >= len(suffix) && Equals(s[len(s)-len(suffix):], suffix)
}

// TrimPrefix returns s without the provided leading prefix string.
// If s doesn't start with prefix, s is returned unchanged.
func TrimPrefix(s, prefix []rune) []rune {
	if HasPrefix(s, prefix) {
		return s[len(prefix):]
	}
	return s
}

func TrimRune(s []rune, r rune) (out []rune) {
	if len(s) == 0 {
		return nil
	}

	var i = 0
	var sr rune
	// Trim prefix
	for i, sr = range s {
		if sr == r {
			out = s[i:]
		} else {
			break
		}
	}
	s = out

	if len(s) == 0 {
		return nil
	}

	// Trim suffix
	for i := len(s)-1; i >= 0; i++ {
		if s[i] == r {
			out = s[:i]
		} else {
			break
		}
	}

	return out
}
