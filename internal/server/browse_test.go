package server

import (
	"strings"
	"testing"
)

func TestParseBrowsePS(t *testing.T) {
	if g := parseBrowsePS("", 50, 100); g != 50 {
		t.Fatalf("default got %d", g)
	}
	if g := parseBrowsePS("100", 50, 80); g != 50 {
		t.Fatalf("invalid ps when max 80 should fall back to default, got %d", g)
	}
	if g := parseBrowsePS("20", 50, 100); g != 20 {
		t.Fatalf("choice got %d", g)
	}
	if g := parseBrowsePS("999", 50, 100); g != 50 {
		t.Fatalf("invalid should fall back, got %d", g)
	}
}

func TestBuildBrowseFilter(t *testing.T) {
	cols := map[string]struct{}{"id": {}, "name": {}}
	_, args, err := buildBrowseFilter("", "", "", cols)
	if err != nil || args != nil {
		t.Fatal(err, args)
	}
	w, args, err := buildBrowseFilter("name", "contains", "ab", cols)
	if err != nil || !strings.Contains(w, "WHERE") || !strings.Contains(w, "LIKE") || len(args) != 1 {
		t.Fatalf("contains: %q %v %v", w, args, err)
	}
	if pat, ok := args[0].(string); !ok || pat != "%ab%" {
		t.Fatalf("pattern %v", args)
	}
	_, _, err = buildBrowseFilter("evil;", "eq", "x", cols)
	if err == nil {
		t.Fatal("want reject col")
	}
	_, _, err = buildBrowseFilter("missing", "eq", "x", cols)
	if err == nil {
		t.Fatal("want unknown col")
	}
	_, _, err = buildBrowseFilter("name", "bogus", "x", cols)
	if err == nil {
		t.Fatal("want bad op")
	}
	w, args, err = buildBrowseFilter("", "contains", "anything", cols)
	if err != nil || w != "" || args != nil {
		t.Fatalf("empty fcol must ignore fop/fval: where=%q args=%v err=%v", w, args, err)
	}
}

func TestEscapeLikePatternWithBang(t *testing.T) {
	if g := escapeLikePatternWithBang("a%b_c!"); g != "a!%b!_c!!" {
		t.Fatal(g)
	}
}
