package client

import (
	"context"
	"strings"
	"testing"
)

// ============================================================================
// Operator Tests
// ============================================================================

func TestOperatorString(t *testing.T) {
	tests := []struct {
		name     string
		operator Operator
		expected string
	}{
		{"Equals", Equals, "="},
		{"NotEquals", NotEquals, "!="},
		{"GreaterThan", GreaterThan, ">"},
		{"LessThan", LessThan, "<"},
		{"GreaterThanOrEqual", GreaterThanOrEqual, ">="},
		{"LessThanOrEqual", LessThanOrEqual, "<="},
		{"Like", Like, "LIKE"},
		{"ILike", ILike, "ILIKE"},
		{"NotLike", NotLike, "NOT LIKE"},
		{"NotILike", NotILike, "NOT ILIKE"},
		{"In", In, "IN"},
		{"NotIn", NotIn, "NOT IN"},
		{"IsNull", IsNull, "IS NULL"},
		{"IsNotNull", IsNotNull, "IS NOT NULL"},
		{"And", And, "AND"},
		{"Or", Or, "OR"},
		{"Not", Not, "NOT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.operator.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestDirectionString(t *testing.T) {
	tests := []struct {
		name      string
		direction Direction
		expected  string
	}{
		{"Ascending", Ascending, "ASC"},
		{"Descending", Descending, "DESC"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.direction.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

// ============================================================================
// QueryBuilder SELECT Tests
// ============================================================================

func TestQueryBuilder_SimpleSelect(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users", "id", "name", "email")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT id, name, email FROM Users;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestQueryBuilder_SelectAllFields(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 0 {
		t.Errorf("Expected 0 params, got %d", len(params))
	}
}

func TestQueryBuilder_WhereClause(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Where("age", GreaterThan, 18)

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE age > $1;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 1 || params[0] != 18 {
		t.Errorf("Expected params [18], got %v", params)
	}
}

func TestQueryBuilder_MultipleWhereWithAnd(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").
		Where("age", GreaterThan, 18).
		And("status", Equals, "active")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE age > $1 AND status = $2;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 2 || params[0] != 18 || params[1] != "active" {
		t.Errorf("Expected params [18, active], got %v", params)
	}
}

func TestQueryBuilder_WhereWithOr(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").
		Where("role", Equals, "admin").
		Or("role", Equals, "moderator")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE role = $1 OR role = $2;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 2 || params[0] != "admin" || params[1] != "moderator" {
		t.Errorf("Expected params [admin, moderator], got %v", params)
	}
}

func TestQueryBuilder_ComplexWhereClause(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").
		Where("age", GreaterThanOrEqual, 18).
		And("age", LessThan, 65).
		And("status", Equals, "active").
		Or("role", Equals, "admin")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE age >= $1 AND age < $2 AND status = $3 OR role = $4;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 4 {
		t.Errorf("Expected 4 params, got %d", len(params))
	}
}

func TestQueryBuilder_IsNullOperator(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Where("deletedAt", IsNull, nil)

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE deletedAt IS NULL;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 0 {
		t.Errorf("Expected 0 params for IS NULL, got %d", len(params))
	}
}

func TestQueryBuilder_IsNotNullOperator(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Where("email", IsNotNull, nil)

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE email IS NOT NULL;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 0 {
		t.Errorf("Expected 0 params for IS NOT NULL, got %d", len(params))
	}
}

func TestQueryBuilder_LikeOperator(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Where("name", Like, "%John%")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE name LIKE $1;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 1 || params[0] != "%John%" {
		t.Errorf("Expected params [%%John%%], got %v", params)
	}
}

func TestQueryBuilder_InOperator(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Where("role", In, []string{"admin", "moderator", "user"})

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users WHERE role IN $1;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 1 {
		t.Errorf("Expected 1 param, got %d", len(params))
	}
}

func TestQueryBuilder_OrderBy(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").OrderBy("name", Ascending)

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users ORDER BY name ASC;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_MultipleOrderBy(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").
		OrderBy("status", Ascending).
		OrderBy("createdAt", Descending)

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users ORDER BY status ASC, createdAt DESC;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_LimitAndOffset(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Limit(10).Offset(20)

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Users LIMIT 10 OFFSET 20;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_FullQuery(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users", "id", "name", "email").
		Where("age", GreaterThan, 18).
		And("status", Equals, "active").
		OrderBy("name", Ascending).
		Limit(50).
		Offset(100)

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT id, name, email FROM Users WHERE age > $1 AND status = $2 ORDER BY name ASC LIMIT 50 OFFSET 100;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 2 || params[0] != 18 || params[1] != "active" {
		t.Errorf("Expected params [18, active], got %v", params)
	}
}

// ============================================================================
// JOIN Tests
// ============================================================================

func TestQueryBuilder_LeftJoin(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Orders").LeftJoin("Customers", "Orders.customerId", "Customers.id")

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Orders LEFT JOIN Customers ON Orders.customerId = Customers.id;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_InnerJoin(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Orders").InnerJoin("Customers", "Orders.customerId", "Customers.id")

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Orders INNER JOIN Customers ON Orders.customerId = Customers.id;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_RightJoin(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Orders").RightJoin("Customers", "Orders.customerId", "Customers.id")

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Orders RIGHT JOIN Customers ON Orders.customerId = Customers.id;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_MultipleJoins(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Orders").
		LeftJoin("Customers", "Orders.customerId", "Customers.id").
		InnerJoin("Products", "Orders.productId", "Products.id")

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	expected := "SELECT * FROM Orders LEFT JOIN Customers ON Orders.customerId = Customers.id INNER JOIN Products ON Orders.productId = Products.id;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}
}

func TestQueryBuilder_JoinWithWhere(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Orders").
		LeftJoin("Customers", "Orders.customerId", "Customers.id").
		Where("Customers.country", Equals, "USA")

	query, params, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	if !strings.Contains(query, "LEFT JOIN") {
		t.Error("Expected query to contain LEFT JOIN")
	}
	if !strings.Contains(query, "WHERE Customers.country = $1") {
		t.Error("Expected query to contain WHERE clause with dot-notation")
	}

	if len(params) != 1 || params[0] != "USA" {
		t.Errorf("Expected params [USA], got %v", params)
	}
}

// ============================================================================
// InsertBuilder Tests
// ============================================================================

func TestInsertBuilder_SimpleInsert(t *testing.T) {
	client := &Client{}
	ib := &InsertBuilder{client: client, bundle: "Users"}
	ib.Values(map[string]interface{}{
		"name":  "John Doe",
		"email": "john@example.com",
		"age":   30,
	})

	query, params := ib.buildInsertQuery()

	if !strings.HasPrefix(query, "ADD DOCUMENT TO \"Users\" (") {
		t.Error("Expected query to start with ADD DOCUMENT TO \"Users\" (")
	}
	if !strings.Contains(query, "WITH (") {
		t.Error("Expected query to contain VALUES (")
	}
	if !strings.HasSuffix(query, ");") {
		t.Error("Expected query to end with );")
	}

	if len(params) != 3 {
		t.Errorf("Expected 3 params, got %d", len(params))
	}
}

func TestInsertBuilder_EmptyValues(t *testing.T) {
	client := &Client{}
	ib := &InsertBuilder{client: client, bundle: "Users"}

	ctx := context.Background()
	_, err := ib.Execute(ctx)
	if err == nil {
		t.Error("Expected error for empty values")
	}

	qe, ok := err.(*QueryError)
	if !ok {
		t.Error("Expected QueryError type")
	}
	if qe.Code != "E_INVALID_QUERY" {
		t.Errorf("Expected error code E_INVALID_QUERY, got %s", qe.Code)
	}
}

// ============================================================================
// UpdateBuilder Tests
// ============================================================================
/*

valid syntax:
UPDATE DOCUMENTS IN BUNDLE "Authors"
 ("AuthorName" = "Dan Strohschein-669" )
 WHERE "DocumentID" == "187320fc9a770e28_33";


*/
func TestUpdateBuilder_SimpleUpdate(t *testing.T) {
	client := &Client{}
	ub := &UpdateBuilder{client: client, bundle: "Users"}
	ub.Set("name", "Jane Doe").
		Set("email", "jane@example.com").
		Where("id", Equals, 123)

	query, params := ub.buildUpdateQuery()

	expected := "UPDATE DOCUMENTS IN BUNDLE \"Users\" (\"name\" = \"Jane Doe\", \"email\" = \"jane@example.com\") WHERE \"id\" == 123;"
	if !strings.HasPrefix(query, expected) {
		t.Errorf("Expected query to start with '%s'", expected)
	}
	if !strings.Contains(query, "WHERE id = ") {
		t.Error("Expected query to contain WHERE clause")
	}

	if len(params) != 3 {
		t.Errorf("Expected 3 params (2 SET + 1 WHERE), got %d", len(params))
	}
}

func TestUpdateBuilder_NoWhereClause(t *testing.T) {
	client := &Client{}
	ub := &UpdateBuilder{client: client, bundle: "Users"}
	ub.Set("name", "Jane Doe")

	ctx := context.Background()
	_, err := ub.Execute(ctx)
	if err == nil {
		t.Error("Expected error for UPDATE without WHERE clause")
	}

	qe, ok := err.(*QueryError)
	if !ok {
		t.Error("Expected QueryError type")
	}
	if !strings.Contains(qe.Message, "WHERE clause required") {
		t.Errorf("Expected error message about WHERE clause, got: %s", qe.Message)
	}
}

func TestUpdateBuilder_NoSetFields(t *testing.T) {
	client := &Client{}
	ub := &UpdateBuilder{client: client, bundle: "Users"}
	ub.Where("id", Equals, 123)

	ctx := context.Background()
	_, err := ub.Execute(ctx)
	if err == nil {
		t.Error("Expected error for UPDATE without SET fields")
	}

	qe, ok := err.(*QueryError)
	if !ok {
		t.Error("Expected QueryError type")
	}
	if !strings.Contains(qe.Message, "no fields to update") {
		t.Errorf("Expected error message about no fields, got: %s", qe.Message)
	}
}

func TestUpdateBuilder_MultipleWhereConditions(t *testing.T) {
	client := &Client{}
	ub := &UpdateBuilder{client: client, bundle: "Users"}
	ub.Set("status", "inactive").
		Where("lastLoginAt", LessThan, "2023-01-01").
		And("role", NotEquals, "admin")

	query, params := ub.buildUpdateQuery()

	if !strings.Contains(query, "WHERE lastLoginAt <") {
		t.Error("Expected query to contain WHERE clause")
	}
	if !strings.Contains(query, "AND role !=") {
		t.Error("Expected query to contain AND clause")
	}

	if len(params) != 3 {
		t.Errorf("Expected 3 params, got %d", len(params))
	}
}

// ============================================================================
// DeleteBuilder Tests
// ============================================================================

func TestDeleteBuilder_SimpleDelete(t *testing.T) {
	client := &Client{}
	db := &DeleteBuilder{client: client, bundle: "Users"}
	db.Where("id", Equals, 123)

	query, params := db.buildDeleteQuery()

	expected := "DELETE FROM Users WHERE id = $1;"
	if query != expected {
		t.Errorf("Expected:\n%s\nGot:\n%s", expected, query)
	}

	if len(params) != 1 || params[0] != 123 {
		t.Errorf("Expected params [123], got %v", params)
	}
}

func TestDeleteBuilder_NoWhereClause(t *testing.T) {
	client := &Client{}
	db := &DeleteBuilder{client: client, bundle: "Users"}

	ctx := context.Background()
	_, err := db.Execute(ctx)
	if err == nil {
		t.Error("Expected error for DELETE without WHERE clause")
	}

	qe, ok := err.(*QueryError)
	if !ok {
		t.Error("Expected QueryError type")
	}
	if !strings.Contains(qe.Message, "WHERE clause required") {
		t.Errorf("Expected error message about WHERE clause, got: %s", qe.Message)
	}
}

func TestDeleteBuilder_MultipleConditions(t *testing.T) {
	client := &Client{}
	db := &DeleteBuilder{client: client, bundle: "Users"}
	db.Where("status", Equals, "inactive").
		And("deletedAt", IsNotNull, nil)

	query, params := db.buildDeleteQuery()

	if !strings.Contains(query, "WHERE status = $1") {
		t.Error("Expected WHERE clause with status")
	}
	if !strings.Contains(query, "AND deletedAt IS NOT NULL") {
		t.Error("Expected AND clause with IS NOT NULL")
	}

	if len(params) != 1 {
		t.Errorf("Expected 1 param, got %d", len(params))
	}
}

// ============================================================================
// Fingerprinting Tests
// ============================================================================

func TestQueryBuilder_Fingerprint(t *testing.T) {
	client := &Client{}

	// Same query pattern should produce same fingerprint
	qb1 := &QueryBuilder{client: client, queryType: selectQuery}
	qb1.Select("Users", "id", "name").Where("age", GreaterThan, 18)

	qb2 := &QueryBuilder{client: client, queryType: selectQuery}
	qb2.Select("Users", "id", "name").Where("age", GreaterThan, 25) // Different value

	fp1 := qb1.Fingerprint()
	fp2 := qb2.Fingerprint()

	// Same structure, different values should produce same fingerprint
	if fp1 != fp2 {
		t.Error("Expected same fingerprint for same query structure with different values")
	}

	// Different query structure should produce different fingerprint
	qb3 := &QueryBuilder{client: client, queryType: selectQuery}
	qb3.Select("Users", "id", "name", "email").Where("age", GreaterThan, 18)

	fp3 := qb3.Fingerprint()
	if fp1 == fp3 {
		t.Error("Expected different fingerprint for different field list")
	}
}

func TestQueryBuilder_FingerprintFormat(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client, queryType: selectQuery}
	qb.Select("Users")

	fp := qb.Fingerprint()

	if !strings.HasPrefix(fp, "qb_") {
		t.Errorf("Expected fingerprint to start with 'qb_', got %s", fp)
	}

	if len(fp) != 19 { // "qb_" + 16 hex chars
		t.Errorf("Expected fingerprint length 19, got %d", len(fp))
	}
}

func TestQueryBuilder_FingerprintUniqueness(t *testing.T) {
	client := &Client{}

	patterns := []struct {
		name  string
		setup func() *QueryBuilder
	}{
		{
			"simple_select",
			func() *QueryBuilder {
				qb := &QueryBuilder{client: client, queryType: selectQuery}
				qb.Select("Users")
				return qb
			},
		},
		{
			"with_fields",
			func() *QueryBuilder {
				qb := &QueryBuilder{client: client, queryType: selectQuery}
				qb.Select("Users", "id", "name")
				return qb
			},
		},
		{
			"with_where",
			func() *QueryBuilder {
				qb := &QueryBuilder{client: client, queryType: selectQuery}
				qb.Select("Users").Where("age", GreaterThan, 18)
				return qb
			},
		},
		{
			"with_order",
			func() *QueryBuilder {
				qb := &QueryBuilder{client: client, queryType: selectQuery}
				qb.Select("Users").OrderBy("name", Ascending)
				return qb
			},
		},
		{
			"with_limit",
			func() *QueryBuilder {
				qb := &QueryBuilder{client: client, queryType: selectQuery}
				qb.Select("Users").Limit(10)
				return qb
			},
		},
	}

	fingerprints := make(map[string]string)
	for _, p := range patterns {
		qb := p.setup()
		fp := qb.Fingerprint()

		if existing, found := fingerprints[fp]; found {
			t.Errorf("Collision detected: %s and %s have same fingerprint %s", p.name, existing, fp)
		}
		fingerprints[fp] = p.name
	}
}

// ============================================================================
// Chaining Tests
// ============================================================================

func TestQueryBuilder_MethodChaining(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}

	// All methods should return *QueryBuilder for chaining
	result := qb.Select("Users").
		Where("age", GreaterThan, 18).
		And("status", Equals, "active").
		Or("role", Equals, "admin").
		OrderBy("name", Ascending).
		Limit(10).
		Offset(5).
		WithValidation(true).
		Include("profile")

	if result != qb {
		t.Error("Method chaining should return same QueryBuilder instance")
	}

	// Verify all operations were applied
	if qb.bundle != "Users" {
		t.Error("Bundle not set correctly")
	}
	if len(qb.whereClauses) != 3 {
		t.Errorf("Expected 3 where clauses, got %d", len(qb.whereClauses))
	}
	if len(qb.orderBys) != 1 {
		t.Errorf("Expected 1 order by, got %d", len(qb.orderBys))
	}
	if qb.limitVal == nil || *qb.limitVal != 10 {
		t.Error("Limit not set correctly")
	}
	if qb.offsetVal == nil || *qb.offsetVal != 5 {
		t.Error("Offset not set correctly")
	}
	if !qb.schemaValidation {
		t.Error("Schema validation not enabled")
	}
	if len(qb.includes) != 1 {
		t.Error("Include not added")
	}
}

func TestInsertBuilder_MethodChaining(t *testing.T) {
	client := &Client{}
	ib := &InsertBuilder{client: client, bundle: "Users"}

	result := ib.Values(map[string]interface{}{
		"name": "John",
	}).WithValidation(true)

	if result != ib {
		t.Error("Method chaining should return same InsertBuilder instance")
	}
}

func TestUpdateBuilder_MethodChaining(t *testing.T) {
	client := &Client{}
	ub := &UpdateBuilder{client: client, bundle: "Users"}

	result := ub.Set("name", "Jane").
		Set("email", "jane@example.com").
		Where("id", Equals, 1).
		And("status", Equals, "active").
		WithValidation(true)

	if result != ub {
		t.Error("Method chaining should return same UpdateBuilder instance")
	}
}

func TestDeleteBuilder_MethodChaining(t *testing.T) {
	client := &Client{}
	db := &DeleteBuilder{client: client, bundle: "Users"}

	result := db.Where("id", Equals, 1).
		And("status", Equals, "deleted").
		WithValidation(true)

	if result != db {
		t.Error("Method chaining should return same DeleteBuilder instance")
	}
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestQueryBuilder_ValidationToggle(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}

	// Default should be disabled
	if qb.schemaValidation {
		t.Error("Schema validation should be disabled by default")
	}

	// Enable validation
	qb.WithValidation(true)
	if !qb.schemaValidation {
		t.Error("Schema validation should be enabled")
	}

	// Disable validation
	qb.WithValidation(false)
	if qb.schemaValidation {
		t.Error("Schema validation should be disabled")
	}
}

// ============================================================================
// Edge Cases
// ============================================================================

func TestQueryBuilder_EmptyBundle(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}

	ctx := context.Background()
	_, err := qb.Execute(ctx)
	if err == nil {
		t.Error("Expected error for empty bundle name")
	}

	qe, ok := err.(*QueryError)
	if !ok {
		t.Error("Expected QueryError type")
	}
	if qe.Code != "E_INVALID_QUERY" {
		t.Errorf("Expected error code E_INVALID_QUERY, got %s", qe.Code)
	}
}

func TestQueryBuilder_NilLimitOffset(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users")

	// Should work fine with nil limit/offset
	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	if strings.Contains(query, "LIMIT") || strings.Contains(query, "OFFSET") {
		t.Error("Query should not contain LIMIT or OFFSET when not set")
	}
}

func TestQueryBuilder_ZeroLimitOffset(t *testing.T) {
	client := &Client{}
	qb := &QueryBuilder{client: client}
	qb.Select("Users").Limit(0).Offset(0)

	query, _, err := qb.buildQuery()
	if err != nil {
		t.Fatalf("buildQuery failed: %v", err)
	}

	if !strings.Contains(query, "LIMIT 0") {
		t.Error("Query should contain LIMIT 0")
	}
	if !strings.Contains(query, "OFFSET 0") {
		t.Error("Query should contain OFFSET 0")
	}
}

// ============================================================================
// Integration Tests (require running SyndrDB server)
// ============================================================================

const (
	integrationTestConnStr = "syndrdb://127.0.0.1:1776:primary:root:root;"
	integrationTestTimeout = 10000
)

// skipIfNoServer skips the test if SyndrDB server is not available
func skipIfNoServer(t *testing.T) *Client {
	opts := DefaultOptions()
	c := NewClient(&opts)

	ctx := context.Background()
	err := c.Connect(ctx, integrationTestConnStr)
	if err != nil {
		t.Skipf("Skipping integration test: SyndrDB server not available: %v", err)
		return nil
	}

	return c
}

// setupTestBundle creates a test bundle and returns a cleanup function
func setupTestBundle(t *testing.T, c *Client, bundleName string) func() {
	ctx := context.Background()

	// Create test bundle
	createCmd := `CREATE BUNDLE "` + bundleName + `"
 WITH FIELDS (
    {"id", "STRING", TRUE, FALSE, ""},
    {"name", "STRING", FALSE, FALSE, ""},
    {"email", "STRING", FALSE, FALSE, ""},
    {"age", "INT", FALSE, FALSE, 0},
    {"status", "STRING", FALSE, FALSE, ""},
    {"createdAt", "DATETIME", FALSE, FALSE, NULL}
);`

	_, err := c.Mutate(createCmd, integrationTestTimeout)
	if err != nil {
		t.Fatalf("Failed to create test bundle: %v", err)
	}

	// Return cleanup function
	return func() {
		dropCmd := "DELETE BUNDLE \"" + bundleName + "\" WITH FORCE;"
		_, _ = c.Mutate(dropCmd, integrationTestTimeout)
		c.Disconnect(ctx)
	}
}

func TestIntegration_QueryBuilder_SimpleSelect(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	insertCmd := `ADD DOCUMENT TO BUNDLE "TestUsers" WITH ({"id"="1"}, {"name"="John Doe"}, {"email"="john@test.com"}, {"age"=30}, {"status"="active"});`
	_, err := c.Mutate(insertCmd, integrationTestTimeout)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test QueryBuilder SELECT
	results, err := c.QueryBuilder().
		Select("TestUsers", "id", "name", "email").
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	t.Logf("Query results: %+v", results)
}

func TestIntegration_QueryBuilder_WhereClause(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers2")
	defer cleanup()

	ctx := context.Background()

	// Insert multiple records
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers2" WITH ({"id"="1"}, {"name"="Alice"}, {"age"=25}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers2" WITH ({"id"="2"}, {"name"="Bob"}, {"age"=35}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers2" WITH ({"id"="3"}, {"name"="Charlie"}, {"age"=45}, {"status"="inactive"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test WHERE with GreaterThan
	results, err := c.QueryBuilder().
		Select("TestUsers2").
		Where("age", GreaterThan, 30).
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute failed: %v", err)
	}

	if results == nil {
		t.Fatal("Expected results, got nil")
	}

	t.Logf("WHERE age > 30 results: %+v", results)
}

func TestIntegration_QueryBuilder_AndOrConditions(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers3")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers3" WITH ({"id"="1"}, {"name"="Alice"}, {"age"=25}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers3" WITH ({"id"="2"}, {"name"="Bob"}, {"age"=35}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers3" WITH ({"id"="3"}, {"name"="Charlie"}, {"age"=45}, {"status"="inactive"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers3" WITH ({"id"="4"}, {"name"="David"}, {"age"=28}, {"status"="active"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test AND conditions
	results, err := c.QueryBuilder().
		Select("TestUsers3").
		Where("age", GreaterThanOrEqual, 25).
		And("status", Equals, "active").
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with AND failed: %v", err)
	}

	t.Logf("AND condition results: %+v", results)

	// Test OR conditions
	results2, err := c.QueryBuilder().
		Select("TestUsers3").
		Where("age", LessThan, 30).
		Or("status", Equals, "inactive").
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with OR failed: %v", err)
	}

	t.Logf("OR condition results: %+v", results2)
}

func TestIntegration_QueryBuilder_OrderByLimitOffset(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers4")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers4" WITH ({"id"="1"}, {"name"="Alice"}, {"age"=25});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers4" WITH ({"id"="2"}, {"name"="Bob"}, {"age"=35});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers4" WITH ({"id"="3"}, {"name"="Charlie"}, {"age"=45});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers4" WITH ({"id"="4"}, {"name"="David"}, {"age"=28});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers4" WITH ({"id"="5"}, {"name"="Eve"}, {"age"=32});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test ORDER BY with LIMIT and OFFSET
	results, err := c.QueryBuilder().
		Select("TestUsers4", "id", "name", "age").
		OrderBy("age", Ascending).
		Limit(2).
		Offset(1).
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with ORDER BY/LIMIT/OFFSET failed: %v", err)
	}

	t.Logf("ORDER BY age ASC LIMIT 2 OFFSET 1 results: %+v", results)
}

func TestIntegration_QueryBuilder_IsNullOperator(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers5")
	defer cleanup()

	ctx := context.Background()

	// Insert records with and without email
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers5" WITH ({"id"="1"}, {"name"="Alice"}, {"email"="alice@test.com"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers5" WITH ({"id"="2"}, {"name"="Bob"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers5" WITH ({"id"="3"}, {"name"="Charlie"}, {"email"="charlie@test.com"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test IS NULL
	results, err := c.QueryBuilder().
		Select("TestUsers5").
		Where("email", IsNull, nil).
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with IS NULL failed: %v", err)
	}

	t.Logf("IS NULL results: %+v", results)

	// Test IS NOT NULL
	results2, err := c.QueryBuilder().
		Select("TestUsers5").
		Where("email", IsNotNull, nil).
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with IS NOT NULL failed: %v", err)
	}

	t.Logf("IS NOT NULL results: %+v", results2)
}

func TestIntegration_InsertBuilder(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers6")
	defer cleanup()

	ctx := context.Background()

	// Test InsertBuilder
	result, err := c.InsertBuilder("TestUsers6").
		Values(map[string]interface{}{
			"id":     "test-1",
			"name":   "Test User",
			"email":  "test@example.com",
			"age":    30,
			"status": "active",
		}).
		Execute(ctx)

	if err != nil {
		t.Fatalf("InsertBuilder Execute failed: %v", err)
	}

	t.Logf("Insert result: %+v", result)

	// Verify insertion with SELECT
	results, err := c.QueryBuilder().
		Select("TestUsers6").
		Where("id", Equals, "test-1").
		Execute(ctx)

	if err != nil {
		t.Fatalf("Failed to verify insertion: %v", err)
	}

	t.Logf("Verification query results: %+v", results)
}

func TestIntegration_UpdateBuilder(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers7")
	defer cleanup()

	ctx := context.Background()

	// Insert initial data
	_, err := c.Mutate(`ADD DOCUMENT TO BUNDLE "TestUsers7" WITH ({"id"="1"}, {"name"="Alice"}, {"status"="active"});`, integrationTestTimeout)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test UpdateBuilder
	result, err := c.UpdateBuilder("TestUsers7").
		Set("status", "inactive").
		Set("name", "Alice Updated").
		Where("id", Equals, "1").
		Execute(ctx)

	if err != nil {
		t.Fatalf("UpdateBuilder Execute failed: %v", err)
	}

	t.Logf("Update result: %+v", result)

	// Check if result contains an error from server
	if resultMap, ok := result.(map[string]interface{}); ok {
		if status, hasStatus := resultMap["status"]; hasStatus && status == "error" {
			t.Fatalf("Server returned error: %v", resultMap["message"])
		}
	}

	// Verify update with SELECT
	results, err := c.QueryBuilder().
		Select("TestUsers7").
		Where("id", Equals, "1").
		Execute(ctx)

	if err != nil {
		t.Fatalf("Failed to verify update: %v", err)
	}

	t.Logf("Verification query results: %+v", results)
}

func TestIntegration_DeleteBuilder(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers8")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers8" WITH ({"id"="1"}, {"name"="Alice"}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers8" WITH ({"id"="2"}, {"name"="Bob"}, {"status"="inactive"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers8" WITH ({"id"="3"}, {"name"="Charlie"}, {"status"="inactive"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test DeleteBuilder
	result, err := c.DeleteBuilder("TestUsers8").
		Where("status", Equals, "inactive").
		Execute(ctx)

	if err != nil {
		t.Fatalf("DeleteBuilder Execute failed: %v", err)
	}

	t.Logf("Delete result: %+v", result)

	// Check if result contains an error from server
	if resultMap, ok := result.(map[string]interface{}); ok {
		if status, hasStatus := resultMap["status"]; hasStatus && status == "error" {
			t.Fatalf("Server returned error: %v", resultMap["message"])
		}
	}

	// Verify deletion with SELECT
	results, err := c.QueryBuilder().
		Select("TestUsers8").
		Execute(ctx)

	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}

	t.Logf("Remaining records after delete: %+v", results)
}

func TestIntegration_QueryBuilder_LikeOperator(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers9")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers9" WITH ({"id"="1"}, {"name"="John Doe"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers9" WITH ({"id"="2"}, {"name"="Jane Doe"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers9" WITH ({"id"="3"}, {"name"="Bob Smith"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test LIKE operator
	results, err := c.QueryBuilder().
		Select("TestUsers9").
		Where("name", Like, "%Doe%").
		Execute(ctx)

	if err != nil {
		t.Fatalf("QueryBuilder Execute with LIKE failed: %v", err)
	}

	t.Logf("LIKE operator results: %+v", results)
}

func TestIntegration_QueryBuilder_ComplexQuery(t *testing.T) {
	c := skipIfNoServer(t)
	if c == nil {
		return
	}

	cleanup := setupTestBundle(t, c, "TestUsers10")
	defer cleanup()

	ctx := context.Background()

	// Insert test data
	records := []string{
		`ADD DOCUMENT TO BUNDLE "TestUsers10" WITH ({"id"="1"}, {"name"="Alice"}, {"age"=25}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers10" WITH ({"id"="2"}, {"name"="Bob"}, {"age"=35}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers10" WITH ({"id"="3"}, {"name"="Charlie"}, {"age"=45}, {"status"="inactive"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers10" WITH ({"id"="4"}, {"name"="David"}, {"age"=28}, {"status"="active"});`,
		`ADD DOCUMENT TO BUNDLE "TestUsers10" WITH ({"id"="5"}, {"name"="Eve"}, {"age"=32}, {"status"="active"});`,
	}

	for _, cmd := range records {
		_, err := c.Mutate(cmd, integrationTestTimeout)
		if err != nil {
			t.Fatalf("Failed to insert test data: %v", err)
		}
	}

	// Test complex query with multiple features
	results, err := c.QueryBuilder().
		Select("TestUsers10", "id", "name", "age", "status").
		Where("age", GreaterThanOrEqual, 25).
		And("status", Equals, "active").
		OrderBy("age", Descending).
		Limit(3).
		Execute(ctx)

	if err != nil {
		t.Fatalf("Complex query failed: %v", err)
	}

	t.Logf("Complex query results: %+v", results)
}
