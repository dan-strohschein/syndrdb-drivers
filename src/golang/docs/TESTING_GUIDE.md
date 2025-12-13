# Testing Guide: For TypeScript/Node Developers

This guide helps TypeScript and Node.js developers understand how to test Go applications using the SyndrDB testutil package. If you're coming from Jest, Mocha, or other JavaScript testing frameworks, this guide will help you translate your knowledge to Go testing.

## Table of Contents

1. [Quick Comparison](#quick-comparison)
2. [MockClient vs @jest/mock](#mockclient-vs-jestmock)
3. [Factory Pattern vs factory-bot/fishery](#factory-pattern-vs-factory-botfishery)
4. [Async Testing: WaitFor vs @testing-library](#async-testing-waitfor-vs-testing-library)
5. [Integration Testing Patterns](#integration-testing-patterns)
6. [Complete Examples](#complete-examples)

## Quick Comparison

### Test Structure

**Jest/Mocha:**
```javascript
describe('UserService', () => {
  it('should get user by ID', async () => {
    const user = await userService.getUser(1);
    expect(user.name).toBe('Alice');
  });
});
```

**Go (testutil):**
```go
func TestUserService_GetUser(t *testing.T) {
    user, err := userService.GetUser(1)
    testutil.RequireNoError(t, err, "should get user")
    testutil.AssertEqual(t, user.Name, "Alice", "name should match")
}
```

### Assertions

| Jest                          | Go testutil                                    |
|-------------------------------|------------------------------------------------|
| `expect(x).toBe(y)`           | `testutil.AssertEqual(t, x, y, "msg")`         |
| `expect(x).not.toBe(y)`       | `testutil.AssertNotEqual(t, x, y, "msg")`      |
| `expect(str).toContain(sub)`  | `testutil.AssertContains(t, str, sub, "msg")`  |
| `expect(err).toBeNull()`      | `testutil.RequireNoError(t, err, "msg")`       |
| `expect(err).toBeDefined()`   | `testutil.RequireError(t, err, "msg")`         |

**Key Difference:** Go has two assertion patterns:
- `Require*`: Fails immediately (like Jest's default behavior)
- `Assert*`: Logs error but continues (like Jest's `test.todo`)

## MockClient vs @jest/mock

### Creating Mocks

**Jest:**
```javascript
const mockClient = {
  query: jest.fn(),
  mutate: jest.fn()
};
```

**Go testutil:**
```go
mock := testutil.NewMockClient()
```

### Setting Expectations

**Jest:**
```javascript
mockClient.query
  .mockResolvedValueOnce({ users: [{ id: 1, name: 'Alice' }] })
  .mockResolvedValueOnce({ users: [] });
```

**Go testutil:**
```go
mock.ExpectQuery("SELECT * FROM users").
    WillReturn(map[string]interface{}{
        "users": []interface{}{
            map[string]interface{}{"id": 1, "name": "Alice"},
        },
    }).
    Once()
```

### Fluent API Comparison

**Jest:**
```javascript
mockClient.query
  .mockResolvedValue({ data: 'cached' })
  .mockResolvedValue({ data: 'updated' });
  
mockClient.query.mockRejectedValue(new Error('Not found'));
```

**Go testutil:**
```go
// Success responses
mock.ExpectQuery("SELECT * FROM cache").
    WillReturn(map[string]interface{}{"data": "cached"}).
    Once()

// Error responses
mock.ExpectQuery("SELECT * FROM missing").
    WillReturnError(testutil.MockError("NOT_FOUND", "bundle not found")).
    Once()
```

### Verifying Calls

**Jest:**
```javascript
expect(mockClient.query).toHaveBeenCalledTimes(3);
expect(mockClient.query).toHaveBeenCalledWith('SELECT * FROM users');
```

**Go testutil:**
```go
mock.VerifyExpectations(t)  // Verifies all expectations met

// Or manual verification
if count := mock.GetCallCount("Query"); count != 3 {
    t.Errorf("expected 3 calls, got %d", count)
}
```

### Call Count Modifiers

| Jest                  | Go testutil     |
|-----------------------|-----------------|
| `mockFn.mockImplementation()` (unlimited) | `.AnyTimes()`   |
| (default behavior - once)     | `.Once()`       |
| N/A                           | `.Twice()`      |
| N/A                           | `.Times(n)`     |

**Example:**
```go
// Caching scenario - unlimited calls OK
mock.ExpectQuery("SELECT * FROM cache").
    WillReturn(map[string]interface{}{"data": "cached"}).
    AnyTimes()

// Expect exactly 3 calls
mock.ExpectQuery("SELECT * FROM users").
    WillReturn(map[string]interface{}{"users": []interface{}{}}).
    Times(3)
```

## Factory Pattern vs factory-bot/fishery

### Creating Test Data

**factory-bot (Node):**
```javascript
import { factory } from 'factory-bot';

const userFactory = factory.define('user', () => ({
  id: factory.sequence('user.id'),
  email: factory.sequence('user.email', n => `user${n}@example.com`),
  name: 'Test User',
  active: true
}));

const user = await userFactory.build();
const users = await userFactory.buildList(5);
```

**Go testutil:**
```go
// Built-in factory
user := testutil.BuildUser()
users := testutil.BuildUsers(5)

// Custom fields
user := testutil.BuildUser(
    testutil.WithField("name", "Custom Name"),
    testutil.WithField("active", false),
)

// Multiple fields at once
user := testutil.BuildUser(
    testutil.WithFields(map[string]interface{}{
        "name":   "Alice",
        "email":  "alice@example.com",
        "active": false,
    }),
)
```

### Sequences and Random Data

**Fishery (Node):**
```javascript
import { Factory } from 'fishery';

const userFactory = Factory.define<User>(({ sequence }) => ({
  id: sequence,
  email: `user${sequence}@example.com`,
  username: `user${sequence}`,
  randomValue: faker.random.number()
}));
```

**Go testutil:**
```go
// Sequences (atomic, thread-safe)
id := testutil.SequenceID()           // Returns incrementing int64
email := testutil.SequenceEmail()     // Returns "user1@example.com", "user2@example.com"...
username := testutil.SequenceUsername() // Returns "user1", "user2"...

// Random data
str := testutil.RandomString(10)      // Random 10-char string
num := testutil.RandomInt(1, 100)     // Random int between 1-100
bool := testutil.RandomBool()         // Random true/false
email := testutil.RandomEmail()       // Random email with random string
date := testutil.RandomDate()         // Random past date
future := testutil.RandomFutureDate() // Random future date
```

### Building Related Data

**Node (factory-bot):**
```javascript
const author = await userFactory.build();
const post = await postFactory.build({ authorId: author.id });
const comments = await commentFactory.buildList(3, { postId: post.id });
```

**Go testutil:**
```go
// Create related entities
author := testutil.BuildUser(
    testutil.WithField("name", "John Doe"),
)

post := testutil.BuildPost(
    testutil.WithField("author_id", author["id"]),
    testutil.WithField("title", "Test Post"),
)

comments := testutil.BuildComments(3,
    testutil.WithField("post_id", post["id"]),
)
```

### Custom Factories

**Node:**
```javascript
import { Factory } from 'fishery';

const customFactory = Factory.define<CustomType>(() => ({
  field1: 'value1',
  field2: 'value2'
}));
```

**Go:**
```go
// Create custom factory
factory := testutil.NewFactory(
    map[string]interface{}{
        "field1": "value1",
        "field2": "value2",
        "id":     testutil.SequenceID,  // Function for lazy evaluation
    },
    func(data map[string]interface{}) interface{} {
        return data  // Or transform to custom type
    },
)

// Register in registry
registry := testutil.DefaultRegistry
registry.Register("custom", factory)

// Build
item, _ := registry.Build("custom")
```

## Async Testing: WaitFor vs @testing-library

### Waiting for Conditions

**@testing-library/react:**
```javascript
import { waitFor } from '@testing-library/react';

await waitFor(() => {
  expect(getByText('Success')).toBeInTheDocument();
}, { timeout: 5000, interval: 100 });
```

**Go testutil:**
```go
testutil.WaitFor(t, 5*time.Second, 100*time.Millisecond, func() bool {
    return elementVisible()
})

// Or use Jest-style naming
testutil.Eventually(t, 5*time.Second, 100*time.Millisecond, func() bool {
    return conditionMet()
})
```

### Example: Waiting for Async Operation

**Node:**
```javascript
test('async operation completes', async () => {
  const promise = performAsyncOperation();
  
  await waitFor(() => {
    expect(isComplete()).toBe(true);
  });
  
  const result = await promise;
  expect(result).toBe(expectedValue);
});
```

**Go:**
```go
func TestAsyncOperation(t *testing.T) {
    done := false
    
    // Start async operation
    go func() {
        performAsyncOperation()
        done = true
    }()
    
    // Wait for completion
    testutil.Eventually(t, 2*time.Second, 100*time.Millisecond, func() bool {
        return done
    })
    
    testutil.AssertEqual(t, done, true, "operation should complete")
}
```

## Integration Testing Patterns

### Environment Variables

**Node (.env files):**
```javascript
// .env.test
DATABASE_URL=postgres://localhost:5432/test_db

// test setup
require('dotenv').config({ path: '.env.test' });
```

**Go (environment variables):**
```go
// Set in shell or CI
export SYNDRDB_TEST_CONN="http://localhost:8080"

// Or in code for tests
os.Setenv("SYNDRDB_TEST_CONN", "http://localhost:8080")

// Use in tests
client, cleanup := testutil.NewTestClientOrSkip(t)
defer cleanup()
// Test will skip if SYNDRDB_TEST_CONN not set
```

### Setup and Teardown

**Jest:**
```javascript
describe('UserRepository', () => {
  let db;
  
  beforeEach(async () => {
    db = await createTestDatabase();
    await db.migrate.latest();
  });
  
  afterEach(async () => {
    await db.destroy();
  });
  
  it('creates user', async () => {
    const user = await db.users.create({ name: 'Alice' });
    expect(user.id).toBeDefined();
  });
});
```

**Go:**
```go
func TestUserRepository(t *testing.T) {
    client, cleanup := testutil.NewTestClientOrSkip(t)
    defer cleanup()
    
    bundleName := testutil.TestBundleName("users")
    bundleCleanup := testutil.SetupTestBundle(t, client, bundleName, []string{
        "id int64 primary",
        "name string",
    })
    defer bundleCleanup()
    
    // Run tests with bundle
    t.Run("creates user", func(t *testing.T) {
        _, err := client.Mutate("INSERT INTO " + bundleName + " (id, name) VALUES (1, 'Alice')", 0)
        testutil.RequireNoError(t, err, "insert should succeed")
    })
}
```

### Table-Driven Tests vs describe/it blocks

**Jest:**
```javascript
describe('calculator', () => {
  it.each([
    [1, 2, 3],
    [2, 3, 5],
    [10, 5, 15]
  ])('adds %i + %i to equal %i', (a, b, expected) => {
    expect(add(a, b)).toBe(expected);
  });
});
```

**Go:**
```go
func TestCalculator_Add(t *testing.T) {
    tests := []struct {
        name     string
        a, b     int
        expected int
    }{
        {"positive numbers", 1, 2, 3},
        {"small numbers", 2, 3, 5},
        {"larger numbers", 10, 5, 15},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := add(tt.a, tt.b)
            testutil.AssertEqual(t, result, tt.expected, "sum should match")
        })
    }
}
```

## Complete Examples

### Example 1: Testing a User Service (with Mocks)

**TypeScript/Jest:**
```typescript
import { UserService } from './UserService';
import { SyndrDBClient } from 'syndrdb';

describe('UserService', () => {
  let mockClient: jest.Mocked<SyndrDBClient>;
  let service: UserService;
  
  beforeEach(() => {
    mockClient = {
      query: jest.fn(),
      mutate: jest.fn()
    } as any;
    service = new UserService(mockClient);
  });
  
  it('gets user by ID', async () => {
    mockClient.query.mockResolvedValue({
      users: [{ id: 1, name: 'Alice', email: 'alice@example.com' }]
    });
    
    const user = await service.getUser(1);
    
    expect(user.name).toBe('Alice');
    expect(mockClient.query).toHaveBeenCalledWith(
      'SELECT * FROM users WHERE id = $1',
      [1]
    );
  });
  
  it('throws error when user not found', async () => {
    mockClient.query.mockRejectedValue(new Error('NOT_FOUND'));
    
    await expect(service.getUser(999)).rejects.toThrow('NOT_FOUND');
  });
});
```

**Go/testutil:**
```go
package service_test

import (
    "context"
    "testing"
    
    "your-project/service"
    "github.com/dan-strohschein/syndrdb-drivers/src/golang/testutil"
)

func TestUserService_GetUser(t *testing.T) {
    mock := testutil.NewMockClient()
    svc := service.NewUserService(mock)
    ctx := context.Background()
    
    // Set up expectation
    mock.ExpectQuery("SELECT * FROM users WHERE id = $1").
        WillReturn(map[string]interface{}{
            "users": []interface{}{
                map[string]interface{}{
                    "id":    1,
                    "name":  "Alice",
                    "email": "alice@example.com",
                },
            },
        }).
        Once()
    
    // Execute
    user, err := svc.GetUser(ctx, 1)
    
    // Verify
    testutil.RequireNoError(t, err, "should get user")
    testutil.AssertEqual(t, user.Name, "Alice", "name should match")
    
    mock.VerifyExpectations(t)
}

func TestUserService_GetUserNotFound(t *testing.T) {
    mock := testutil.NewMockClient()
    svc := service.NewUserService(mock)
    ctx := context.Background()
    
    // Expect error
    mock.ExpectQuery("SELECT * FROM users WHERE id = $1").
        WillReturnError(testutil.MockError("NOT_FOUND", "user not found")).
        Once()
    
    // Execute
    _, err := svc.GetUser(ctx, 999)
    
    // Verify
    testutil.RequireError(t, err, "should return error")
    
    mock.VerifyExpectations(t)
}
```

### Example 2: Integration Test with Test Fixtures

**TypeScript/Jest:**
```typescript
describe('User CRUD Integration', () => {
  let client: SyndrDBClient;
  
  beforeAll(async () => {
    client = new SyndrDBClient(process.env.SYNDRDB_TEST_CONN);
    await client.connect();
  });
  
  afterAll(async () => {
    await client.disconnect();
  });
  
  it('performs full CRUD cycle', async () => {
    // Create
    await client.mutate('INSERT INTO users (id, name, email) VALUES (1, "Alice", "alice@example.com")');
    
    // Read
    const result = await client.query('SELECT * FROM users WHERE id = 1');
    expect(result.users).toHaveLength(1);
    expect(result.users[0].name).toBe('Alice');
    
    // Update
    await client.mutate('UPDATE users SET name = "Alice Updated" WHERE id = 1');
    const updated = await client.query('SELECT * FROM users WHERE id = 1');
    expect(updated.users[0].name).toBe('Alice Updated');
    
    // Delete
    await client.mutate('DELETE FROM users WHERE id = 1');
    const deleted = await client.query('SELECT * FROM users');
    expect(deleted.users).toHaveLength(0);
  });
});
```

**Go/testutil:**
```go
func TestUserCRUD_Integration(t *testing.T) {
    client, clientCleanup := testutil.NewTestClientOrSkip(t)
    defer clientCleanup()
    
    // Setup test bundle
    bundleName := testutil.TestBundleName("users")
    cleanup := testutil.SetupTestBundle(t, client, bundleName, []string{
        "id int64 primary",
        "email string",
        "name string",
    })
    defer cleanup()
    
    ctx := context.Background()
    
    // Create user using factory
    user := testutil.BuildUser(
        testutil.WithField("id", int64(1)),
        testutil.WithField("email", "alice@example.com"),
        testutil.WithField("name", "Alice"),
    )
    testutil.InsertTestData(t, client, bundleName, []map[string]interface{}{user})
    
    // Read
    result, err := client.Query(ctx, "SELECT * FROM "+bundleName+" WHERE id = 1", 0)
    testutil.RequireNoError(t, err, "query should succeed")
    
    data := result.(map[string]interface{})["data"].([]interface{})
    testutil.AssertEqual(t, len(data), 1, "should have 1 user")
    
    // Update
    _, err = client.Mutate("UPDATE "+bundleName+" SET name = 'Alice Updated' WHERE id = 1", 0)
    testutil.RequireNoError(t, err, "update should succeed")
    
    // Verify update
    result, err = client.Query(ctx, "SELECT * FROM "+bundleName+" WHERE id = 1", 0)
    testutil.RequireNoError(t, err, "query should succeed")
    
    data = result.(map[string]interface{})["data"].([]interface{})
    userData := data[0].(map[string]interface{})
    testutil.AssertEqual(t, userData["name"], "Alice Updated", "name should be updated")
    
    // Delete
    _, err = client.Mutate("DELETE FROM "+bundleName+" WHERE id = 1", 0)
    testutil.RequireNoError(t, err, "delete should succeed")
    
    // Verify deletion
    result, err = client.Query(ctx, "SELECT * FROM "+bundleName, 0)
    testutil.RequireNoError(t, err, "query should succeed")
    
    data = result.(map[string]interface{})["data"].([]interface{})
    testutil.AssertEqual(t, len(data), 0, "should have 0 users after deletion")
}
```

### Example 3: Testing with Factories

**TypeScript:**
```typescript
import { buildUser, buildPost, buildComments } from './factories';

describe('Blog Post with Comments', () => {
  it('creates post with comments', async () => {
    const author = buildUser({ name: 'John Doe' });
    const post = buildPost({ authorId: author.id, title: 'Test Post' });
    const comments = buildComments(3, { postId: post.id });
    
    expect(comments).toHaveLength(3);
    comments.forEach(comment => {
      expect(comment.postId).toBe(post.id);
    });
  });
});
```

**Go:**
```go
func TestBlogPostWithComments(t *testing.T) {
    // Create author
    author := testutil.BuildUser(
        testutil.WithField("name", "John Doe"),
    )
    
    // Create post by this author
    post := testutil.BuildPost(
        testutil.WithField("author_id", author["id"]),
        testutil.WithField("title", "Test Post"),
    )
    
    // Create comments on this post
    comments := testutil.BuildComments(3,
        testutil.WithField("post_id", post["id"]),
    )
    
    testutil.AssertEqual(t, len(comments), 3, "should have 3 comments")
    
    // Verify all comments belong to the post
    for i, comment := range comments {
        testutil.AssertEqual(t, comment["post_id"], post["id"],
            fmt.Sprintf("comment %d should belong to post", i))
    }
}
```

## Running Tests

### Jest Commands

```bash
# Run all tests
npm test

# Run specific test file
npm test user.test.ts

# Run in watch mode
npm test -- --watch

# Run with coverage
npm test -- --coverage
```

### Go Test Commands

```bash
# Run all tests
go test ./...

# Run specific package
go test ./testutil

# Run specific test
go test -run TestUserService

# Run with verbose output
go test -v ./testutil

# Run with coverage
go test -cover ./testutil

# Run tests with build tag
go test -tags=milestone2 ./testutil
```

## Best Practices

### 1. Use Subtests for Grouping (like describe blocks)

**Jest:**
```javascript
describe('Calculator', () => {
  describe('add', () => {
    it('adds positive numbers', () => {});
    it('adds negative numbers', () => {});
  });
});
```

**Go:**
```go
func TestCalculator(t *testing.T) {
    t.Run("add", func(t *testing.T) {
        t.Run("positive numbers", func(t *testing.T) {
            // test
        })
        t.Run("negative numbers", func(t *testing.T) {
            // test
        })
    })
}
```

### 2. Use Table-Driven Tests for Multiple Cases

```go
func TestAdd(t *testing.T) {
    tests := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 1, 2, 3},
        {"negative", -1, -2, -3},
        {"mixed", 1, -1, 0},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := add(tt.a, tt.b)
            testutil.AssertEqual(t, got, tt.want, "sum should match")
        })
    }
}
```

### 3. Use Factories for Consistent Test Data

```go
// Instead of manually creating data
user := map[string]interface{}{
    "id": int64(1),
    "email": "test@example.com",
    "username": "testuser",
    // ... lots of fields
}

// Use factories
user := testutil.BuildUser(
    testutil.WithField("id", int64(1)),
)
```

### 4. Clean Up Resources with Defer

```go
func TestSomething(t *testing.T) {
    client, cleanup := testutil.NewTestClientOrSkip(t)
    defer cleanup()  // Ensures cleanup even if test fails
    
    // test code
}
```

## Troubleshooting

### Common Issues

**Issue:** "SYNDRDB_TEST_CONN not set" - test skipped
```go
// Solution: Set environment variable
export SYNDRDB_TEST_CONN="http://localhost:8080"

// Or use NewTestClient for mandatory connection
client, cleanup := testutil.NewTestClient(t)  // Fails if not set
```

**Issue:** Mock expectations not met
```go
// Always call VerifyExpectations at the end
mock.VerifyExpectations(t)  // Will fail test if expectations not met
```

**Issue:** Cleanup not running
```go
// Always use defer immediately after resource creation
cleanup := testutil.SetupTestBundle(...)
defer cleanup()  // Don't put defer later in function
```

## Additional Resources

- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Table-Driven Tests in Go](https://dave.cheney.net/2019/05/07/prefer-table-driven-tests)
- [SyndrDB Client Documentation](../client/README.md)
- [testutil Package Reference](../testutil/)

## Need Help?

- Check the [examples directory](../examples/) for more code samples
- Review [integration tests](../tests/integration/) for real-world patterns
- Open an issue on GitHub for questions
