package testutil_test

import (
	"context"
	"testing"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/testutil"
)

func TestMockClient_BasicExpectations(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM users").
		WillReturn(map[string]interface{}{
			"users": []interface{}{
				map[string]interface{}{"id": 1, "name": "Alice"},
				map[string]interface{}{"id": 2, "name": "Bob"},
			},
		})

	result, err := mock.Query(ctx, "SELECT * FROM users", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result == nil {
		t.Fatal("expected result, got nil")
	}

	mock.VerifyExpectations(t)
}

func TestMockClient_MultipleExpectations(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM users").WillReturn(map[string]interface{}{"count": 10})
	mock.ExpectMutate("INSERT INTO users (name) VALUES ('Alice')").WillReturn(map[string]interface{}{"id": 1})
	mock.ExpectQuery("SELECT * FROM users WHERE id = 1").WillReturn(map[string]interface{}{"name": "Alice"})

	_, _ = mock.Query(ctx, "SELECT * FROM users", 0)
	_, _ = mock.Mutate(ctx, "INSERT INTO users (name) VALUES ('Alice')", 0)
	_, _ = mock.Query(ctx, "SELECT * FROM users WHERE id = 1", 0)

	mock.VerifyExpectations(t)
}

func TestMockClient_ErrorExpectations(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM nonexistent").
		WillReturnError(testutil.MockError("NOT_FOUND", "bundle not found"))

	_, err := mock.Query(ctx, "SELECT * FROM nonexistent", 0)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	mock.VerifyExpectations(t)
}

func TestMockClient_Times(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM users").
		WillReturn(map[string]interface{}{"users": []interface{}{}}).
		Times(3)

	for i := 0; i < 3; i++ {
		_, _ = mock.Query(ctx, "SELECT * FROM users", 0)
	}

	mock.VerifyExpectations(t)
}

func TestMockClient_AnyTimes(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM cache").
		WillReturn(map[string]interface{}{"data": "cached"}).
		AnyTimes()

	for i := 0; i < 10; i++ {
		_, _ = mock.Query(ctx, "SELECT * FROM cache", 0)
	}

	mock.VerifyExpectations(t)
}

func TestMockClient_CallHistory(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT * FROM users").WillReturn(nil).AnyTimes()
	mock.ExpectMutate("INSERT INTO users").WillReturn(nil).AnyTimes()

	_, _ = mock.Query(ctx, "SELECT * FROM users", 0)
	_, _ = mock.Mutate(ctx, "INSERT INTO users", 0)
	_, _ = mock.Query(ctx, "SELECT * FROM users", 0)

	if count := mock.GetCallCount("Query"); count != 2 {
		t.Errorf("expected 2 Query calls, got %d", count)
	}

	if count := mock.GetCallCount("Mutate"); count != 1 {
		t.Errorf("expected 1 Mutate call, got %d", count)
	}

	calls := mock.GetCalls()
	if len(calls) != 3 {
		t.Errorf("expected 3 total calls, got %d", len(calls))
	}

	mock.VerifyExpectations(t)
}

func TestMockClient_Reset(t *testing.T) {
	mock := testutil.NewMockClient()
	ctx := context.Background()

	mock.ExpectQuery("SELECT 1").WillReturn(1).Once()
	_, _ = mock.Query(ctx, "SELECT 1", 0)

	mock.Reset()

	mock.ExpectQuery("SELECT 2").WillReturn(2).Once()
	_, _ = mock.Query(ctx, "SELECT 2", 0)

	if count := mock.GetCallCount("Query"); count != 1 {
		t.Errorf("expected 1 call after reset, got %d", count)
	}
}
