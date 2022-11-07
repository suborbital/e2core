package wasmtest

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/docker/go-connections/nat"
	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/suborbital/appspec/capabilities"
	"github.com/suborbital/vektor/vlog"

	"github.com/suborbital/e2core/sat/api"
	"github.com/suborbital/e2core/sat/engine"
)

func setupPostgresContainer(ctx context.Context, schema string) (testcontainers.Container, string, error) {
	port := "5432"
	password := "reactr"
	dbName := "reactr"

	urlFunc := func(port nat.Port) string {
		return fmt.Sprintf("postgres://postgres:%s@localhost:%s/%s?sslmode=disable", password, port.Port(), dbName)
	}

	tcreq := testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image: "postgres:alpine",

			ExposedPorts: []string{port},
			Cmd:          []string{"postgres"},
			Env: map[string]string{
				"POSTGRES_PASSWORD": password,
				"POSTGRES_DB":       dbName,
			},
			WaitingFor: wait.ForSQL(nat.Port(port), "postgres", urlFunc).Timeout(time.Second * 10),
		},
		Started: true,
	}

	container, err := testcontainers.GenericContainer(ctx, tcreq)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to create GenericContainer")
	}

	externalPort, err := container.MappedPort(ctx, nat.Port(port))
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to MappedPort")
	}

	connStr := urlFunc(externalPort)
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to sql.Open")
	}

	// apply schema
	if _, err := db.Exec(schema); err != nil {
		return nil, "", errors.Wrap(err, "failed to db.Exec")
	}

	if err := db.Close(); err != nil {
		return nil, "", errors.Wrap(err, "failed to db.Close")
	}

	return container, connStr, nil
}

func TestPGDBQueries(t *testing.T) {
	schema := `CREATE TABLE users (uuid varchar(64), email varchar(255), created_at timestamp, state varchar(3), identifier int);`

	ctx := context.Background()

	container, dbConnString, err := setupPostgresContainer(ctx, schema)
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to setupPostgresContainer"))
	}

	defer container.Terminate(ctx)

	q1 := capabilities.Query{
		Type:     capabilities.QueryTypeInsert,
		Name:     "PGInsertUser",
		VarCount: 2,
		Query: `
		INSERT INTO users (uuid, email, created_at, state, identifier)
		VALUES ($1, $2, NOW(), 'A', 12345)`,
	}

	q2 := capabilities.Query{
		Type:     capabilities.QueryTypeSelect,
		Name:     "PGSelectUserWithUUID",
		VarCount: 1,
		Query: `
		SELECT * FROM users
		WHERE uuid = $1`,
	}

	q3 := capabilities.Query{
		Type:     capabilities.QueryTypeUpdate,
		Name:     "PGUpdateUserWithUUID",
		VarCount: 1,
		Query: `
		UPDATE users SET state='B' WHERE uuid = $1`,
	}

	q4 := capabilities.Query{
		Type:     capabilities.QueryTypeDelete,
		Name:     "PGDeleteUserWithUUID",
		VarCount: 1,
		Query: `
		DELETE FROM users WHERE uuid = $1`,
	}

	config := capabilities.DefaultConfigWithDB(vlog.Default(), capabilities.DBTypePostgres, dbConnString, []capabilities.Query{q1, q2, q3, q4})

	api, _ := api.NewWithConfig(config)

	e := engine.NewWithAPI(api)

	tests := []struct {
		jobtype  string
		filepath string
	}{
		{
			"rs-dbtest",
			"../testdata/rs-dbtest/rs-dbtest.wasm",
		},
		{
			"tinygo-dbtest",
			"../testdata/tinygo-db/tinygo-db.wasm",
		},
	}

	for _, test := range tests {
		t.Run(test.jobtype, func(t *testing.T) {
			doWasm, _ := e.RegisterFromFile(test.jobtype, test.filepath)

			res, err := doWasm(nil).Then()
			if err != nil {
				t.Error(errors.Wrap(err, "failed to doWasm"))
				return
			}

			if string(res.([]byte)) != "all good!" {
				t.Error("something went wrong...")
			}
		})
	}
}
