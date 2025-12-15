package client

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/cespare/xxhash"
)

// Operator represents a comparison operator for WHERE clauses.
type Operator int

const (
	// Comparison operators
	Equals Operator = iota
	NotEquals
	GreaterThan
	LessThan
	GreaterThanOrEqual
	LessThanOrEqual

	// Pattern matching
	Like
	ILike
	NotLike
	NotILike

	// Set operators
	In
	NotIn

	// Null checks
	IsNull
	IsNotNull

	// Logical operators (for complex conditions)
	And
	Or
	Not
)

// String returns the SyndrQL representation of the operator.
func (o Operator) String() string {
	switch o {
	case Equals:
		return "=="
	case NotEquals:
		return "!="
	case GreaterThan:
		return ">"
	case LessThan:
		return "<"
	case GreaterThanOrEqual:
		return ">="
	case LessThanOrEqual:
		return "<="
	case Like:
		return "LIKE"
	case ILike:
		return "ILIKE"
	case NotLike:
		return "NOT LIKE"
	case NotILike:
		return "NOT ILIKE"
	case In:
		return "IN"
	case NotIn:
		return "NOT IN"
	case IsNull:
		return "IS NULL"
	case IsNotNull:
		return "IS NOT NULL"
	case And:
		return "AND"
	case Or:
		return "OR"
	case Not:
		return "NOT"
	default:
		return "="
	}
}

// Direction represents the sort direction for ORDER BY clauses.
type Direction int

const (
	Ascending Direction = iota
	Descending
)

// String returns the SyndrQL representation of the direction.
func (d Direction) String() string {
	switch d {
	case Ascending:
		return "ASC"
	case Descending:
		return "DESC"
	default:
		return "ASC"
	}
}

// queryType represents the type of query being built.
type queryType int

const (
	selectQuery queryType = iota
	insertQuery
	updateQuery
	deleteQuery
)

// whereClause represents a WHERE condition with its logical connector.
type whereClause struct {
	field     string
	operator  Operator
	value     interface{}
	connector Operator // And or Or
}

// orderByClause represents an ORDER BY clause.
type orderByClause struct {
	field     string
	direction Direction
}

// joinClause represents a JOIN clause with ON conditions.
type joinClause struct {
	joinType         string // "INNER", "LEFT", "RIGHT"
	targetBundle     string
	onSourceField    string
	onTargetField    string
	alias            string // Optional table alias
	relationshipName string // For relationship-based joins
}

// QueryBuilder provides a fluent API for building type-safe SELECT queries.
type QueryBuilder struct {
	client           *Client
	bundle           string
	fields           []string
	whereClauses     []whereClause
	orderBys         []orderByClause
	joinClauses      []joinClause // Explicit JOIN clauses
	limitVal         *int
	offsetVal        *int
	includes         []string // For relationship eager loading
	params           []interface{}
	paramCount       int
	schemaValidation bool
	queryType        queryType
}

// InsertBuilder provides a fluent API for building INSERT queries.
type InsertBuilder struct {
	client           *Client
	bundle           string
	values           map[string]interface{}
	params           []interface{}
	paramCount       int
	schemaValidation bool
}

// UpdateBuilder provides a fluent API for building UPDATE queries.
type UpdateBuilder struct {
	client           *Client
	bundle           string
	setFields        map[string]interface{}
	whereClauses     []whereClause
	params           []interface{}
	paramCount       int
	schemaValidation bool
}

// DeleteBuilder provides a fluent API for building DELETE queries.
type DeleteBuilder struct {
	client           *Client
	bundle           string
	whereClauses     []whereClause
	params           []interface{}
	paramCount       int
	schemaValidation bool
}

// TODO: Implement Upsert(bundle, data, conflictFields) for INSERT ... ON CONFLICT
// operations pending server protocol specification for conflict resolution syntax.

// ============================================================================
// QueryBuilder SELECT Methods
// ============================================================================

// Select initializes a SELECT query for the specified bundle and fields.
// If no fields are specified, SELECT * is assumed.
func (qb *QueryBuilder) Select(bundle string, fields ...string) *QueryBuilder {
	qb.bundle = bundle
	qb.fields = fields
	qb.queryType = selectQuery
	return qb
}

// Where adds a WHERE condition with implicit AND connector.
// Subsequent calls to Where() are combined with AND.
func (qb *QueryBuilder) Where(field string, op Operator, value interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return qb
}

// And explicitly adds a WHERE condition with AND connector.
// Functionally equivalent to Where() but more explicit in complex queries.
func (qb *QueryBuilder) And(field string, op Operator, value interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return qb
}

// Or adds a WHERE condition with OR connector.
func (qb *QueryBuilder) Or(field string, op Operator, value interface{}) *QueryBuilder {
	qb.whereClauses = append(qb.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: Or,
	})
	return qb
}

// OrderBy adds an ORDER BY clause.
func (qb *QueryBuilder) OrderBy(field string, dir Direction) *QueryBuilder {
	qb.orderBys = append(qb.orderBys, orderByClause{
		field:     field,
		direction: dir,
	})
	return qb
}

// Limit sets the maximum number of results to return.
func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limitVal = &n
	return qb
}

// Offset sets the number of results to skip.
func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offsetVal = &n
	return qb
}

// WithValidation enables or disables schema validation for this query.
// Validation is disabled by default for maximum performance.
func (qb *QueryBuilder) WithValidation(enabled bool) *QueryBuilder {
	qb.schemaValidation = enabled
	return qb
}

// Include adds a relationship for eager loading via JOIN.
// The relationship name should match the relationship defined in the schema.
func (qb *QueryBuilder) Include(relationship string) *QueryBuilder {
	qb.includes = append(qb.includes, relationship)
	return qb
}

// LeftJoin adds an explicit LEFT JOIN clause with ON condition.
// Usage: LeftJoin("Orders", "Orders.CustomerId", "Customers.Id")
// The onSourceField refers to the field in the joining table,
// and onTargetField refers to the field in the source/previously joined table.
func (qb *QueryBuilder) LeftJoin(targetBundle, onSourceField, onTargetField string) *QueryBuilder {
	qb.joinClauses = append(qb.joinClauses, joinClause{
		joinType:      "LEFT",
		targetBundle:  targetBundle,
		onSourceField: onSourceField,
		onTargetField: onTargetField,
	})
	return qb
}

// InnerJoin adds an INNER JOIN clause with ON condition.
// Usage: InnerJoin("Orders", "Orders.CustomerId", "Customers.Id")
func (qb *QueryBuilder) InnerJoin(targetBundle, onSourceField, onTargetField string) *QueryBuilder {
	qb.joinClauses = append(qb.joinClauses, joinClause{
		joinType:      "INNER",
		targetBundle:  targetBundle,
		onSourceField: onSourceField,
		onTargetField: onTargetField,
	})
	return qb
}

// RightJoin adds a RIGHT JOIN clause with ON condition.
// Usage: RightJoin("Orders", "Orders.CustomerId", "Customers.Id")
func (qb *QueryBuilder) RightJoin(targetBundle, onSourceField, onTargetField string) *QueryBuilder {
	qb.joinClauses = append(qb.joinClauses, joinClause{
		joinType:      "RIGHT",
		targetBundle:  targetBundle,
		onSourceField: onSourceField,
		onTargetField: onTargetField,
	})
	return qb
}

// ============================================================================
// InsertBuilder Methods
// ============================================================================

// Values sets the field values for the INSERT operation.
// Note: Server handles multiple INSERT calls as batches automatically.
func (ib *InsertBuilder) Values(data map[string]interface{}) *InsertBuilder {
	ib.values = data
	return ib
}

// WithValidation enables or disables schema validation for this insert.
func (ib *InsertBuilder) WithValidation(enabled bool) *InsertBuilder {
	ib.schemaValidation = enabled
	return ib
}

// ============================================================================
// UpdateBuilder Methods
// ============================================================================

// Set adds or updates a field to be set in the UPDATE operation.
func (ub *UpdateBuilder) Set(field string, value interface{}) *UpdateBuilder {
	if ub.setFields == nil {
		ub.setFields = make(map[string]interface{})
	}
	ub.setFields[field] = value
	return ub
}

// Where adds a WHERE condition for the UPDATE operation.
func (ub *UpdateBuilder) Where(field string, op Operator, value interface{}) *UpdateBuilder {
	ub.whereClauses = append(ub.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return ub
}

// And adds a WHERE condition with AND connector.
func (ub *UpdateBuilder) And(field string, op Operator, value interface{}) *UpdateBuilder {
	ub.whereClauses = append(ub.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return ub
}

// Or adds a WHERE condition with OR connector.
func (ub *UpdateBuilder) Or(field string, op Operator, value interface{}) *UpdateBuilder {
	ub.whereClauses = append(ub.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: Or,
	})
	return ub
}

// WithValidation enables or disables schema validation for this update.
func (ub *UpdateBuilder) WithValidation(enabled bool) *UpdateBuilder {
	ub.schemaValidation = enabled
	return ub
}

// ============================================================================
// DeleteBuilder Methods
// ============================================================================

// Where adds a WHERE condition for the DELETE operation.
func (db *DeleteBuilder) Where(field string, op Operator, value interface{}) *DeleteBuilder {
	db.whereClauses = append(db.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return db
}

// And adds a WHERE condition with AND connector.
func (db *DeleteBuilder) And(field string, op Operator, value interface{}) *DeleteBuilder {
	db.whereClauses = append(db.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: And,
	})
	return db
}

// Or adds a WHERE condition with OR connector.
func (db *DeleteBuilder) Or(field string, op Operator, value interface{}) *DeleteBuilder {
	db.whereClauses = append(db.whereClauses, whereClause{
		field:     field,
		operator:  op,
		value:     value,
		connector: Or,
	})
	return db
}

// WithValidation enables or disables schema validation for this delete.
func (db *DeleteBuilder) WithValidation(enabled bool) *DeleteBuilder {
	db.schemaValidation = enabled
	return db
}

// ============================================================================
// Execute Methods
// ============================================================================

// Execute builds and executes the SELECT query, returning results.
func (qb *QueryBuilder) Execute(ctx context.Context) (interface{}, error) {
	if qb.bundle == "" {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle name is required",
		}
	}

	// Build the query string
	query, params, err := qb.buildQuery()
	if err != nil {
		return nil, err
	}

	// TODO: Validate schema if enabled
	if qb.schemaValidation && qb.client.schemaValidator != nil {
		if err := qb.client.schemaValidator.ValidateQuery(qb.bundle, qb.fields, qb.whereClauses); err != nil {
			return nil, err
		}
	}

	// For now, inline parameters into query (prepared statements not yet fully supported)
	inlineQuery := inlineParameters(query, params)

	// Execute query using Query method
	return qb.client.Query(inlineQuery, 10000)
}

// Execute builds and executes the INSERT query, returning the result.
func (ib *InsertBuilder) Execute(ctx context.Context) (interface{}, error) {
	if ib.bundle == "" {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle name is required",
		}
	}
	if len(ib.values) == 0 {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "no values specified for insert",
		}
	}

	// Build the query string
	query, params := ib.buildInsertQuery()

	// TODO: Validate schema if enabled
	if ib.schemaValidation && ib.client.schemaValidator != nil {
		if err := ib.client.schemaValidator.ValidateInsert(ib.bundle, ib.values); err != nil {
			return nil, err
		}
	}

	// For now, inline parameters into query (prepared statements not yet fully supported)
	inlineQuery := inlineParameters(query, params)

	// Execute mutation using Mutate method
	return ib.client.Mutate(inlineQuery, 10000)
}

// Execute builds and executes the UPDATE query, returning the result.
func (ub *UpdateBuilder) Execute(ctx context.Context) (interface{}, error) {
	if ub.bundle == "" {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle name is required",
		}
	}
	if len(ub.setFields) == 0 {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "no fields to update",
		}
	}
	if len(ub.whereClauses) == 0 {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "WHERE clause required for UPDATE (use Where() to specify conditions)",
		}
	}

	// Build the query string
	query, params := ub.buildUpdateQuery()

	// TODO: Validate schema if enabled
	if ub.schemaValidation && ub.client.schemaValidator != nil {
		if err := ub.client.schemaValidator.ValidateUpdate(ub.bundle, ub.setFields, ub.whereClauses); err != nil {
			return nil, err
		}
	}

	// For now, inline parameters into query (prepared statements not yet fully supported)
	inlineQuery := inlineParameters(query, params)

	// Execute mutation
	return ub.client.Mutate(inlineQuery, 10000)
}

// Execute builds and executes the DELETE query, returning the result.
func (db *DeleteBuilder) Execute(ctx context.Context) (interface{}, error) {
	if db.bundle == "" {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle name is required",
		}
	}
	if len(db.whereClauses) == 0 {
		return nil, &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "WHERE clause required for DELETE (use Where() to specify conditions)",
		}
	}

	// Build the query string
	query, params := db.buildDeleteQuery()

	// TODO: Validate schema if enabled
	if db.schemaValidation && db.client.schemaValidator != nil {
		if err := db.client.schemaValidator.ValidateDelete(db.bundle, db.whereClauses); err != nil {
			return nil, err
		}
	}

	// For now, inline parameters into query (prepared statements not yet fully supported)
	inlineQuery := inlineParameters(query, params)

	// Execute mutation
	return db.client.Mutate(inlineQuery, 10000)
}

// ============================================================================
// Query Building Helpers
// ============================================================================

// buildQuery constructs the SELECT query string with parameterized values.
func (qb *QueryBuilder) buildQuery() (string, []interface{}, error) {
	var query strings.Builder
	var params []interface{}
	paramCount := 0

	// SELECT clause
	query.WriteString("SELECT ")
	if len(qb.fields) == 0 {
		query.WriteString("*")
	} else {
		for i, field := range qb.fields {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteString(field)
		}
	}

	// FROM clause
	query.WriteString(" FROM ")
	query.WriteString(qb.bundle)

	// JOIN clauses from Include() relationships
	if len(qb.includes) > 0 && qb.client.schemaValidator != nil {
		// Fetch schema to resolve relationships
		ctx := context.Background()
		schemaDefn, err := qb.client.schemaValidator.getSchema(ctx)
		if err == nil && schemaDefn != nil {
			// Find the current bundle in schema
			for _, bundle := range schemaDefn.Bundles {
				if bundle.Name == qb.bundle {
					// Search relationships in this bundle
					for _, relationshipName := range qb.includes {
						for _, rel := range bundle.Relationships {
							if rel.Name == relationshipName {
								// Generate JOIN based on relationship
								query.WriteString(" LEFT JOIN ")
								query.WriteString(rel.DestBundle)
								query.WriteString(" ON ")
								query.WriteString(qb.bundle)
								query.WriteString(".")
								query.WriteString(rel.SourceField)
								query.WriteString(" = ")
								query.WriteString(rel.DestBundle)
								query.WriteString(".")
								query.WriteString(rel.DestField)
								break
							}
						}
					}
					break
				}
			}
		}
	}

	// Explicit JOIN clauses
	for _, join := range qb.joinClauses {
		query.WriteString(" ")
		query.WriteString(join.joinType)
		query.WriteString(" JOIN ")
		query.WriteString(join.targetBundle)
		query.WriteString(" ON ")
		query.WriteString(join.onSourceField)
		query.WriteString(" = ")
		query.WriteString(join.onTargetField)
	}

	// WHERE clause
	if len(qb.whereClauses) > 0 {
		query.WriteString(" WHERE ")
		for i, clause := range qb.whereClauses {
			if i > 0 {
				query.WriteString(" ")
				query.WriteString(clause.connector.String())
				query.WriteString(" ")
			}

			// Handle dot-notation for relationship traversal (e.g., "Author.Name")
			// Dot-notation allows querying related bundle fields directly
			query.WriteString(clause.field)
			query.WriteString(" ")
			query.WriteString(clause.operator.String())

			// Handle NULL checks specially (no parameter)
			if clause.operator == IsNull || clause.operator == IsNotNull {
				// No parameter needed
			} else {
				paramCount++
				query.WriteString(" $")
				query.WriteString(strconv.Itoa(paramCount))
				params = append(params, clause.value)
			}
		}
	}

	// ORDER BY clause
	if len(qb.orderBys) > 0 {
		query.WriteString(" ORDER BY ")
		for i, orderBy := range qb.orderBys {
			if i > 0 {
				query.WriteString(", ")
			}
			query.WriteString(orderBy.field)
			query.WriteString(" ")
			query.WriteString(orderBy.direction.String())
		}
	}

	// LIMIT clause
	if qb.limitVal != nil {
		query.WriteString(" LIMIT ")
		query.WriteString(strconv.Itoa(*qb.limitVal))
	}

	// OFFSET clause
	if qb.offsetVal != nil {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(*qb.offsetVal))
	}

	query.WriteString(";")

	return query.String(), params, nil
}

// buildInsertQuery constructs the INSERT query string with parameterized values.
func (ib *InsertBuilder) buildInsertQuery() (string, []interface{}) {
	var query strings.Builder
	var params []interface{}

	// Add DOCUMENTS clause
	query.WriteString("ADD DOCUMENT TO BUNDLE  ")
	query.WriteString("\"" + ib.bundle + "\"")
	query.WriteString(" WITH (")

	// Field names

	fieldCount := 1
	for field, value := range ib.values {
		query.WriteString("{")
		query.WriteString("\"" + field + "\"")
		query.WriteString(" =  ")
		//Determine the data type
		switch value.(type) {
		case string:
			query.WriteString("\"" + fmt.Sprintf("%v", value) + "\"")
		default:
			query.WriteString(fmt.Sprintf("%v", value))
		}

		query.WriteString("}")
		// If this is the last field, don't add a comma
		if fieldCount < len(ib.values) {
			query.WriteString(", ")
		}
		fieldCount++

	}
	//fieldOrder = append(fieldOrder, field)

	// for _, field := range fieldOrder {
	// 	if !first {
	// 		query.WriteString(", ")
	// 	}
	// 	query.WriteString(field)
	// 	first = false
	// }

	//query.WriteString(") VALUES (")

	// Values as parameters
	// first = true
	// for _, field := range fieldOrder {
	// 	if !first {
	// 		query.WriteString(", ")
	// 	}
	// 	paramCount++
	// 	query.WriteString("$")
	// 	query.WriteString(strconv.Itoa(paramCount))
	// 	params = append(params, ib.values[field])
	// 	first = false
	// }

	query.WriteString(");")

	return query.String(), params
}

// buildUpdateQuery constructs the UPDATE query string with parameterized values.
func (ub *UpdateBuilder) buildUpdateQuery() (string, []interface{}) {

	/* PROPER SyndrQL Syntax:

		UPDATE DOCUMENTS IN BUNDLE "Authors"
	 ("AuthorName" = "Dan Strohschein-669" )
	 WHERE "DocumentID" == "187320fc9a770e28_33";

	*/

	var query strings.Builder
	var params []interface{}
	paramCount := 0

	// UPDATE clause
	query.WriteString("UPDATE DOCUMENTS IN BUNDLE \"")
	query.WriteString(ub.bundle)
	query.WriteString("\" ( ")

	// SET clause
	first := true
	fieldCount := 1
	for field, value := range ub.setFields {
		if !first {
			query.WriteString(", ")
		}
		query.WriteString("\"" + field + "\"")
		query.WriteString(" = ")
		switch value.(type) {
		case string:
			query.WriteString("\"" + fmt.Sprintf("%v", value) + "\"")
		default:
			query.WriteString(fmt.Sprintf("%v", value))
		}
		//paramCount++
		//query.WriteString(strconv.Itoa(paramCount))
		//params = append(params, value)
		//first = false
		if fieldCount < len(ub.setFields) {
			query.WriteString(", ")
		}
		fieldCount++

	}

	query.WriteString(")")

	// WHERE clause
	query.WriteString(" WHERE ")
	for i, clause := range ub.whereClauses {
		if i > 0 {
			query.WriteString(" ")
			query.WriteString(clause.connector.String())
			query.WriteString(" ")
		}

		query.WriteString("\"" + clause.field + "\"")
		query.WriteString(" ")
		query.WriteString(clause.operator.String())

		if clause.operator == IsNull || clause.operator == IsNotNull {
			// No parameter needed
		} else {
			paramCount++
			//query.WriteString(" $")
			switch clause.value.(type) {
			case string:
				query.WriteString(" \"" + fmt.Sprintf("%v", clause.value) + "\"")
			default:
				query.WriteString(" " + fmt.Sprintf("%v", clause.value))
			}

			//query.WriteString(strconv.Itoa(paramCount))
			//params = append(params, clause.value)
		}
	}

	query.WriteString(";")

	return query.String(), params
}

// buildDeleteQuery constructs the DELETE query string with parameterized values.
func (db *DeleteBuilder) buildDeleteQuery() (string, []interface{}) {
	var query strings.Builder
	var params []interface{}
	paramCount := 0

	// DELETE DOCUMENTS FROM clause
	query.WriteString("DELETE DOCUMENTS FROM \"")
	query.WriteString(db.bundle)
	query.WriteString("\"")

	// WHERE clause
	query.WriteString(" WHERE ")
	for i, clause := range db.whereClauses {
		if i > 0 {
			query.WriteString(" ")
			query.WriteString(clause.connector.String())
			query.WriteString(" ")
		}

		// Field names are always quoted
		query.WriteString("\"" + clause.field + "\"")
		query.WriteString(" ")
		query.WriteString(clause.operator.String())

		if clause.operator == IsNull || clause.operator == IsNotNull {
			// No parameter needed
		} else {
			paramCount++
			//query.WriteString(" $")
			//query.WriteString(strconv.Itoa(paramCount))
			//params = append(params, clause.value)
			//query.WriteString(" $")
			switch clause.value.(type) {
			case string:
				query.WriteString(" \"" + fmt.Sprintf("%v", clause.value) + "\"")
			default:
				query.WriteString(" " + fmt.Sprintf("%v", clause.value))
			}
		}
	}

	query.WriteString(";")

	return query.String(), params
}

// ============================================================================
// Fingerprinting for Query Caching
// ============================================================================

// Fingerprint generates a unique cache key for the query pattern using xxhash.
// This enables statement caching for frequently executed query patterns.
func (qb *QueryBuilder) Fingerprint() string {
	var pattern strings.Builder

	// Bundle and query type
	pattern.WriteString(qb.bundle)
	pattern.WriteString(":")
	switch qb.queryType {
	case selectQuery:
		pattern.WriteString("SELECT")
	case insertQuery:
		pattern.WriteString("INSERT")
	case updateQuery:
		pattern.WriteString("UPDATE")
	case deleteQuery:
		pattern.WriteString("DELETE")
	}

	// Fields
	pattern.WriteString(":")
	if len(qb.fields) > 0 {
		pattern.WriteString(strings.Join(qb.fields, ","))
	} else {
		pattern.WriteString("*")
	}

	// WHERE operators (not values, just structure)
	if len(qb.whereClauses) > 0 {
		pattern.WriteString(":WHERE:")
		for i, clause := range qb.whereClauses {
			if i > 0 {
				pattern.WriteString(",")
			}
			pattern.WriteString(clause.field)
			pattern.WriteString(clause.operator.String())
		}
	}

	// ORDER BY
	if len(qb.orderBys) > 0 {
		pattern.WriteString(":ORDER:")
		for i, orderBy := range qb.orderBys {
			if i > 0 {
				pattern.WriteString(",")
			}
			pattern.WriteString(orderBy.field)
			pattern.WriteString(orderBy.direction.String())
		}
	}

	// LIMIT/OFFSET
	if qb.limitVal != nil {
		pattern.WriteString(":LIMIT:")
		pattern.WriteString(strconv.Itoa(*qb.limitVal))
	}
	if qb.offsetVal != nil {
		pattern.WriteString(":OFFSET:")
		pattern.WriteString(strconv.Itoa(*qb.offsetVal))
	}

	// Hash with xxhash for speed
	hash := xxhash.Sum64String(pattern.String())
	return fmt.Sprintf("qb_%016x", hash)
}

// ============================================================================
// Helper Functions
// ============================================================================

// inlineParameters replaces parameter placeholders ($1, $2, etc.) with actual values.
// This is a temporary solution until full prepared statement support is available.
func inlineParameters(query string, params []interface{}) string {
	result := query
	for i, param := range params {
		placeholder := fmt.Sprintf("$%d", i+1)
		value := formatParameterValue(param)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// formatParameterValue converts a parameter value to its string representation for inline SQL.
func formatParameterValue(param interface{}) string {
	if param == nil {
		return "NULL"
	}

	switch v := param.(type) {
	case string:
		// Escape single quotes in strings
		escaped := strings.ReplaceAll(v, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		if v {
			return "TRUE"
		}
		return "FALSE"
	default:
		// For other types, convert to string and quote
		str := fmt.Sprintf("%v", v)
		escaped := strings.ReplaceAll(str, "'", "''")
		return fmt.Sprintf("'%s'", escaped)
	}
}
