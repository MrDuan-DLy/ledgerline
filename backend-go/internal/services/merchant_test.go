package services

import (
	"testing"

	_ "modernc.org/sqlite"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"strips suffix PLC", "Tesco PLC", "tesco"},
		{"strips suffix Ltd and trims", "  UBER EATS Ltd  ", "uber eats"},
		{"removes apostrophe", "McDonald's", "mcdonald s"},
		{"empty string", "", ""},
		{"strips multiple suffixes", "Amazon Inc UK", "amazon"},
		{"collapses multiple spaces", "Some   Company   Name", "some company name"},
		{"lowercases everything", "ALL UPPER CASE", "all upper case"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := Normalize(tc.input)
			if got != tc.expected {
				t.Errorf("Normalize(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestFilterTokens(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		minLen   int
		expected int
	}{
		{"filters short tokens", []string{"AB", "HELLO", "XY", "WORLD"}, 3, 2},
		{"keeps all when all long enough", []string{"HELLO", "WORLD"}, 3, 2},
		{"empty input", []string{}, 3, 0},
		{"all too short", []string{"A", "BC"}, 3, 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := filterTokens(tc.tokens, tc.minLen)
			if len(got) != tc.expected {
				t.Errorf("filterTokens() returned %d tokens, want %d", len(got), tc.expected)
			}
		})
	}
}

func TestAnyTokenIn(t *testing.T) {
	tests := []struct {
		name     string
		tokens   []string
		text     string
		expected bool
	}{
		{"token found in text", []string{"TESCO", "ASDA"}, "TESCO STORES LONDON", true},
		{"no token found", []string{"UBER", "AMAZON"}, "TESCO STORES LONDON", false},
		{"empty tokens", []string{}, "ANYTHING", false},
		{"partial token match", []string{"TES"}, "TESCO", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := anyTokenIn(tc.tokens, tc.text)
			if got != tc.expected {
				t.Errorf("anyTokenIn() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestMerchantService_Match(t *testing.T) {
	db := setupTestDB(t)

	// Insert merchants with patterns
	db.MustExec(`INSERT INTO merchants (id, name, patterns, category_id) VALUES
		(1, 'Tesco', '["tesco express", "tesco metro"]', 1),
		(2, 'Uber Eats', '["ubereats", "uber eats"]', 2),
		(3, 'Amazon', '["amzn", "amazon prime"]', 3)`)
	db.MustExec(`INSERT INTO categories (id, name) VALUES (1, 'Groceries'), (2, 'Food'), (3, 'Shopping')`)

	svc := NewMerchantService(db)

	t.Run("exact match on canonical name", func(t *testing.T) {
		id, name, score, matchType := svc.Match("Tesco")
		if id == nil {
			t.Fatal("expected match, got nil")
		}
		if *id != 1 {
			t.Errorf("expected merchant 1, got %d", *id)
		}
		if name == nil || *name != "Tesco" {
			t.Errorf("expected name 'Tesco', got %v", name)
		}
		if score != 1.0 {
			t.Errorf("expected score 1.0, got %f", score)
		}
		if matchType != "exact" {
			t.Errorf("expected match_type 'exact', got %q", matchType)
		}
	})

	t.Run("exact match on pattern", func(t *testing.T) {
		id, name, _, matchType := svc.Match("tesco express")
		if id == nil {
			t.Fatal("expected match, got nil")
		}
		if *id != 1 {
			t.Errorf("expected merchant 1, got %d", *id)
		}
		if name == nil || *name != "Tesco" {
			t.Errorf("expected name 'Tesco', got %v", name)
		}
		if matchType != "exact" {
			t.Errorf("expected match_type 'exact', got %q", matchType)
		}
	})

	t.Run("token subset or fuzzy match", func(t *testing.T) {
		id, name, _, matchType := svc.Match("Uber Eats London Bridge")
		if id == nil {
			t.Fatal("expected match, got nil")
		}
		if name == nil || *name != "Uber Eats" {
			t.Errorf("expected name 'Uber Eats', got %v", name)
		}
		if matchType != "token_subset" && matchType != "exact" && matchType != "fuzzy" {
			t.Errorf("expected token_subset, exact, or fuzzy match, got %q", matchType)
		}
		_ = id
	})

	t.Run("no match for unrelated name", func(t *testing.T) {
		id, _, score, matchType := svc.Match("Completely Random Store XYZ")
		if matchType != "none" && id != nil {
			t.Errorf("expected no match, got id=%v score=%f type=%s", id, score, matchType)
		}
	})

	t.Run("empty string returns none", func(t *testing.T) {
		id, _, _, matchType := svc.Match("")
		if id != nil {
			t.Errorf("expected nil for empty string, got %v", *id)
		}
		if matchType != "none" {
			t.Errorf("expected 'none', got %q", matchType)
		}
	})
}

func TestMerchantService_GetOrCreate(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMerchantService(db)

	t.Run("create new merchant", func(t *testing.T) {
		id, err := svc.GetOrCreate("Tesco", "TESCO STORES LTD")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id <= 0 {
			t.Errorf("expected positive ID, got %d", id)
		}

		// Verify it exists
		var name string
		db.Get(&name, `SELECT name FROM merchants WHERE id = ?`, id)
		if name != "Tesco" {
			t.Errorf("expected name 'Tesco', got %q", name)
		}

		// Verify pattern was learned
		var patterns string
		db.Get(&patterns, `SELECT patterns FROM merchants WHERE id = ?`, id)
		if patterns == "[]" {
			t.Error("expected patterns to include normalized raw name")
		}
	})

	t.Run("get existing merchant", func(t *testing.T) {
		id1, _ := svc.GetOrCreate("Tesco", "")
		id2, _ := svc.GetOrCreate("Tesco", "")
		if id1 != id2 {
			t.Errorf("expected same ID for existing merchant, got %d and %d", id1, id2)
		}
	})

	t.Run("learn new pattern from raw", func(t *testing.T) {
		svc.GetOrCreate("Tesco", "TESCO METRO CHELSEA")

		var patterns string
		db.Get(&patterns, `SELECT patterns FROM merchants WHERE name = 'Tesco'`)
		if patterns == "[]" {
			t.Error("expected patterns to include new raw name")
		}
	})
}
