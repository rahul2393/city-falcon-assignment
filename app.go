package main

import (
	"fmt"
	"github.com/go-pg/pg/v10/orm"
	"github.com/gofiber/fiber/v2/middleware/cache"
	"github.com/google/uuid"
	"github.com/rahul2393/city-falcon-assignment/internal/model"
	"github.com/rahul2393/city-falcon-assignment/internal/repository/dataprovider"
	"github.com/rahul2393/city-falcon-assignment/internal/repository/dataprovider/postgres"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
)

type Service struct {
	provider dataprovider.Provider
	logger   *logrus.Entry
}

type Options struct {
	DBURL             string
	LogQuery          string
	ListenAddressHTTP string
}

// MustGet retrieves the value of the environment variable named key. It panics if the variable is not present.
func MustGet(key string) string {
	val := os.Getenv(key)
	if val == "" {
		panic(fmt.Sprint("Required environment variable not set: ", key))
	}

	return val
}

func main() {
	options := Options{
		DBURL:             MustGet("DB_URL"),
		LogQuery:          os.Getenv("LOG_QUERY"),
		ListenAddressHTTP: MustGet("LISTEN_ADDRESS_HTTP"),
	}
	logger := NewLogger()
	repo, err := postgres.NewRepository(options.DBURL, options.LogQuery != "", logger)
	if err != nil {
		logger.Fatalf("failed to connect to DB, check connection string: %w", err)
	}
	svc := Service{provider: repo, logger: logger}
	app := fiber.New()
	app.Use(cache.New(cache.Config{
		Next: func(c *fiber.Ctx) bool {
			return c.Query("refresh") == "true"
		},
		Expiration:   30 * time.Second,
		CacheControl: true,
	}))
	app.Get("/slow-queries", func(c *fiber.Ctx) error {
		req := model.SlowQueriesRequest{
			PageSize:   100,
			PageOffset: 0,
			OrderBy:    c.Query("orderBy", "pid"),
			Filter:     c.Query("filter", ""),
		}
		if v, err := strconv.Atoi(c.Query("pageSize", "100")); err == nil {
			req.PageSize = v
		}
		if v, err := strconv.Atoi(c.Query("pageOffset", "0")); err == nil {
			req.PageOffset = v
		}
		resp, err := svc.provider.SlowQuery(c.Context(), req)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(resp)
	})

	app.Post("/entry", func(c *fiber.Ctx) error {
		var reqBody model.Entry
		if err := c.BodyParser(&reqBody); err != nil {
			return err
		}
		reqBody.ID = uuid.New()
		resp, err := svc.provider.Create(c.Context(), &reqBody)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(resp)
	})

	app.Get("/entries", func(c *fiber.Ctx) error {
		req := model.ListEntriesRequest{
			PageSize:   100,
			PageOffset: 0,
			OrderBy:    c.Query("orderBy", "create_time"),
			Filter:     c.Query("filter", ""),
		}
		if v, err := strconv.Atoi(c.Query("pageSize", "100")); err == nil {
			req.PageSize = v
		}
		if v, err := strconv.Atoi(c.Query("pageOffset", "0")); err == nil {
			req.PageOffset = v
		}
		resp, err := svc.provider.ListEntries(c.Context(), req)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(model.ListEntriesResponse{Entries: resp})
	})

	app.Get("/entry/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		showDeleted := false
		if c.Query("showDeleted") == "true" {
			showDeleted = true
		}
		resp, err := svc.provider.GetByID(c.Context(), id, showDeleted, func(query *orm.Query) {
			query.WherePK()
		})
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(resp)
	})

	// updates the entry record
	app.Put("/entry/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		var reqBody model.Entry
		if err := c.BodyParser(&reqBody); err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		reqBody.ID = id
		resp, err := svc.provider.Update(c.Context(), &reqBody,
			strings.Split(c.Query("updateMask", ""), ","), func(query *orm.Query) {
				query.WherePK()
			})
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(resp)
	})

	// delete entry by id
	app.Delete("/entry/:id", func(c *fiber.Ctx) error {
		id, err := uuid.Parse(c.Params("id"))
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return err
		}
		resp, err := svc.provider.Delete(c.Context(), &model.Entry{ID: id}, nil)
		if err != nil {
			c.Status(http.StatusInternalServerError)
			return nil
		}
		return c.JSON(resp)
	})

	log.Fatal(app.Listen(":" + options.ListenAddressHTTP))
}

func NewLogger() *logrus.Entry {
	l := logrus.New()
	if os.Getenv("LOG_JSON") != "" {
		l.Formatter = &logrus.JSONFormatter{
			DataKey:         "data",
			TimestampFormat: time.RFC3339Nano,
		}
	}
	if lvl, err := logrus.ParseLevel(os.Getenv("LOG_LEVEL")); err == nil {
		l.Level = lvl
	} else {
		l.Level = logrus.TraceLevel
	}

	lo := l.WithFields(logrus.Fields{
		"app": map[string]string{
			"host":    os.Getenv("HOST"),
			"version": os.Getenv("APP_VERSION"),
		},
	})

	return lo
}
