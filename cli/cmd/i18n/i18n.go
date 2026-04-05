package i18n

// Language represents the CLI display language
type Language string

const (
	// LangZh represents Chinese
	LangZh Language = "zh"
	// LangEn represents English
	LangEn Language = "en"
)

// CurrentLanguage holds the currently selected language
var CurrentLanguage Language = LangZh

// I18n contains translations for CLI strings
type I18n struct {
	// Root command
	RootShort map[Language]string
	RootLong  map[Language]string
	RootHelp  map[Language]string
	RootVer   map[Language]string

	// Flags
	FlagConfig   map[Language]string
	FlagLogLevel map[Language]string
	FlagVerbose  map[Language]string
	FlagLang     map[Language]string

	// Built-in commands
	CmdHelpShort       map[Language]string
	CmdHelpLong        map[Language]string
	CmdCompletionShort map[Language]string
	CmdCompletionLong  map[Language]string

	// Main commands
	CmdAgentShort                   map[Language]string
	CmdAgentLong                    map[Language]string
	CmdAgentArgInst                 map[Language]string
	CmdAnalyticsShort               map[Language]string
	CmdAuditShort                   map[Language]string
	CmdAuditLong                    map[Language]string
	CmdAuditArgBook                 map[Language]string
	CmdAuditArgChapter              map[Language]string
	CmdBookShort                    map[Language]string
	CmdBookLong                     map[Language]string
	CmdBookCreateShort              map[Language]string
	CmdBookCreateLong               map[Language]string
	CmdBookCreateArgTitle           map[Language]string
	CmdBookCreateArgGenre           map[Language]string
	CmdBookCreateArgPlatform        map[Language]string
	CmdBookCreateFlagTitle          map[Language]string
	CmdBookCreateFlagGenre          map[Language]string
	CmdBookCreateFlagPlatform       map[Language]string
	CmdBookCreateFlagTargetChapters map[Language]string
	CmdBookCreateFlagWords          map[Language]string
	CmdBookCreateFlagBrief          map[Language]string
	CmdBookCreateFlagLang           map[Language]string
	CmdBookUpdateShort              map[Language]string
	CmdBookUpdateLong               map[Language]string
	CmdBookListShort                map[Language]string
	CmdBookDeleteShort              map[Language]string
	CmdBookDeleteLong               map[Language]string
	CmdBookDeleteArgBook            map[Language]string
	CmdComposeShort                 map[Language]string
	CmdComposeLong                  map[Language]string
	CmdConfigShort                  map[Language]string
	CmdConfigLong                   map[Language]string
	CmdConfigSetShort               map[Language]string
	CmdConfigSetLong                map[Language]string
	CmdConfigGetShort               map[Language]string
	CmdConsolidateShort             map[Language]string
	CmdConsolidateLong              map[Language]string
	CmdDaemonShort                  map[Language]string
	CmdDaemonLong                   map[Language]string
	CmdDaemonUpShort                map[Language]string
	CmdDaemonDownShort              map[Language]string
	CmdDetectShort                  map[Language]string
	CmdDetectLong                   map[Language]string
	CmdDoctorShort                  map[Language]string
	CmdDoctorLong                   map[Language]string
	CmdDraftShort                   map[Language]string
	CmdDraftLong                    map[Language]string
	CmdDraftCreateShort             map[Language]string
	CmdDraftListShort               map[Language]string
	CmdEvalShort                    map[Language]string
	CmdEvalLong                     map[Language]string
	CmdExportShort                  map[Language]string
	CmdExportLong                   map[Language]string
	CmdFanficShort                  map[Language]string
	CmdFanficLong                   map[Language]string
	CmdGenreShort                   map[Language]string
	CmdGenreLong                    map[Language]string
	CmdGenreListShort               map[Language]string
	CmdGenreShowShort               map[Language]string
	CmdImportShort                  map[Language]string
	CmdImportLong                   map[Language]string
	CmdImportChaptersShort          map[Language]string
	CmdImportChaptersLong           map[Language]string
	CmdImportCanonShort             map[Language]string
	CmdImportCanonLong              map[Language]string
	CmdInitShort                    map[Language]string
	CmdInitLong                     map[Language]string
	CmdPlanShort                    map[Language]string
	CmdPlanLong                     map[Language]string
	CmdPlanChapterShort             map[Language]string
	CmdPlanChapterLong              map[Language]string
	CmdRadarShort                   map[Language]string
	CmdRadarLong                    map[Language]string
	CmdReviewShort                  map[Language]string
	CmdReviewLong                   map[Language]string
	CmdReviewListShort              map[Language]string
	CmdReviewApproveShort           map[Language]string
	CmdReviseShort                  map[Language]string
	CmdReviseLong                   map[Language]string
	CmdStatusShort                  map[Language]string
	CmdStatusLong                   map[Language]string
	CmdStudioShort                  map[Language]string
	CmdStudioLong                   map[Language]string
	CmdStyleShort                   map[Language]string
	CmdStyleLong                    map[Language]string
	CmdUpdateShort                  map[Language]string
	CmdUpdateLong                   map[Language]string
	CmdWriteShort                   map[Language]string
	CmdWriteLong                    map[Language]string
	CmdWriteNextShort               map[Language]string
	CmdWriteNextLong                map[Language]string
	CmdWriteRewriteShort            map[Language]string
	CmdWriteRewriteLong             map[Language]string
	CmdWriteRepairShort             map[Language]string
	CmdWriteRepairLong              map[Language]string

	// Usage strings
	UsageHelp map[Language]string

	// Examples (kept in i18n for consistency)
	ExConfigSet      map[Language]string
	ExConfigGet      map[Language]string
	ExConfigSetModel map[Language]string
	ExConfigRmModel  map[Language]string
}

// Translations holds all translation data
var Translations = I18n{
	RootShort: map[Language]string{
		LangZh: "iPen - 多智能体小说创作系统",
		LangEn: "iPen - Multi-agent novel production system",
	},
	RootLong: map[Language]string{
		LangZh: "iPen 是一个多智能体小说创作系统，使用 AI 帮助作者高效地创作、管理和发布小说。",
		LangEn: "iPen is a multi-agent novel production system that uses AI to help writers create, manage, and publish novels efficiently.",
	},
	RootHelp: map[Language]string{
		LangZh: "获取关于 ipen 的帮助信息。",
		LangEn: "Get help information for ipen",
	},
	RootVer: map[Language]string{
		LangZh: "显示 ipen 版本号。",
		LangEn: "Show ipen version number",
	},

	FlagConfig: map[Language]string{
		LangZh: "配置文件路径 (默认 $HOME/.ipen.yaml)",
		LangEn: "Config file path (default $HOME/.ipen.yaml)",
	},
	FlagLogLevel: map[Language]string{
		LangZh: "日志级别 (debug, info, warn, error)",
		LangEn: "Log level (debug, info, warn, error)",
	},
	FlagVerbose: map[Language]string{
		LangZh: "详细输出",
		LangEn: "Verbose output",
	},
	FlagLang: map[Language]string{
		LangZh: "界面显示语言: zh (中文) 或 en (英文)",
		LangEn: "UI display language: zh (Chinese) or en (English)",
	},

	CmdHelpShort: map[Language]string{
		LangZh: "获取关于任何命令的帮助信息。",
		LangEn: "Get help about any command",
	},
	CmdHelpLong: map[Language]string{
		LangZh: "获取关于任何命令的详细帮助信息和使用示例。",
		LangEn: "Get detailed help information and usage examples for any command.",
	},
	CmdCompletionShort: map[Language]string{
		LangZh: "为指定的 Shell 生成自动补全脚本",
		LangEn: "Generate autocompletion script for the specified shell",
	},
	CmdCompletionLong: map[Language]string{
		LangZh: `为指定的 Shell (bash, zsh, fish, powershell) 生成自动补全脚本。
将补全脚本加入 Shell 配置后，可使用 Tab 自动补全 ipen 命令。`,
		LangEn: `Generate autocompletion script for the specified shell (bash, zsh, fish, powershell).

After adding the completion script to your shell configuration, you can use Tab key to autocomplete ipen commands.`,
	},

	// Agent
	CmdAgentShort: map[Language]string{
		LangZh: "自然语言智能体模式",
		LangEn: "Natural language agent mode",
	},
	CmdAgentLong: map[Language]string{
		LangZh: "自然语言智能体模式（LLM 通过工具调用编排任务）",
		LangEn: "Natural language agent mode (LLM orchestrates via tool-use)",
	},
	CmdAgentArgInst: map[Language]string{
		LangZh: "给智能体的自然语言指令",
		LangEn: "Natural language instruction for the agent",
	},

	// Analytics
	CmdAnalyticsShort: map[Language]string{
		LangZh: "查看数据分析",
		LangEn: "View analytics",
	},

	// Audit
	CmdAuditShort: map[Language]string{
		LangZh: "审计章节",
		LangEn: "Audit chapters",
	},
	CmdAuditLong: map[Language]string{
		LangZh: "审计章节的连续性问题。如果不指定章节号，则审计最新一章。",
		LangEn: "Audit a chapter for continuity issues. Defaults to latest chapter if omitted.",
	},
	CmdAuditArgBook: map[Language]string{
		LangZh: "书籍的唯一标识符。如果省略且只有一本书，则自动检测。",
		LangEn: "Book ID. Auto-detected if only one book exists.",
	},
	CmdAuditArgChapter: map[Language]string{
		LangZh: "要审计的章节号（可选，默认为最新一章）",
		LangEn: "Chapter number (optional, defaults to latest)",
	},

	// Book
	CmdBookShort: map[Language]string{
		LangZh: "管理书籍",
		LangEn: "Manage books",
	},
	CmdBookLong: map[Language]string{
		LangZh: "在 iPen 项目中创建、更新、列表和删除书籍。",
		LangEn: "Create, update, list, and delete books in your iPen project.",
	},
	CmdBookCreateShort: map[Language]string{
		LangZh: "创建新书，AI 自动生成基础设定",
		LangEn: "Create a new book with AI-generated foundation",
	},
	CmdBookCreateLong: map[Language]string{
		LangZh: "创建新书，AI 自动生成基础设定（故事圣经、大纲、书籍规则）。",
		LangEn: "Create a new book with AI-generated foundation (story bible, outline, book rules).",
	},
	CmdBookCreateArgTitle: map[Language]string{
		LangZh: "小说标题（--title 或位置参数 <title>）",
		LangEn: "Book title (--title or positional <title>)",
	},
	CmdBookCreateArgGenre: map[Language]string{
		LangZh: "小说类型",
		LangEn: "Genre",
	},
	CmdBookCreateArgPlatform: map[Language]string{
		LangZh: "目标平台",
		LangEn: "Target platform",
	},
	CmdBookCreateFlagTitle: map[Language]string{
		LangZh: "小说标题（--title 或位置参数 <title>）",
		LangEn: "Book title (--title or positional <title>)",
	},
	CmdBookCreateFlagGenre: map[Language]string{
		LangZh: "小说类型",
		LangEn: "Genre",
	},
	CmdBookCreateFlagPlatform: map[Language]string{
		LangZh: "目标平台",
		LangEn: "Target platform",
	},
	CmdBookCreateFlagTargetChapters: map[Language]string{
		LangZh: "目标章节数",
		LangEn: "Target chapter count",
	},
	CmdBookCreateFlagWords: map[Language]string{
		LangZh: "每章字数",
		LangEn: "Words per chapter",
	},
	CmdBookCreateFlagBrief: map[Language]string{
		LangZh: "创意大纲文件路径 (.md/.txt)",
		LangEn: "Path to creative brief file (.md/.txt)",
	},
	CmdBookCreateFlagLang: map[Language]string{
		LangZh: "写作语言: zh (中文) 或 en (英文)",
		LangEn: "Writing language: zh (Chinese) or en (English)",
	},
	CmdBookUpdateShort: map[Language]string{
		LangZh: "更新书籍设置",
		LangEn: "Update book settings",
	},
	CmdBookUpdateLong: map[Language]string{
		LangZh: "更新书籍设置。如果省略书籍ID且只有一本书，则自动检测。",
		LangEn: "Update book settings. Auto-detects book ID if only one book exists.",
	},
	CmdBookListShort: map[Language]string{
		LangZh: "列出所有书籍",
		LangEn: "List all books",
	},
	CmdBookDeleteShort: map[Language]string{
		LangZh: "删除书籍及其所有章节",
		LangEn: "Delete a book and all its chapters",
	},
	CmdBookDeleteLong: map[Language]string{
		LangZh: "删除指定书籍及其所有章节、真实文件和快照。此操作不可逆！",
		LangEn: "Delete a book and all its chapters, truth files, and snapshots. This action is irreversible!",
	},
	CmdBookDeleteArgBook: map[Language]string{
		LangZh: "要删除的书籍的唯一标识符（必填）。",
		LangEn: "Book ID to delete (required).",
	},

	// Compose
	CmdComposeShort: map[Language]string{
		LangZh: "创作内容",
		LangEn: "Compose content",
	},
	CmdComposeLong: map[Language]string{
		LangZh: "基于规划结果创作章节内容",
		LangEn: "Compose chapter content based on planning results",
	},

	// Config
	CmdConfigShort: map[Language]string{
		LangZh: "管理 iPen 配置",
		LangEn: "Manage iPen configuration",
	},
	CmdConfigLong: map[Language]string{
		LangZh: "管理全局和项目级别的 iPen 配置设置。",
		LangEn: "Manage global and project-level iPen configuration settings.",
	},
	CmdConfigSetShort: map[Language]string{
		LangZh: "设置配置值",
		LangEn: "Set a configuration value",
	},
	CmdConfigSetLong: map[Language]string{
		LangZh: "设置配置值，例如 llm.apiKey、llm.model 等。",
		LangEn: "Set a configuration value, e.g., llm.apiKey, llm.model",
	},
	CmdConfigGetShort: map[Language]string{
		LangZh: "获取配置值",
		LangEn: "Get a configuration value",
	},

	// Consolidate
	CmdConsolidateShort: map[Language]string{
		LangZh: "整合内容",
		LangEn: "Consolidate content",
	},
	CmdConsolidateLong: map[Language]string{
		LangZh: "整合多个来源的内容。",
		LangEn: "Consolidate content from multiple sources",
	},

	// Daemon
	CmdDaemonShort: map[Language]string{
		LangZh: "管理 iPen 守护进程",
		LangEn: "Manage iPen daemon",
	},
	CmdDaemonLong: map[Language]string{
		LangZh: "启动和停止 iPen 后台守护进程。",
		LangEn: "Start and stop the iPen background daemon.",
	},
	CmdDaemonUpShort: map[Language]string{
		LangZh: "启动守护进程",
		LangEn: "Start the daemon",
	},
	CmdDaemonDownShort: map[Language]string{
		LangZh: "停止守护进程",
		LangEn: "Stop the daemon",
	},

	// Detect
	CmdDetectShort: map[Language]string{
		LangZh: "检测内容问题",
		LangEn: "Detect issues in content",
	},
	CmdDetectLong: map[Language]string{
		LangZh: "检测内容中的问题（如时代错误、不一致等）。",
		LangEn: "Detect issues in content (e.g., anachronisms, inconsistencies)",
	},

	// Doctor
	CmdDoctorShort: map[Language]string{
		LangZh: "检查 iPen 健康和配置",
		LangEn: "Check iPen health and configuration",
	},
	CmdDoctorLong: map[Language]string{
		LangZh: "诊断常见问题并检查 iPen 是否正确配置。",
		LangEn: "Diagnose common issues and check if iPen is properly configured.",
	},

	// Draft
	CmdDraftShort: map[Language]string{
		LangZh: "管理草稿",
		LangEn: "Manage drafts",
	},
	CmdDraftLong: map[Language]string{
		LangZh: "创建和管理草稿章节。",
		LangEn: "Create and manage draft chapters",
	},
	CmdDraftCreateShort: map[Language]string{
		LangZh: "创建草稿",
		LangEn: "Create draft",
	},
	CmdDraftListShort: map[Language]string{
		LangZh: "列出草稿",
		LangEn: "List drafts",
	},

	// Eval
	CmdEvalShort: map[Language]string{
		LangZh: "评估内容质量",
		LangEn: "Evaluate content quality",
	},
	CmdEvalLong: map[Language]string{
		LangZh: "评估和评分内容质量。",
		LangEn: "Evaluate and score content quality.",
	},

	// Export
	CmdExportShort: map[Language]string{
		LangZh: "导出书籍内容",
		LangEn: "Export book content",
	},
	CmdExportLong: map[Language]string{
		LangZh: "导出书籍内容为各种格式。",
		LangEn: "Export book content to various formats",
	},

	// Fanfic
	CmdFanficShort: map[Language]string{
		LangZh: "管理同人小说",
		LangEn: "Manage fanfiction",
	},
	CmdFanficLong: map[Language]string{
		LangZh: "创建和管理同人小说（基于原作正典）。",
		LangEn: "Create and manage fanfiction (based on parent canon)",
	},

	// Genre
	CmdGenreShort: map[Language]string{
		LangZh: "管理小说类型",
		LangEn: "Manage genres",
	},
	CmdGenreLong: map[Language]string{
		LangZh: "列表和配置类型配置文件。",
		LangEn: "List and configure genre profiles",
	},
	CmdGenreListShort: map[Language]string{
		LangZh: "列出所有类型",
		LangEn: "List all genres",
	},
	CmdGenreShowShort: map[Language]string{
		LangZh: "显示类型详情",
		LangEn: "Show genre details",
	},

	// Import
	CmdImportShort: map[Language]string{
		LangZh: "导入内容",
		LangEn: "Import content",
	},
	CmdImportLong: map[Language]string{
		LangZh: "从其他来源导入章节或正典内容",
		LangEn: "Import chapters or canon from other sources",
	},
	CmdImportChaptersShort: map[Language]string{
		LangZh: "导入章节",
		LangEn: "Import chapters",
	},
	CmdImportChaptersLong: map[Language]string{
		LangZh: "从指定目录导入章节文件。",
		LangEn: "Import chapter files from a directory.",
	},
	CmdImportCanonShort: map[Language]string{
		LangZh: "导入正典",
		LangEn: "Import canon",
	},
	CmdImportCanonLong: map[Language]string{
		LangZh: "从原作书籍导入正典内容到目标书籍。",
		LangEn: "Import canon content from parent book to target book.",
	},

	// Init
	CmdInitShort: map[Language]string{
		LangZh: "初始化 iPen 项目",
		LangEn: "Initialize an iPen project",
	},
	CmdInitLong: map[Language]string{
		LangZh: "初始化 iPen 项目（默认当前目录）。如果提供名称，则会创建子目录。",
		LangEn: "Initialize an iPen project (current directory by default). If name is provided, creates subdirectory.",
	},

	// Plan
	CmdPlanShort: map[Language]string{
		LangZh: "管理故事规划",
		LangEn: "Manage story planning",
	},
	CmdPlanLong: map[Language]string{
		LangZh: "规划章节输入材料",
		LangEn: "Plan chapter input artifacts",
	},
	CmdPlanChapterShort: map[Language]string{
		LangZh: "规划下一章章节意图",
		LangEn: "Plan chapter intent for next chapter",
	},
	CmdPlanChapterLong: map[Language]string{
		LangZh: "为下一章生成章节输入意图。",
		LangEn: "Generate chapter intent for the next chapter",
	},

	// Radar
	CmdRadarShort: map[Language]string{
		LangZh: "监控和检测变化",
		LangEn: "Monitor and detect changes",
	},
	CmdRadarLong: map[Language]string{
		LangZh: "监控网络并根据配置源检测变化。",
		LangEn: "Monitor web and detect changes based on configured sources.",
	},

	// Review
	CmdReviewShort: map[Language]string{
		LangZh: "审查和批准章节",
		LangEn: "Review and approve chapters",
	},
	CmdReviewLong: map[Language]string{
		LangZh: "审查和批准已创作的章节。",
		LangEn: "Review and approve written chapters",
	},
	CmdReviewListShort: map[Language]string{
		LangZh: "列出待审查章节",
		LangEn: "List chapters pending review",
	},
	CmdReviewApproveShort: map[Language]string{
		LangZh: "批准章节",
		LangEn: "Approve chapter",
	},

	// Revise
	CmdReviseShort: map[Language]string{
		LangZh: "修订章节",
		LangEn: "Revise chapter",
	},
	CmdReviseLong: map[Language]string{
		LangZh: "根据审计问题修订章节。如果不指定章节号，则修订最新一章。",
		LangEn: "Revise an chapter based on audit issues. Defaults to latest chapter if omitted.",
	},

	// Status
	CmdStatusShort: map[Language]string{
		LangZh: "显示项目状态",
		LangEn: "Show project status",
	},
	CmdStatusLong: map[Language]string{
		LangZh: "显示所有书籍或指定书籍的状态。",
		LangEn: "Show status for all books or a specific book.",
	},

	// Studio
	CmdStudioShort: map[Language]string{
		LangZh: "启动 iPen Studio",
		LangEn: "Launch iPen Studio",
	},
	CmdStudioLong: map[Language]string{
		LangZh: "启动 iPen Studio Web 界面。",
		LangEn: "Launch the iPen Studio web interface.",
	},

	// Style
	CmdStyleShort: map[Language]string{
		LangZh: "管理写作风格",
		LangEn: "Manage writing style",
	},
	CmdStyleLong: map[Language]string{
		LangZh: "管理和应用写作风格设置。",
		LangEn: "Manage and apply writing style settings",
	},

	// Update
	CmdUpdateShort: map[Language]string{
		LangZh: "更新 iPen",
		LangEn: "Update iPen",
	},
	CmdUpdateLong: map[Language]string{
		LangZh: "将 iPen 更新到最新版本。",
		LangEn: "Update iPen to the latest version.",
	},

	// Write
	CmdWriteShort: map[Language]string{
		LangZh: "创作章节",
		LangEn: "Write chapters",
	},
	CmdWriteLong: map[Language]string{
		LangZh: "为书籍创作、重写和修复章节。",
		LangEn: "Write, rewrite, and repair chapters for your books.",
	},
	CmdWriteNextShort: map[Language]string{
		LangZh: "为书籍创作下一章",
		LangEn: "Write the next chapter for a book",
	},
	CmdWriteNextLong: map[Language]string{
		LangZh: "为书籍创作下一章。如果省略书籍ID且只有一本书，则自动检测。",
		LangEn: "Write the next chapter for a book. Auto-detects book ID if only one book exists.",
	},
	CmdWriteRewriteShort: map[Language]string{
		LangZh: "重新生成指定章节",
		LangEn: "Re-generate a specific chapter",
	},
	CmdWriteRewriteLong: map[Language]string{
		LangZh: "重新生成指定章节。会删除该章节及之后的所有章节，然后重新创作。",
		LangEn: "Re-generate a specific chapter. Deletes this chapter and all later chapters, then re-writes.",
	},
	CmdWriteRepairShort: map[Language]string{
		LangZh: "重建已降级章节的真实文件",
		LangEn: "Rebuild truth files for degraded chapter",
	},
	CmdWriteRepairLong: map[Language]string{
		LangZh: "重建已降级章节的真实文件，但不重新创作章节正文。",
		LangEn: "Rebuild truth files for a degraded chapter without re-writing the chapter body.",
	},

	// Usage strings
	UsageHelp: map[Language]string{
		LangZh: `使用 "ipen [command] --help" 获取更多命令帮助。`,
		LangEn: `Use "ipen [command] --help" for more information about a command.`,
	},

	// Examples
	ExConfigSet: map[Language]string{
		LangZh: "  ipen config set llm.model gpt-4\n  ipen config set language zh",
		LangEn: "  ipen config set llm.model gpt-4\n  ipen config set language zh",
	},
	ExConfigGet: map[Language]string{
		LangZh: "  ipen config get llm.model\n  ipen config get language",
		LangEn: "  ipen config get llm.model\n  ipen config get language",
	},
	ExConfigSetModel: map[Language]string{
		LangZh: "  ipen config set-model writer gpt-4\n  ipen config set-model auditor claude-3 --provider anthropic",
		LangEn: "  ipen config set-model writer gpt-4\n  ipen config set-model auditor claude-3 --provider anthropic",
	},
	ExConfigRmModel: map[Language]string{
		LangZh: "  ipen config remove-model writer",
		LangEn: "  ipen config remove-model writer",
	},
}

// T returns the translated string for the given key and current language
func T(translations map[Language]string) string {
	if val, ok := translations[CurrentLanguage]; ok {
		return val
	}
	// Fallback to Chinese if translation not found
	if val, ok := translations[LangZh]; ok {
		return val
	}
	return ""
}
