package benchmarks

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/client"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/codegen"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/migration"
	"github.com/dan-strohschein/syndrdb-drivers/src/golang/schema"
)

const benchConnString = "syndrdb://localhost:1776:primary:root:root;"

// BenchmarkConnectionEstablishment measures connection setup/teardown time
func BenchmarkConnectionEstablishment(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		c := client.NewClient(&client.ClientOptions{
			DefaultTimeoutMs: 10000,
			DebugMode:        false,
			MaxRetries:       3,
		})

		err := c.Connect(benchConnString)
		if err != nil {
			b.Fatalf("Failed to connect: %v", err)
		}

		err = c.Disconnect()
		if err != nil {
			b.Fatalf("Failed to disconnect: %v", err)
		}
	}
}

// BenchmarkSimpleQuery measures query execution time
func BenchmarkSimpleQuery(b *testing.B) {
	c := client.NewClient(&client.ClientOptions{
		DefaultTimeoutMs: 10000,
		DebugMode:        false,
		MaxRetries:       3,
	})

	err := c.Connect(benchConnString)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := c.Query("SHOW BUNDLES;", 10000)
		if err != nil {
			b.Fatalf("Query failed: %v", err)
		}
	}
}

// BenchmarkMutation measures mutation execution time
func BenchmarkMutation(b *testing.B) {
	c := client.NewClient(&client.ClientOptions{
		DefaultTimeoutMs: 10000,
		DebugMode:        false,
		MaxRetries:       3,
	})

	err := c.Connect(benchConnString)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}
	defer c.Disconnect()

	// Setup: Create a test bundle
	createCmd := `CREATE BUNDLE "bench_test" WITH FIELDS (
		id INT REQUIRED UNIQUE,
		name STRING REQUIRED,
		value INT
	);`
	_, err = c.Mutate(createCmd, 10000)
	if err != nil {
		b.Fatalf("Failed to create bundle: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		insertCmd := fmt.Sprintf(`INSERT INTO "bench_test" (id, name, value) VALUES (%d, "test_%d", %d);`, i, i, i*10)
		_, err := c.Mutate(insertCmd, 10000)
		if err != nil {
			b.Fatalf("Mutation failed: %v", err)
		}
	}

	b.StopTimer()

	// Cleanup
	_, _ = c.Mutate(`DROP BUNDLE "bench_test";`, 10000)
}

// BenchmarkSchemaComparison measures schema comparison performance
func BenchmarkSchemaComparison(b *testing.B) {
	oldSchema := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	newSchema := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
					{Name: "age", Type: "int", Required: false},
				},
				Indexes: []schema.IndexDefinition{
					{Name: "idx_email", Type: "hash", Fields: []string{"email"}},
				},
				Relationships: []schema.RelationshipDefinition{},
			},
			{
				Name: "posts",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "user_id", Type: "int", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "string", Required: false},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = schema.CompareSchemas(&oldSchema, &newSchema)
	}
}

// BenchmarkMigrationGeneration measures migration generation performance
func BenchmarkMigrationGeneration(b *testing.B) {
	rollbackGen := migration.NewRollbackGenerator()

	// Sample UP commands to generate DOWN commands for
	upCommands := []string{
		`CREATE BUNDLE "users" WITH FIELDS (id INT REQUIRED UNIQUE, name STRING REQUIRED, email STRING REQUIRED UNIQUE);`,
		`UPDATE BUNDLE "users" SET ADD FIELD age INT;`,
		`UPDATE BUNDLE "users" SET ADD INDEX idx_email ON email TYPE HASH;`,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = rollbackGen.GenerateDown(upCommands)
	}
}

// BenchmarkJSONSchemaGeneration measures JSON Schema generation performance
func BenchmarkJSONSchemaGeneration(b *testing.B) {
	schemaDef := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
					{Name: "age", Type: "int", Required: false},
					{Name: "active", Type: "bool", Required: true},
				},
				Indexes: []schema.IndexDefinition{
					{Name: "idx_email", Type: "hash", Fields: []string{"email"}},
				},
				Relationships: []schema.RelationshipDefinition{},
			},
			{
				Name: "posts",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "user_id", Type: "int", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "string", Required: false},
					{Name: "published", Type: "bool", Required: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	gen := codegen.NewJSONSchemaGenerator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = gen.GenerateSingle(&schemaDef)
	}
}

// BenchmarkGraphQLSchemaGeneration measures GraphQL Schema generation performance
func BenchmarkGraphQLSchemaGeneration(b *testing.B) {
	schemaDef := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
					{Name: "age", Type: "int", Required: false},
					{Name: "active", Type: "bool", Required: true},
				},
				Indexes: []schema.IndexDefinition{
					{Name: "idx_email", Type: "hash", Fields: []string{"email"}},
				},
				Relationships: []schema.RelationshipDefinition{},
			},
			{
				Name: "posts",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "user_id", Type: "int", Required: true},
					{Name: "title", Type: "string", Required: true},
					{Name: "content", Type: "string", Required: false},
					{Name: "published", Type: "bool", Required: true},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	gen := codegen.NewGraphQLSchemaGenerator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = gen.Generate(&schemaDef)
	}
}

// BenchmarkTypeMapping measures the performance of type mapping
func BenchmarkTypeMapping(b *testing.B) {
	// Create a simple schema for type mapping benchmark
	schemaDef := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "test",
				Fields: []schema.FieldDefinition{
					{Name: "f_int", Type: "int", Required: true},
					{Name: "f_string", Type: "string", Required: true},
					{Name: "f_bool", Type: "bool", Required: true},
					{Name: "f_float", Type: "float", Required: false},
					{Name: "f_double", Type: "double", Required: false},
					{Name: "f_bytes", Type: "bytes", Required: false},
					{Name: "f_timestamp", Type: "timestamp", Required: false},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	jsonGen := codegen.NewJSONSchemaGenerator()
	gqlGen := codegen.NewGraphQLSchemaGenerator()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = jsonGen.GenerateSingle(&schemaDef)
		_, _ = gqlGen.Generate(&schemaDef)
	}
}

// BenchmarkLargeSchemaDiff measures performance with large schemas
func BenchmarkLargeSchemaDiff(b *testing.B) {
	oldBundles := make([]schema.BundleDefinition, 100)
	newBundles := make([]schema.BundleDefinition, 100)

	for i := 0; i < 100; i++ {
		oldBundles[i] = schema.BundleDefinition{
			Name: fmt.Sprintf("bundle_%d", i),
			Fields: []schema.FieldDefinition{
				{Name: "id", Type: "int", Required: true, Unique: true},
				{Name: "field1", Type: "string", Required: true},
				{Name: "field2", Type: "int", Required: false},
			},
			Indexes:       []schema.IndexDefinition{},
			Relationships: []schema.RelationshipDefinition{},
		}

		newBundles[i] = schema.BundleDefinition{
			Name: fmt.Sprintf("bundle_%d", i),
			Fields: []schema.FieldDefinition{
				{Name: "id", Type: "int", Required: true, Unique: true},
				{Name: "field1", Type: "string", Required: true},
				{Name: "field2", Type: "int", Required: false},
				{Name: "field3", Type: "bool", Required: false},
			},
			Indexes:       []schema.IndexDefinition{},
			Relationships: []schema.RelationshipDefinition{},
		}
	}

	oldSchema := schema.SchemaDefinition{Bundles: oldBundles}
	newSchema := schema.SchemaDefinition{Bundles: newBundles}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = schema.CompareSchemas(&oldSchema, &newSchema)
	}
}

// BenchmarkJSONSerialization measures JSON marshaling/unmarshaling performance
func BenchmarkJSONSerialization(b *testing.B) {
	schemaDef := schema.SchemaDefinition{
		Bundles: []schema.BundleDefinition{
			{
				Name: "users",
				Fields: []schema.FieldDefinition{
					{Name: "id", Type: "int", Required: true, Unique: true},
					{Name: "name", Type: "string", Required: true},
					{Name: "email", Type: "string", Required: true, Unique: true},
					{Name: "age", Type: "int", Required: false},
				},
				Indexes:       []schema.IndexDefinition{},
				Relationships: []schema.RelationshipDefinition{},
			},
		},
	}

	b.Run("Marshal", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, err := json.Marshal(schemaDef)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("Unmarshal", func(b *testing.B) {
		data, _ := json.Marshal(schemaDef)
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var s schema.SchemaDefinition
			err := json.Unmarshal(data, &s)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
