package listing

import (
	"strings"
	"testing"
	"time"

	"github.com/go-pg/pg/v10/orm"
)

func Test_escapeLike(t *testing.T) {
	tests := []struct {
		name string
		expr string
		want string
	}{
		{"alphanum", "foo123", "foo123"},
		{"slash", "foo\\bar", "foo\\\\bar"},
		{"percent", "foo%bar", "foo\\%bar"},
		{"dot", "foo. bar", "foo\\. bar"},
		{"combined 1", "\\.%", "\\\\\\.\\%"},
		{"combined 2", "a\\.%%.1", "a\\\\\\.\\%\\%\\.1"},
		{"combined 3", "b\\\\%\\%.2", "b\\\\\\\\\\%\\\\\\%\\.2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeLike(tt.expr); got != tt.want {
				t.Errorf("escapeLike() = %v, want %v", got, tt.want)
			}
		})
	}
}

type testUser struct {
	CreateTime time.Time
	CreateBy   string `filter:",ops:=;!=;in"`
	FirstName  string
	LastName   string
	IsAdmin    bool
	LoginCount uint
	Props      map[string]string
}

func queryString(f orm.QueryAppender) (string, error) {
	// Adapted from go-pg select_test.go
	fmter := orm.NewFormatter().WithModel(f)
	b, err := f.AppendQuery(fmter, nil)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func selectQueryString(q *orm.Query) (string, error) {
	// Adapted from go-pg select_test.go
	sel := orm.NewSelectQuery(q)
	s, err := queryString(sel)
	if err != nil {
		return "", err
	}
	return s, nil
}

func Test_ApplyFilters(t *testing.T) {
	const where = " WHERE "

	fc := FilterConfig{}
	fc.Hooks = map[string]FilterHook{
		"firstName": RenameFieldAndMapValuesFilterHook("first_name", func(value interface{}) (interface{}, error) {
			return value.(string), nil
		}),
	}

	type test struct {
		name    string
		expr    string
		wantErr bool
		want    string
	}
	modelTests := []struct {
		config FilterConfig
		prefix string
		tests  []test
	}{
		{
			fc, `SELECT "test_user"."create_time", "test_user"."create_by", "test_user"."first_name", "test_user"."last_name", "test_user"."is_admin", "test_user"."login_count", "test_user"."props" FROM "test_users" AS "test_user" WHERE `, []test{
				{"empty", "", false, ""}, // Special case, the " WHERE " will be ignored when comparing to the prefix above

				// Operators
				{"eq and eq", `firstName = "Attila" lastName = "Molnar"`, false, `("test_user"."first_name" = 'Attila') AND ("test_user"."last_name" = 'Molnar')`},
				{"not eq and eq", `not firstName = "Attila" lastName = "Molnar"`, false, `( NOT "test_user"."first_name" = 'Attila') AND ("test_user"."last_name" = 'Molnar')`},

				{"ne and ne", `firstName != "fn" lastName != "ln"`, false, `("test_user"."first_name" <> 'fn') AND ("test_user"."last_name" <> 'ln')`},
				{"not ne and not ne", `not firstName != "fn" not lastName != "ln"`, false, `( NOT "test_user"."first_name" <> 'fn') AND ( NOT "test_user"."last_name" <> 'ln')`},

				{"greater than and less than", `loginCount > 0 loginCount < 5`, false, `("test_user"."login_count" > 0) AND ("test_user"."login_count" < 5)`},
				{"not greater than and not less than", `not loginCount > 0 not loginCount < 5`, false, `( NOT "test_user"."login_count" > 0) AND ( NOT "test_user"."login_count" < 5)`},

				{"greater than or equal and less than or equal", `loginCount >= 0 loginCount <= 5`, false, `("test_user"."login_count" >= 0) AND ("test_user"."login_count" <= 5)`},
				{"not greater than or equal and not less than or equal", `not loginCount >= 0 not loginCount <= 5`, false, `( NOT "test_user"."login_count" >= 0) AND ( NOT "test_user"."login_count" <= 5)`},

				{"string in", `firstName in ("A", "B", "C")`, false, `("test_user"."first_name" IN ('A','B','C'))`},
				{"not string in", `not firstName in ("foo", "Bar", "BAZ")`, false, `("test_user"."first_name" NOT  IN ('foo','Bar','BAZ'))`},

				{"uint in", `loginCount in ( 1 , 2 ,  3 )`, false, `("test_user"."login_count" IN (1,2,3))`},
				{"not uint in", `not loginCount in (1 ,2,  3)`, false, `("test_user"."login_count" NOT  IN (1,2,3))`},

				{"int range", "loginCount: [1, 100]", false, `("test_user"."login_count" BETWEEN 1 AND 100)`},
				{"not int range", "not loginCount: [1, 100]", false, `("test_user"."login_count" NOT  BETWEEN 1 AND 100)`},

				{"date range", `createTime: ["2020-10-01T12:34:56Z", "2022-11-02T21:43:46Z"]`, false, `("test_user"."create_time" BETWEEN '2020-10-01T12:34:56Z' AND '2022-11-02T21:43:46Z')`},
				{"not date range", `not createTime: ["2020-10-01T12:34:56Z", "2022-11-02T21:43:46Z"]`, false, `("test_user"."create_time" NOT  BETWEEN '2020-10-01T12:34:56Z' AND '2022-11-02T21:43:46Z')`},

				{"string contains", `firstName: "A%.\\"`, false, `("test_user"."first_name" LIKE '%A\%\.\\%')`},
				{"not string contains", `not firstName: "B%.\\"`, false, `("test_user"."first_name" NOT  LIKE '%B\%\.\\%')`},

				// Bools
				{"bool eq", `isAdmin = true`, false, `("test_user"."is_admin" = TRUE)`},
				{"not bool eq", `not isAdmin = true`, false, `( NOT "test_user"."is_admin" = TRUE)`},

				{"bool ne", `isAdmin != false`, false, `("test_user"."is_admin" <> FALSE)`},
				{"not bool ne", `not isAdmin != false`, false, `( NOT "test_user"."is_admin" <> FALSE)`},

				// Nested fields
				{"nested field eq", `props.prop_one = "42"`, false, `("test_user"."props"->>'prop_one' = '42')`},
				{"not nested field eq", `not props.prop_one = "42"`, false, `( NOT "test_user"."props"->>'prop_one' = '42')`},

				{"nested field ne", `props.prop_one != "42" props.prop_two != "1"`, false, `("test_user"."props"->>'prop_one' <> '42') AND ("test_user"."props"->>'prop_two' <> '1')`},
				{"not nested field ne", `NOT props.prop_one != "42"`, false, `( NOT "test_user"."props"->>'prop_one' <> '42')`},

				{"nested field in", `props.prop_one IN ("1", "2") props.prop_two IN ("3", "4")`, false, `("test_user"."props"->>'prop_one' IN ('1','2')) AND ("test_user"."props"->>'prop_two' IN ('3','4'))`},
				{"not nested field in", `not props.prop_one IN ("1", "2") not props.prop_two IN ("3", "4")`, false, `("test_user"."props"->>'prop_one' NOT  IN ('1','2')) AND ("test_user"."props"->>'prop_two' NOT  IN ('3','4'))`},

				// Miscellaneous
				{"combined", `not createBy in ("users/9e0b2436-3646-440a-a737-a59729870d5e","users/67b6ed54-bcc1-496c-8c7e-f1e8e20755f6") props.p in ("3","2","1") firstName: "l" createTime: ["2020-10-01T00:00:00Z", "2025-10-01T00:00:00Z"]`, false, `("test_user"."create_by" NOT  IN ('users/9e0b2436-3646-440a-a737-a59729870d5e','users/67b6ed54-bcc1-496c-8c7e-f1e8e20755f6')) AND ("test_user"."props"->>'p' IN ('3','2','1')) AND ("test_user"."first_name" LIKE '%l%') AND ("test_user"."create_time" BETWEEN '2020-10-01T00:00:00Z' AND '2025-10-01T00:00:00Z')`},
				{"20 in", `props.inTest in ("1","2","3","4","5","6","7","8","9","10","11","12","13","14","15","16","17","18","19","20")`, false, `("test_user"."props"->>'inTest' IN ('1','2','3','4','5','6','7','8','9','10','11','12','13','14','15','16','17','18','19','20'))`},

				// Errors

				// No operator
				{"known field without op", `lastName`, true, ""},
				{"unknown field without op", `unknown`, true, ""},

				// Unknown field
				{"unknown field", `unknown = "foo"`, true, ""},
				{"known and unknown fields", `firstName = "fn" unknown = "foo"`, true, ""},

				// Miscellaneous problems
				{"unterminated string", `lastName "`, true, ""},
				{"unterminated string inside in", `lastName in ("d)`, true, ""},
				{"empty in", `lastName in ()`, true, ""},
				{"nesting for non-map", `lastName.foo = "bar"`, true, ""},
				{"nested field named as keyword", `props.in IN ("1", "2") props.not = "3" props.true = "t" props.false != "f"`, true, ""},
			},
		},
	}
	for _, mt := range modelTests {
		if !strings.HasSuffix(mt.prefix, where) {
			panic("prefix must end with \"" + where + "\"")
		}

		for _, tt := range mt.tests {
			t.Run(tt.name, func(t *testing.T) {
				q := orm.NewQuery(nil, &testUser{})

				if err := ApplyFilters(tt.expr, mt.config, q); (err != nil) != tt.wantErr {
					t.Errorf("applyFilters() error = %v, wantErr %v", err, tt.wantErr)
					return
				}

				if tt.wantErr {
					return
				}

				s, err := selectQueryString(q)
				if err != nil {
					t.Errorf("selectQueryString(): %v", err)
					return
				}

				if !strings.HasPrefix(s, mt.prefix) || s[len(mt.prefix):] != tt.want {
					// Special case for empty expression, there will be no WHERE clause hence the HasPrefix() check doesn't pass
					if tt.expr == "" && s == mt.prefix[:len(mt.prefix)-len(where)] {
						// Pass
						return
					}

					t.Errorf("mismatch\ngot  = %v\nwant = %v%v", s, mt.prefix, tt.want)
					return
				}
			})
		}
	}
}
