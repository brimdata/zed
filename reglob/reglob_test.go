package reglob_test

import (
	"testing"

	"github.com/mccanne/zq/reglob"
)

func TestReglob(t *testing.T) {
	expected := "^S.*$"
	actual := reglob.Reglob("S*")

	if actual != expected {
		t.Fatalf("Expected '%s' to equal '%s'", actual, expected)
	}
}

func Test_SingleStar(t *testing.T) {
	expected := "^.*$"
	actual := reglob.Reglob("*")

	if actual != expected {
		t.Fatalf("Expected '%s' to equal '%s'", actual, expected)
	}
}
