package services

import (
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"github.com/adrg/strutil"
	"github.com/adrg/strutil/metrics"
	"github.com/jmoiron/sqlx"
)

var (
	suffixRE    = regexp.MustCompile(`(?i)\b(ltd|limited|plc|inc|incorporated|corp|corporation|co|llc|llp|uk|group)\b`)
	nonAlphaRE  = regexp.MustCompile(`[^a-z0-9\s]`)
	multiSpaceRE = regexp.MustCompile(`\s+`)
)

// Normalize lowercases, strips suffixes, removes non-alpha, collapses spaces.
func Normalize(raw string) string {
	text := strings.ToLower(strings.TrimSpace(raw))
	text = suffixRE.ReplaceAllString(text, "")
	text = nonAlphaRE.ReplaceAllString(text, " ")
	text = multiSpaceRE.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

type merchantRow struct {
	ID         int    `db:"id"`
	Name       string `db:"name"`
	Patterns   string `db:"patterns"`
	CategoryID *int64 `db:"category_id"`
}

func (m *merchantRow) getPatterns() []string {
	var patterns []string
	json.Unmarshal([]byte(m.Patterns), &patterns)
	return patterns
}

// MerchantService matches raw merchant names to canonical merchants.
type MerchantService struct {
	db *sqlx.DB
}

func NewMerchantService(db *sqlx.DB) *MerchantService {
	return &MerchantService{db: db}
}

func (s *MerchantService) loadAll() []merchantRow {
	var merchants []merchantRow
	if err := s.db.Select(&merchants, `SELECT id, name, patterns, category_id FROM merchants`); err != nil {
		log.Printf("loadAll merchants error: %v", err)
	}
	return merchants
}

// Match attempts to find a matching merchant. Returns (merchantID, merchantName, score, matchType).
func (s *MerchantService) Match(rawName string) (*int, *string, float64, string) {
	if strings.TrimSpace(rawName) == "" {
		return nil, nil, 0, "none"
	}

	norm := Normalize(rawName)
	if norm == "" {
		return nil, nil, 0, "none"
	}

	merchants := s.loadAll()
	normTokens := toSet(strings.Fields(norm))

	var bestID *int
	var bestName *string
	bestScore := 0.0
	bestType := "none"

	jw := metrics.NewJaroWinkler()

	for _, m := range merchants {
		canonicalNorm := Normalize(m.Name)

		// 1. Exact match on canonical name
		if norm == canonicalNorm {
			id := m.ID
			name := m.Name
			return &id, &name, 1.0, "exact"
		}

		// 2. Exact match on patterns
		for _, pattern := range m.getPatterns() {
			if norm == pattern {
				id := m.ID
				name := m.Name
				return &id, &name, 1.0, "exact"
			}
		}

		// 3. Token subset
		canonicalTokens := toSet(strings.Fields(canonicalNorm))
		if len(canonicalTokens) > 0 && isSubset(canonicalTokens, normTokens) {
			coverage := float64(len(canonicalTokens)) / float64(len(normTokens))
			score := 0.8 + (0.15 * coverage)
			if score > bestScore {
				id := m.ID
				name := m.Name
				bestID = &id
				bestName = &name
				bestScore = score
				bestType = "token_subset"
			}
		}

		for _, pattern := range m.getPatterns() {
			patTokens := toSet(strings.Fields(pattern))
			if len(patTokens) > 0 && isSubset(patTokens, normTokens) {
				coverage := float64(len(patTokens)) / float64(len(normTokens))
				score := 0.8 + (0.15 * coverage)
				if score > bestScore {
					id := m.ID
					name := m.Name
					bestID = &id
					bestName = &name
					bestScore = score
					bestType = "token_subset"
				}
			}
		}

		// 4. Fuzzy match on canonical
		ratio := strutil.Similarity(norm, canonicalNorm, jw)
		if ratio >= 0.65 && ratio > bestScore {
			id := m.ID
			name := m.Name
			bestID = &id
			bestName = &name
			bestScore = ratio
			bestType = "fuzzy"
		}

		for _, pattern := range m.getPatterns() {
			ratio := strutil.Similarity(norm, pattern, jw)
			if ratio >= 0.65 && ratio > bestScore {
				id := m.ID
				name := m.Name
				bestID = &id
				bestName = &name
				bestScore = ratio
				bestType = "fuzzy"
			}
		}
	}

	if bestID != nil {
		return bestID, bestName, Round2(bestScore*1000) / 1000, bestType
	}
	return nil, nil, 0, "none"
}

// Resolve returns canonical name if matched, else the raw name.
func (s *MerchantService) Resolve(rawName string) string {
	if rawName == "" {
		return rawName
	}
	_, name, _, matchType := s.Match(rawName)
	if name != nil && matchType != "none" {
		return *name
	}
	return rawName
}

// GetOrCreate gets existing or creates a new merchant. Learns pattern from raw.
func (s *MerchantService) GetOrCreate(canonical string, raw string) (int, error) {
	normCanonical := Normalize(canonical)

	// Try exact canonical match
	var m merchantRow
	err := s.db.Get(&m, `SELECT id, name, patterns, category_id FROM merchants WHERE name = ?`, canonical)
	if err == nil {
		if raw != "" {
			normRaw := Normalize(raw)
			if normRaw != "" && normRaw != Normalize(m.Name) {
				addPattern(s.db, m.ID, m.Patterns, normRaw)
			}
		}
		return m.ID, nil
	}

	// Try matching by normalized name
	merchants := s.loadAll()
	for _, existing := range merchants {
		if Normalize(existing.Name) == normCanonical {
			if raw != "" {
				normRaw := Normalize(raw)
				if normRaw != "" && normRaw != Normalize(existing.Name) {
					addPattern(s.db, existing.ID, existing.Patterns, normRaw)
				}
			}
			return existing.ID, nil
		}
	}

	// Create new
	patterns := []string{}
	if raw != "" {
		normRaw := Normalize(raw)
		if normRaw != "" && normRaw != normCanonical {
			patterns = append(patterns, normRaw)
		}
	}
	patternsJSON, _ := json.Marshal(patterns)

	res, err := s.db.Exec(
		`INSERT INTO merchants (name, patterns) VALUES (?, ?)`,
		canonical, string(patternsJSON))
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	return int(id), nil
}

func addPattern(db *sqlx.DB, id int, existingJSON string, pattern string) {
	var patterns []string
	json.Unmarshal([]byte(existingJSON), &patterns)
	for _, p := range patterns {
		if p == pattern {
			return
		}
	}
	patterns = append(patterns, pattern)
	pj, _ := json.Marshal(patterns)
	if _, err := db.Exec(`UPDATE merchants SET patterns = ? WHERE id = ?`, string(pj), id); err != nil {
		log.Printf("addPattern update error: %v", err)
	}
}

func toSet(tokens []string) map[string]bool {
	s := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		s[t] = true
	}
	return s
}

func isSubset(subset, superset map[string]bool) bool {
	for k := range subset {
		if !superset[k] {
			return false
		}
	}
	return true
}
