# sqlc-go-builder

`sqlc` implements a Dynamic Query Builder for [SQLC](https://github.com/kyleconroy/sqlc) and more specifically `MySQL` queries.

It implements a parser using [vitess-go-sqlparser](https://vitess.io/docs/contributing/contributing-to-ast-parser/) to parse and rewrite Complex MySQL queries on the fly using their `AST` and `Rewrite` functionality provided by the `sqlparser` package.

## Features

1. Allows `Where`, `Order`, `In`, `Offset`, `Limit`, `Group` dynamically.
2. Supports complex `SELECT` queries.
3. Safe from SQLi by using Parameterized Arguments for everything dynamic.
4. Sanitizes code by using Vitess SQL parser.

## Example

### Using as standalone

```go
package main

import (
	"fmt"

	sqlc "github.com/projectdiscovery/sqlc-go-builder"
)

func main() {
	builder := sqlc.New().
		In("id", 1, 2, 3).
		Where("name = ?", "John").
		Where("age > ?", 18).
        	Group("name, age").
		Order("name ASC, age DESC").
		Offset(10).
		Limit(5)

	query, args, err := builder.Build("select id from user_items")
	if err != nil {
		panic(err)
	}
	fmt.Printf("query= %s\n args= %v\n", query, args)
}

// Output:
// query= select id from user_items where id in (?, ?, ?) and `name` = ? and age > ? group by `name`, age order by `name` asc, age desc limit 10, 5
// args= [1 2 3 John 18]
```

### Wrapping with SQLC

```go
package background

import (
	"context"
	"fmt"
	"testing"

	v2 "github.com/test-repo/pkg/db/v2"
	"github.com/test-repo/pkg/db/v2/dbsql"
	sqlc "github.com/projectdiscovery/sqlc-go-builder"
)

func TestPaginationDynamic(t *testing.T) {
	db, err := v2.New("<db-url>")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	querier := dbsql.New(sqlc.Wrap(db.Pool))
	//querier := db.Queries()
	data, err := querier.GetData(
		sqlc.Build(context.Background(), func(builder *sqlc.Builder) {
			builder.Limit(10)
		}),
	)
	if err != nil {
		t.Fatal(err)
	}

	for _, item := range data {
		item := item
		fmt.Printf("%v\n", item)
	}
}
```

### TODO

- [ ] Better error handling
- [ ] More tests


### Credits

1. https://github.com/yiplee/sqlc - Original inspiration for this library. The concepts have been extended to support AST rewrite instead of string formatting and things have been made safer (No SQLi).
