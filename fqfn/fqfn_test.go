package fqfn

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type FQFNSuite struct {
	suite.Suite
}

func TestFQFNSuite(t *testing.T) {
	suite.Run(t, &FQFNSuite{})
}

func (s *FQFNSuite) TestParse() {
	for _, tt := range []struct {
		name string
		text string
		fqfn FQFN
		error
	}{
		{"fully-qualified single-level namespace", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api-users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api-users",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"fully-qualified two-level namespace", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api/users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"fully-qualified multi-level namespace", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api/users/auora/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users/auora",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for single-application func with single-level namespace", "/api-users/add-user", FQFN{
			Namespace: "api-users",
			Fn:        "add-user",
		}, nil},
		{"uri for single-application func with two-level namespace", "/api/users/add-user", FQFN{
			Namespace: "api/users",
			Fn:        "add-user",
		}, nil},
		{"uri for single-application func with multi-level namespace", "/api/users/auora/add-user", FQFN{
			Namespace: "api/users/auora",
			Fn:        "add-user",
		}, nil},
		{"uri for versioned single-application func with single-level namespace", "/ref/98qhrfgo3089hafrouhqf48/api-users/add-user", FQFN{
			Namespace: "api-users",
			Fn:        "add-user",
			Hash:      "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for versioned single-application func with two-level namespace", "/ref/98qhrfgo3089hafrouhqf48/api/users/add-user", FQFN{
			Namespace: "api/users",
			Fn:        "add-user",
			Hash:      "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for versioned single-application func with multi-level namespace", "/ref/98qhrfgo3089hafrouhqf48/api/users/auora/add-user", FQFN{
			Namespace: "api/users/auora",
			Fn:        "add-user",
			Hash:      "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for multi-application func with single-level namespace", "/com.suborbital.acmeco/api-users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api-users",
			Fn:         "add-user",
		}, nil},
		{"uri for multi-application func with two-level namespace", "/com.suborbital.acmeco/api/users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users",
			Fn:         "add-user",
		}, nil},
		{"uri for multi-application func with multi-level namespace", "/com.suborbital.acmeco/api/users/auora/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users/auora",
			Fn:         "add-user",
		}, nil},
		{"uri for versioned multi-application func with single-level namespace", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco/api-users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api-users",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for versioned multi-application func with two-level namespace", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco/api/users/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"uri for versioned multi-application func with multi-level namespace", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco/api/users/auora/add-user", FQFN{
			Identifier: "com.suborbital.acmeco",
			Namespace:  "api/users/auora",
			Fn:         "add-user",
			Hash:       "98qhrfgo3089hafrouhqf48",
		}, nil},
		{"malformed—doesn't start with right prefix 1", "fqfn:com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api/users/auora/add-user", FQFN{}, errWrongPrefix},
		{"malformed—doesn't start with right prefix 2", "https://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api/users/auora/add-user", FQFN{}, errWrongPrefix},
		{"malformed—malformed identifier", "fqfn://com-suborbital-acmeco/98qhrfgo3089hafrouhqf48/api/users/auora/add-user", FQFN{}, errMalformedIdentifier},
		{"malformed—not fully-qualified 1", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48", FQFN{}, errMustBeFullyQualified},
		{"malformed—not fully-qualified 2", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/add-user", FQFN{}, errMustBeFullyQualified},
		{"malformed—not enough parts 1", "/add-user", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 2", "/com.suborbital.acmeco", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 3", "/com.suborbital.acmeco/add-user", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 4", "/ref/98qhrfgo3089hafrouhqf48", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 5", "/ref/98qhrfgo3089hafrouhqf48/add-user", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 6", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco", FQFN{}, errTooFewParts},
		{"malformed—not enough parts 7", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco/add-user", FQFN{}, errTooFewParts},
		{"malformed—trailing slash 1", "fqfn://com.suborbital.acmeco/98qhrfgo3089hafrouhqf48/api/users/add-user/", FQFN{}, errTrailingSlash},
		{"malformed—trailing slash 2", "/ref/98qhrfgo3089hafrouhqf48/com.suborbital.acmeco/api/users/auora/add-user/", FQFN{}, errTrailingSlash},
	} {
		s.Run(tt.name, func() {
			fqfn, err := Parse(tt.text)

			if err != nil {
				s.Assertions.ErrorIs(err, tt.error)
				return
			}

			s.Assertions.Equal(fqfn, tt.fqfn)
		})
	}
}
