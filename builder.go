package sqlc

import (
	"fmt"
	"strings"

	"vitess.io/vitess/go/vt/sqlparser"
)

type (
	// Builder is a SQL query builder.
	//
	// It supports dynamic WHERE conditions, ORDER BY, LIMIT, and OFFSET.
	// For dynamic WHERE and ORDER BY clauses, only the parent table rows can be used.
	// It uses vitess sqlparser to parse the query and rewrite it.
	Builder struct {
		filters       []filter
		order         *sqlparser.OrderBy
		offset, limit *int
		group         *sqlparser.GroupBy

		RowCount int
	}

	filter struct {
		expression   string
		args         []interface{}
		placeholders []string
	}
)

// New creates a new Builder.
func New() *Builder {
	return &Builder{}
}

func (b *Builder) clone() *Builder {
	cb := *b
	return &cb
}

// Where set conditions of where in SELECT
// Where("user = ?","tom")
// Where("a = ? OR b = ?",1,2)
//
// Requires 3 arguments, first is key, second is operator
// and third is a argument to be replaced with a placeholder.
//
// Requires spaces between conditions to work.
func (b *Builder) Where(query string, args ...interface{}) *Builder {
	b.filters = append(b.filters, filter{
		expression: query,
		args:       args,
	})

	return b
}

func (b *Builder) whereWithPlaceholders(query string, args []interface{}, placeholders []string) *Builder {
	b.filters = append(b.filters, filter{
		expression:   query,
		args:         args,
		placeholders: placeholders,
	})

	return b
}

// In is an equivalent of Where("column IN (?,?,?)", args...).
// In("id", 1, 2, 3)
func (b *Builder) In(column string, args ...interface{}) *Builder {
	placeholders := make([]string, len(args))
	for i := range args {
		placeholders[i] = "?"
	}

	colIdent, err := getTableRowIdentifier(column)
	if err != nil {
		fmt.Printf("could not get table row identifier %s: %s", column, err)
		return b
	}
	quotedColumn := sqlparser.String(colIdent)

	query := fmt.Sprintf("%s IN (%s)", quotedColumn, strings.Join(placeholders, ","))
	return b.whereWithPlaceholders(query, args, placeholders)
}

// Order sets columns of ORDER BY in SELECT.
// Order("name, age DESC")
func (b *Builder) Order(cols string) *Builder {
	columns, err := extractOrderBy(cols)
	if err != nil {
		fmt.Printf("could not extract order by %s: %s", cols, err)
		return b
	}
	b.order = columns
	return b
}

// Group sets columns of GROUP BY in SELECT.
// Group("name")
func (b *Builder) Group(cols string) *Builder {
	groups := sqlparser.GroupBy{}
	parts := strings.Split(cols, ",")
	for _, item := range parts {
		item := item

		value := strings.Trim(item, " ")
		colIdent, err := getTableRowIdentifier(value)
		if err != nil {
			fmt.Printf("could not get table row identifier %s: %s", cols, err)
			return b
		}
		groups = append(groups, colIdent)
	}
	b.group = &groups
	return b
}

// Offset sets the offset in SELECT.
func (b *Builder) Offset(x int) *Builder {
	b.offset = &x
	return b
}

// Limit sets the limit in SELECT.
func (b *Builder) Limit(x int) *Builder {
	b.limit = &x
	return b
}
