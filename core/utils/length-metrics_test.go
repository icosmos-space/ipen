package utils

import (
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestLengthMetrics_CountingModes(t *testing.T) {
	if got := CountChapterLength("他抬头看天。", models.CountingModeZHChars); got != 6 {
		t.Fatalf("expected 6 zh chars, got %d", got)
	}
	if got := CountChapterLength("He looked at the sky.", models.CountingModeENWords); got != 5 {
		t.Fatalf("expected 5 english words, got %d", got)
	}
}

func TestLengthMetrics_CountsProseOnlyForMarkdown(t *testing.T) {
	chapter := "---\ntitle: 第1章 归来\n---\n\n# 第1章 归来\n\n陈风抬头看天。"
	if got := CountChapterLength(chapter, models.CountingModeZHChars); got != len([]rune("陈风抬头看天。")) {
		t.Fatalf("expected prose-only count, got %d", got)
	}
}

func TestLengthMetrics_BuildLengthSpec(t *testing.T) {
	spec := BuildLengthSpec(2200, LanguageZH)
	if spec.Target != 2200 || spec.SoftMin != 1900 || spec.SoftMax != 2500 || spec.HardMin != 1600 || spec.HardMax != 2800 || spec.CountingMode != models.CountingModeZHChars || spec.NormalizeMode != models.NormalizeModeNone {
		t.Fatalf("unexpected zh spec: %#v", spec)
	}

	enSpec := BuildLengthSpec(2200, LanguageEN)
	if enSpec.CountingMode != models.CountingModeENWords || enSpec.SoftMin != 1900 || enSpec.SoftMax != 2500 || enSpec.HardMin != 1600 || enSpec.HardMax != 2800 {
		t.Fatalf("unexpected en spec: %#v", enSpec)
	}

	small := BuildLengthSpec(220, LanguageZH)
	if small.SoftMin != 190 || small.SoftMax != 250 || small.HardMin != 160 || small.HardMax != 280 {
		t.Fatalf("unexpected scaled spec: %#v", small)
	}
}

func TestLengthMetrics_RangeAndNormalizeMode(t *testing.T) {
	spec := BuildLengthSpec(2200, LanguageZH)
	if !IsOutsideSoftRange(1800, spec) || IsOutsideSoftRange(2200, spec) {
		t.Fatalf("soft range detection mismatch")
	}
	if !IsOutsideHardRange(1500, spec) || IsOutsideHardRange(2200, spec) {
		t.Fatalf("hard range detection mismatch")
	}

	if mode := ChooseNormalizeMode(1800, spec); mode != models.NormalizeModeExpand {
		t.Fatalf("expected expand, got %s", mode)
	}
	if mode := ChooseNormalizeMode(2200, spec); mode != models.NormalizeModeNone {
		t.Fatalf("expected none, got %s", mode)
	}
	if mode := ChooseNormalizeMode(2600, spec); mode != models.NormalizeModeCompress {
		t.Fatalf("expected compress, got %s", mode)
	}
}
