/**
 * Canonical state files — 每本书三大真相来源
 * 会作为markdown文件进行持久化，在这里进行解析和验证
 */
package models

// CurrentState 当前状态
// 用于存储当前章节的状态信息，包括位置、角色、敌人、已知真相、当前冲突、锚点等。
type CurrentState struct {
	// 章节
	Chapter int `json:"chapter"`
	// 位置
	Location string `json:"location"`
	// 角色
	Protagonist ProtagonistState `json:"protagonist"`
	// 敌人
	Enemies []EnemyInfo `json:"enemies"`
	// 已知真相
	KnownTruths []string `json:"knownTruths"`
	// 当前冲突
	CurrentConflict string `json:"currentConflict"`
	// 锚点
	Anchor string `json:"anchor"`
}

// ProtagonistState 表示角色的当前状态
type ProtagonistState struct {
	// 角色状态
	Status string `json:"status"`
	// 当前目标
	CurrentGoal string `json:"currentGoal"`
	// 限制条件
	// 约束条件
	Constraints string `json:"constraints"`
}

// EnemyInfo 表示敌人的信息
type EnemyInfo struct {
	// 敌人名称
	Name string `json:"name"`
	// 与角色的关系
	Relationship string `json:"relationship"`
	// 威胁等级
	Threat string `json:"threat"`
}

// LedgerEntry 表示粒子账本中的一个条目
type LedgerEntry struct {
	// 章节
	Chapter int `json:"chapter"`
	// 开始值
	OpeningValue float64 `json:"openingValue"`
	// 来源
	Source string `json:"source"`
	// 资源完整性
	ResourceCompleteness string `json:"resourceCompleteness"`
	// 变化量
	Delta float64 `json:"delta"`
	// 结束值
	ClosingValue float64 `json:"closingValue"`
	// 基础值
	Basis string `json:"basis"`
}

// ParticleLedger 表示粒子账本
type ParticleLedger struct {
	// 硬上限
	HardCap float64 `json:"hardCap"`
	// 当前总值
	CurrentTotal float64 `json:"currentTotal"`
	// 条目
	Entries []LedgerEntry `json:"entries"`
}

// HookStatus 表示钩子的状态
type BookHookStatus string

const (
	// 未解决
	HookStatusOpen BookHookStatus = "未解决"
	// 进行中
	HookStatusProgressing BookHookStatus = "进行中"
	// 已解决
	HookStatusResolved BookHookStatus = "已解决"
)

// PendingHook 表示处于pending状态的钩子
type PendingHook struct {
	// 钩子ID
	ID string `json:"id"`
	// 原始章节
	OriginChapter int `json:"originChapter"`
	// 钩子类型
	Type string `json:"type"`
	// 钩子状态
	Status BookHookStatus `json:"status" validate:"required,oneof: 未解决 进行中 已解决"`
	// 最新进度
	LastProgress string `json:"lastProgress"`
	// 预期解决进度
	ExpectedResolution string `json:"expectedResolution"`
	// 备注
	Note string `json:"note"`
}

// PendingHooks 表示处于pending状态的钩子集合
type PendingHooks struct {
	Hooks []PendingHook `json:"hooks"`
}
