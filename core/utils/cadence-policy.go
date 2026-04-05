package utils

// CadenceWindowDefaults 存储default lookback windows for cadence analysis。
type CadenceWindowDefaults struct {
	SummaryLookback             int
	EnglishVarianceLookback     int
	RecentBoundaryPatternBodies int
}

// CadencePressureThresholds 存储scene/mood/title cadence thresholds。
type CadencePressureThresholds struct {
	Scene CadencePressureRule
	Mood  CadencePressureRule
	Title CadencePressureRule
}

// CadencePressureRule controls pressure level transitions.
type CadencePressureRule struct {
	HighCount         int
	MediumCount       int
	MediumWindowFloor int
}

// LongSpanFatigueThresholds 存储repeated-pattern detection thresholds。
type LongSpanFatigueThresholds struct {
	BoundarySimilarityFloor   float64
	BoundarySentenceMinLength int
	BoundaryPatternMinBodies  int
}

var CADENCE_WINDOW_DEFAULTS = CadenceWindowDefaults{
	SummaryLookback:             4,
	EnglishVarianceLookback:     24,
	RecentBoundaryPatternBodies: 2,
}

var CADENCE_PRESSURE_THRESHOLDS = CadencePressureThresholds{
	Scene: CadencePressureRule{HighCount: 3, MediumCount: 2, MediumWindowFloor: 4},
	Mood:  CadencePressureRule{HighCount: 3, MediumCount: 2, MediumWindowFloor: 4},
	Title: CadencePressureRule{HighCount: 3, MediumCount: 2, MediumWindowFloor: 4},
}

var LONG_SPAN_FATIGUE_THRESHOLDS = LongSpanFatigueThresholds{
	BoundarySimilarityFloor:   0.72,
	BoundarySentenceMinLength: 18,
	BoundaryPatternMinBodies:  3,
}

// ResolveCadencePressure 返回"high", "medium", or empty string。
func ResolveCadencePressure(count int, total int, highThreshold int, mediumThreshold int, mediumWindowFloor int) string {
	if count >= highThreshold {
		return "high"
	}
	if count >= mediumThreshold && total >= mediumWindowFloor {
		return "medium"
	}
	return ""
}
