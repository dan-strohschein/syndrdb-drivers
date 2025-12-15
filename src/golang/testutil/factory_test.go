package testutil_test

import (
	"testing"

	"github.com/dan-strohschein/syndrdb-drivers/src/golang/testutil"
)

func TestUserFactory_Build(t *testing.T) {
	factory := testutil.NewUserFactory()
	user := factory.Build()
	userMap := user.(map[string]interface{})

	requiredFields := []string{"id", "email", "username", "name", "created_at", "active"}
	for _, field := range requiredFields {
		if _, ok := userMap[field]; !ok {
			t.Errorf("missing required field: %s", field)
		}
	}

	if userMap["active"] != true {
		t.Errorf("expected active=true, got %v", userMap["active"])
	}
}

func TestUserFactory_BuildWithOptions(t *testing.T) {
	factory := testutil.NewUserFactory()
	user := factory.Build(
		testutil.WithField("name", "Custom Name"),
		testutil.WithField("active", false),
	)
	userMap := user.(map[string]interface{})

	if userMap["name"] != "Custom Name" {
		t.Errorf("expected name='Custom Name', got %v", userMap["name"])
	}
}

func TestUserFactory_BuildList(t *testing.T) {
	factory := testutil.NewUserFactory()
	users := factory.BuildList(5)
	if len(users) != 5 {
		t.Errorf("expected 5 users, got %d", len(users))
	}
}

func TestBuildUsers_Shorthand(t *testing.T) {
	users := testutil.BuildUsers(3)
	if len(users) != 3 {
		t.Errorf("expected 3 users, got %d", len(users))
	}
}

func TestSequenceGenerators(t *testing.T) {
	email1 := testutil.SequenceEmail()
	email2 := testutil.SequenceEmail()
	if email1 == email2 {
		t.Error("expected unique emails")
	}

	id1 := testutil.SequenceID()
	id2 := testutil.SequenceID()
	if id2 <= id1 {
		t.Error("expected increasing IDs")
	}
}

func TestRandomGenerators(t *testing.T) {
	str := testutil.RandomString(10)
	if len(str) != 10 {
		t.Errorf("expected string length 10, got %d", len(str))
	}

	val := testutil.RandomInt(1, 10)
	if val < 1 || val > 10 {
		t.Errorf("expected value between 1-10, got %d", val)
	}
}
