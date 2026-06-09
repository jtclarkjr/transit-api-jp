package handler

import (
	"strings"
	"testing"
)

func TestResolveTransitNodeAcceptsNodeID(t *testing.T) {
	got, err := resolveTransitNode("00004212")
	if err != nil {
		t.Fatalf("resolveTransitNode returned error: %v", err)
	}
	if got != "00004212" {
		t.Fatalf("resolveTransitNode() = %q, want node ID unchanged", got)
	}
}

func TestIsNodeID(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "00004212", want: true},
		{value: "Hinode (Tokyo)", want: false},
		{value: "東京駅", want: false},
		{value: "1234", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := isNodeID(tt.value); got != tt.want {
				t.Fatalf("isNodeID(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}

func TestBuildTransportNodeURLEncodesStation(t *testing.T) {
	got := buildTransportNodeURL("example.test", "Hinode (Tokyo)")

	if !strings.Contains(got, "word=Hinode+%28Tokyo%29") {
		t.Fatalf("URL = %q, want encoded word query", got)
	}
}

func TestResolveTransitNodeRejectsTranslatedDisplayName(t *testing.T) {
	_, err := resolveTransitNode("Hinode (Tokyo)")
	if err == nil {
		t.Fatal("expected translated display name to be rejected")
	}
	if !strings.Contains(err.Error(), "translated display name") {
		t.Fatalf("error = %q, want translated display name guidance", err.Error())
	}
}

func TestLooksLikeTranslatedDisplayName(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{value: "Hinode (Tokyo)", want: true},
		{value: "Tokyo", want: true},
		{value: "東京駅", want: false},
		{value: "JR難波", want: false},
		{value: "00004212", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			if got := looksLikeTranslatedDisplayName(tt.value); got != tt.want {
				t.Fatalf("looksLikeTranslatedDisplayName(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}
}
