package fqfn

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

////////////////////////////////////////////////////////////////////
// An FQFN (fully-qualified function name) is a 'globally unique'
// name for a specific function from a specific application version
// example: com.suborbital.test#default::get-file@v0.0.1
// i.e. identifier # namespace :: name @ version
////////////////////////////////////////////////////////////////////

// NamespaceDefault and others represent conts for namespaces.
const (
	NamespaceDefault = "default"
)

// FQFN is a parsed fqfn.
type FQFN struct {
	Identifier string `json:"identifier"`
	Namespace  string `json:"namespace"`
	Fn         string `json:"fn"`
	Version    string `json:"version"`
}

func Parse(name string) FQFN {
	// if the name contains a #, parse that out as the identifier.
	identifier := ""
	identParts := strings.SplitN(name, "#", 2)
	if len(identParts) == 2 {
		identifier = identParts[0]
		name = identParts[1]
	}

	// if a Runnable is referenced with its namespace, i.e. users#getUser
	// then we need to parse that and ensure we only match that namespace.

	namespace := NamespaceDefault
	namespaceParts := strings.SplitN(name, "::", 2)
	if len(namespaceParts) == 2 {
		namespace = namespaceParts[0]
		name = namespaceParts[1]
	}

	// next, if the name contains an @, parse the app version.
	appVersion := ""
	versionParts := strings.SplitN(name, "@", 2)
	if len(versionParts) == 2 {
		name = versionParts[0]
		appVersion = versionParts[1]
	}

	fqfn := FQFN{
		Identifier: identifier,
		Namespace:  namespace,
		Fn:         name,
		Version:    appVersion,
	}

	return fqfn
}

// HeadlessURLPath returns the headless URL path for a function.
func (f FQFN) HeadlessURLPath() string {
	return fmt.Sprintf("/%s/%s/%s/%s", f.Identifier, f.Namespace, f.Fn, f.Version)
}

func FromParts(ident, namespace, fn, version string) string {
	return fmt.Sprintf("%s#%s::%s@%s", ident, namespace, fn, version)
}

func FromURL(u *url.URL) (string, error) {
	path := strings.TrimPrefix(u.Path, "/")

	parts := strings.Split(path, "/")
	if len(parts) != 4 {
		return "", errors.New("path is not an FQFN")
	}

	ident := parts[0]
	namespace := parts[1]
	fn := parts[2]
	version := parts[3]

	fqfn := FromParts(ident, namespace, fn, version)

	return fqfn, nil
}
