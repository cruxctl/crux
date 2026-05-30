package mcp

import (
	"strings"
	"unicode"
)

// InvertedIndex is a simple full-text index.
type InvertedIndex struct {
	index map[string]map[string]bool // term -> docID set
}

// NewInvertedIndex creates an empty index.
func NewInvertedIndex() *InvertedIndex {
	return &InvertedIndex{index: map[string]map[string]bool{}}
}

// Build reconstructs the index from documents.
func (ii *InvertedIndex) Build(docs []SearchDoc) {
	ii.index = map[string]map[string]bool{}
	for _, doc := range docs {
		for _, term := range tokenize(doc.Content) {
			if ii.index[term] == nil {
				ii.index[term] = map[string]bool{}
			}
			ii.index[term][doc.ID] = true
		}
	}
}

// Search finds doc IDs matching all query terms.
func (ii *InvertedIndex) Search(query string) []string {
	terms := tokenize(query)
	if len(terms) == 0 {
		return nil
	}
	// Intersect posting lists.
	var result map[string]bool
	for _, term := range terms {
		set := ii.index[term]
		if set == nil {
			return nil
		}
		if result == nil {
			result = make(map[string]bool, len(set))
			for id := range set {
				result[id] = true
			}
		} else {
			for id := range result {
				if !set[id] {
					delete(result, id)
				}
			}
		}
	}
	out := make([]string, 0, len(result))
	for id := range result {
		out = append(out, id)
	}
	return out
}

func tokenize(text string) []string {
	var terms []string
	for _, word := range strings.FieldsFunc(text, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		word = strings.ToLower(word)
		if len(word) > 2 {
			terms = append(terms, word)
		}
	}
	return terms
}
