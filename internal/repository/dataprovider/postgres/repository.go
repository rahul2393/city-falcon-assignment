package postgres

import (
	"context"
	"fmt"

	pg "github.com/go-pg/pg/v10"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/rahul2393/city-falcon-assignment/internal/model"
	"github.com/rahul2393/city-falcon-assignment/internal/repository/dataprovider"
	"github.com/rahul2393/city-falcon-assignment/pkg/listing"
)

const (
	defaultLimit = 100
)

var entryConfig = listing.FilterConfig{
	Hooks: map[string]listing.FilterHook{
		"database_name": listing.RenameFieldAndMapValuesFilterHook("datname", func(value interface{}) (interface{}, error) {
			return value.(string), nil
		}),
	},
}

type PGRepository struct {
	db *pg.DB
}

func NewRepository(dbURL string, enableQueryLog bool, logger *logrus.Entry) (dataprovider.Provider, error) {
	dbopts, err := pg.ParseURL(dbURL)
	if err != nil {
		return nil, fmt.Errorf("pg.ParseURL(): %w", err)
	}
	db := pg.Connect(dbopts)
	if err := db.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("db.Ping(): %w", err)
	}
	if err := db.Model(&model.Entry{}).CreateTable(&orm.CreateTableOptions{
		IfNotExists: true,
	}); err != nil {
		return nil, err
	}
	if enableQueryLog {
		db.AddQueryHook(dbLogger{log: logger})
	}
	return &PGRepository{db: db}, nil
}

type dbLogger struct {
	log *logrus.Entry
}

func (d dbLogger) BeforeQuery(ctx context.Context, q *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

func (d dbLogger) AfterQuery(ctx context.Context, q *pg.QueryEvent) error {
	bytes, err := q.FormattedQuery()
	if err == nil {
		d.log.Debug(string(bytes))
	}

	return nil
}

func (p PGRepository) SlowQuery(ctx context.Context, req model.SlowQueriesRequest) ([]*model.SlowQueryRecord, error) {
	var resources []*model.SlowQueryRecord
	query := p.db.ModelContext(ctx, &model.SlowQueryRecord{})
	if err := listing.ApplyFilters(req.Filter, entryConfig, query); err != nil {
		return nil, fmt.Errorf("[slowQuery] error in filter: %v", err)
	}
	query.Order(req.OrderBy)
	if req.PageSize > defaultLimit {
		req.PageSize = defaultLimit
	}
	if req.PageOffset < 0 {
		req.PageOffset = 0
	}
	if err := query.Limit(req.PageSize).Offset(req.PageOffset).Select(&resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (p PGRepository) Create(ctx context.Context, resource *model.Entry) (*model.Entry, error) {
	if _, err := p.db.Model(resource).Insert(); err != nil {
		return nil, err
	}
	return resource, nil
}

func (p PGRepository) ListEntries(ctx context.Context, req model.ListEntriesRequest) ([]*model.Entry, error) {
	var resources []*model.Entry
	query := p.db.ModelContext(ctx, &model.Entry{})
	if err := listing.ApplyFilters(req.Filter, entryConfig, query); err != nil {
		return nil, fmt.Errorf("error in filter: %v", err)
	}
	query.Order(req.OrderBy)
	if req.PageSize > defaultLimit {
		req.PageSize = defaultLimit
	}
	if req.PageOffset < 0 {
		req.PageOffset = 0
	}
	if err := query.Limit(req.PageSize).Offset(req.PageOffset).Select(&resources); err != nil {
		return nil, err
	}
	return resources, nil
}

func (p PGRepository) GetByID(ctx context.Context, id uuid.UUID, showDeleted bool, queryHook dataprovider.QueryHook) (*model.Entry, error) {
	resource := &model.Entry{ID: id}
	query := p.db.ModelContext(ctx, resource)
	if showDeleted {
		query.AllWithDeleted()
	}
	queryHook(query)
	if err := query.Select(); err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}
	return resource, nil
}

func (p PGRepository) Update(ctx context.Context, resource *model.Entry, fields []string, queryHook dataprovider.QueryHook) (*model.Entry, error) {
	if err := p.db.WithContext(ctx).RunInTransaction(ctx, func(tx *pg.Tx) error {
		query := tx.Model(resource).Returning("*").Column("update_time")
		for _, col := range fields {
			query.Column(col)
		}

		queryHook(query)

		if _, err := query.Update(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return resource, nil
}

func (p PGRepository) Delete(ctx context.Context, resource *model.Entry, queryHook dataprovider.QueryHook) (*model.Entry, error) {
	if err := p.db.WithContext(ctx).RunInTransaction(ctx, func(tx *pg.Tx) error {
		query := tx.Model(resource).WherePK().Returning("*")
		if queryHook != nil {
			queryHook(query)
		}
		if _, err := query.Delete(); err != nil {
			return err
		}
		return nil
	}); err != nil {
		if err == pg.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return resource, nil
}
