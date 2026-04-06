package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/icosmos-space/ipen/cli/cmd/commands"
	"github.com/icosmos-space/ipen/cli/cmd/i18n"
	"github.com/spf13/cobra"
)

// Version 构建时设置
var Version = "dev"

var rootCmd *cobra.Command

func init() {
	//
	cobra.MousetrapHelpText = "这是个命令行程序，请从终端启动。"
}

// detectLanguageFromArgs detects language from command line args
func detectLanguageFromArgs() {
	for i, arg := range os.Args {
		if arg == "--lang" && i+1 < len(os.Args) {
			setLanguage(os.Args[i+1])
			return
		}
		if strings.HasPrefix(arg, "--lang=") {
			setLanguage(arg[7:])
			return
		}
		if arg == "-g" && i+1 < len(os.Args) {
			setLanguage(os.Args[i+1])
			return
		}
	}
}

func setLanguage(lang string) {
	switch lang {
	case "zh", "cn", "zh-CN", "zh-Hans":
		i18n.CurrentLanguage = i18n.LangZh
	case "en", "eng", "en-US":
		i18n.CurrentLanguage = i18n.LangEn
	}
}

// T is a shorthand for i18n.T
func T(m map[i18n.Language]string) string { return i18n.T(m) }

// Translations is a shorthand for i18n.Translations
var Translations = i18n.Translations

func init() {
	// Detect language early from args before cobra parses them
	detectLanguageFromArgs()

	// Create root command
	rootCmd = &cobra.Command{
		Use:     "ipen",
		Short:   T(Translations.RootShort),
		Long:    T(Translations.RootLong),
		Version: Version,
	}

	// Global flags
	rootCmd.PersistentFlags().StringP("config", "c", "", T(Translations.FlagConfig))
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", T(Translations.FlagLogLevel))
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, T(Translations.FlagVerbose))
	rootCmd.PersistentFlags().VarP(&langFlag{}, "lang", "g", T(Translations.FlagLang))

	// Silence usage on errors
	rootCmd.SilenceUsage = true
	rootCmd.SilenceErrors = true

	// Custom help command
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [command]",
		Short: T(Translations.CmdHelpShort),
		Long:  T(Translations.CmdHelpLong),
	})

	rootCmd.CompletionOptions.DisableDefaultCmd = false

	// Flag descriptions
	rootCmd.Flags().BoolP("help", "h", false, T(Translations.RootHelp))
	rootCmd.Flags().Bool("version", false, T(Translations.RootVer))

	// Register completion command
	registerCompletionCommand()

	// Register all subcommands
	registerCommands()
}

func registerCompletionCommand() {
	completionCmd := &cobra.Command{
		Use:                   "completion [bash|zsh|fish|powershell]",
		Short:                 T(Translations.CmdCompletionShort),
		Long:                  T(Translations.CmdCompletionLong),
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
			}
			return nil
		},
	}

	rootCmd.AddCommand(completionCmd)
}

func registerCommands() {
	// Add all subcommands to root
	rootCmd.AddCommand(commands.InitCommand())
	rootCmd.AddCommand(commands.ConfigCommand())
	rootCmd.AddCommand(commands.BookCommand())
	rootCmd.AddCommand(commands.WriteCommand())
	rootCmd.AddCommand(commands.ReviewCommand())
	rootCmd.AddCommand(commands.StatusCommand())
	rootCmd.AddCommand(commands.RadarCommand())
	rootCmd.AddCommand(commands.DaemonCommand())
	rootCmd.AddCommand(commands.DaemonUpAliasCommand())
	rootCmd.AddCommand(commands.DaemonDownAliasCommand())
	rootCmd.AddCommand(commands.DoctorCommand())
	rootCmd.AddCommand(commands.ExportCommand())
	rootCmd.AddCommand(commands.DraftCommand())
	rootCmd.AddCommand(commands.AuditCommand())
	rootCmd.AddCommand(commands.ReviseCommand())
	rootCmd.AddCommand(commands.AgentCommand())
	rootCmd.AddCommand(commands.PlanCommand())
	rootCmd.AddCommand(commands.ComposeCommand())
	rootCmd.AddCommand(commands.GenreCommand())
	rootCmd.AddCommand(commands.UpdateCommand())
	rootCmd.AddCommand(commands.DetectCommand())
	rootCmd.AddCommand(commands.StyleCommand())
	rootCmd.AddCommand(commands.AnalyticsCommand())
	rootCmd.AddCommand(commands.EvalCommand())
	rootCmd.AddCommand(commands.ImportCommand())
	rootCmd.AddCommand(commands.FanficCommand())
	rootCmd.AddCommand(commands.StudioCommand())
	rootCmd.AddCommand(commands.ConsolidateCommand())
}

// Execute executes the root command
func Execute() error {
	return rootCmd.Execute()
}

// langFlag implements pflag.Value interface for language flag
type langFlag struct{}

func (f *langFlag) String() string {
	return string(i18n.CurrentLanguage)
}

func (f *langFlag) Set(value string) error {
	setLanguage(value)
	return nil
}

func (f *langFlag) Type() string {
	return "string"
}

// ExitOnError exits with error code 1 if err is not nil
func ExitOnError(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
