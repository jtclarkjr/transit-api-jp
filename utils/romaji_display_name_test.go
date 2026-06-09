package utils

import "testing"

func TestRomajiDisplayNameFormatsBracketedQualifier(t *testing.T) {
	got, err := RomajiDisplayName("押上[スカイツリー前]")
	if err != nil {
		t.Fatalf("RomajiDisplayName returned error: %v", err)
	}

	want := "Oshiage [Skytree front]"
	if got != want {
		t.Fatalf("RomajiDisplayName() = %q, want %q", got, want)
	}
}

func TestRomajiDisplayNameConvertsStationSuffix(t *testing.T) {
	got, err := RomajiDisplayName("日の出駅")
	if err != nil {
		t.Fatalf("RomajiDisplayName returned error: %v", err)
	}

	want := "Hinode Station"
	if got != want {
		t.Fatalf("RomajiDisplayName() = %q, want %q", got, want)
	}
}
