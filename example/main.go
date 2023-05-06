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
