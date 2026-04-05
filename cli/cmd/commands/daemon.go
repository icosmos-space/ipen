package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/icosmos-space/ipen/core/pipeline"
	"github.com/spf13/cobra"
)

const daemonPIDFile = "ipen.pid"

// DaemonCommand 管理守护进程生命周期。
func DaemonCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "daemon",
		Short: T(TR.CmdDaemonShort),
		Long:  T(TR.CmdDaemonLong),
	}
	cmd.AddCommand(daemonUpCommand())
	cmd.AddCommand(daemonDownCommand())
	return cmd
}

func daemonUpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "up",
		Short: T(TR.CmdDaemonUpShort),
		RunE:  runDaemonUp,
	}
	cmd.Flags().BoolP("quiet", "q", false, "Suppress console output")
	return cmd
}

func runDaemonUp(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	config, err := loadConfig(root)
	if err != nil {
		return err
	}
	quiet, _ := cmd.Flags().GetBool("quiet")

	pidPath := filepath.Join(root, daemonPIDFile)
	if _, err := os.Stat(pidPath); err == nil {
		existing, _ := os.ReadFile(pidPath)
		return fmt.Errorf("daemon appears to be running (PID: %s). run `ipen daemon down` first", strings.TrimSpace(string(existing)))
	}
	if err := os.WriteFile(pidPath, []byte(strconv.Itoa(os.Getpid())), 0644); err != nil {
		return err
	}
	defer func() {
		_ = os.Remove(pidPath)
	}()

	if !quiet {
		fmt.Println("Starting daemon...")
		fmt.Printf("  Write cycle: %s\n", config.Daemon.Schedule.WriteCron)
		fmt.Printf("  Radar scan: %s\n", config.Daemon.Schedule.RadarCron)
		fmt.Printf("  Max concurrent books: %d\n", config.Daemon.MaxConcurrentBooks)
	}

	scheduler := pipeline.NewScheduler(pipeline.SchedulerConfig{
		PipelineConfig:         buildPipelineConfig(config, root, quiet),
		RadarCron:              config.Daemon.Schedule.RadarCron,
		WriteCron:              config.Daemon.Schedule.WriteCron,
		MaxConcurrentBooks:     config.Daemon.MaxConcurrentBooks,
		ChaptersPerCycle:       config.Daemon.ChaptersPerCycle,
		RetryDelayMs:           config.Daemon.RetryDelayMs,
		CooldownAfterChapterMs: config.Daemon.CooldownAfterChapterMs,
		MaxChaptersPerDay:      config.Daemon.MaxChaptersPerDay,
		QualityGates:           &config.Daemon.QualityGates,
		Detection:              config.Detection,
		OnChapterComplete: func(bookID string, chapter int, status string) {
			if quiet {
				return
			}
			fmt.Printf("[chapter] %s Ch.%d => %s\n", bookID, chapter, status)
		},
		OnError: func(bookID string, err error) {
			fmt.Printf("[error] %s: %v\n", bookID, err)
		},
		OnPause: func(bookID string, reason string) {
			fmt.Printf("[pause] %s: %s\n", bookID, reason)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		return err
	}
	defer scheduler.Stop()

	if !quiet {
		fmt.Println("Daemon running. Press Ctrl+C to stop.")
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	<-stop

	if !quiet {
		fmt.Println("Shutting down daemon...")
	}
	return nil
}

func daemonDownCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "down",
		Short: T(TR.CmdDaemonDownShort),
		RunE:  runDaemonDown,
	}
}

func runDaemonDown(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	pidPath := filepath.Join(root, daemonPIDFile)
	raw, err := os.ReadFile(pidPath)
	if err != nil {
		fmt.Println("No daemon running.")
		return nil
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil {
		_ = os.Remove(pidPath)
		return fmt.Errorf("invalid pid file content")
	}

	process, err := os.FindProcess(pid)
	if err == nil {
		_ = process.Kill()
	}
	_ = os.Remove(pidPath)
	fmt.Printf("Daemon stopped (PID: %d).\n", pid)
	return nil
}

// DaemonUpAliasCommand 保持与旧版 `ipen up` 的兼容。
func DaemonUpAliasCommand() *cobra.Command {
	cmd := daemonUpCommand()
	cmd.Use = "up"
	cmd.Hidden = true
	return cmd
}

// DaemonDownAliasCommand 保持与旧版 `ipen down` 的兼容。
func DaemonDownAliasCommand() *cobra.Command {
	cmd := daemonDownCommand()
	cmd.Use = "down"
	cmd.Hidden = true
	return cmd
}
