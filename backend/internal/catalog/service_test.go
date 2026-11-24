package catalog

import (
	"context"
	"errors"
	"testing"
)

func TestListReturnsEmptySliceNotNil(t *testing.T) {
	res, err := NewService(&memRepository{}).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Items == nil || len(res.Items) != 0 {
		t.Fatalf("items = %#v, want empty non-nil slice", res.Items)
	}
}

func TestListKeepsOrderAndParents(t *testing.T) {
	parent := int64(1)
	repo := &memRepository{items: []Category{
		{ID: 1, Name: "Electronics"},
		{ID: 2, Name: "Phones", ParentID: &parent},
	}}
	res, err := NewService(repo).List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 2 || res.Items[0].ID != 1 || res.Items[1].ParentID == nil || *res.Items[1].ParentID != 1 {
		t.Fatalf("items = %#v", res.Items)
	}
}

func TestListPassesRepositoryError(t *testing.T) {
	boom := errors.New("boom")
	if _, err := NewService(&memRepository{err: boom}).List(context.Background()); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want boom", err)
	}
}
