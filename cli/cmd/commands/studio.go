package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"
)

type studioLaunchSpec struct {
	StudioEntry string
	Command     string
	Args        []string
}

func firstExistingPath(paths []string) string {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func resolveStudioLaunch(root string) *studioLaunchSpec {
	sourceEntry := firstExistingPath([]string{
		filepath.Join(root, "packages", "studio", "src", "api", "index.ts"),
		filepath.Join(root, "..", "packages", "studio", "src", "api", "index.ts"),
		filepath.Join(root, "..", "studio", "src", "api", "index.ts"),
	})
	if sourceEntry != "" {
		studioPackageRoot := filepath.Dir(filepath.Dir(filepath.Dir(sourceEntry)))
		localLoader := firstExistingPath([]string{
			filepath.Join(studioPackageRoot, "node_modules", "tsx", "dist", "loader.mjs"),
		})
		if localLoader != "" {
			return &studioLaunchSpec{
				StudioEntry: sourceEntry,
				Command:     "node",
				Args:        []string{"--import", localLoader, sourceEntry, root},
			}
		}

		localTSX := firstExistingPath([]string{
			filepath.Join(studioPackageRoot, "node_modules", ".bin", "tsx.cmd"),
			filepath.Join(studioPackageRoot, "node_modules", ".bin", "tsx"),
		})
		if localTSX != "" {
			return &studioLaunchSpec{
				StudioEntry: sourceEntry,
				Command:     localTSX,
				Args:        []string{sourceEntry, root},
			}
		}

		return &studioLaunchSpec{
			StudioEntry: sourceEntry,
			Command:     "npx",
			Args:        []string{"tsx", sourceEntry, root},
		}
	}

	builtEntry := firstExistingPath([]string{
		filepath.Join(root, "node_modules", "@actalk", "ipen-studio", "dist", "api", "index.js"),
		filepath.Join(root, "node_modules", "@actalk", "ipen-studio", "server.cjs"),
		filepath.Join(root, "cli", "node_modules", "@actalk", "ipen-studio", "dist", "api", "index.js"),
		filepath.Join(root, "cli", "node_modules", "@actalk", "ipen-studio", "server.cjs"),
	})
	if builtEntry != "" {
		return &studioLaunchSpec{
			StudioEntry: builtEntry,
			Command:     "node",
			Args:        []string{builtEntry, root},
		}
	}
	return nil
}

// StudioCommand 启动 iPen Studio 工作台服务。
func StudioCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "studio",
		Short: T(TR.CmdStudioShort),
		Long:  T(TR.CmdStudioLong),
		RunE:  runStudio,
	}
	cmd.Flags().IntP("port", "p", 4567, "Studio server port")
	return cmd
}

func runStudio(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	port, _ := cmd.Flags().GetInt("port")
	launch := resolveStudioLaunch(root)
	if launch == nil {
		return fmt.Errorf(
			"iPen Studio not found. If this is a source checkout, build studio first and then rerun `ipen studio`",
		)
	}

	fmt.Printf("Starting iPen Studio on http://localhost:%d\n", port)
	child := exec.Command(launch.Command, launch.Args...)
	child.Dir = root
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr
	child.Stdin = os.Stdin
	child.Env = append(os.Environ(), "IPEN_STUDIO_PORT="+strconv.Itoa(port))
	return child.Run()
}
