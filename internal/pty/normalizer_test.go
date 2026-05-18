package pty

import (
	"context"
	"testing"
)

func TestNormalizerRendersFinalScreenWithCursorMoves(t *testing.T) {
	normalizer := NewNormalizer()
	result, err := normalizer.Normalize(context.Background(), PTYRawOutput{
		RawBytes: []byte("\x1b[2J\x1b[1;1HStatus: loading\x1b[1;9Hready\x1b[K"),
	}, NormalizeSpec{
		StripANSI:           true,
		StripControlChars:   true,
		NormalizeWhitespace: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.FinalScreen != "Status: ready" {
		t.Fatalf("expected rendered final screen, got %q", result.FinalScreen)
	}
}

func TestNormalizerStripsSingleCharacterEscapes(t *testing.T) {
	normalizer := NewNormalizer()
	result, err := normalizer.Normalize(context.Background(), PTYRawOutput{
		RawBytes: []byte("usage\n\x1b7\x1b[r\x1b8"),
	}, NormalizeSpec{
		StripANSI:           true,
		StripControlChars:   true,
		NormalizeWhitespace: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.CleanText != "usage" {
		t.Fatalf("expected single-character escapes to be removed, got %q", result.CleanText)
	}
}
