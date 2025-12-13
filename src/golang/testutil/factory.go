//go:build milestone2

package testutil

import (
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"
)

// Factory generates test data with customizable options.
// This interface allows for flexible test data generation patterns.
type Factory interface {
	// Build creates a single instance of test data
	Build(options ...Option) interface{}

	// BuildList creates multiple instances of test data
	BuildList(count int, options ...Option) []interface{}
}

// Option is a function that modifies factory behavior.
type Option func(map[string]interface{})

// BaseFactory provides common factory functionality.
type BaseFactory struct {
	defaults map[string]interface{}
	builder  func(map[string]interface{}) interface{}
}

// NewBaseFactory creates a new base factory with default values.
func NewBaseFactory(defaults map[string]interface{}, builder func(map[string]interface{}) interface{}) *BaseFactory {
	return &BaseFactory{
		defaults: defaults,
		builder:  builder,
	}
}

// Build creates a single instance with optional overrides.
func (f *BaseFactory) Build(options ...Option) interface{} {
	// Copy defaults
	data := make(map[string]interface{})
	for k, v := range f.defaults {
		data[k] = v
	}

	// Apply options
	for _, opt := range options {
		opt(data)
	}

	return f.builder(data)
}

// BuildList creates multiple instances.
func (f *BaseFactory) BuildList(count int, options ...Option) []interface{} {
	results := make([]interface{}, count)
	for i := 0; i < count; i++ {
		results[i] = f.Build(options...)
	}
	return results
}

// Common option builders

// WithField sets a specific field value.
func WithField(name string, value interface{}) Option {
	return func(data map[string]interface{}) {
		data[name] = value
	}
}

// WithFields sets multiple field values.
func WithFields(fields map[string]interface{}) Option {
	return func(data map[string]interface{}) {
		for k, v := range fields {
			data[k] = v
		}
	}
}

// Sequence generators for unique values

var (
	emailSequence    uint64
	usernameSequence uint64
	idSequence       uint64
)

// SequenceEmail generates unique email addresses.
func SequenceEmail() string {
	n := atomic.AddUint64(&emailSequence, 1)
	return fmt.Sprintf("user%d@example.com", n)
}

// SequenceUsername generates unique usernames.
func SequenceUsername() string {
	n := atomic.AddUint64(&usernameSequence, 1)
	return fmt.Sprintf("user%d", n)
}

// SequenceID generates unique IDs.
func SequenceID() int64 {
	return int64(atomic.AddUint64(&idSequence, 1))
}

// Random generators for realistic test data

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// RandomString generates a random string of the specified length.
func RandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rng.Intn(len(charset))]
	}
	return string(b)
}

// RandomInt generates a random integer between min and max (inclusive).
func RandomInt(min, max int) int {
	return min + rng.Intn(max-min+1)
}

// RandomBool generates a random boolean.
func RandomBool() bool {
	return rng.Intn(2) == 1
}

// RandomEmail generates a random email address.
func RandomEmail() string {
	return fmt.Sprintf("%s@%s.com", RandomString(8), RandomString(6))
}

// RandomDate generates a random date within the last year.
func RandomDate() time.Time {
	now := time.Now()
	daysAgo := rng.Intn(365)
	return now.AddDate(0, 0, -daysAgo)
}

// RandomFutureDate generates a random date within the next year.
func RandomFutureDate() time.Time {
	now := time.Now()
	daysAhead := rng.Intn(365)
	return now.AddDate(0, 0, daysAhead)
}

// Built-in Factories

// UserFactory creates user test data.
type UserFactory struct {
	*BaseFactory
}

// NewUserFactory creates a factory for generating user data.
// This matches common user schema patterns.
func NewUserFactory() *UserFactory {
	return &UserFactory{
		BaseFactory: NewBaseFactory(
			map[string]interface{}{
				"id":         SequenceID,
				"email":      SequenceEmail,
				"username":   SequenceUsername,
				"name":       "Test User",
				"created_at": time.Now,
				"active":     true,
			},
			func(data map[string]interface{}) interface{} {
				// Resolve lazy values (functions)
				resolved := make(map[string]interface{})
				for k, v := range data {
					switch fn := v.(type) {
					case func() int64:
						resolved[k] = fn()
					case func() string:
						resolved[k] = fn()
					case func() time.Time:
						resolved[k] = fn()
					default:
						resolved[k] = v
					}
				}
				return resolved
			},
		),
	}
}

// PostFactory creates post/article test data.
type PostFactory struct {
	*BaseFactory
}

// NewPostFactory creates a factory for generating post data.
func NewPostFactory() *PostFactory {
	return &PostFactory{
		BaseFactory: NewBaseFactory(
			map[string]interface{}{
				"id":         SequenceID,
				"title":      func() string { return "Test Post " + RandomString(5) },
				"content":    func() string { return "This is test content. " + RandomString(50) },
				"author_id":  SequenceID,
				"created_at": time.Now,
				"published":  true,
				"views":      func() int { return RandomInt(0, 1000) },
			},
			func(data map[string]interface{}) interface{} {
				resolved := make(map[string]interface{})
				for k, v := range data {
					switch fn := v.(type) {
					case func() int64:
						resolved[k] = fn()
					case func() string:
						resolved[k] = fn()
					case func() time.Time:
						resolved[k] = fn()
					case func() int:
						resolved[k] = fn()
					default:
						resolved[k] = v
					}
				}
				return resolved
			},
		),
	}
}

// CommentFactory creates comment test data.
type CommentFactory struct {
	*BaseFactory
}

// NewCommentFactory creates a factory for generating comment data.
func NewCommentFactory() *CommentFactory {
	return &CommentFactory{
		BaseFactory: NewBaseFactory(
			map[string]interface{}{
				"id":         SequenceID,
				"post_id":    SequenceID,
				"user_id":    SequenceID,
				"content":    func() string { return "This is a test comment. " + RandomString(30) },
				"created_at": time.Now,
				"likes":      func() int { return RandomInt(0, 100) },
			},
			func(data map[string]interface{}) interface{} {
				resolved := make(map[string]interface{})
				for k, v := range data {
					switch fn := v.(type) {
					case func() int64:
						resolved[k] = fn()
					case func() string:
						resolved[k] = fn()
					case func() time.Time:
						resolved[k] = fn()
					case func() int:
						resolved[k] = fn()
					default:
						resolved[k] = v
					}
				}
				return resolved
			},
		),
	}
}

// FactoryRegistry manages multiple factories.
type FactoryRegistry struct {
	factories map[string]Factory
}

// NewFactoryRegistry creates a new factory registry.
func NewFactoryRegistry() *FactoryRegistry {
	return &FactoryRegistry{
		factories: make(map[string]Factory),
	}
}

// Register adds a factory to the registry.
func (r *FactoryRegistry) Register(name string, factory Factory) {
	r.factories[name] = factory
}

// Get retrieves a factory by name.
func (r *FactoryRegistry) Get(name string) (Factory, bool) {
	f, ok := r.factories[name]
	return f, ok
}

// Build creates a single instance from a named factory.
func (r *FactoryRegistry) Build(name string, options ...Option) (interface{}, error) {
	factory, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("factory not found: %s", name)
	}
	return factory.Build(options...), nil
}

// BuildList creates multiple instances from a named factory.
func (r *FactoryRegistry) BuildList(name string, count int, options ...Option) ([]interface{}, error) {
	factory, ok := r.Get(name)
	if !ok {
		return nil, fmt.Errorf("factory not found: %s", name)
	}
	return factory.BuildList(count, options...), nil
}

// DefaultRegistry is a global registry with common factories.
var DefaultRegistry = NewFactoryRegistry()

func init() {
	// Register built-in factories
	DefaultRegistry.Register("user", NewUserFactory())
	DefaultRegistry.Register("post", NewPostFactory())
	DefaultRegistry.Register("comment", NewCommentFactory())
}

// Helper functions for common patterns

// BuildUser is a shorthand for building a user.
func BuildUser(options ...Option) map[string]interface{} {
	result, _ := DefaultRegistry.Build("user", options...)
	return result.(map[string]interface{})
}

// BuildUsers builds multiple users.
func BuildUsers(count int, options ...Option) []map[string]interface{} {
	results, _ := DefaultRegistry.BuildList("user", count, options...)
	users := make([]map[string]interface{}, len(results))
	for i, r := range results {
		users[i] = r.(map[string]interface{})
	}
	return users
}

// BuildPost is a shorthand for building a post.
func BuildPost(options ...Option) map[string]interface{} {
	result, _ := DefaultRegistry.Build("post", options...)
	return result.(map[string]interface{})
}

// BuildPosts builds multiple posts.
func BuildPosts(count int, options ...Option) []map[string]interface{} {
	results, _ := DefaultRegistry.BuildList("post", count, options...)
	posts := make([]map[string]interface{}, len(results))
	for i, r := range results {
		posts[i] = r.(map[string]interface{})
	}
	return posts
}

// BuildComment is a shorthand for building a comment.
func BuildComment(options ...Option) map[string]interface{} {
	result, _ := DefaultRegistry.Build("comment", options...)
	return result.(map[string]interface{})
}

// BuildComments builds multiple comments.
func BuildComments(count int, options ...Option) []map[string]interface{} {
	results, _ := DefaultRegistry.BuildList("comment", count, options...)
	comments := make([]map[string]interface{}, len(results))
	for i, r := range results {
		comments[i] = r.(map[string]interface{})
	}
	return comments
}
