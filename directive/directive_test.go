package directive

import (
	"fmt"
	"testing"
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
				Steps: []Executable{
					{
						Group: []CallableFn{
							{
								Fn: "db#getUser",
							},
							{
								Fn: "db#getUserDetails",
							},
						},
					},
					{
						CallableFn: CallableFn{
							Fn: "api#returnUser",
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
				Steps: []Executable{
					{
						CallableFn: CallableFn{
							Fn: "api#returnUser",
						},
					},
					{
						Group: []CallableFn{
							{
								Fn: "db#getUser",
							},
							{
								Fn: "db#getUserDetails",
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
				Steps: []Executable{
					{
						CallableFn: CallableFn{
							Fn: "api#returnUser",
							OnErr: &FnOnErr{
								Code: map[int]string{
									400: "continue",
								},
								Any: "return",
							},
						},
					},
					{
						CallableFn: CallableFn{
							Fn: "api#returnUser",
							OnErr: &FnOnErr{
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
				Steps: []Executable{
					{
						Group: []CallableFn{
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
	dir := Directive{
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

	fqfn1, err := dir.FQFN("getUser")
	if err != nil {
		t.Error("fqfn1 err", err)
	}

	if fqfn1 != "default#getUser@v0.1.1" {
		t.Error("fqfn1 should be 'default#getUser@v0.1.1', got", fqfn1)
	}

	fqfn2, err := dir.FQFN("db#getUserDetails")
	if err != nil {
		t.Error("fqfn2 err", err)
	}

	if fqfn2 != "db#getUserDetails@v0.1.1" {
		t.Error("fqfn2 should be 'db#getUserDetails@v0.1.1', got", fqfn2)
	}

	fqfn3, err := dir.FQFN("api#returnUser")
	if err != nil {
		t.Error("fqfn3 err", err)
	}

	if fqfn3 != "api#returnUser@v0.1.1" {
		t.Error("fqfn3 should be 'api#returnUser@v0.1.1', got", fqfn3)
	}

	_, err = dir.FQFN("foo#bar")
	if err == nil {
		t.Error("foo#bar should have errored")
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
				Steps: []Executable{
					{
						Group: []CallableFn{
							{
								Fn: "getUser",
								With: []string{
									"data: someData",
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
