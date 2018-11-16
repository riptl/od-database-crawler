// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fasturl

import (
	"bytes"
	encodingPkg "encoding"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"reflect"
	"testing"
)

type URLTest struct {
	in        string
	out       *URL   // expected parse; RawPath="" means same as Path
	roundtrip string // expected result of reserializing the URL; empty means same as "in".
}

var urltests = []URLTest{
	// no path
	{
		"http://www.google.com",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "www.google.com",
		},
		"",
	},
	// path
	{
		"http://www.google.com/",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "www.google.com",
			Path:   "/",
		},
		"",
	},
	// %20 outside query
	{
		"http://www.google.com/a%20b",
		&URL{
			Scheme:   SchemeHTTP,
			Host:     "www.google.com",
			Path:     "/a%20b",
		},
		"",
	},
	// leading // without scheme should create an authority
	{
		"//foo",
		&URL{
			Host: "foo",
		},
		"",
	},
	// Three leading slashes isn't an authority, but doesn't return an error.
	// (We can't return an error, as this code is also used via
	// ServeHTTP -> ReadRequest -> Parse, which is arguably a
	// different URL parsing context, but currently shares the
	// same codepath)
	{
		"///threeslashes",
		&URL{
			Path: "///threeslashes",
		},
		"",
	},
	// unescaped @ in username should not confuse host
	{
		"http://j@ne:password@google.com",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "google.com",
		},
		"http://google.com",
	},
	// unescaped @ in password should not confuse host
	{
		"http://jane:p@ssword@google.com",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "google.com",
		},
		"http://google.com",
	},
	// Relative path
	{
		"a/b/c",
		&URL{
			Path: "a/b/c",
		},
		"a/b/c",
	},
	// host subcomponent; IPv4 address in RFC 3986
	{
		"http://192.168.0.1/",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "192.168.0.1",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv4 address in RFC 3986
	{
		"http://192.168.0.1:8080/",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "192.168.0.1:8080",
			Path:   "/",
		},
		"",
	},
	// host subcomponent; IPv6 address in RFC 3986
	{
		"http://[fe80::1]/",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1]",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv6 address in RFC 3986
	{
		"http://[fe80::1]:8080/",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1]:8080",
			Path:   "/",
		},
		"",
	},
	// host subcomponent; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25en0]/", // alphanum zone identifier
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1%en0]",
			Path:   "/",
		},
		"",
	},
	// host and port subcomponents; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25en0]:8080/", // alphanum zone identifier
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1%en0]:8080",
			Path:   "/",
		},
		"",
	},
	// host subcomponent; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25%65%6e%301-._~]/", // percent-encoded+unreserved zone identifier
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1%en01-._~]",
			Path:   "/",
		},
		"http://[fe80::1%25en01-._~]/",
	},
	// host and port subcomponents; IPv6 address with zone identifier in RFC 6874
	{
		"http://[fe80::1%25%65%6e%301-._~]:8080/", // percent-encoded+unreserved zone identifier
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[fe80::1%en01-._~]:8080",
			Path:   "/",
		},
		"http://[fe80::1%25en01-._~]:8080/",
	},
	// golang.org/issue/12200 (colon with empty port)
	{
		"http://192.168.0.2:8080/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "192.168.0.2:8080",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://192.168.0.2:/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "192.168.0.2:",
			Path:   "/foo",
		},
		"",
	},
	{
		// Malformed IPv6 but still accepted.
		"http://2b01:e34:ef40:7730:8e70:5aff:fefe:edac:8080/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "2b01:e34:ef40:7730:8e70:5aff:fefe:edac:8080",
			Path:   "/foo",
		},
		"",
	},
	{
		// Malformed IPv6 but still accepted.
		"http://2b01:e34:ef40:7730:8e70:5aff:fefe:edac:/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "2b01:e34:ef40:7730:8e70:5aff:fefe:edac:",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:8080/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:8080",
			Path:   "/foo",
		},
		"",
	},
	{
		"http://[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "[2b01:e34:ef40:7730:8e70:5aff:fefe:edac]:",
			Path:   "/foo",
		},
		"",
	},
	// golang.org/issue/7991 and golang.org/issue/12719 (non-ascii %-encoded in host)
	{
		"http://hello.世界.com/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
	},
	{
		"http://hello.%e4%b8%96%e7%95%8c.com/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
	},
	{
		"http://hello.%E4%B8%96%E7%95%8C.com/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "hello.世界.com",
			Path:   "/foo",
		},
		"",
	},
	// golang.org/issue/10433 (path beginning with //)
	{
		"http://example.com//foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Path:   "//foo",
		},
		"",
	},
	// test that we can reparse the host names we accept.
	{
		"http://authority<\"hi\">/foo",
		&URL{
			Scheme: SchemeHTTP,
			Host:   "authority<\"hi\">",
			Path:   "/foo",
		},
		"",
	},
}

// more useful string for debugging than fmt's struct printer
func ufmt(u *URL) string {
	return fmt.Sprintf("opaque=%q, scheme=%q, host=%q, path=%q",
		u.Opaque, Schemes[u.Scheme], u.Host, u.Path)
}

func BenchmarkString(b *testing.B) {
	b.StopTimer()
	b.ReportAllocs()
	for _, tt := range urltests {
		var u URL
		err := u.Parse(tt.in)
		if err != nil {
			b.Errorf("Parse(%q) returned error %s", tt.in, err)
			continue
		}
		if tt.roundtrip == "" {
			continue
		}
		b.StartTimer()
		var g string
		for i := 0; i < b.N; i++ {
			g = u.String()
		}
		b.StopTimer()
		if w := tt.roundtrip; b.N > 0 && g != w {
			b.Errorf("Parse(%q).String() == %q, want %q", tt.in, g, w)
		}
	}
}

func TestParse(t *testing.T) {
	for _, tt := range urltests {
		var u URL
		err := u.Parse(tt.in)
		if err != nil {
			t.Errorf("Parse(%q) returned error %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(&u, tt.out) {
			t.Errorf("Parse(%q):\n\tgot  %v\n\twant %v\n", tt.in, ufmt(&u), ufmt(tt.out))
		}
	}
}

const pathThatLooksSchemeRelative = "//not.a.user@not.a.host/just/a/path"

var parseRequestURLTests = []struct {
	url           string
	expectedValid bool
}{
	{"http://foo.com", true},
	{"http://foo.com/", true},
	{"http://foo.com/path", true},
	{"/", true},
	{pathThatLooksSchemeRelative, true},
	{"//not.a.user@%66%6f%6f.com/just/a/path/also", true},
	{"*", true},
	{"http://192.168.0.1/", true},
	{"http://192.168.0.1:8080/", true},
	{"http://[fe80::1]/", true},
	{"http://[fe80::1]:8080/", true},

	// Tests exercising RFC 6874 compliance:
	{"http://[fe80::1%25en0]/", true},                 // with alphanum zone identifier
	{"http://[fe80::1%25en0]:8080/", true},            // with alphanum zone identifier
	{"http://[fe80::1%25%65%6e%301-._~]/", true},      // with percent-encoded+unreserved zone identifier
	{"http://[fe80::1%25%65%6e%301-._~]:8080/", true}, // with percent-encoded+unreserved zone identifier

	{"foo.html", false},
	{"../dir/", false},
	{"http://192.168.0.%31/", false},
	{"http://192.168.0.%31:8080/", false},
	{"http://[fe80::%31]/", false},
	{"http://[fe80::%31]:8080/", false},
	{"http://[fe80::%31%25en0]/", false},
	{"http://[fe80::%31%25en0]:8080/", false},

	// These two cases are valid as textual representations as
	// described in RFC 4007, but are not valid as address
	// literals with IPv6 zone identifiers in URIs as described in
	// RFC 6874.
	{"http://[fe80::1%en0]/", false},
	{"http://[fe80::1%en0]:8080/", false},
}

func TestParseRequestURI(t *testing.T) {
	for _, test := range parseRequestURLTests {
		var u URL
		err := u.ParseRequestURI(test.url)
		if test.expectedValid && err != nil {
			t.Errorf("ParseRequestURI(%q) gave err %v; want no error", test.url, err)
		} else if !test.expectedValid && err == nil {
			t.Errorf("ParseRequestURI(%q) gave nil error; want some error", test.url)
		}
	}

	var url URL
	err := url.ParseRequestURI(pathThatLooksSchemeRelative)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}
	if url.Path != pathThatLooksSchemeRelative {
		t.Errorf("ParseRequestURI path:\ngot  %q\nwant %q", url.Path, pathThatLooksSchemeRelative)
	}
}

var stringURLTests = []struct {
	url  URL
	want string
}{
	// No leading slash on path should prepend slash on String() call
	{
		url: URL{
			Scheme: SchemeHTTP,
			Host:   "www.google.com",
			Path:   "search",
		},
		want: "http://www.google.com/search",
	},
	// Relative path with first element containing ":" should be prepended with "./", golang.org/issue/17184
	{
		url: URL{
			Path: "this:that",
		},
		want: "./this:that",
	},
	// Relative path with second element containing ":" should not be prepended with "./"
	{
		url: URL{
			Path: "here/this:that",
		},
		want: "here/this:that",
	},
	// Non-relative path with first element containing ":" should not be prepended with "./"
	{
		url: URL{
			Scheme: SchemeHTTP,
			Host:   "www.google.com",
			Path:   "this:that",
		},
		want: "http://www.google.com/this:that",
	},
}

func TestURLString(t *testing.T) {
	for _, tt := range urltests {
		var u URL
		err := u.Parse(tt.in)
		if err != nil {
			t.Errorf("Parse(%q) returned error %s", tt.in, err)
			continue
		}
		expected := tt.in
		if tt.roundtrip != "" {
			expected = tt.roundtrip
		}
		s := u.String()
		if s != expected {
			t.Errorf("Parse(%q).String() == %q (expected %q)", tt.in, s, expected)
		}
	}

	for _, tt := range stringURLTests {
		if got := tt.url.String(); got != tt.want {
			t.Errorf("%+v.String() = %q; want %q", tt.url, got, tt.want)
		}
	}
}

var resolvePathTests = []struct {
	base, ref, expected string
}{
	{"a/b", ".", "/a/"},
	{"a/b", "c", "/a/c"},
	{"a/b", "..", "/"},
	{"a/", "..", "/"},
	{"a/", "../..", "/"},
	{"a/b/c", "..", "/a/"},
	{"a/b/c", "../d", "/a/d"},
	{"a/b/c", ".././d", "/a/d"},
	{"a/b", "./..", "/"},
	{"a/./b", ".", "/a/"},
	{"a/../", ".", "/"},
	{"a/.././b", "c", "/c"},
}

func TestResolvePath(t *testing.T) {
	for _, test := range resolvePathTests {
		got := resolvePath(test.base, test.ref)
		if got != test.expected {
			t.Errorf("For %q + %q got %q; expected %q", test.base, test.ref, got, test.expected)
		}
	}
}

var resolveReferenceTests = []struct {
	base, rel, expected string
}{
	// Absolute URL references
	{"http://foo.com?a=b", "https://bar.com/", "https://bar.com/"},
	{"http://foo.com/", "https://bar.com/?a=b", "https://bar.com/"},
	{"http://foo.com/", "https://bar.com/?", "https://bar.com/"},

	// Path-absolute references
	{"http://foo.com/bar", "/baz", "http://foo.com/baz"},
	{"http://foo.com/bar?a=b#f", "/baz", "http://foo.com/baz"},
	{"http://foo.com/bar?a=b", "/baz?", "http://foo.com/baz"},
	{"http://foo.com/bar?a=b", "/baz?c=d", "http://foo.com/baz"},

	// Multiple slashes
	{"http://foo.com/bar", "http://foo.com//baz", "http://foo.com//baz"},
	{"http://foo.com/bar", "http://foo.com///baz/quux", "http://foo.com///baz/quux"},

	// Scheme-relative
	{"https://foo.com/bar?a=b", "//bar.com/quux", "https://bar.com/quux"},

	// Path-relative references:

	// ... current directory
	{"http://foo.com", ".", "http://foo.com/"},
	{"http://foo.com/bar", ".", "http://foo.com/"},
	{"http://foo.com/bar/", ".", "http://foo.com/bar/"},

	// ... going down
	{"http://foo.com", "bar", "http://foo.com/bar"},
	{"http://foo.com/", "bar", "http://foo.com/bar"},
	{"http://foo.com/bar/baz", "quux", "http://foo.com/bar/quux"},

	// ... going up
	{"http://foo.com/bar/baz", "../quux", "http://foo.com/quux"},
	{"http://foo.com/bar/baz", "../../../../../quux", "http://foo.com/quux"},
	{"http://foo.com/bar", "..", "http://foo.com/"},
	{"http://foo.com/bar/baz", "./..", "http://foo.com/"},
	// ".." in the middle (issue 3560)
	{"http://foo.com/bar/baz", "quux/dotdot/../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/.././tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/./../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/././../../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/./.././../tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/dotdot/dotdot/./../../.././././tail", "http://foo.com/bar/quux/tail"},
	{"http://foo.com/bar/baz", "quux/./dotdot/../dotdot/../dot/./tail/..", "http://foo.com/bar/quux/dot/"},

	// Remove any dot-segments prior to forming the target URI.
	// http://tools.ietf.org/html/rfc3986#section-5.2.4
	{"http://foo.com/dot/./dotdot/../foo/bar", "../baz", "http://foo.com/dot/baz"},

	// Triple dot isn't special
	{"http://foo.com/bar", "...", "http://foo.com/..."},

	// Fragment
	{"http://foo.com/bar", ".#frag", "http://foo.com/"},
	{"http://example.org/", "#!$&%27()*+,;=", "http://example.org/"},

	// Paths with escaping (issue 16947).
	{"http://foo.com/foo%2fbar/", "../baz", "http://foo.com/baz"},
	{"http://foo.com/1/2%2f/3%2f4/5", "../../a/b/c", "http://foo.com/1/a/b/c"},
	{"http://foo.com/1/2/3", "./a%2f../../b/..%2fc", "http://foo.com/1/2/b/..%2fc"},
	{"http://foo.com/1/2%2f/3%2f4/5", "./a%2f../b/../c", "http://foo.com/1/2%2f/3%2f4/a%2f../c"},
	{"http://foo.com/foo%20bar/", "../baz", "http://foo.com/baz"},
	{"http://foo.com/foo", "../bar%2fbaz", "http://foo.com/bar%2fbaz"},
	{"http://foo.com/foo%2dbar/", "./baz-quux", "http://foo.com/foo%2dbar/baz-quux"},

	// RFC 3986: Normal Examples
	// http://tools.ietf.org/html/rfc3986#section-5.4.1
	{"http://a/b/c/d;p?q", "g", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "./g", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g/", "http://a/b/c/g/"},
	{"http://a/b/c/d;p?q", "/g", "http://a/g"},
	{"http://a/b/c/d;p?q", "//g", "http://g"},
	{"http://a/b/c/d;p?q", "?y", "http://a/b/c/d;p"},
	{"http://a/b/c/d;p?q", "g?y", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "#s", "http://a/b/c/d;p"},
	{"http://a/b/c/d;p?q", "g#s", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g?y#s", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", ";x", "http://a/b/c/;x"},
	{"http://a/b/c/d;p?q", "g;x", "http://a/b/c/g;x"},
	{"http://a/b/c/d;p?q", "g;x?y#s", "http://a/b/c/g;x"},
	{"http://a/b/c/d;p?q", "", "http://a/b/c/d;p"},
	{"http://a/b/c/d;p?q", ".", "http://a/b/c/"},
	{"http://a/b/c/d;p?q", "./", "http://a/b/c/"},
	{"http://a/b/c/d;p?q", "..", "http://a/b/"},
	{"http://a/b/c/d;p?q", "../", "http://a/b/"},
	{"http://a/b/c/d;p?q", "../g", "http://a/b/g"},
	{"http://a/b/c/d;p?q", "../..", "http://a/"},
	{"http://a/b/c/d;p?q", "../../", "http://a/"},
	{"http://a/b/c/d;p?q", "../../g", "http://a/g"},

	// RFC 3986: Abnormal Examples
	// http://tools.ietf.org/html/rfc3986#section-5.4.2
	{"http://a/b/c/d;p?q", "../../../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "../../../../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "/./g", "http://a/g"},
	{"http://a/b/c/d;p?q", "/../g", "http://a/g"},
	{"http://a/b/c/d;p?q", "g.", "http://a/b/c/g."},
	{"http://a/b/c/d;p?q", ".g", "http://a/b/c/.g"},
	{"http://a/b/c/d;p?q", "g..", "http://a/b/c/g.."},
	{"http://a/b/c/d;p?q", "..g", "http://a/b/c/..g"},
	{"http://a/b/c/d;p?q", "./../g", "http://a/b/g"},
	{"http://a/b/c/d;p?q", "./g/.", "http://a/b/c/g/"},
	{"http://a/b/c/d;p?q", "g/./h", "http://a/b/c/g/h"},
	{"http://a/b/c/d;p?q", "g/../h", "http://a/b/c/h"},
	{"http://a/b/c/d;p?q", "g;x=1/./y", "http://a/b/c/g;x=1/y"},
	{"http://a/b/c/d;p?q", "g;x=1/../y", "http://a/b/c/y"},
	{"http://a/b/c/d;p?q", "g?y/./x", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g?y/../x", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g#s/./x", "http://a/b/c/g"},
	{"http://a/b/c/d;p?q", "g#s/../x", "http://a/b/c/g"},

	// Extras.
	{"https://a/b/c/d;p?q", "//g?q", "https://g"},
	{"https://a/b/c/d;p?q", "//g#s", "https://g"},
	{"https://a/b/c/d;p?q", "//g/d/e/f?y#s", "https://g/d/e/f"},
	{"https://a/b/c/d;p#s", "?y", "https://a/b/c/d;p"},
	{"https://a/b/c/d;p?q#s", "?y", "https://a/b/c/d;p"},
}

func TestResolveReference(t *testing.T) {
	mustParse := func(url string) *URL {
		u := new(URL)
		err := u.Parse(url)
		if err != nil {
			t.Fatalf("Parse(%q) got err %v", url, err)
		}
		return u
	}
	for _, test := range resolveReferenceTests {
		base := mustParse(test.base)
		rel := mustParse(test.rel)
		var url URL
		base.ResolveReference(&url, rel)
		if got := url.String(); got != test.expected {
			t.Errorf("URL(%q).ResolveReference(%q)\ngot  %q\nwant %q", test.base, test.rel, got, test.expected)
		}
	}
}

type RequestURITest struct {
	url *URL
	out string
}

var requritests = []RequestURITest{
	{
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Path:   "",
		},
		"/",
	},
	{
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Path:   "/a b",
		},
		"/a%20b",
	},
	// golang.org/issue/4860 variant 1
	{
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Opaque: "/%2F/%2F/",
		},
		"/%2F/%2F/",
	},
	// golang.org/issue/4860 variant 2
	{
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Opaque: "//other.example.com/%2F/%2F/",
		},
		"http://other.example.com/%2F/%2F/",
	},
	{
		&URL{
			Scheme: SchemeHTTP,
			Opaque: "opaque",
		},
		"opaque",
	},
	{
		&URL{
			Scheme: SchemeHTTP,
			Host:   "example.com",
			Path:   "//foo",
		},
		"//foo",
	},
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		in      string
		wantErr bool
	}{
		{"http://[::1]", false},
		{"http://[::1]:80", false},
		{"http://[::1]:namedport", true}, // rfc3986 3.2.3
		{"http://[::1]/", false},
		{"http://[::1]a", true},
		{"http://[::1]%23", true},
		{"http://[::1%25en0]", false},     // valid zone id
		{"http://[::1]:", false},          // colon, but no port OK
		{"http://[::1]:%38%30", true},     // not allowed: % encoding only for non-ASCII
		{"http://[::1%25%41]", false},     // RFC 6874 allows over-escaping in zone
		{"http://[%10::1]", true},         // no %xx escapes in IP address
		{"http://[::1]/%48", false},       // %xx in path is fine
		{"http://%41:8080/", true},        // not allowed: % encoding only for non-ASCII

		{"http://[]%20%48%54%54%50%2f%31%2e%31%0a%4d%79%48%65%61%64%65%72%3a%20%31%32%33%0a%0a/", true}, // golang.org/issue/11208
		{"http://a b.com/", true},    // no space in host name please
	}
	for _, tt := range tests {
		var u URL
		err := u.Parse(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("Parse(%q) = %#v; want an error", tt.in, u)
			}
			continue
		}
		if err != nil {
			t.Logf("Parse(%q) = %v; want no error", tt.in, err)
		}
	}
}

type shouldEscapeTest struct {
	in     byte
	mode   encoding
	escape bool
}

var shouldEscapeTests = []shouldEscapeTest{
	// Unreserved characters (§2.3)
	{'a', encodePath, false},
	{'a', encodeUserPassword, false},
	{'a', encodeQueryComponent, false},
	{'a', encodeFragment, false},
	{'a', encodeHost, false},
	{'z', encodePath, false},
	{'A', encodePath, false},
	{'Z', encodePath, false},
	{'0', encodePath, false},
	{'9', encodePath, false},
	{'-', encodePath, false},
	{'-', encodeUserPassword, false},
	{'-', encodeQueryComponent, false},
	{'-', encodeFragment, false},
	{'.', encodePath, false},
	{'_', encodePath, false},
	{'~', encodePath, false},

	// User information (§3.2.1)
	{':', encodeUserPassword, true},
	{'/', encodeUserPassword, true},
	{'?', encodeUserPassword, true},
	{'@', encodeUserPassword, true},
	{'$', encodeUserPassword, false},
	{'&', encodeUserPassword, false},
	{'+', encodeUserPassword, false},
	{',', encodeUserPassword, false},
	{';', encodeUserPassword, false},
	{'=', encodeUserPassword, false},

	// Host (IP address, IPv6 address, registered name, port suffix; §3.2.2)
	{'!', encodeHost, false},
	{'$', encodeHost, false},
	{'&', encodeHost, false},
	{'\'', encodeHost, false},
	{'(', encodeHost, false},
	{')', encodeHost, false},
	{'*', encodeHost, false},
	{'+', encodeHost, false},
	{',', encodeHost, false},
	{';', encodeHost, false},
	{'=', encodeHost, false},
	{':', encodeHost, false},
	{'[', encodeHost, false},
	{']', encodeHost, false},
	{'0', encodeHost, false},
	{'9', encodeHost, false},
	{'A', encodeHost, false},
	{'z', encodeHost, false},
	{'_', encodeHost, false},
	{'-', encodeHost, false},
	{'.', encodeHost, false},
}

func TestShouldEscape(t *testing.T) {
	for _, tt := range shouldEscapeTests {
		if shouldEscape(tt.in, tt.mode) != tt.escape {
			t.Errorf("shouldEscape(%q, %v) returned %v; expected %v", tt.in, tt.mode, !tt.escape, tt.escape)
		}
	}
}

type timeoutError struct {
	timeout bool
}

func (e *timeoutError) Error() string { return "timeout error" }
func (e *timeoutError) Timeout() bool { return e.timeout }

type temporaryError struct {
	temporary bool
}

func (e *temporaryError) Error() string   { return "temporary error" }
func (e *temporaryError) Temporary() bool { return e.temporary }

type timeoutTemporaryError struct {
	timeoutError
	temporaryError
}

func (e *timeoutTemporaryError) Error() string { return "timeout/temporary error" }

var netErrorTests = []struct {
	err       error
	timeout   bool
	temporary bool
}{{
	err:       &Error{"Get", "http://google.com/", &timeoutError{timeout: true}},
	timeout:   true,
	temporary: false,
}, {
	err:       &Error{"Get", "http://google.com/", &timeoutError{timeout: false}},
	timeout:   false,
	temporary: false,
}, {
	err:       &Error{"Get", "http://google.com/", &temporaryError{temporary: true}},
	timeout:   false,
	temporary: true,
}, {
	err:       &Error{"Get", "http://google.com/", &temporaryError{temporary: false}},
	timeout:   false,
	temporary: false,
}, {
	err:       &Error{"Get", "http://google.com/", &timeoutTemporaryError{timeoutError{timeout: true}, temporaryError{temporary: true}}},
	timeout:   true,
	temporary: true,
}, {
	err:       &Error{"Get", "http://google.com/", &timeoutTemporaryError{timeoutError{timeout: false}, temporaryError{temporary: true}}},
	timeout:   false,
	temporary: true,
}, {
	err:       &Error{"Get", "http://google.com/", &timeoutTemporaryError{timeoutError{timeout: true}, temporaryError{temporary: false}}},
	timeout:   true,
	temporary: false,
}, {
	err:       &Error{"Get", "http://google.com/", &timeoutTemporaryError{timeoutError{timeout: false}, temporaryError{temporary: false}}},
	timeout:   false,
	temporary: false,
}, {
	err:       &Error{"Get", "http://google.com/", io.EOF},
	timeout:   false,
	temporary: false,
}}

// Test that url.Error implements net.Error and that it forwards
func TestURLErrorImplementsNetError(t *testing.T) {
	for i, tt := range netErrorTests {
		err, ok := tt.err.(net.Error)
		if !ok {
			t.Errorf("%d: %T does not implement net.Error", i+1, tt.err)
			continue
		}
		if err.Timeout() != tt.timeout {
			t.Errorf("%d: err.Timeout(): got %v, want %v", i+1, err.Timeout(), tt.timeout)
			continue
		}
		if err.Temporary() != tt.temporary {
			t.Errorf("%d: err.Temporary(): got %v, want %v", i+1, err.Temporary(), tt.temporary)
		}
	}
}

var _ encodingPkg.BinaryMarshaler = (*URL)(nil)
var _ encodingPkg.BinaryUnmarshaler = (*URL)(nil)

func TestJSON(t *testing.T) {
	var u URL
	err := u.Parse("https://www.google.com/x?y=z")
	if err != nil {
		t.Fatal(err)
	}
	js, err := json.Marshal(&u)
	if err != nil {
		t.Fatal(err)
	}

	// If only we could implement TextMarshaler/TextUnmarshaler,
	// this would work:
	//
	// if string(js) != strconv.Quote(u.String()) {
	// 	t.Errorf("json encoding: %s\nwant: %s\n", js, strconv.Quote(u.String()))
	// }

	u1 := new(URL)
	err = json.Unmarshal(js, u1)
	if err != nil {
		t.Fatal(err)
	}
	if u1.String() != u.String() {
		t.Errorf("json decoded to: %s\nwant: %s\n", u1, &u)
	}
}

func TestGob(t *testing.T) {
	var u URL
	err := u.Parse("https://www.google.com/x?y=z")
	if err != nil {
		t.Fatal(err)
	}
	var w bytes.Buffer
	err = gob.NewEncoder(&w).Encode(&u)
	if err != nil {
		t.Fatal(err)
	}

	u1 := new(URL)
	err = gob.NewDecoder(&w).Decode(u1)
	if err != nil {
		t.Fatal(err)
	}
	if u1.String() != u.String() {
		t.Errorf("json decoded to: %s\nwant: %s\n", u1, &u)
	}
}
