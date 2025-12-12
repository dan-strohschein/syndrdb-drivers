package schema

import (
	"fmt"
	"strings"
)

// SerializeCreateBundle generates a CREATE BUNDLE command.
// Format matches SchemaSerializer.ts lines 15-40.
func SerializeCreateBundle(bundle *BundleDefinition) string {
	var fields []string
	for _, field := range bundle.Fields {
		defaultVal := serializeDefaultValue(field.DefaultValue)
		fields = append(fields, fmt.Sprintf(
			`    {"%s", "%s", %s, %s, %s}`,
			field.Name,
			field.Type,
			boolToUpper(field.Required),
			boolToUpper(field.Unique),
			defaultVal,
		))
	}

	return fmt.Sprintf(
		"CREATE BUNDLE \"%s\"\nWITH FIELDS (\n%s\n);",
		bundle.Name,
		strings.Join(fields, ",\n"),
	)
}

// SerializeUpdateBundle generates an UPDATE BUNDLE SET command.
// Format matches SchemaSerializer.ts lines 42-80.
func SerializeUpdateBundle(bundleName string, changes *BundleChange) string {
	var modifications []string

	for _, fieldChange := range changes.FieldChanges {
		switch fieldChange.Type {
		case "add":
			defaultVal := serializeDefaultValue(fieldChange.NewField.DefaultValue)
			modifications = append(modifications, fmt.Sprintf(
				`    {ADD "%s" = "%s", "%s", %s, %s, %s}`,
				fieldChange.FieldName,
				fieldChange.FieldName,
				fieldChange.NewField.Type,
				boolToUpper(fieldChange.NewField.Required),
				boolToUpper(fieldChange.NewField.Unique),
				defaultVal,
			))

		case "remove":
			modifications = append(modifications, fmt.Sprintf(
				`    {REMOVE "%s" = "", "", FALSE, FALSE, NULL}`,
				fieldChange.FieldName,
			))

		case "modify":
			defaultVal := serializeDefaultValue(fieldChange.NewField.DefaultValue)
			modifications = append(modifications, fmt.Sprintf(
				`    {MODIFY "%s" = "%s", "%s", %s, %s, %s}`,
				fieldChange.FieldName,
				fieldChange.NewField.Name,
				fieldChange.NewField.Type,
				boolToUpper(fieldChange.NewField.Required),
				boolToUpper(fieldChange.NewField.Unique),
				defaultVal,
			))
		}
	}

	if len(modifications) == 0 {
		return ""
	}

	return fmt.Sprintf(
		"UPDATE BUNDLE \"%s\"\nSET (\n%s\n);",
		bundleName,
		strings.Join(modifications, ",\n"),
	)
}

// SerializeCreateIndex generates a CREATE INDEX command.
// Format matches SchemaSerializer.ts lines 82-95.
// TODO: Support multi-field composite indexes (see SchemaSerializer.ts line 58).
func SerializeCreateIndex(index *IndexDefinition, bundleName string) string {
	fieldsStr := ""
	if len(index.Fields) > 0 {
		quotedFields := make([]string, len(index.Fields))
		for i, field := range index.Fields {
			quotedFields[i] = fmt.Sprintf(`"%s"`, field)
		}
		fieldsStr = strings.Join(quotedFields, ", ")
	}

	if index.Type == HASH {
		return fmt.Sprintf(
			`CREATE HASH INDEX "%s" ON BUNDLE "%s" WITH FIELDS (%s);`,
			index.Name,
			bundleName,
			fieldsStr,
		)
	} else if index.Type == BTREE {
		return fmt.Sprintf(
			`CREATE B-INDEX "%s" ON BUNDLE "%s" WITH FIELDS (%s);`,
			index.Name,
			bundleName,
			fieldsStr,
		)
	}

	return ""
}

// SerializeDropIndex generates a DROP INDEX command.
func SerializeDropIndex(indexName string) string {
	return fmt.Sprintf(`DROP INDEX "%s";`, indexName)
}

// SerializeAddRelationship generates an UPDATE BUNDLE ADD RELATIONSHIP command.
// Format matches SchemaSerializer.ts lines 100-120.
func SerializeAddRelationship(bundleName string, rel *RelationshipDefinition) string {
	return fmt.Sprintf(
		`UPDATE BUNDLE "%s" ADD RELATIONSHIP ("%s" {"%s", "%s", "%s", "%s", "%s"});`,
		bundleName,
		rel.Name,
		rel.Type,
		rel.SourceBundle,
		rel.SourceField,
		rel.DestBundle,
		rel.DestField,
	)
}

// SerializeRemoveRelationship generates an UPDATE BUNDLE REMOVE RELATIONSHIP command.
func SerializeRemoveRelationship(bundleName string, relName string) string {
	return fmt.Sprintf(
		`UPDATE BUNDLE "%s" REMOVE RELATIONSHIP "%s";`,
		bundleName,
		relName,
	)
}

// SerializeDeleteBundle generates a DROP BUNDLE command.
func SerializeDeleteBundle(bundleName string) string {
	return fmt.Sprintf(`DROP BUNDLE "%s";`, bundleName)
}

// boolToUpper converts boolean to uppercase string (TRUE/FALSE).
func boolToUpper(b bool) string {
	if b {
		return "TRUE"
	}
	return "FALSE"
}

// serializeDefaultValue converts a default value to its string representation.
func serializeDefaultValue(val interface{}) string {
	if val == nil {
		return "NULL"
	}

	switch v := val.(type) {
	case string:
		return fmt.Sprintf(`"%s"`, v)
	case int, int32, int64, float32, float64:
		return fmt.Sprintf("%v", v)
	case bool:
		return boolToUpper(v)
	default:
		return "NULL"
	}
}
