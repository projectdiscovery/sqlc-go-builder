package sqlc

import (
	"reflect"
	"testing"
)

func TestBuilder(t *testing.T) {
	tests := []struct {
		name         string
		setup        func(*Builder)
		baseQuery    string
		wantQuery    string
		wantArgs     []interface{}
		wantBuildErr bool
	}{
		{
			name: "Valid Where",
			setup: func(b *Builder) {
				b.Where("name = ?", "John")
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users where `name` = ?",
			wantArgs:  []interface{}{"John"},
		},
		{
			name: "Valid In",
			setup: func(b *Builder) {
				b.In("age", 25, 30, 35)
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users where age in (?, ?, ?)",
			wantArgs:  []interface{}{25, 30, 35},
		},
		{
			name: "Valid Order",
			setup: func(b *Builder) {
				b.Order("name ASC, age DESC")
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users order by `name` asc, age desc",
		},
		{
			name: "Valid Offset",
			setup: func(b *Builder) {
				b.Offset(10)
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users limit 10, 100",
		},
		{
			name: "Valid Limit",
			setup: func(b *Builder) {
				b.Limit(5)
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users limit 0, 5",
		},
		{
			name: "Valid Where, In, Order, Offset, Limit",
			setup: func(b *Builder) {
				b.Where("name = ?", "John").
					In("age", 25, 30, 35).
					Order("name ASC, age DESC").
					Offset(10).
					Limit(5)
			},
			baseQuery: "SELECT * FROM users",
			wantQuery: "select * from users where `name` = ? and age in (?, ?, ?) order by `name` asc, age desc limit 10, 5",
			wantArgs:  []interface{}{"John", 25, 30, 35},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{}
			tt.setup(b)
			gotQuery, gotArgs, err := b.Build(tt.baseQuery)
			if (err != nil) != tt.wantBuildErr {
				t.Errorf("Builder.Build() error = %v, wantBuildErr %v", err, tt.wantBuildErr)
				return
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("Builder.Build() gotQuery = %v, wantQuery %v", gotQuery, tt.wantQuery)
			}
			if !reflect.DeepEqual(gotArgs, tt.wantArgs) {
				t.Errorf("Builder.Build() gotArgs = %v, wantArgs %v", gotArgs, tt.wantArgs)
			}
		})
	}
}

func TestBuilder_Order_SQLInjection(t *testing.T) {
	sanitized := "select * from users order by `name` asc, age asc"

	tests := []struct {
		name   string
		cols   string
		result string
	}{
		{
			name:   "SQL injection attempt - column name with comment",
			cols:   "name --, age",
			result: sanitized,
		},
		{
			name:   "SQL injection attempt - order direction",
			cols:   "name ASC, age DESC; DROP TABLE users",
			result: sanitized,
		},
		{
			name:   "SQL injection attempt - order direction with comment",
			cols:   "name ASC --, age ASC",
			result: sanitized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &Builder{}
			b.Order(tt.cols)
			final, _, err := b.Build("SELECT * FROM users")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if final != tt.result {
				t.Errorf("expected %q, got %q", tt.result, final)
			}
		})
	}
}
