package agents

import (
	"reflect"
	"strings"
	"testing"

	"github.com/icosmos-space/ipen/core/models"
)

func TestGetFanficDimensionConfig_ActiveAndDeactivated(t *testing.T) {
	expectedActive := []int{34, 35, 36, 37}
	expectedDeactivated := []int{28, 29, 30, 31}
	modes := []models.FanficMode{models.FanficModeCanon, models.FanficModeAU, models.FanficModeOOC, models.FanficModeCP}

	for _, mode := range modes {
		cfg := GetFanficDimensionConfig(mode, nil)
		if !reflect.DeepEqual(cfg.ActiveIDs, expectedActive) {
			t.Fatalf("mode %s active ids mismatch: %#v", mode, cfg.ActiveIDs)
		}
		if !reflect.DeepEqual(cfg.DeactivatedIDs, expectedDeactivated) {
			t.Fatalf("mode %s deactivated ids mismatch: %#v", mode, cfg.DeactivatedIDs)
		}
	}
}

func TestGetFanficDimensionConfig_SeverityOverrides(t *testing.T) {
	canon := GetFanficDimensionConfig(models.FanficModeCanon, nil)
	if canon.SeverityOverrides[34] != "critical" || canon.SeverityOverrides[35] != "critical" || canon.SeverityOverrides[36] != "warning" || canon.SeverityOverrides[37] != "critical" {
		t.Fatalf("canon severity overrides mismatch: %#v", canon.SeverityOverrides)
	}

	au := GetFanficDimensionConfig(models.FanficModeAU, nil)
	if au.SeverityOverrides[34] != "critical" || au.SeverityOverrides[35] != "info" || au.SeverityOverrides[37] != "info" {
		t.Fatalf("au severity overrides mismatch: %#v", au.SeverityOverrides)
	}

	ooc := GetFanficDimensionConfig(models.FanficModeOOC, nil)
	if ooc.SeverityOverrides[1] != "info" || ooc.SeverityOverrides[34] != "info" {
		t.Fatalf("ooc severity overrides mismatch: %#v", ooc.SeverityOverrides)
	}

	cp := GetFanficDimensionConfig(models.FanficModeCP, nil)
	if cp.SeverityOverrides[36] != "critical" {
		t.Fatalf("cp expected dim 36 to be critical, got %q", cp.SeverityOverrides[36])
	}
}

func TestGetFanficDimensionConfig_NotesCoverage(t *testing.T) {
	cfg := GetFanficDimensionConfig(models.FanficModeCanon, nil)
	if note, ok := cfg.Notes[1]; !ok || note == "" || !strings.Contains(note, "fanfic_canon.md") {
		t.Fatalf("expected canon note for dim1 to mention fanfic_canon.md, got %q", note)
	}

	for _, dim := range FANFIC_DIMENSIONS {
		note, ok := cfg.Notes[dim.ID]
		if !ok || note == "" {
			t.Fatalf("missing note for dim %d", dim.ID)
		}
	}
}
