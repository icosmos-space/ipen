package models

// CurrentState 表示the canonical current state。
type CurrentState struct {
	Chapter         int              `json:"chapter"`
	Location        string           `json:"location"`
	Protagonist     ProtagonistState `json:"protagonist"`
	Enemies         []EnemyInfo      `json:"enemies"`
	KnownTruths     []string         `json:"knownTruths"`
	CurrentConflict string           `json:"currentConflict"`
	Anchor          string           `json:"anchor"`
}

// ProtagonistState 表示the protagonist's current state。
type ProtagonistState struct {
	Status      string `json:"status"`
	CurrentGoal string `json:"currentGoal"`
	Constraints string `json:"constraints"`
}

// EnemyInfo 表示enemy information。
type EnemyInfo struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	Threat       string `json:"threat"`
}

// LedgerEntry 表示an entry in the particle ledger。
type LedgerEntry struct {
	Chapter              int     `json:"chapter"`
	OpeningValue         float64 `json:"openingValue"`
	Source               string  `json:"source"`
	ResourceCompleteness string  `json:"resourceCompleteness"`
	Delta                float64 `json:"delta"`
	ClosingValue         float64 `json:"closingValue"`
	Basis                string  `json:"basis"`
}

// ParticleLedger 表示the particle ledger。
type ParticleLedger struct {
	HardCap      float64       `json:"hardCap"`
	CurrentTotal float64       `json:"currentTotal"`
	Entries      []LedgerEntry `json:"entries"`
}

// HookStatus 表示the status of a hook。
type BookHookStatus string

const (
	HookStatusOpen        BookHookStatus = "open"
	HookStatusProgressing BookHookStatus = "progressing"
	HookStatusResolved    BookHookStatus = "resolved"
)

// PendingHook 表示a pending hook。
type PendingHook struct {
	ID                 string         `json:"id"`
	OriginChapter      int            `json:"originChapter"`
	Type               string         `json:"type"`
	Status             BookHookStatus `json:"status"`
	LastProgress       string         `json:"lastProgress"`
	ExpectedResolution string         `json:"expectedResolution"`
	Note               string         `json:"note"`
}

// PendingHooks 表示a collection of pending hooks。
type PendingHooks struct {
	Hooks []PendingHook `json:"hooks"`
}
