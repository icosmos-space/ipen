package utils

import "github.com/icosmos-space/ipen/core/models"

// HookPhase 表示where the story is in macro progression。
type HookPhase string

const (
	HookPhaseOpening HookPhase = "opening"
	HookPhaseMiddle  HookPhase = "middle"
	HookPhaseLate    HookPhase = "late"
)

// HookAgendaLoad 表示agenda load tiers。
type HookAgendaLoad string

const (
	HookAgendaLoadLight  HookAgendaLoad = "light"
	HookAgendaLoadMedium HookAgendaLoad = "medium"
	HookAgendaLoadHeavy  HookAgendaLoad = "heavy"
)

// HookLifecycleProfile defines timing behavior for a payoff timing bucket.
type HookLifecycleProfile struct {
	EarliestResolveAge int
	StaleDormancy      int
	OverdueAge         int
	MinimumPhase       HookPhase
	ResolveBias        int
}

var HOOK_TIMING_PROFILES = map[models.HookPayoffTiming]HookLifecycleProfile{
	models.TimingImmediate: {EarliestResolveAge: 1, StaleDormancy: 1, OverdueAge: 3, MinimumPhase: HookPhaseOpening, ResolveBias: 5},
	models.TimingNearTerm:  {EarliestResolveAge: 1, StaleDormancy: 2, OverdueAge: 5, MinimumPhase: HookPhaseOpening, ResolveBias: 4},
	models.TimingMidArc:    {EarliestResolveAge: 2, StaleDormancy: 4, OverdueAge: 8, MinimumPhase: HookPhaseOpening, ResolveBias: 3},
	models.TimingSlowBurn:  {EarliestResolveAge: 4, StaleDormancy: 5, OverdueAge: 12, MinimumPhase: HookPhaseMiddle, ResolveBias: 2},
	models.TimingEndgame:   {EarliestResolveAge: 6, StaleDormancy: 6, OverdueAge: 16, MinimumPhase: HookPhaseLate, ResolveBias: 1},
}

var HOOK_PHASE_WEIGHT = map[HookPhase]int{
	HookPhaseOpening: 0,
	HookPhaseMiddle:  1,
	HookPhaseLate:    2,
}

var HOOK_PHASE_THRESHOLDS = struct {
	MiddleProgress float64
	LateProgress   float64
	MiddleChapter  int
	LateChapter    int
}{
	MiddleProgress: 0.33,
	LateProgress:   0.72,
	MiddleChapter:  8,
	LateChapter:    24,
}

var HOOK_PRESSURE_WEIGHTS = struct {
	StaleAdvanceBonus         int
	OverdueAdvanceBonus       int
	ResolveBiasMultiplier     int
	ProgressingResolveBonus   int
	DormancyResolveMultiplier int
	MaxDormancyResolveBonus   int
	OverdueResolveBonus       int
	MustAdvancePressureFloor  int
	CriticalResolvePressure   int
}{
	StaleAdvanceBonus:         8,
	OverdueAdvanceBonus:       6,
	ResolveBiasMultiplier:     10,
	ProgressingResolveBonus:   5,
	DormancyResolveMultiplier: 2,
	MaxDormancyResolveBonus:   12,
	OverdueResolveBonus:       10,
	MustAdvancePressureFloor:  8,
	CriticalResolvePressure:   40,
}

var HOOK_ACTIVITY_THRESHOLDS = struct {
	RecentlyTouchedDormancy     int
	LongArcQuietHoldMaxAge      int
	LongArcQuietHoldMaxDormancy int
	RefreshDormancy             int
	FreshPromiseAge             int
}{
	RecentlyTouchedDormancy:     1,
	LongArcQuietHoldMaxAge:      2,
	LongArcQuietHoldMaxDormancy: 1,
	RefreshDormancy:             2,
	FreshPromiseAge:             1,
}

var HOOK_AGENDA_LIMITS = map[HookAgendaLoad]struct {
	StaleDebt       int
	MustAdvance     int
	EligibleResolve int
	AvoidFamilies   int
}{
	HookAgendaLoadLight:  {StaleDebt: 1, MustAdvance: 2, EligibleResolve: 1, AvoidFamilies: 2},
	HookAgendaLoadMedium: {StaleDebt: 2, MustAdvance: 2, EligibleResolve: 1, AvoidFamilies: 3},
	HookAgendaLoadHeavy:  {StaleDebt: 3, MustAdvance: 3, EligibleResolve: 2, AvoidFamilies: 4},
}

var HOOK_AGENDA_LOAD_THRESHOLDS = struct {
	HeavyReadyCount         int
	HeavyStaleCount         int
	HeavyCriticalCount      int
	HeavyPressuredCount     int
	MediumReadyCount        int
	MediumStaleCount        int
	MediumCriticalCount     int
	MediumPressuredFamilies int
}{
	HeavyReadyCount:         3,
	HeavyStaleCount:         4,
	HeavyCriticalCount:      3,
	HeavyPressuredCount:     6,
	MediumReadyCount:        2,
	MediumStaleCount:        2,
	MediumCriticalCount:     1,
	MediumPressuredFamilies: 3,
}

var HOOK_VISIBILITY_WINDOWS = map[models.HookPayoffTiming]int{
	models.TimingImmediate: 5,
	models.TimingNearTerm:  5,
	models.TimingMidArc:    6,
	models.TimingSlowBurn:  8,
	models.TimingEndgame:   10,
}

var HOOK_RELEVANT_SELECTION_DEFAULTS = struct {
	Primary struct {
		BaseLimit               int
		PressuredExpansionLimit int
		PressuredThreshold      int
	}
	Stale struct {
		DefaultLimit          int
		ExpandedLimit         int
		OverdueThreshold      int
		FamilySpreadThreshold int
	}
}{}

var HOOK_HEALTH_DEFAULTS = struct {
	MaxActiveHooks        int
	StaleAfterChapters    int
	NoAdvanceWindow       int
	NewHookBurstThreshold int
}{
	MaxActiveHooks:        12,
	StaleAfterChapters:    10,
	NoAdvanceWindow:       5,
	NewHookBurstThreshold: 2,
}

func init() {
	HOOK_RELEVANT_SELECTION_DEFAULTS.Primary.BaseLimit = 3
	HOOK_RELEVANT_SELECTION_DEFAULTS.Primary.PressuredExpansionLimit = 4
	HOOK_RELEVANT_SELECTION_DEFAULTS.Primary.PressuredThreshold = 4
	HOOK_RELEVANT_SELECTION_DEFAULTS.Stale.DefaultLimit = 1
	HOOK_RELEVANT_SELECTION_DEFAULTS.Stale.ExpandedLimit = 2
	HOOK_RELEVANT_SELECTION_DEFAULTS.Stale.OverdueThreshold = 2
	HOOK_RELEVANT_SELECTION_DEFAULTS.Stale.FamilySpreadThreshold = 2
}

// ResolveHookVisibilityWindow 解析per-timing visibility lookback。
func ResolveHookVisibilityWindow(timing models.HookPayoffTiming) int {
	if window, ok := HOOK_VISIBILITY_WINDOWS[timing]; ok {
		return window
	}
	return 6
}
