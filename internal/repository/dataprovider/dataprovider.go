package dataprovider

import (
	"context"
	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"

	"github.com/rahul2393/city-falcon-assignment/internal/model"
)

type QueryHook func(query *orm.Query)

type Provider interface {
	SlowQuery(ctx context.Context, req model.SlowQueriesRequest) ([]*model.SlowQueryRecord, error)

	Create(ctx context.Context, resource *model.Entry) (*model.Entry, error)
	ListEntries(ctx context.Context, req model.ListEntriesRequest) ([]*model.Entry, error)
	GetByID(ctx context.Context, id uuid.UUID, showDeleted bool, queryHook QueryHook) (*model.Entry, error)
	Update(ctx context.Context, resource *model.Entry, fields []string, queryHook QueryHook) (*model.Entry, error)
	Delete(ctx context.Context, resource *model.Entry, queryHook QueryHook) (*model.Entry, error)
}
