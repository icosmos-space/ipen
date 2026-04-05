package models

// ChapterConflict 表示a chapter conflict。
type ChapterConflict struct {
	Type       string  `json:"type"`
	Resolution string  `json:"resolution"`
	Detail     *string `json:"detail,omitempty"`
}

// HookPressurePhase 表示the phase of hook pressure。
type HookPressurePhase string

const (
	PhaseOpening HookPressurePhase = "opening"
	PhaseMiddle  HookPressurePhase = "middle"
	PhaseLate    HookPressurePhase = "late"
)

// HookMovement 表示the movement of a hook。
type HookMovement string

const (
	MovementQuietHold     HookMovement = "quiet-hold"
	MovementRefresh       HookMovement = "refresh"
	MovementAdvance       HookMovement = "advance"
	MovementPartialPayoff HookMovement = "partial-payoff"
	MovementFullPayoff    HookMovement = "full-payoff"
)

// HookPressureLevel 表示the pressure level of a hook。
type HookPressureLevel string

const (
	PressureLow      HookPressureLevel = "low"
	PressureMedium   HookPressureLevel = "medium"
	PressureHigh     HookPressureLevel = "high"
	PressureCritical HookPressureLevel = "critical"
)

// HookPressureReason 表示the reason for hook pressure。
type HookPressureReason string

const (
	ReasonFreshPromise  HookPressureReason = "fresh-promise"
	ReasonBuildingDebt  HookPressureReason = "building-debt"
	ReasonStalePromise  HookPressureReason = "stale-promise"
	ReasonRipePayoff    HookPressureReason = "ripe-payoff"
	ReasonOverduePayoff HookPressureReason = "overdue-payoff"
	ReasonLongArcHold   HookPressureReason = "long-arc-hold"
)

// HookPressure 表示the pressure on a hook。
type HookPressure struct {
	HookID            string             `json:"hookId"`
	Type              string             `json:"type"`
	Movement          HookMovement       `json:"movement"`
	Pressure          HookPressureLevel  `json:"pressure"`
	PayoffTiming      *HookPayoffTiming  `json:"payoffTiming,omitempty"`
	Phase             HookPressurePhase  `json:"phase"`
	Reason            HookPressureReason `json:"reason"`
	BlockSiblingHooks bool               `json:"blockSiblingHooks"`
}

// HookAgenda 表示the hook agenda for a chapter。
type HookAgenda struct {
	PressureMap          []HookPressure `json:"pressureMap"`
	MustAdvance          []string       `json:"mustAdvance"`
	EligibleResolve      []string       `json:"eligibleResolve"`
	StaleDebt            []string       `json:"staleDebt"`
	AvoidNewHookFamilies []string       `json:"avoidNewHookFamilies"`
}

// ChapterIntent 表示the intent for a chapter。
type ChapterIntent struct {
	Chapter        int               `json:"chapter"`
	Goal           string            `json:"goal"`
	OutlineNode    *string           `json:"outlineNode,omitempty"`
	SceneDirective *string           `json:"sceneDirective,omitempty"`
	ArcDirective   *string           `json:"arcDirective,omitempty"`
	MoodDirective  *string           `json:"moodDirective,omitempty"`
	TitleDirective *string           `json:"titleDirective,omitempty"`
	MustKeep       []string          `json:"mustKeep"`
	MustAvoid      []string          `json:"mustAvoid"`
	StyleEmphasis  []string          `json:"styleEmphasis"`
	Conflicts      []ChapterConflict `json:"conflicts"`
	HookAgenda     HookAgenda        `json:"hookAgenda"`
}

// ContextSource 表示a context source。
type ContextSource struct {
	Source  string  `json:"source"`
	Reason  string  `json:"reason"`
	Excerpt *string `json:"excerpt,omitempty"`
}

// ContextPackage 表示a context package。
type ContextPackage struct {
	Chapter         int             `json:"chapter"`
	SelectedContext []ContextSource `json:"selectedContext"`
}

// RuleLayerScope 表示the scope of a rule layer。
type RuleLayerScope string

const (
	ScopeGlobal RuleLayerScope = "global"
	ScopeBook   RuleLayerScope = "book"
	ScopeArc    RuleLayerScope = "arc"
	ScopeLocal  RuleLayerScope = "local"
)

// RuleLayer 表示a rule layer。
type RuleLayer struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Precedence int            `json:"precedence"`
	Scope      RuleLayerScope `json:"scope"`
}

// OverrideEdge 表示an override edge。
type OverrideEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Allowed bool   `json:"allowed"`
	Scope   string `json:"scope"`
}

// ActiveOverride 表示an active override。
type ActiveOverride struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Target string `json:"target"`
	Reason string `json:"reason"`
}

// RuleStackSections 表示sections of a rule stack。
type RuleStackSections struct {
	Hard       []string `json:"hard"`
	Soft       []string `json:"soft"`
	Diagnostic []string `json:"diagnostic"`
}

// RuleStack 表示a rule stack。
type RuleStack struct {
	Layers          []RuleLayer       `json:"layers"`
	Sections        RuleStackSections `json:"sections"`
	OverrideEdges   []OverrideEdge    `json:"overrideEdges"`
	ActiveOverrides []ActiveOverride  `json:"activeOverrides"`
}

// ChapterTrace 表示the trace for a chapter。
type ChapterTrace struct {
	Chapter         int      `json:"chapter"`
	PlannerInputs   []string `json:"plannerInputs"`
	ComposerInputs  []string `json:"composerInputs"`
	SelectedSources []string `json:"selectedSources"`
	Notes           []string `json:"notes"`
}
