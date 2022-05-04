package directive

import (
	"fmt"
	"testing"

	"github.com/suborbital/velocity/directive/executable"
)

func TestYAMLMarshalUnmarshal(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []executable.Executable{
					{
						Group: []executable.CallableFn{
							{
								Fn: "db::getUser",
							},
							{
								Fn: "db::getUserDetails",
							},
						},
					},
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
						},
					},
				},
			},
		},
	}

	yamlBytes, err := dir.Marshal()
	if err != nil {
		t.Error(err)
		return
	}

	dir2 := Directive{}
	if err := dir2.Unmarshal(yamlBytes); err != nil {
		t.Error(err)
		return
	}

	if err := dir2.Validate(); err != nil {
		t.Error(err)
	}

	if len(dir2.Handlers[0].Steps) != 2 {
		t.Error("wrong number of steps")
		return
	}

	if len(dir2.Runnables) != 3 {
		t.Error("wrong number of steps")
		return
	}
}

func TestDirectiveValidatorGroupLast(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
						},
					},
					{
						Group: []executable.CallableFn{
							{
								Fn: "db::getUser",
							},
							{
								Fn: "db::getUserDetails",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}

func TestDirectiveValidatorInvalidOnErr(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Code: map[int]string{
									400: "continue",
								},
								Any: "return",
							},
						},
					},
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Other: "continue",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}

func TestDirectiveValidatorDuplicateParameterizedResourceMethod(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/:hello/world",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Any: "continue",
							},
						},
					},
				},
			},
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/:goodbye/moon",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Any: "continue",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}

func TestDirectiveValidatorDuplicateResourceMethod(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/hello",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Any: "continue",
							},
						},
					},
				},
			},
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/hello",
				},
				Steps: []executable.Executable{
					{
						CallableFn: executable.CallableFn{
							Fn: "api::returnUser",
							OnErr: &executable.ErrHandler{
								Any: "continue",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}

func TestDirectiveValidatorMissingFns(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []executable.Executable{
					{
						Group: []executable.CallableFn{
							{
								Fn: "getUser",
							},
							{
								Fn: "getFoobar",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}

func TestDirectiveFQFNs(t *testing.T) {
	dir := &Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "default",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
	}

	if err := dir.Validate(); err != nil {
		t.Error("failed to Validate directive")
		return
	}

	run1 := dir.FindRunnable("getUser")
	if run1 == nil {
		t.Error("failed to find Runnable for getUser")
		return
	}

	fqfn1 := dir.fqfnForFunc(run1.Namespace, run1.Name)

	if fqfn1 != "dev.suborbital.appname#default::getUser@v0.1.1" {
		t.Error("fqfn1 should be 'dev.suborbital.appname#default::getUser@v0.1.1', got", fqfn1)
	}

	if fqfn1 != run1.FQFN {
		t.Errorf("fqfn1 %q did not match run1.FQFN %q", fqfn1, run1.FQFN)
	}

	run2 := dir.FindRunnable("db::getUserDetails")
	if run2 == nil {
		t.Error("failed to find Runnable for db::getUserDetails")
		return
	}

	fqfn2 := dir.fqfnForFunc(run2.Namespace, run2.Name)

	if fqfn2 != "dev.suborbital.appname#db::getUserDetails@v0.1.1" {
		t.Error("fqfn2 should be 'dev.suborbital.appname#db::getUserDetails@v0.1.1', got", fqfn2)
	}

	if fqfn2 != run2.FQFN {
		t.Error("fqfn2 did not match run2.FQFN")
	}

	run3 := dir.FindRunnable("api::returnUser")
	if run3 == nil {
		t.Error("failed to find Runnable for api::returnUser")
		return
	}

	fqfn3 := dir.fqfnForFunc(run3.Namespace, run3.Name)

	if fqfn3 != "dev.suborbital.appname#api::returnUser@v0.1.1" {
		t.Error("fqfn3 should be 'dev.suborbital.appname#api::returnUser@v0.1.1', got", fqfn3)
	}

	if fqfn3 != run3.FQFN {
		t.Error("fqfn1 did not match run1.FQFN")
	}

	run4 := dir.FindRunnable("dev.suborbital.appname#api::returnUser@v0.1.1")
	if run4 == nil {
		t.Error("failed to find Runnable for dev.suborbital.appname#api::returnUser@v0.1.1")
		return
	}

	fqfn4 := dir.fqfnForFunc(run3.Namespace, run3.Name)

	if fqfn4 != "dev.suborbital.appname#api::returnUser@v0.1.1" {
		t.Error("fqfn4 should be 'dev.suborbital.appname#api::returnUser@v0.1.1', got", fqfn3)
	}

	if fqfn4 != run4.FQFN {
		t.Error("fqfn1 did not match run1.FQFN")
	}

	run5 := dir.FindRunnable("foo::bar")
	if run5 != nil {
		t.Error("should not have found a Runnable for foo::bar")
	}
}

func TestDirectiveValidatorWithMissingState(t *testing.T) {
	dir := Directive{
		Identifier:  "dev.suborbital.appname",
		AppVersion:  "v0.1.1",
		AtmoVersion: "v0.0.6",
		Runnables: []Runnable{
			{
				Name:      "getUser",
				Namespace: "db",
			},
			{
				Name:      "getUserDetails",
				Namespace: "db",
			},
			{
				Name:      "returnUser",
				Namespace: "api",
			},
		},
		Handlers: []Handler{
			{
				Input: Input{
					Type:     "request",
					Method:   "GET",
					Resource: "/api/v1/user",
				},
				Steps: []executable.Executable{
					{
						Group: []executable.CallableFn{
							{
								Fn: "getUser",
								With: map[string]string{
									"data": "someData",
								},
							},
							{
								Fn: "getFoobar",
							},
						},
					},
				},
			},
		},
	}

	if err := dir.Validate(); err == nil {
		t.Error("directive validation should have failed")
	} else {
		fmt.Println("directive validation properly failed:", err)
	}
}
