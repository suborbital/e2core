package fqfn

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////////
// An FQFN (fully-qualified function name) is a "globally unique"
// name for a specific function from a specific application hash
// example: fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api-users/add-user
// i.e. fqfn://<identifier>/<hash>/<namespace>/<funcname>
//
// These URI forms are also supported:
//
// 		/api/users/add-user
// 		(single-application function)
//
// 		/ref/f0e4c2f76c58916ec258f246851be/api/users/add-user
// 		(reference to a version of a single-application function)
//
// 		/com.suborbital.acmeco/api/users/add-user
// 		(multi-application, single-domain function)
//
// 		/ref/f0e4c2f76c58916ec258f246851be/com.suborbital.acmeco/api/users/add-user
// 		(reference to a version of a multi-application, single-domain function)
//
// Additionally, a URL form assumes the function identifier is the reverse domain of
// the URL, but otherwise is the same as the URI form.
// example: https://acmeco.suborbital.com/api-users/add-user
////////////////////////////////////////////////////////////////////////////////////

// NamespaceDefault and others represent conts for namespaces.
const (
	NamespaceDefault = "default"
)

// FQFN is a parsed fqfn.
type FQFN struct {
	Identifier string `json:"identifier"`
	Namespace  string `json:"namespace"`
	Fn         string `json:"fn"`
	Hash       string `json:"hash"`
}

var errWrongPrefix = errors.New("FQFN must begin with 'fqfn://' or '/'")
var errMustBeFullyQualified = errors.New("FQFN text format must contain an identifier, hash, namespace, and function name")
var errTooFewParts = errors.New("FQFN must contain at least a namespace and function name")
var errMalformedIdentifier = errors.New("identifier must contain exactly two dots")
var errTrailingSlash = errors.New("FQFN must not end in a trailing slash")

func Parse(fqfnString string) (FQFN, error) {
	if strings.HasPrefix(fqfnString, "fqfn://") {
		return parseTextFormat(fqfnString)
	}

	if strings.HasPrefix(fqfnString, "/") {
		return parseUriFormat(fqfnString)
	}

	return FQFN{}, errWrongPrefix
}

func parseTextFormat(fqfnString string) (FQFN, error) {
	fqfnString = strings.TrimPrefix(fqfnString, "fqfn://")

	segments := strings.Split(fqfnString, "/")

	// There should be at least four segments representing the ident, hash, namespace, and name.
	// Additional segments would be the result of multi-level namespaces.
	if len(segments) < 4 {
		return FQFN{}, errMustBeFullyQualified
	}

	// If the last segment is empty, there was a trailing slash
	if segments[len(segments)-1] == "" {
		return FQFN{}, errTrailingSlash
	}

	identifier := segments[0]

	if strings.Count(identifier, ".") != 2 {
		return FQFN{}, errMalformedIdentifier
	}

	hash := segments[1]

	// Reconstruct the namespace
	namespace := strings.Join(segments[2:len(segments)-1], "/")

	// The function name is just the last element
	fn := segments[len(segments)-1]

	fqfn := FQFN{
		Identifier: identifier,
		Namespace:  namespace,
		Fn:         fn,
		Hash:       hash,
	}

	return fqfn, nil
}

func parseUriFormat(fqfnString string) (FQFN, error) {
	segments := strings.Split(fqfnString, "/")
	// The first segment will be empty since the string starts with '/'
	segments = segments[1:]

	// There should be at least two segments
	if len(segments) < 2 {
		return FQFN{}, errTooFewParts
	}

	// If the last segment is empty, there was a trailing slash
	if segments[len(segments)-1] == "" {
		return FQFN{}, errTrailingSlash
	}

	// Check for a hash
	var hash string
	if segments[0] == "ref" {
		hash = segments[1]
		segments = segments[2:]

		// There should be at least two more segments
		if len(segments) < 2 {
			return FQFN{}, errTooFewParts
		}
	}

	// Check for an identifier
	var identifier string
	if strings.Count(segments[0], ".") == 2 {
		identifier = segments[0]
		segments = segments[1:]

		// There _still_ should be at least two more segments
		if len(segments) < 2 {
			return FQFN{}, errTooFewParts
		}
	}

	// Reconstruct the namespace
	namespace := strings.Join(segments[:len(segments)-1], "/")

	// The function name is just the last element
	fn := segments[len(segments)-1]

	fqfn := FQFN{
		Identifier: identifier,
		Namespace:  namespace,
		Fn:         fn,
		Hash:       hash,
	}

	return fqfn, nil
}

// HeadlessURLPath returns the headless URL path for a function.
func (f FQFN) HeadlessURLPath() string {
	return fmt.Sprintf("/%s/%s/%s/%s", f.Identifier, f.Namespace, f.Fn, f.Hash)
}

func FromParts(ident, namespace, fn, hash string) string {
	return fmt.Sprintf("fqfn://%s/%s/%s/%s", ident, hash, namespace, fn)
}

func FromURL(u *url.URL) (string, error) {
	fqfn, err := parseUriFormat(u.Path)
	if err != nil {
		return "", err
	}

	identParts := strings.Split(u.Host, ".")
	if len(identParts) != 3 {
		return "", errMalformedIdentifier
	}

	ident := strings.Join([]string{identParts[2], identParts[1], identParts[0]}, ".")

	return FromParts(ident, fqfn.Namespace, fqfn.Fn, fqfn.Hash), nil
}
