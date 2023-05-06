package sqlc

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"vitess.io/vitess/go/vt/sqlparser"
)

var defaultRowCount = 100

// replaceVitessRegex is a regex to replace vitess placeholders with a single
// question mark.
var replaceVitessRegex = regexp.MustCompile(`:v\d+`)

// Build generates a query and args from the builder.
func (b *Builder) Build(query string, args ...interface{}) (string, []interface{}, error) {
	stmt, err := sqlparser.Parse(query)
	if err != nil {
		return "", nil, errors.Wrap(err, "could not parse sql")
	}

	var finalArgs []interface{}
	finalArgs = append(finalArgs, args...)

	var finalError error
	post := func(cursor *sqlparser.Cursor) bool {
		switch n := cursor.Node().(type) {
		case *sqlparser.Select:
			args, err := b.modifySelectStatement(n, len(args))
			if err != nil {
				finalError = err
				return false
			}
			finalArgs = append(finalArgs, args...)
		}
		return true
	}
	if finalError != nil {
		return "", nil, errors.Wrap(finalError, "could not build query")
	}
	s := sqlparser.Rewrite(stmt, nil, post)
	data := sqlparser.String(s)
	replaced := replaceVitessRegex.ReplaceAllString(data, "?")
	return replaced, finalArgs, nil
}

func (b *Builder) modifySelectStatement(stmt *sqlparser.Select, previousIndex int) ([]interface{}, error) {
	if b.order != nil {
		stmt.OrderBy = *b.order
	}
	if b.limit != nil {
		if stmt.Limit != nil {
			stmt.Limit.Rowcount = sqlparser.NewIntLiteral(strconv.Itoa(*b.limit))
		} else {
			stmt.Limit = &sqlparser.Limit{
				Rowcount: sqlparser.NewIntLiteral(strconv.Itoa(*b.limit)),
				Offset:   sqlparser.NewIntLiteral(strconv.Itoa(0)),
			}
		}
	}
	if b.offset != nil {
		if stmt.Limit != nil {
			stmt.Limit.Offset = sqlparser.NewIntLiteral(strconv.Itoa(*b.offset))
		} else {
			stmt.Limit = &sqlparser.Limit{
				Offset: sqlparser.NewIntLiteral(strconv.Itoa(*b.offset)),
			}
			if b.RowCount > 0 {
				stmt.Limit.Rowcount = sqlparser.NewIntLiteral(strconv.Itoa(b.RowCount))
			} else {
				stmt.Limit.Rowcount = sqlparser.NewIntLiteral(strconv.Itoa(defaultRowCount))
			}
		}
	}
	if b.group != nil {
		stmt.GroupBy = *b.group
	}
	var filterErr error
	var args []interface{}
mainLoop:
	for i, filter := range b.filters {
		filter := filter

		comparison, err := extractWhereStatement(filter.expression, i+1+previousIndex, filter.placeholders)
		if err != nil {
			filterErr = errors.Errorf("could not extract where statement: %s", err)
			break
		}
		if stmt.Where == nil {
			stmt.Where = &sqlparser.Where{Expr: comparison}
			args = append(args, filter.args...)
			continue
		}
		switch v := stmt.Where.Expr.(type) {
		case *sqlparser.ComparisonExpr:
			stmt.Where.Expr = &sqlparser.AndExpr{
				Left:  v,
				Right: comparison,
			}
		case *sqlparser.AndExpr:
			stmt.Where.Expr = &sqlparser.AndExpr{
				Left:  v,
				Right: comparison,
			}
		case *sqlparser.OrExpr:
			stmt.Where.Expr = &sqlparser.AndExpr{
				Left:  v,
				Right: comparison,
			}
		default:
			filterErr = errors.Errorf("unsupported where expression: %T", stmt.Where.Expr)
			break mainLoop
		}
		args = append(args, filter.args...)
	}
	if filterErr != nil {
		return nil, filterErr
	}
	return args, nil
}

// extractOrderBy parses the orderByStr and returns a sqlparser.OrderBy.
func extractOrderBy(orderByStr string) (*sqlparser.OrderBy, error) {
	// Split the orderByStr by commas
	parts := strings.Split(orderByStr, ",")

	orderBy := make(sqlparser.OrderBy, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		parts := strings.SplitN(part, " ", 2)
		if len(parts) == 0 {
			return nil, errors.Errorf("invalid order by statement: %v", part)
		}
		var direction string
		column := parts[0]
		if len(parts) < 2 {
			direction = "asc"
		} else {
			direction = parts[1]
		}

		// Determine the order direction
		var dir sqlparser.OrderDirection
		if strings.HasSuffix(strings.ToLower(direction), "desc") {
			dir = sqlparser.DescOrder
		} else if strings.HasSuffix(strings.ToLower(part), "asc") {
			dir = sqlparser.AscOrder
		}
		leftStmt, err := getTableRowIdentifier(column)
		if err != nil {
			return nil, errors.Wrap(err, "could not get table row identifier")
		}
		// Create the OrderBy data structure
		orderBy[i] = &sqlparser.Order{
			Expr:      leftStmt,
			Direction: dir,
		}
	}
	return &orderBy, nil
}

func extractWhereStatement(whereStr string, i int, placeholders []string) (*sqlparser.ComparisonExpr, error) {
	parts := strings.SplitN(whereStr, " ", 3)
	if len(parts) < 3 {
		return nil, errors.New("invalid where statement")
	}

	key := parts[0]
	operator := parts[1]

	var op sqlparser.ComparisonExprOperator
	switch operator {
	case "=":
		op = sqlparser.EqualOp
	case "!=":
		op = sqlparser.NotEqualOp
	case ">":
		op = sqlparser.GreaterThanOp
	case "<":
		op = sqlparser.LessThanOp
	case ">=":
		op = sqlparser.GreaterEqualOp
	case "<=":
		op = sqlparser.LessEqualOp
	case "IN":
		op = sqlparser.InOp
	case "NOT IN":
		op = sqlparser.NotInOp
	case "LIKE":
		op = sqlparser.LikeOp
	case "NOT LIKE":
		op = sqlparser.NotLikeOp
	default:
		return nil, errors.Errorf("invalid operator: %s", operator)
	}

	// Handle placeholders only as of now
	value := parts[2]
	var right sqlparser.Expr
	if value == "?" {
		right = buildPositionalArgument(i)
	} else if len(placeholders) > 0 {
		var valueTuples sqlparser.ValTuple
		for ni := range placeholders {
			valueTuples = append(valueTuples, buildPositionalArgument(i+ni))
		}
		right = valueTuples
	}

	// Check the key to see if we have a table name as well
	leftStmt, err := getTableRowIdentifier(key)
	if err != nil {
		return nil, errors.Wrap(err, "could not get table row identifier")
	}

	expr := &sqlparser.ComparisonExpr{
		Operator: op,
		Left:     leftStmt,
		Right:    right,
	}
	return expr, nil
}

func getTableRowIdentifier(key string) (sqlparser.Expr, error) {
	keyNames := strings.SplitN(key, ".", 2)
	if len(keyNames) == 0 {
		return nil, errors.New("invalid where statement key")
	}
	var tableName string
	var rowName string
	if len(keyNames) == 2 {
		tableName = keyNames[0]
		rowName = keyNames[1]
	} else {
		rowName = keyNames[0]
	}

	var leftStmt sqlparser.Expr
	if tableName != "" {
		leftStmt = sqlparser.NewColNameWithQualifier(rowName, sqlparser.TableName{
			Name: sqlparser.NewIdentifierCS(tableName),
		})
	} else {
		leftStmt = sqlparser.NewColName(key)
	}
	return leftStmt, nil
}

func buildPositionalArgument(i int) sqlparser.Argument {
	var builder strings.Builder
	builder.WriteRune('v')
	builder.WriteString(strconv.Itoa(i))
	built := sqlparser.NewArgument(builder.String())
	return built
}
