package postgres_test

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/go-pg/pg/v10/orm"
	"github.com/google/uuid"

	"github.com/rahul2393/city-falcon-assignment/internal/model"
	testutil "github.com/rahul2393/city-falcon-assignment/tests/dbutils"
)

var (
	util           *testutil.TestUtil
	invalidEntryID = uuid.MustParse("6e9412ec-34eb-4c17-91d4-d5591b8c1190")
)

func init() {
	testutil := testutil.New()
	if err := testutil.InitDB(); err != nil {
		testutil.Log.Panicf("testutil.initDB(): %v", err)
	}
	util = testutil
}

func cleanEntryData() {
	_, err := util.DB.Exec(`drop table if exists "entries" cascade`)
	if err != nil {
		util.Log.Infof("query execution error %v", err)
	}
}

func TestPGDataProvider_SlowQuery(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}
	defer cleanEntryData()

	tests := []struct {
		name    string
		args    model.SlowQueriesRequest
		want    int
		wantErr string
	}{
		{
			name: "valid input",
			args: model.SlowQueriesRequest{
				PageSize:   1001,
				PageOffset: 0,
				OrderBy:    "pid",
				Filter:     "",
			},
		},
		{
			name: "success with invalid pageOffset",
			args: model.SlowQueriesRequest{
				PageSize:   1000000000,
				PageOffset: -1,
				OrderBy:    "pid",
				Filter:     "",
			},
		},
		{
			name: "success with valid filter",
			args: model.SlowQueriesRequest{
				PageSize:   100,
				PageOffset: 0,
				OrderBy:    "pid",
				Filter:     `database_name!=""`,
			},
		},
		{
			name: "invalid filter",
			args: model.SlowQueriesRequest{
				PageSize:   100,
				PageOffset: 0,
				OrderBy:    "pid",
				Filter:     "ad!=><",
			},
			wantErr: "[slowQuery] error in filter: parse: 1:5: unexpected token \">\" (expected <int> | <float> | <string> | \"TRUE\" | \"FALSE\")",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.SlowQuery(ctx, tt.args)
			if err != nil && strings.Compare(err.Error(), tt.wantErr) != 0 {
				t.Errorf("name: %v, Persist.SlowQuery() error = %v, wantErr %v", tt.name, err.Error(), tt.wantErr)
				return
			}
		})
	}
}

func TestPGDataProvider_ListEntries(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}
	defer cleanEntryData()

	tests := []struct {
		name    string
		args    model.ListEntriesRequest
		want    int
		wantErr string
	}{
		{
			name: "valid input",
			args: model.ListEntriesRequest{
				PageSize:   100,
				PageOffset: 0,
				OrderBy:    "version",
				Filter:     "",
			},
		},
		{
			name: "success with invalid pageOffset",
			args: model.ListEntriesRequest{
				PageSize:   1000000000,
				PageOffset: -1,
				OrderBy:    "version",
				Filter:     "",
			},
		},
		{
			name: "success with valid filter",
			args: model.ListEntriesRequest{
				PageSize:   100,
				PageOffset: 0,
				OrderBy:    "version",
				Filter:     `version!="4"`,
			},
		},
		{
			name: "invalid filter",
			args: model.ListEntriesRequest{
				PageSize:   100,
				PageOffset: 0,
				OrderBy:    "version",
				Filter:     "ad!=><",
			},
			wantErr: "error in filter: parse: 1:5: unexpected token \">\" (expected <int> | <float> | <string> | \"TRUE\" | \"FALSE\")",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := p.ListEntries(ctx, tt.args)
			if err != nil && strings.Compare(err.Error(), tt.wantErr) != 0 {
				t.Errorf("name: %v, Persist.ListEntries() error = %v, wantErr %v", tt.name, err.Error(), tt.wantErr)
				return
			}
		})
	}
}

func TestPGDataProvider_CreateEntry(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}

	defer cleanEntryData()
	type args struct {
		res *model.Entry
	}
	newEntryID := uuid.MustParse("1a5263a4-94af-47c1-8cd2-c5ca3ae1c4e0")
	tests := []struct {
		name    string
		args    args
		want    *model.Entry
		wantErr string
	}{
		{
			name: "create failure",
			args: args{
				res: &model.Entry{
					Version: 5,
				},
			},
			wantErr: `ERROR #23502 null value in column "id" of relation "entries" violates not-null constraint`,
		},
		{
			name: "create success",
			args: args{
				res: &model.Entry{
					ID:      newEntryID,
					Version: 5,
				},
			},
			want: &model.Entry{
				ID:      newEntryID,
				Version: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Create(ctx, tt.args.res)
			if err != nil && strings.Compare(err.Error(), tt.wantErr) != 0 {
				t.Errorf("name: %v, Persist.Create() error = %v, wantErr %v", tt.name, err.Error(), tt.wantErr)
				return
			}
			if got != nil {
				got.CreateTime = time.Time{}
				got.UpdateTime = time.Time{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("name: %v, Persist.Create() \ngot  %v\nwant %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPGDataProvider_GetByID(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}

	defer cleanEntryData()

	type args struct {
		showDeleted bool
		id          uuid.UUID
	}

	tests := []struct {
		name string
		args args
		want *model.Entry
	}{

		{
			name: "entry not existed",
			args: args{
				showDeleted: true,
				id:          invalidEntryID,
			},
			want: nil,
		},
		{
			name: "entry deleted not return",
			args: args{
				showDeleted: false,
				id:          uuid.MustParse("f3fa60c1-02a4-496a-8c9b-c5418c9d3e68"),
			},
			want: nil,
		},
		{
			name: "entry deleted return",
			args: args{
				showDeleted: true,
				id:          uuid.MustParse("f3fa60c1-02a4-496a-8c9b-c5418c9d3e68"),
			},
			want: &model.Entry{
				ID:      uuid.MustParse("f3fa60c1-02a4-496a-8c9b-c5418c9d3e68"),
				Version: 3,
			},
		},
		{
			name: "entry normal return",
			args: args{
				showDeleted: false,
				id:          uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
			},
			want: &model.Entry{
				ID:      uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
				Version: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.GetByID(ctx, tt.args.id, tt.args.showDeleted, func(query *orm.Query) {
				query.WherePK()
			})
			if err != nil {
				t.Errorf("name: %v, Persist.GetByID() error = %v, wantErr nil", tt.name, err)
				return
			}

			if got != nil {
				got.CreateTime = time.Time{}
				got.UpdateTime = time.Time{}
				got.DeleteTime = nil
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("name: %v, Persist.GetByID() \ngot  %v\nwant %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPGDataProvider_Update(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}

	defer cleanEntryData()

	type args struct {
		res    *model.Entry
		fields []string
	}

	tests := []struct {
		name string
		args args
		want *model.Entry
	}{
		{
			name: "invalid entry id",
			args: args{
				res: &model.Entry{
					ID: invalidEntryID,
				},
			},
		},
		{
			name: "update success",
			args: args{
				res: &model.Entry{
					ID:      uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
					Version: 100,
				},
				fields: []string{"version"},
			},
			want: &model.Entry{
				ID:      uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
				Version: 100,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Update(ctx, tt.args.res, tt.args.fields, func(query *orm.Query) {
				query.WherePK()
			})
			if err != nil {
				t.Errorf("name: %v, Persist.Update() error = %v, wantErr nil", tt.name, err)
				return
			}
			if got != nil {
				got.CreateTime = time.Time{}
				got.DeleteTime = nil
				got.UpdateTime = time.Time{}
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("name: %v, Persist.Update() \ngot  %v\nwant %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestPGDataProvider_DeleteEntry(t *testing.T) {
	ctx := util.Context
	p := util.Persist

	if err := util.SetupDB(); err != nil {
		util.Log.Panicf("util.SetupDB(): %v", err)
	}

	defer cleanEntryData()
	type args struct {
		id uuid.UUID
	}
	tests := []struct {
		name string
		args args
		want *model.Entry
	}{
		{
			name: "with not existence id",
			args: args{
				id: uuid.New(),
			},
		},
		{
			name: "with existence entry id",
			args: args{
				id: uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
			},
			want: &model.Entry{
				ID:      uuid.MustParse("50321353-d4a8-4e5d-810a-44f60a056fc4"),
				Version: 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := p.Delete(ctx, &model.Entry{ID: tt.args.id}, nil)
			if err != nil {
				t.Errorf("Persist.Delete() error = %v, wantErr nil", err)
				return
			}

			if got != nil {
				if got.DeleteTime.IsZero() {
					t.Errorf("Persist.DeleteTime must not be nil")
				}
				// Ignore fields
				got.DeleteTime = nil
				got.CreateTime = time.Time{}
				got.UpdateTime = time.Time{}
			}

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Persist.Delete() \ngot  %v\nwant %v", got, tt.want)
			}
		})
	}
}
