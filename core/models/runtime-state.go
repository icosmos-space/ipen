package models

// RuntimeStateLanguage 运行时状态语言
type RuntimeStateLanguage string

const (
	LanguageZH RuntimeStateLanguage = "zh"
	LanguageEN RuntimeStateLanguage = "en"
)

// StateManifest 状态清单
// 用于存储状态清单信息，包括模式、语言、最后应用的章节、投影版本、迁移警告等。
type StateManifest struct {
	// 模式版本
	SchemaVersion int `json:"schemaVersion"` // always 2
	// 语言
	Language RuntimeStateLanguage `json:"language"`
	// 最后应用的章节
	LastAppliedChapter int `json:"lastAppliedChapter"`
	// 投影版本
	ProjectionVersion int `json:"projectionVersion"`
	// 迁移警告
	MigrationWarnings []string `json:"migrationWarnings"`
}

// HookStatus 钩子状态
// 表示一个钩子的状态（运行时状态版本）。
type HookStatus string

const (
	HookStatusOpenRT        HookStatus = "open"
	HookStatusProgressingRT HookStatus = "progressing"
	HookStatusDeferred      HookStatus = "已延迟"
	HookStatusResolvedRT    HookStatus = "已解决"
)

// HookPayoffTiming 钩子支付时间
// 表示一个钩子的支付时间（运行时状态版本）。
type HookPayoffTiming string

const (
	TimingImmediate HookPayoffTiming = "immediate"
	TimingNearTerm  HookPayoffTiming = "near-term"
	TimingMidArc    HookPayoffTiming = "mid-arc"
	TimingSlowBurn  HookPayoffTiming = "slow-burn"
	TimingEndgame   HookPayoffTiming = "endgame"
)

// HookRecord 表示a hook record。
type HookRecord struct {
	HookID              string            `json:"hookId"`
	StartChapter        int               `json:"startChapter"`
	Type                string            `json:"type"`
	Status              HookStatus        `json:"status"`
	LastAdvancedChapter int               `json:"lastAdvancedChapter"`
	ExpectedPayoff      string            `json:"expectedPayoff"`
	PayoffTiming        *HookPayoffTiming `json:"payoffTiming,omitempty"`
	Notes               string            `json:"notes"`
}

// HooksState 表示the hooks state。
type HooksState struct {
	Hooks []HookRecord `json:"hooks"`
}

// ChapterSummaryRow 表示a chapter summary row。
type ChapterSummaryRow struct {
	Chapter      int    `json:"chapter"`
	Title        string `json:"title"`
	Characters   string `json:"characters"`
	Events       string `json:"events"`
	StateChanges string `json:"stateChanges"`
	HookActivity string `json:"hookActivity"`
	Mood         string `json:"mood"`
	ChapterType  string `json:"chapterType"`
}

// ChapterSummariesState 表示the chapter summaries state。
type ChapterSummariesState struct {
	Rows []ChapterSummaryRow `json:"rows"`
}

// CurrentStateFact 表示a fact in the current state。
type CurrentStateFact struct {
	Subject           string `json:"subject"`
	Predicate         string `json:"predicate"`
	Object            string `json:"object"`
	ValidFromChapter  int    `json:"validFromChapter"`
	ValidUntilChapter *int   `json:"validUntilChapter"` // nullable
	SourceChapter     int    `json:"sourceChapter"`
}

// CurrentStateState 表示the current state facts。
type CurrentStateState struct {
	Chapter int                `json:"chapter"`
	Facts   []CurrentStateFact `json:"facts"`
}

// CurrentStatePatch 表示a patch to the current state。
type CurrentStatePatch struct {
	CurrentLocation   *string `json:"currentLocation,omitempty"`
	ProtagonistState  *string `json:"protagonistState,omitempty"`
	CurrentGoal       *string `json:"currentGoal,omitempty"`
	CurrentConstraint *string `json:"currentConstraint,omitempty"`
	CurrentAlliances  *string `json:"currentAlliances,omitempty"`
	CurrentConflict   *string `json:"currentConflict,omitempty"`
}

// HookOps 表示hook operations。
type HookOps struct {
	Upsert  []HookRecord `json:"upsert"`
	Mention []string     `json:"mention"`
	Resolve []string     `json:"resolve"`
	Defer   []string     `json:"defer"`
}

// NewHookCandidate 表示a new hook candidate。
type NewHookCandidate struct {
	Type           string            `json:"type"`
	ExpectedPayoff string            `json:"expectedPayoff"`
	PayoffTiming   *HookPayoffTiming `json:"payoffTiming,omitempty"`
	Notes          string            `json:"notes"`
}

// RuntimeStateDelta 表示a delta to apply to the runtime state。
type RuntimeStateDelta struct {
	// 章节
	Chapter int `json:"chapter"`
	// 当前状态补丁
	CurrentStatePatch *CurrentStatePatch `json:"currentStatePatch,omitempty"`
	// 钩子操作
	HookOps HookOps `json:"hookOps"`
	// 新钩子候选
	NewHookCandidates []NewHookCandidate `json:"newHookCandidates"`
	// 章节摘要
	ChapterSummary     *ChapterSummaryRow `json:"chapterSummary,omitempty"`
	SubplotOps         []map[string]any   `json:"subplotOps"`
	EmotionalArcOps    []map[string]any   `json:"emotionalArcOps"`
	CharacterMatrixOps []map[string]any   `json:"characterMatrixOps"`
	Notes              []string           `json:"notes"`
}
