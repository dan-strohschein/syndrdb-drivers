package testutil_test

import (
	"testing"
	"time"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/testutil"
)

func TestTestDBName(t *testing.T) {
	name1 := testutil.TestDBName("test")
	name2 := testutil.TestDBName("test")
	if name1 == name2 {
		t.Error("expected unique names")
	}
}

func TestTestBundleName(t *testing.T) {
	name1 := testutil.TestBundleName("test")
	name2 := testutil.TestBundleName("test")
	if name1 == name2 {
		t.Error("expected unique names")
	}
}

func TestWithTimeout(t *testing.T) {
	ctx, _ := testutil.WithTimeout(t, 100*time.Millisecond)
	select {
	case <-ctx.Done():
		t.Fatal("context canceled too early")
	default:
	}
}

func TestAssertEqual(t *testing.T) {
	testutil.AssertEqual(t, 42, 42, "values should be equal")
}

func TestAssertNotEqual(t *testing.T) {
	testutil.AssertNotEqual(t, 42, 43, "values should not be equal")
}

func TestAssertContains(t *testing.T) {
	testutil.AssertContains(t, "hello world", "world", "should contain substring")
}

func TestWaitFor(t *testing.T) {
	counter := 0
	condition := func() bool {
		counter++
		return counter >= 3
	}
	testutil.WaitFor(t, 1*time.Second, 100*time.Millisecond, condition)
	if counter < 3 {
		t.Errorf("expected counter >= 3, got %d", counter)
	}
}

func TestEventually(t *testing.T) {
	counter := 0
	condition := func() bool {
		counter++
		return counter >= 2
	}
	testutil.Eventually(t, 1*time.Second, 100*time.Millisecond, condition)
	if counter < 2 {
		t.Errorf("expected counter >= 2, got %d", counter)
	}
}

func TestParallel(t *testing.T) {
	result := testutil.Parallel(t)
	if result != t {
		t.Error("expected Parallel to return the test instance")
	}
}

func TestSkipIf(t *testing.T) {
	testutil.SkipIf(t, false, "should not skip")
}

func TestSkipUnless(t *testing.T) {
	testutil.SkipUnless(t, true, "should not skip")
}
