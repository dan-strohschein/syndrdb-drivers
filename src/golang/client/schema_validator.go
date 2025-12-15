package client

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

// SchemaValidator provides schema-based validation for QueryBuilder operations.
type SchemaValidator struct {
	client      *Client
	schema      *schema.SchemaDefinition
	schemaMu    sync.RWMutex
	lastFetch   time.Time
	cacheTTL    time.Duration
	autoRefresh bool
}

// NewSchemaValidator creates a new schema validator with the specified cache TTL.
func NewSchemaValidator(client *Client, cacheTTL time.Duration, autoRefresh bool) *SchemaValidator {
	return &SchemaValidator{
		client:      client,
		cacheTTL:    cacheTTL,
		autoRefresh: autoRefresh,
	}
}

// fetchSchema retrieves the schema from the server using SHOW BUNDLES.
func (sv *SchemaValidator) fetchSchema(ctx context.Context) error {
	// Query for schema
	query := "SHOW BUNDLES;"
	result, err := sv.client.Query(query, 0)
	if err != nil {
		return &QueryError{
			Code:    "E_SCHEMA_FETCH_FAILED",
			Type:    "QueryError",
			Message: "failed to fetch schema",
			Cause:   err,
		}
	}

	// Parse schema from response
	// Convert result to JSON bytes for parsing
	var responseBytes []byte
	switch v := result.(type) {
	case string:
		responseBytes = []byte(v)
	case []byte:
		responseBytes = v
	default:
		// Try JSON marshaling
		responseBytes, err = json.Marshal(v)
		if err != nil {
			return &QueryError{
				Code:    "E_SCHEMA_PARSE_FAILED",
				Type:    "QueryError",
				Message: "failed to marshal schema response",
				Cause:   err,
			}
		}
	}

	parsedSchema, err := schema.ParseServerSchema(responseBytes)
	if err != nil {
		return &QueryError{
			Code:    "E_SCHEMA_PARSE_FAILED",
			Type:    "QueryError",
			Message: "failed to parse schema response",
			Cause:   err,
		}
	}

	sv.schemaMu.Lock()
	sv.schema = parsedSchema
	sv.lastFetch = time.Now()
	sv.schemaMu.Unlock()

	return nil
}

// getSchema returns the cached schema, fetching it if necessary or expired.
func (sv *SchemaValidator) getSchema(ctx context.Context) (*schema.SchemaDefinition, error) {
	sv.schemaMu.RLock()
	needsFetch := sv.schema == nil || time.Since(sv.lastFetch) > sv.cacheTTL
	sv.schemaMu.RUnlock()

	if needsFetch {
		if err := sv.fetchSchema(ctx); err != nil {
			return nil, err
		}
	}

	sv.schemaMu.RLock()
	defer sv.schemaMu.RUnlock()
	return sv.schema, nil
}

// InvalidateCache forces a schema refresh on the next validation.
func (sv *SchemaValidator) InvalidateCache() {
	sv.schemaMu.Lock()
	sv.schema = nil
	sv.schemaMu.Unlock()
}

// DetectDDL checks if a query contains DDL operations that require schema refresh.
func DetectDDL(query string) bool {
	upperQuery := strings.ToUpper(strings.TrimSpace(query))

	// Check for bundle DDL operations
	if strings.HasPrefix(upperQuery, "CREATE BUNDLE") ||
		strings.HasPrefix(upperQuery, "UPDATE BUNDLE") ||
		strings.HasPrefix(upperQuery, "DROP BUNDLE") ||
		strings.HasPrefix(upperQuery, "ALTER BUNDLE") {
		return true
	}

	return false
}

// ValidateQuery validates a SELECT query against the schema.
func (sv *SchemaValidator) ValidateQuery(bundle string, fields []string, whereClauses []whereClause) error {
	ctx := context.Background()
	schemaDefn, err := sv.getSchema(ctx)
	if err != nil {
		return err
	}

	// Find bundle definition
	bundleDefn := sv.findBundle(schemaDefn, bundle)
	if bundleDefn == nil {
		return &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle not found: " + bundle,
		}
	}

	// Validate field names (if specific fields are requested)
	if len(fields) > 0 {
		for _, field := range fields {
			if !sv.hasField(bundleDefn, field) {
				return &QueryError{
					Code:    "E_INVALID_QUERY",
					Type:    "QueryError",
					Message: "field not found in bundle: " + field,
				}
			}
		}
	}

	// Validate WHERE clause fields
	for _, clause := range whereClauses {
		// Handle dot-notation for relationship traversal
		if strings.Contains(clause.field, ".") {
			// TODO: Validate relationship traversal
			continue
		}

		if !sv.hasField(bundleDefn, clause.field) {
			return &QueryError{
				Code:    "E_INVALID_QUERY",
				Type:    "QueryError",
				Message: "WHERE field not found in bundle: " + clause.field,
			}
		}
	}

	return nil
}

// ValidateInsert validates an INSERT operation against the schema.
func (sv *SchemaValidator) ValidateInsert(bundle string, values map[string]interface{}) error {
	ctx := context.Background()
	schemaDefn, err := sv.getSchema(ctx)
	if err != nil {
		return err
	}

	// Find bundle definition
	bundleDefn := sv.findBundle(schemaDefn, bundle)
	if bundleDefn == nil {
		return &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle not found: " + bundle,
		}
	}

	// Validate all fields exist
	for field := range values {
		if !sv.hasField(bundleDefn, field) {
			return &QueryError{
				Code:    "E_INVALID_QUERY",
				Type:    "QueryError",
				Message: "field not found in bundle: " + field,
			}
		}
	}

	// TODO: Validate required fields are present
	// TODO: Validate field types match values

	return nil
}

// ValidateUpdate validates an UPDATE operation against the schema.
func (sv *SchemaValidator) ValidateUpdate(bundle string, setFields map[string]interface{}, whereClauses []whereClause) error {
	ctx := context.Background()
	schemaDefn, err := sv.getSchema(ctx)
	if err != nil {
		return err
	}

	// Find bundle definition
	bundleDefn := sv.findBundle(schemaDefn, bundle)
	if bundleDefn == nil {
		return &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle not found: " + bundle,
		}
	}

	// Validate SET fields
	for field := range setFields {
		if !sv.hasField(bundleDefn, field) {
			return &QueryError{
				Code:    "E_INVALID_QUERY",
				Type:    "QueryError",
				Message: "SET field not found in bundle: " + field,
			}
		}
	}

	// Validate WHERE fields
	for _, clause := range whereClauses {
		if !sv.hasField(bundleDefn, clause.field) {
			return &QueryError{
				Code:    "E_INVALID_QUERY",
				Type:    "QueryError",
				Message: "WHERE field not found in bundle: " + clause.field,
			}
		}
	}

	return nil
}

// ValidateDelete validates a DELETE operation against the schema.
func (sv *SchemaValidator) ValidateDelete(bundle string, whereClauses []whereClause) error {
	ctx := context.Background()
	schemaDefn, err := sv.getSchema(ctx)
	if err != nil {
		return err
	}

	// Find bundle definition
	bundleDefn := sv.findBundle(schemaDefn, bundle)
	if bundleDefn == nil {
		return &QueryError{
			Code:    "E_INVALID_QUERY",
			Type:    "QueryError",
			Message: "bundle not found: " + bundle,
		}
	}

	// Validate WHERE fields
	for _, clause := range whereClauses {
		if !sv.hasField(bundleDefn, clause.field) {
			return &QueryError{
				Code:    "E_INVALID_QUERY",
				Type:    "QueryError",
				Message: "WHERE field not found in bundle: " + clause.field,
			}
		}
	}

	return nil
}

// findBundle searches for a bundle definition in the schema.
func (sv *SchemaValidator) findBundle(schemaDefn *schema.SchemaDefinition, bundleName string) *schema.BundleDefinition {
	for _, bundle := range schemaDefn.Bundles {
		if bundle.Name == bundleName {
			return &bundle
		}
	}
	return nil
}

// hasField checks if a field exists in the bundle definition.
func (sv *SchemaValidator) hasField(bundle *schema.BundleDefinition, fieldName string) bool {
	for _, field := range bundle.Fields {
		if field.Name == fieldName {
			return true
		}
	}
	return false
}
