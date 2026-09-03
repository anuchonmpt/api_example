package app

import (
	"reflect"
	"testing"
)

func TestResourceStackClosesInReverseOrderOnce(t *testing.T) {
	t.Parallel()
	var order []string
	stack := &resourceStack{}
	stack.Add(func() error { order = append(order, "database"); return nil })
	stack.Add(func() error { order = append(order, "redis"); return nil })
	if err := stack.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stack.Close(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"redis", "database"}; !reflect.DeepEqual(order, want) {
		t.Fatalf("close order = %v, want %v", order, want)
	}
}
