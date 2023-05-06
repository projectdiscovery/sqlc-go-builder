# sqlc-go-builder

`sqlc` implements a Dynamic Query Builder for [SQLC](https://github.com/kyleconroy/sqlc) and more specifically `MySQL` queries.

It implements a parser using [vitess-go-sqlparser](https://vitess.io/docs/contributing/contributing-to-ast-parser/) to parse and rewrite Complex MySQL queries on the fly using their `AST` and `Rewrite` functionality provided by the `sqlparser` package.

## Example

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
// query= select id from user_items where id in (?, ?, ?) and `name` = ? and age > ? order by `name` asc, age desc limit 10, 5
// args= [1 2 3 John 18]
```

### Credits

1. https://github.com/yiplee/sqlc - Original inspiration for this library. The concepts have been extended to support AST rewrite instead of string formatting and things have been made safer (No SQLi).