// Package testutil provides utility func(s) to help integration test.
package testutil

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/sirupsen/logrus"

	"github.com/rahul2393/city-falcon-assignment/internal/model"
	"github.com/rahul2393/city-falcon-assignment/internal/repository/dataprovider"
	"github.com/rahul2393/city-falcon-assignment/internal/repository/dataprovider/postgres"
)

const (
	expirationInSeconds = 120
	dbPoolSize          = 40
	dbPoolTimeout       = 20 * time.Second
	dbPoolMaxWait       = 120 * time.Second
)

// TestUtil represents data to help running integration tests.
type TestUtil struct {
	DBURL string

	Persist dataprovider.Provider
	DB      *pg.DB
	Log     *logrus.Entry
	Context context.Context
}

// New creates a pointer to the instance of test-util
func New() *TestUtil {
	return &TestUtil{
		Context: context.Background(),
	}
}

func (util *TestUtil) bootstrapDB() error {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return err
	}

	// pulls an image, creates a container based on it and runs
	res, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "13",
		Env: []string{
			"POSTGRES_USER=postgres",
			"POSTGRES_PASSWORD=postgres",
			"POSTGRES_DB=city_falcon_test",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		return err
	}

	if err := res.Expire(expirationInSeconds); err != nil {
		return err
	}

	if err := util.connectDB(pool, res); err != nil {
		return err
	}

	return nil
}

func (util *TestUtil) connectDB(pool *dockertest.Pool, containerResource *dockertest.Resource) error {
	address := containerResource.GetHostPort("5432/tcp")
	util.DBURL = fmt.Sprintf("postgres://postgres:postgres@%s/city_falcon_test?sslmode=disable", address)

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = dbPoolMaxWait
	return pool.Retry(func() error {
		dbopts, err := pg.ParseURL(util.DBURL)
		dbopts.PoolSize = dbPoolSize
		dbopts.PoolTimeout = dbPoolTimeout
		if err != nil {
			return err
		}

		util.DB = pg.Connect(dbopts)
		if err := util.DB.Ping(util.Context); err != nil {
			return err
		}
		return nil
	})
}

// InitDB intitializes db.
func (util *TestUtil) InitDB() error {
	if err := util.bootstrapDB(); err != nil {
		return err
	}

	persist, err := postgres.NewRepository(util.DBURL, true, logrus.New().WithField("test", true))
	if err != nil {
		return err
	}

	util.Persist = persist
	util.Log = logrus.New().WithField("test", true)
	return nil
}

// SetupDB populate data to database
func (util *TestUtil) SetupDB() error {

	if err := util.DB.Model(&model.Entry{}).CreateTable(&orm.CreateTableOptions{
		IfNotExists: true,
	}); err != nil {
		return err
	}
	if err := util.doSeeding(); err != nil {
		return err
	}

	return nil
}

func (util *TestUtil) doSeeding() error {
	createdAt, _ := time.Parse("2006-01-02T15:04:05.000Z", "2021-09-11T11:45:26.371Z")
	entries := []*model.Entry{
		{
			ID:      uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
			Version: 1,
		},
		{
			ID:         uuid.MustParse("f3fa60c1-02a4-496a-8c9b-c5418c9d3e67"),
			CreateTime: createdAt,
			UpdateTime: createdAt,
			Version:    2,
		},
		{
			ID:         uuid.MustParse("f3fa60c1-02a4-496a-8c9b-c5418c9d3e68"),
			CreateTime: createdAt,
			UpdateTime: createdAt,
			DeleteTime: &createdAt,
			Version:    3,
		},
	}

	for _, e := range entries {
		_, err := util.Persist.Create(util.Context, e)
		if err != nil {
			fmt.Printf("query execution Create() failed %v", err)
			return err
		}
	}
	return nil
}
