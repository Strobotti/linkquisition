# uri
![Lint](https://github.com/fredbi/uri/actions/workflows/01-golang-lint.yaml/badge.svg)
![CI](https://github.com/fredbi/uri/actions/workflows/02-test.yaml/badge.svg)
[![Coverage Status](https://coveralls.io/repos/github/fredbi/uri/badge.svg?branch=master)](https://coveralls.io/github/fredbi/uri?branch=master)
![Vulnerability Check](https://github.com/fredbi/uri/actions/workflows/03-govulncheck.yaml/badge.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/fredbi/uri)](https://goreportcard.com/report/github.com/fredbi/uri)

![GitHub tag (latest by date)](https://img.shields.io/github/v/tag/fredbi/uri)
[![Go Reference](https://pkg.go.dev/badge/github.com/fredbi/uri.svg)](https://pkg.go.dev/github.com/fredbi/uri)
[![license](http://img.shields.io/badge/license/License-MIT-yellow.svg)](https://raw.githubusercontent.com/fredbi/uri/master/LICENSE.md)


Package `uri` is meant to be an RFC 3986 compliant URI builder, parser and validator for `golang`.

It supports strict RFC validation for URIs and URI relative references.

This allows for stricter conformance than the `net/url` package in the `go` standard libary,
which provides a workable but loose implementation of the RFC for URLs.

**Requires go1.19**.

## What's new?

### V1.2 announcement

To do before I cut a v1.2.0:
* [] handle empty fragment, empty query.
  Ex: `https://host?` is not equivalent to `http://host`.
  Similarly `https://host#` is not equivalent to `http://host`.
* [] IRI UCS charset compliance
* [] URI normalization (like [PuerkitoBio/purell](https://github.com/PuerkitoBio/purell))
* [] more explicit errors, with context

See also [TODOs](./docs/TODO.md).

### V2 announcement

V2 is getting closer to completion. It comes with:
* very significant performance improvement (x 1.5).
  Eventually `uri` gets significantly faster than `net/url` (-50% ns/op)
* a simplified API: no interface, no `Validate()`, no `Builder()`
* options for tuning validation strictness
* exported package level variables disappear

### Current master (unreleased)

**Fixes**

* stricter scheme validation (no longer support non-ASCII letters).
  Ex: `Smørrebrød://` is not a valid scheme.
* stricter IP validation (do not support escaping in addresses, excepted for IPv6 zones)
* stricter percent-escape validation: an escaped character **MUST** decode to a valid UTF8 endpoint (1).
  Ex: %C3 is an incomplete escaped UTF8 sequence. Should be %C3%B8 to escape the full UTF8 rune.
* stricter port validation. A port is an integer less than or equal to 65535.

> (1)
> `go` natively manipulates UTF8 strings only. Even though the standards are not strict about the character
>  encoding of escaped sequences, it seems natural to prevent invalid UTF8 to propagate via percent escaping.
>  Notice that this approach is not the one followed by `net/url.PathUnescape()`, which leaves invalid runes.

**Features**

* feat: added `IsIP()` bool and `IPAddr() netip.Addr` methods

**Performances**

* perf: slight improvement. Now only 8-25% slower than net/url.Parse, depending on the workload

### [Older releases](#release-notes)

## Usage

### Parsing

```go
	u, err := Parse("https://example.com:8080/path")
	if err != nil {
		fmt.Printf("Invalid URI")
	} else {
		fmt.Printf("%s", u.Scheme())
	}
	// Output: https
```

```go
	u, err := ParseReference("//example.com/path")
	if err != nil {
		fmt.Printf("Invalid URI reference")
	} else {
		fmt.Printf("%s", u.Authority().Path())
	}
	// Output: /path
```

### Validating

```go
    isValid := IsURI("urn://example.com?query=x#fragment/path") // true

    isValid= IsURI("//example.com?query=x#fragment/path") // false

    isValid= IsURIReference("//example.com?query=x#fragment/path") // true
```

#### Caveats

* **Registered name vs DNS name**: RFC3986 defines a super-permissive "registered name" for hosts, for URIs
  not specifically related to an Internet name. Our validation performs a stricter host validation according
  to DNS rules whenever the scheme is a well-known IANA-registered scheme
  (the function `UsesDNSHostValidation(string) bool` is customizable).

> Examples:
> `ftp://host`, `http://host` default to validating a proper DNS hostname.

* **IPv6 validation** relies on IP parsing from the standard library. It is not super strict
  regarding the full-fledged IPv6 specification, but abides by the URI RFC's.

* **URI vs URL**: every URL should be a URI, but the converse does not always hold. This module intends to perform
  stricter validation than the pragmatic standard library `net/url`, which currently remains about 30% faster.

* **URI vs IRI**: at this moment, this module checks for URI, while supporting unicode letters as `ALPHA` tokens.
  This is not strictly compliant with the IRI specification (see known issues).

### Building

The exposed type `URI` can be transformed into a fluent `Builder` to set the parts of an URI.

```go
	aURI, _ := Parse("mailto://user@domain.com")
	newURI := auri.Builder().SetUserInfo(test.name).SetHost("newdomain.com").SetScheme("http").SetPort("443")
```

### Canonicalization

Not supported for now (contemplated as a topic for V2).

For URL normalization, see [PuerkitoBio/purell](https://github.com/PuerkitoBio/purell).

## Reference specifications

The librarian's corner (still WIP).

|Title|Reference|Notes|
|---------------------------------------------|-------------------------------------------------------|----------------|
| Uniform Resource Identifier (URI)           | [RFC3986](https://www.rfc-editor.org/rfc/rfc3986)     | (1)(2) |
| Uniform Resource Locator (URL)              | [RFC1738](https://www.rfc-editor.org/info/rfc1738)    | |
| Relative URL                                | [RFC1808](https://www.rfc-editor.org/info/rfc1808)    | |
| Internationalized Resource Identifier (IRI) | [RFC3987](https://tools.ietf.org/html/rfc3987)        | (1) |
| Practical standardization guidelines        | [URL WhatWG Living Standard](https://url.spec.whatwg.org/) |(4)|
| Domain names implementation                 | [RFC1035](https://datatracker.ietf.org/doc/html/rfc1035) ||
||||
| **IPv6** |||
| Representing IPv6 Zone Identifiers          | [RFC6874](https://www.rfc-editor.org/rfc/rfc6874.txt) |      | |
| IPv6 Addressing architecture                | [RFC3513](https://www.rfc-editor.org/rfc/rfc3513.txt) ||
| **URI Schemes** |||
||||
| IANA registered URI schemes                 | [IANA](https://www.iana.org/assignments/uri-schemes/uri-schemes.xhtml) | (5) | 
||||
| **Port numbers** |||
| IANA port assignments by service            | [IANA](https://www.iana.org/assignments/service-names-port-numbers/service-names-port-numbers.txt) ||
| Well-known TCP and UDP port numbers         | [Wikipedia)(https://en.wikipedia.org/wiki/List_of_TCP_and_UDP_port_numbers) ||
| 

**Notes**

(1) Deviations from the RFC:
* Tokens: ALPHAs are tolerated to be Unicode Letter codepoints. Schemes remain constrained to ASCII letters (`[a-z]|[A-Z]`)
* DIGITs are ASCII digits as required by the RFC. Unicode Digit codepoints are rejected (ex: ६ (6), ① , 六 (6), Ⅶ (7) are not considered legit URI DIGITS).

> Some improvements are still needed to abide more strictly to IRI's provisions for internationalization. Working on it...

(2) Percent-escape:
* Escape sequences, e.g. `%hh` _must_ decode to valid UTF8 runes (standard says _should_).

(2) IP addresses:
* As per RFC3986, `[hh::...]` literals _must_ be IPv6 and `ddd.ddd.ddd.ddd` litterals _must_ be IPv4.
* As per RFC3986, notice that `[]` is illegal, although the golang IP parser translates this to `[::]` (zero value IP).
  In `go`, the zero value for `netip.Addr` is invalid just a well.
* IPv6 zones are supported, with the '%' escaped as '%25' to denote an IPv6 zoneID (RFC6974)
* IPvFuture addresses _are_ supported, with escape sequences (which are not part of RFC3986, but natural since IPv6 do support escaping)

(4) Deviations from the WhatWG recommendation
* `[]` IPv6 address is invalid
* invalid percent-encoded characters trigger an error rather than being ignored

(5) Most _permanently_ registered schemes have been accounted for when checking whether Domain Names apply for hosts rather than the
    "registered name" from RFC3986. Quite a few commonly used found, either unregistered or with a provisional status have been added as well.
    Feel free to create an issue or contribute a change to enrich this list of well-known URI schemes.

## [FAQ](docs/FAQ.md)

## [Benchmarks](docs/BENCHMARKS.md)

## Credits

* Tests have been aggregated from the  test suites of URI validators from other languages:
Perl, Python, Scala, .Net. and the Go url standard library.

* This package was initially based on the work from ttacon/uri (credits: Trey Tacon).
> Extra features like MySQL URIs present in the original repo have been removed.

* A lot of improvements and suggestions have been brought by the incredible guys at [`fyne-io`](https://github.com/fyne-io). Thanks all.

## Release notes

### v1.1.0

**Build**

* requires go1.19

**Features**

* Typed errors: parsing and validation now returns errors of type `uri.Error`,
  with a more accurate pinpointing of the error provided by the value.
  Errors support the go1.20 addition to standard errors with `Join()` and `Cause()`.
  For go1.19, backward compatibility is ensured (errors.Join() is emulated).
* DNS schemes can be overridden at the package level

**Performances**

* Significantly improved parsing speed by dropping usage of regular expressions and reducing allocations (~ x20 faster).

**Fixes**

* stricter compliance regarding paths beginning with a double '/'
* stricter compliance regarding the length of DNS names and their segments
* stricter compliance regarding IPv6 addresses with an empty zone
* stricter compliance regarding IPv6 vs IPv4 litterals
* an empty IPv6 litteral `[]` is invalid

**Known open issues**

* IRI validation lacks strictness
* IPv6 validation relies on the standard library and lacks strictness

**Other**

Major refactoring to enhance code readability, esp. for testing code.

* Refactored validations
* Refactored test suite
* Added support for fuzzing, dependabots & codeQL scans

