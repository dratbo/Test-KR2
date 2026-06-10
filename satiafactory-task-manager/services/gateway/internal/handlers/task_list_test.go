package handlers

import "testing"

func TestFilterTaskViews(t *testing.T) {
	views := []TaskView{
		{Title: "Железные пластины", RecipeName: "Железная пластина", AssigneeName: "Иван"},
		{Title: "Турбомоторы", RecipeName: "Турбомотор", CreatorName: "Мария"},
	}
	filtered := filterTaskViews(views, "турбо")
	if len(filtered) != 1 || filtered[0].Title != "Турбомоторы" {
		t.Fatalf("expected one turbo task, got %v", filtered)
	}
}

func TestPaginateTaskViews(t *testing.T) {
	views := make([]TaskView, 12)
	for i := range views {
		views[i].Title = "Task"
	}
	paged, page, totalPages := paginateTaskViews(views, 2)
	if len(paged) != 5 {
		t.Fatalf("expected 5 tasks on page, got %d", len(paged))
	}
	if page != 2 || totalPages != 3 {
		t.Fatalf("expected page 2 of 3, got %d of %d", page, totalPages)
	}
}
