package main

import (
	"fmt"

	"github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable"
	"github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable/db"
	"github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable/db/query"
	"github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable/errors"
	"github.com/suborbital/e2core/sat/engine/runtime/api/tinygo/runnable/log"
)

type TinygoDb struct{}

func (h TinygoDb) Run(input []byte) ([]byte, error) {
	// `uuid.Generate().String()` doesn't work in old versions of TinyGo so
	// this will have to do: https://xkcd.com/221/
	uuidArg := query.NewArgument("uuid", "d7e00ff6-1b30-48e9-aa7d-dd3db34cb8b5")

	// insert
	_, err := db.Insert("PGInsertUser",
		uuidArg,
		query.NewArgument("email", "connor@suborbital.dev"))
	if err != nil {
		return nil, errors.WithCode(err, 500)
	}
	log.Info("insert successful")

	// update
	result, err := db.Update("PGUpdateUserWithUUID", uuidArg)
	if err != nil {
		return nil, errors.WithCode(err, 500)
	}
	log.Info(fmt.Sprintf("update: %s", string(result)))

	// select
	result, err = db.Select("PGSelectUserWithUUID", uuidArg)
	if err != nil {
		return nil, errors.WithCode(err, 500)
	}
	log.Info(fmt.Sprintf("select: %s", string(result)))

	// delete
	result, err = db.Delete("PGDeleteUserWithUUID", uuidArg)
	if err != nil {
		return nil, errors.WithCode(err, 500)
	}
	log.Info(fmt.Sprintf("delete: %s", string(result)))

	// this one should fail
	result, err = db.Select("PGSelectUserWithUUID", uuidArg)
	if err != nil {
		return nil, errors.WithCode(err, 500)
	}
	res := string(result)

	if res != "[]" {
		return nil, errors.New(fmt.Sprintf("select should have returning nothing, but didn't, got: %s", res), 500)
	}
	return []byte("all good!"), nil
}

// initialize runnable, do not edit //
func main() {
	runnable.Use(TinygoDb{})
}
