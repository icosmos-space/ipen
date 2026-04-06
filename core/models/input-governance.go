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
	PayoffTiming      *HookPayoffTiming  `json:"payoffTiming" validate:"omitempty"`
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
	Chapter        int               `json:"chapter" validate:"required,min=1"`
	Goal           string            `json:"goal" validate:"required,min=1"`
	OutlineNode    *string           `json:"outlineNode" validate:"omitempty"`
	SceneDirective *string           `json:"sceneDirective" validate:"omitempty"`
	ArcDirective   *string           `json:"arcDirective" validate:"omitempty"`
	MoodDirective  *string           `json:"moodDirective" validate:"omitempty"`
	TitleDirective *string           `json:"titleDirective" validate:"omitempty"`
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
	Excerpt *string `json:"excerpt" validate:"omitempty"`
}

// ContextPackage 表示a context package。
type ContextPackage struct {
	Chapter         int             `json:"chapter" validate:"required,min=1"`
	SelectedContext []ContextSource `json:"selectedContext"`
}

// RuleLayerScope 规则层范围
type RuleLayerScope string

const (
	ScopeGlobal RuleLayerScope = "全局"
	ScopeBook   RuleLayerScope = "书籍"
	ScopeArc    RuleLayerScope = "弧"
	ScopeLocal  RuleLayerScope = "本地"
)

// RuleLayer 规则层。
type RuleLayer struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Precedence int            `json:"precedence"`
	Scope      RuleLayerScope `json:"scope"`
}

// OverrideEdge 覆写边缘。
type OverrideEdge struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Allowed bool   `json:"allowed"`
	Scope   string `json:"scope"`
}

// ActiveOverride 活动覆写
type ActiveOverride struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Target string `json:"target"`
	Reason string `json:"reason"`
}

// RuleStackSections 规则栈部分
type RuleStackSections struct {
	Hard       []string `json:"hard"`
	Soft       []string `json:"soft"`
	Diagnostic []string `json:"diagnostic"`
}

// RuleStack 规则栈
type RuleStack struct {
	Layers          []RuleLayer       `json:"layers"`
	Sections        RuleStackSections `json:"sections"`
	OverrideEdges   []OverrideEdge    `json:"overrideEdges"`
	ActiveOverrides []ActiveOverride  `json:"activeOverrides"`
}

// ChapterTrace 章节追踪
type ChapterTrace struct {
	Chapter         int      `json:"chapter" validate:"required,min=1"`
	PlannerInputs   []string `json:"plannerInputs"`
	ComposerInputs  []string `json:"composerInputs"`
	SelectedSources []string `json:"selectedSources"`
	Notes           []string `json:"notes"`
}
