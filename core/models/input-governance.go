package models

// ChapterConflict 章节冲突
type ChapterConflict struct {
	// 冲突类型
	Type string `json:"type"`
	// 解决方案
	Resolution string `json:"resolution"`
	// 详细信息
	Detail *string `json:"detail,omitempty"`
}

// HookPressurePhase 钩子压力阶段
type HookPressurePhase string

const (
	PhaseOpening HookPressurePhase = "开启阶段"
	PhaseMiddle  HookPressurePhase = "中间阶段"
	PhaseLate    HookPressurePhase = "晚阶段"
)

// HookMovement 钩子移动
type HookMovement string

const (
	MovementQuietHold     HookMovement = "安静保持"
	MovementRefresh       HookMovement = "刷新"
	MovementAdvance       HookMovement = "推进"
	MovementPartialPayoff HookMovement = "部分结算"
	MovementFullPayoff    HookMovement = "完整结算"
)

// HookPressureLevel 钩子压力等级
type HookPressureLevel string

const (
	PressureLow      HookPressureLevel = "低"
	PressureMedium   HookPressureLevel = "中"
	PressureHigh     HookPressureLevel = "高"
	PressureCritical HookPressureLevel = "严重"
)

// HookPressureReason 表示the reason for hook pressure。
type HookPressureReason string

const (
	ReasonFreshPromise  HookPressureReason = "新鲜承诺"
	ReasonBuildingDebt  HookPressureReason = "债务建设"
	ReasonStalePromise  HookPressureReason = "陈旧承诺"
	ReasonRipePayoff    HookPressureReason = "成熟结算"
	ReasonOverduePayoff HookPressureReason = "逾期结算"
	ReasonLongArcHold   HookPressureReason = "长期弧保持"
)

// HookPressure 钩子压力
type HookPressure struct {
	// 钩子ID
	HookID string `json:"hookId"`
	// 钩子类型
	Type              string             `json:"type"`
	Movement          HookMovement       `json:"movement"`
	Pressure          HookPressureLevel  `json:"pressure"`
	PayoffTiming      *HookPayoffTiming  `json:"payoffTiming,omitempty"`
	Phase             HookPressurePhase  `json:"phase"`
	Reason            HookPressureReason `json:"reason"`
	BlockSiblingHooks bool               `json:"blockSiblingHooks"`
}

// HookAgenda 钩子议程
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
