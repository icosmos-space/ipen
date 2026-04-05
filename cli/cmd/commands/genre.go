package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/icosmos-space/ipen/core/agents"
	"github.com/spf13/cobra"
)

// GenreCommand 管理类型配置。
func GenreCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genre",
		Short: T(TR.CmdGenreShort),
		Long:  T(TR.CmdGenreLong),
	}

	cmd.AddCommand(genreListCommand())
	cmd.AddCommand(genreShowCommand())
	cmd.AddCommand(genreCreateCommand())
	cmd.AddCommand(genreCopyCommand())
	return cmd
}

func genreListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: T(TR.CmdGenreListShort),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			genres, err := agents.ListAvailableGenres(root)
			if err != nil {
				return err
			}
			if len(genres) == 0 {
				fmt.Println("未找到类型配置。")
				return nil
			}

			fmt.Println("Available genres:")
			for _, genre := range genres {
				fmt.Printf("  %-14s %-20s [%s]\n", genre.ID, genre.Name, genre.Source)
			}
			fmt.Printf("\nTotal: %d\n", len(genres))
			return nil
		},
	}
}

func genreShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: T(TR.CmdGenreShowShort),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := findProjectRoot()
			if err != nil {
				return err
			}

			id := strings.TrimSpace(args[0])
			genres, err := agents.ListAvailableGenres(root)
			if err != nil {
				return err
			}
			exact := false
			available := make([]string, 0, len(genres))
			for _, genre := range genres {
				available = append(available, genre.ID)
				if genre.ID == id {
					exact = true
				}
			}
			if !exact {
				sort.Strings(available)
				list := "(none)"
				if len(available) > 0 {
					list = strings.Join(available, ", ")
				}
				return fmt.Errorf("类型 %q 不存在，可用类型: %s", id, list)
			}

			profile, err := agents.ReadGenreProfile(root, id)
			if err != nil {
				return err
			}

			fmt.Printf("Genre: %s (%s)\n", profile.Profile.Name, id)
			if profile.Profile.Language != "" {
				fmt.Printf("Language: %s\n", profile.Profile.Language)
			}
			if profile.Profile.Description != "" {
				fmt.Printf("Description: %s\n", profile.Profile.Description)
			}
			if profile.Body != "" {
				fmt.Printf("\n--- Body ---\n%s\n", profile.Body)
			}
			return nil
		},
	}
}

func genreCreateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <id>",
		Short: "创建项目级类型配置",
		Args:  cobra.ExactArgs(1),
		RunE:  runGenreCreate,
	}
	cmd.Flags().String("name", "", "类型显示名称")
	cmd.Flags().Bool("numerical", false, "启用数值体系")
	cmd.Flags().Bool("power", false, "启用战力体系")
	cmd.Flags().Bool("era", false, "启用时代考据")
	return cmd
}

func runGenreCreate(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("类型 ID 不能为空")
	}

	name, _ := cmd.Flags().GetString("name")
	name = strings.TrimSpace(name)
	if name == "" {
		name = id
	}
	numerical, _ := cmd.Flags().GetBool("numerical")
	power, _ := cmd.Flags().GetBool("power")
	era, _ := cmd.Flags().GetBool("era")

	genresDir := filepath.Join(root, "genres")
	filePath := filepath.Join(genresDir, id+".md")
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("类型配置已存在: %s", filepath.ToSlash(filePath))
	}
	if err := os.MkdirAll(genresDir, 0755); err != nil {
		return err
	}

	template := fmt.Sprintf(`---
name: %s
id: %s
chapterTypes: ["progress", "setup", "transition", "payoff"]
fatigueWords: ["suddenly", "unbelievable", "without warning"]
numericalSystem: %t
powerScaling: %t
eraResearch: %t
pacingRule: "Deliver a clear progression or reveal every 2-3 chapters."
satisfactionTypes: ["goal_achieved", "obstacle_overcome", "truth_revealed"]
auditDimensions: [1,2,3,6,7,8,9,10,13,14,15,16,17,18,19]
---

## 题材禁忌

- 在此补充该题材不应触碰的禁忌内容。

## 叙事指引

- 在此描述该题材的叙事重心、节奏和文风要求。
`, name, id, numerical, power, era)
	if err := os.WriteFile(filePath, []byte(template), 0644); err != nil {
		return err
	}

	fmt.Printf("已创建类型配置: %s\n", filepath.ToSlash(filePath))
	fmt.Println("可继续编辑该文件，自定义章节类型、疲劳词和叙事规则。")
	return nil
}

func genreCopyCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "copy <id>",
		Short: "复制内置类型配置到项目",
		Args:  cobra.ExactArgs(1),
		RunE:  runGenreCopy,
	}
}

func runGenreCopy(cmd *cobra.Command, args []string) error {
	root, err := findProjectRoot()
	if err != nil {
		return err
	}
	id := strings.TrimSpace(args[0])
	if id == "" {
		return fmt.Errorf("类型 ID 不能为空")
	}

	genresDir := filepath.Join(root, "genres")
	destPath := filepath.Join(genresDir, id+".md")
	if _, err := os.Stat(destPath); err == nil {
		return fmt.Errorf("项目类型配置已存在: %s", filepath.ToSlash(destPath))
	}

	builtinDir := agents.GetBuiltinGenresDir()
	if strings.TrimSpace(builtinDir) == "" {
		return fmt.Errorf("未找到内置类型目录")
	}
	srcPath := filepath.Join(builtinDir, id+".md")
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("未找到内置类型 %q，可先执行 `ipen genre list` 查看", id)
	}

	if err := os.MkdirAll(genresDir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(destPath, content, 0644); err != nil {
		return err
	}

	fmt.Printf("已复制到: %s\n", filepath.ToSlash(destPath))
	fmt.Println("项目级副本会覆盖同名内置类型。")
	return nil
}
