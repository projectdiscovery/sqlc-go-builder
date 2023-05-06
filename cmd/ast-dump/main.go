package main

import (
	"github.com/davecgh/go-spew/spew"
	"vitess.io/vitess/go/vt/sqlparser"
)

func main() {
	query := "select id from user_items WHERE user_items.name=?"
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		panic(err)
	}

	post := func(cursor *sqlparser.Cursor) bool {
		switch n := cursor.Node().(type) {
		case *sqlparser.Select:
			spew.Dump(n)
		}
		return true
	}
	s := sqlparser.Rewrite(stmt, nil, post)
	_ = s
}
