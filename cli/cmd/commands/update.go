package commands

import (
	"fmt"
	"os/exec"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

const updateModulePath = "github.com/icosmos-space/ipen/cli"

// UpdateCommand 更新 iPen Go CLI。
func UpdateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update",
		Short: T(TR.CmdUpdateShort),
		Long:  T(TR.CmdUpdateLong),
		RunE:  runUpdate,
	}
	cmd.Flags().Bool("check", false, "Only print current version and update command")
	return cmd
}

func runUpdate(cmd *cobra.Command, args []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	current := "dev"
	if info, ok := debug.ReadBuildInfo(); ok && strings.TrimSpace(info.Main.Version) != "" {
		current = info.Main.Version
	}

	fmt.Printf("Current version: %s\n", current)
	fmt.Printf("Update command: go install %s@latest\n", updateModulePath)
	if checkOnly {
		return nil
	}

	installCmd := exec.Command("go", "install", updateModulePath+"@latest")
	output, err := installCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("update failed: %v\n%s", err, strings.TrimSpace(string(output)))
	}

	if msg := strings.TrimSpace(string(output)); msg != "" {
		fmt.Println(msg)
	}
	fmt.Println("Update complete. Restart your terminal if command path changed.")
	return nil
}
