package cache

import "testing"

func TestListKey(t *testing.T) {
	if listKey("all", 0) != "tasks:list:all" {
		t.Fatal("all key")
	}
	if listKey("completed", 0) != "tasks:list:completed" {
		t.Fatal("completed key")
	}
	if listKey("mine", 42) != "tasks:list:mine:42" {
		t.Fatal("mine key")
	}
	if listKey("", 0) != "tasks:list:all" {
		t.Fatal("default key")
	}
}
