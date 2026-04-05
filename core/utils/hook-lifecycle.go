package utils

import (
	"regexp"
	"strings"

	"github.com/icosmos-space/ipen/core/models"
)

var hookTimingLabels = map[string]map[models.HookPayoffTiming]string{
	"en": {
		models.TimingImmediate: "immediate",
		models.TimingNearTerm:  "near-term",
		models.TimingMidArc:    "mid-arc",
		models.TimingSlowBurn:  "slow-burn",
		models.TimingEndgame:   "endgame",
	},
	"zh": {
		models.TimingImmediate: "立即",
		models.TimingNearTerm:  "近期",
		models.TimingMidArc:    "中程",
		models.TimingSlowBurn:  "慢烧",
		models.TimingEndgame:   "终局",
	},
}

var timingAliases = []struct {
	Timing  models.HookPayoffTiming
	Pattern *regexp.Regexp
}{
	{models.TimingImmediate, regexp.MustCompile(`(?i)^(立即|马上|本章|下[一1]章|immediate|instant|right\s+away|next(?:\s+chapter)?)$`)},
	{models.TimingNearTerm, regexp.MustCompile(`(?i)^(近期|近几章|短线|soon|near\s*-?\s*term|short\s*run)$`)},
	{models.TimingMidArc, regexp.MustCompile(`(?i)^(中程|中期|卷中|mid\s*-?\s*arc|mid\s*-?\s*book|middle)$`)},
	{models.TimingSlowBurn, regexp.MustCompile(`(?i)^(慢烧|长线|后续|later|slow\s*-?\s*burn|long\s*-?\s*arc)$`)},
	{models.TimingEndgame, regexp.MustCompile(`(?i)^(终局|终章|大结局|climax|finale|endgame)$`)},
}

var signalPatterns = []struct {
	Timing  models.HookPayoffTiming
	Pattern *regexp.Regexp
}{
	{models.TimingEndgame, regexp.MustCompile(`(?i)(终局|终章|大结局|climax|finale|endgame|last act|final reveal)`)},
	{models.TimingImmediate, regexp.MustCompile(`(?i)(当章|本章|下一章|马上|立刻|immediate|next chapter|right away|at once)`)},
	{models.TimingNearTerm, regexp.MustCompile(`(?i)(近期|近几章|短线|soon|near\s*-?\s*term|short run|current sequence)`)},
	{models.TimingMidArc, regexp.MustCompile(`(?i)(中期|卷中|mid\s*-?\s*book|mid\s*-?\s*arc|middle of the arc)`)},
	{models.TimingSlowBurn, regexp.MustCompile(`(?i)(长线|慢烧|后续发酵|later|slow burn|long arc|long tail)`)},
}

// HookLifecycleDescription 描述lifecycle pressure for one hook at one chapter。
type HookLifecycleDescription struct {
	Timing          models.HookPayoffTiming
	Phase           HookPhase
	Age             int
	Dormancy        int
	ReadyToResolve  bool
	Stale           bool
	Overdue         bool
	AdvancePressure int
	ResolvePressure int
}

// NormalizeHookPayoffTiming 规范化aliases into canonical payoff timing。
func NormalizeHookPayoffTiming(value string) (models.HookPayoffTiming, bool) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", false
	}

	for _, candidate := range timingAliases {
		if candidate.Pattern.MatchString(normalized) {
			return candidate.Timing, true
		}
	}
	return "", false
}

// InferHookPayoffTiming 推断timing from expected payoff/notes。
func InferHookPayoffTiming(expectedPayoff string, notes string) models.HookPayoffTiming {
	combined := strings.TrimSpace(strings.TrimSpace(expectedPayoff) + " " + strings.TrimSpace(notes))
	if combined == "" {
		return models.TimingMidArc
	}
	for _, candidate := range signalPatterns {
		if candidate.Pattern.MatchString(combined) {
			return candidate.Timing
		}
	}
	return models.TimingMidArc
}

// ResolveHookPayoffTiming 解析normalized timing or inference fallback。
func ResolveHookPayoffTiming(payoffTiming string, expectedPayoff string, notes string) models.HookPayoffTiming {
	if timing, ok := NormalizeHookPayoffTiming(payoffTiming); ok {
		return timing
	}
	return InferHookPayoffTiming(expectedPayoff, notes)
}

// LocalizeHookPayoffTiming 本地化payoff timing labels。
func LocalizeHookPayoffTiming(timing models.HookPayoffTiming, language string) string {
	lang := "zh"
	if strings.EqualFold(language, "en") {
		lang = "en"
	}
	if label, ok := hookTimingLabels[lang][timing]; ok {
		return label
	}
	return string(timing)
}

// DescribeHookLifecycle 计算timing/phase and pressure metrics。
func DescribeHookLifecycle(
	payoffTiming string,
	expectedPayoff string,
	notes string,
	startChapter int,
	lastAdvancedChapter int,
	status string,
	chapterNumber int,
	targetChapters int,
) HookLifecycleDescription {
	timing := ResolveHookPayoffTiming(payoffTiming, expectedPayoff, notes)
	profile, ok := HOOK_TIMING_PROFILES[timing]
	if !ok {
		profile = HOOK_TIMING_PROFILES[models.TimingMidArc]
	}

	phase := resolveHookPhase(chapterNumber, targetChapters)
	age := maxInt(0, chapterNumber-maxInt(1, startChapter))
	lastTouchChapter := maxInt(startChapter, lastAdvancedChapter)
	dormancy := maxInt(0, chapterNumber-maxInt(1, lastTouchChapter))
	explicitProgressing := regexp.MustCompile(`(?i)^(progressing|advanced|重大推进|持续推进)$`).MatchString(strings.TrimSpace(status))
	phaseReady := HOOK_PHASE_WEIGHT[phase] >= HOOK_PHASE_WEIGHT[profile.MinimumPhase]
	recentlyTouched := dormancy <= HOOK_ACTIVITY_THRESHOLDS.RecentlyTouchedDormancy
	overdue := phaseReady && age >= profile.OverdueAge

	cadenceReady := true
	if timing == models.TimingSlowBurn {
		cadenceReady = phase == HookPhaseLate || overdue
	} else if timing == models.TimingEndgame {
		cadenceReady = phase == HookPhaseLate
	}

	momentum := explicitProgressing || recentlyTouched
	stale := phaseReady && (dormancy >= profile.StaleDormancy || (overdue && !momentum))
	readyToResolve := phaseReady && cadenceReady && age >= profile.EarliestResolveAge && (momentum || (overdue && explicitProgressing))

	resolvePressure := 0
	if readyToResolve {
		resolvePressure = profile.ResolveBias*HOOK_PRESSURE_WEIGHTS.ResolveBiasMultiplier +
			ternaryInt(explicitProgressing, HOOK_PRESSURE_WEIGHTS.ProgressingResolveBonus, 0) +
			minInt(HOOK_PRESSURE_WEIGHTS.MaxDormancyResolveBonus, dormancy*HOOK_PRESSURE_WEIGHTS.DormancyResolveMultiplier) +
			ternaryInt(overdue, HOOK_PRESSURE_WEIGHTS.OverdueResolveBonus, 0)
	}

	return HookLifecycleDescription{
		Timing:         timing,
		Phase:          phase,
		Age:            age,
		Dormancy:       dormancy,
		ReadyToResolve: readyToResolve,
		Stale:          stale,
		Overdue:        overdue,
		AdvancePressure: age + dormancy +
			ternaryInt(stale, HOOK_PRESSURE_WEIGHTS.StaleAdvanceBonus, 0) +
			ternaryInt(overdue, HOOK_PRESSURE_WEIGHTS.OverdueAdvanceBonus, 0),
		ResolvePressure: resolvePressure,
	}
}

func resolveHookPhase(chapterNumber int, targetChapters int) HookPhase {
	if targetChapters > 0 {
		progress := float64(chapterNumber) / float64(targetChapters)
		if progress >= HOOK_PHASE_THRESHOLDS.LateProgress {
			return HookPhaseLate
		}
		if progress >= HOOK_PHASE_THRESHOLDS.MiddleProgress {
			return HookPhaseMiddle
		}
		return HookPhaseOpening
	}

	if chapterNumber >= HOOK_PHASE_THRESHOLDS.LateChapter {
		return HookPhaseLate
	}
	if chapterNumber >= HOOK_PHASE_THRESHOLDS.MiddleChapter {
		return HookPhaseMiddle
	}
	return HookPhaseOpening
}

func ternaryInt(cond bool, a int, b int) int {
	if cond {
		return a
	}
	return b
}
