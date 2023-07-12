package model

import (
	"context"
	"time"

	"github.com/google/uuid"
)

type Entry struct {
	tableName struct{} `pg:",discard_unknown_columns"`

	ID         uuid.UUID  `pg:",pk,type:uuid" `
	CreateTime time.Time  `pg:",notnull"`
	UpdateTime time.Time  `pg:",notnull"`
	DeleteTime *time.Time `pg:",soft_delete" filter:"-"`
	Version    uint64     `pg:",notnull,default:1" filter:"-"`
}

func (e *Entry) BeforeInsert(ctx context.Context) (context.Context, error) {
	if e.CreateTime.IsZero() {
		e.CreateTime = time.Now()
		e.UpdateTime = e.CreateTime
	}
	return ctx, nil
}

func (e *Entry) BeforeUpdate(ctx context.Context) (context.Context, error) {
	e.UpdateTime = time.Now()
	return ctx, nil
}

type ListEntriesRequest struct {
	PageSize   int    `json:"page_size,omitempty"`
	PageOffset int    `json:"page_offset,omitempty"`
	OrderBy    string `json:"order_by,omitempty"`
	Filter     string
}

type ListEntriesResponse struct {
	Entries []*Entry `json:"entries,omitempty"`
}

type SlowQueryRecord struct {
	tableName struct{} `pg:"pg_stat_activity,discard_unknown_columns"`

	DatabaseName  string `pg:"datname" json:"database_name"`
	PID           string `pg:"pid" json:"pid"`
	UserName      string `pg:"usename" json:"user_name"`
	ClientAddress string `pg:"client_addr" json:"client_address"`
	BackendStart  string `pg:"backend_start" json:"backend_start"`
	QueryStart    string `pg:"query_start" json:"query_start"`
	State         string `pg:"state" json:"state"`
	Query         string `pg:"query"  json:"query"`
}

type SlowQueriesRequest struct {
	PageSize   int    `json:"page_size,omitempty"`
	PageOffset int    `json:"page_offset,omitempty"`
	OrderBy    string `json:"order_by,omitempty"`
	Filter     string
}

type SlowQueriesResponse struct {
	SlowQueries []*SlowQueryRecord `json:"slow_queries,omitempty"`
}
