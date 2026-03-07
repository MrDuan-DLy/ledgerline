package services

import (
	"regexp"
	"strings"
	"sync"

	"github.com/jmoiron/sqlx"
)

type cachedRule struct {
	ID          int    `db:"id"`
	Pattern     string `db:"pattern"`
	PatternType string `db:"pattern_type"`
	CategoryID  int    `db:"category_id"`
	Priority    int    `db:"priority"`
	compiledRE  *regexp.Regexp
}

// ClassifyService handles automatic transaction classification.
type ClassifyService struct {
	db    *sqlx.DB
	mu    sync.RWMutex
	rules []cachedRule
}

// NewClassifyService creates a new ClassifyService.
func NewClassifyService(db *sqlx.DB) *ClassifyService {
	return &ClassifyService{db: db}
}

// loadRules loads active rules ordered by priority (highest first).
// Caller must hold s.mu write lock.
func (s *ClassifyService) loadRules() error {
	var rules []cachedRule
	err := s.db.Select(&rules,
		`SELECT id, pattern, pattern_type, category_id, priority
		 FROM rules WHERE is_active = 1 ORDER BY priority DESC`)
	if err != nil {
		return err
	}
	for i := range rules {
		if rules[i].PatternType == "regex" {
			rules[i].compiledRE, _ = regexp.Compile("(?i)" + rules[i].Pattern)
		}
	}
	s.rules = rules
	return nil
}

// InvalidateCache clears the rules cache.
func (s *ClassifyService) InvalidateCache() {
	s.mu.Lock()
	s.rules = nil
	s.mu.Unlock()
}

// Classify returns (categoryID, source) for a description.
func (s *ClassifyService) Classify(description string) (categoryID *int, source string) {
	s.mu.RLock()
	rules := s.rules
	s.mu.RUnlock()

	if rules == nil {
		s.mu.Lock()
		if s.rules == nil {
			if err := s.loadRules(); err != nil {
				s.mu.Unlock()
				return nil, "unclassified"
			}
		}
		rules = s.rules
		s.mu.Unlock()
	}

	upper := strings.ToUpper(description)
	for _, rule := range rules {
		if matches(rule, upper) {
			id := rule.CategoryID
			return &id, "rule"
		}
	}
	return nil, "unclassified"
}

func matches(rule cachedRule, descUpper string) bool {
	patUpper := strings.ToUpper(rule.Pattern)
	switch rule.PatternType {
	case "exact":
		return descUpper == patUpper
	case "contains":
		return strings.Contains(descUpper, patUpper)
	case "regex":
		if rule.compiledRE != nil {
			return rule.compiledRE.MatchString(descUpper)
		}
		return false
	}
	return false
}

// ReclassifyAll re-runs rules on all non-manual transactions.
func (s *ClassifyService) ReclassifyAll() (int, error) {
	s.InvalidateCache()

	type txnRow struct {
		ID             int    `db:"id"`
		RawDescription string `db:"raw_description"`
		CategoryID     *int   `db:"category_id"`
	}

	var txns []txnRow
	err := s.db.Select(&txns,
		`SELECT id, raw_description, category_id FROM transactions WHERE category_source != 'manual'`)
	if err != nil {
		return 0, err
	}

	updated := 0
	for _, txn := range txns {
		catID, source := s.Classify(txn.RawDescription)

		// Check if changed
		changed := false
		if catID == nil && txn.CategoryID != nil {
			changed = true
		} else if catID != nil && txn.CategoryID == nil {
			changed = true
		} else if catID != nil && txn.CategoryID != nil && *catID != *txn.CategoryID {
			changed = true
		}

		if changed {
			if catID != nil {
				_, err = s.db.Exec(
					`UPDATE transactions SET category_id = ?, category_source = ? WHERE id = ?`,
					*catID, source, txn.ID)
			} else {
				_, err = s.db.Exec(
					`UPDATE transactions SET category_id = NULL, category_source = ? WHERE id = ?`,
					source, txn.ID)
			}
			if err != nil {
				return updated, err
			}
			updated++
		}
	}

	return updated, nil
}
