package agents

import (
	"strings"
	"testing"
)

func TestConsolidator_ParseVolumeBoundaries_ChineseFullWidthRange(t *testing.T) {
	agent := &ConsolidatorAgent{}
	outline := stringsJoinLines(
		"# Volume Outline",
		"",
		"### 第一卷：死而复生的实习月（1-20章）",
		"- 主角重返公司，卷入第一起异常事件",
		"",
		"### 第二卷：时间线上的猎手（21-60章）",
		"- 追查时间裂隙背后的操控者",
		"",
	)

	boundaries := agent.parseVolumeBoundaries(outline)
	if len(boundaries) != 2 {
		t.Fatalf("expected 2 boundaries, got %#v", boundaries)
	}
	if boundaries[0].Name != "第一卷：死而复生的实习月" || boundaries[0].StartCh != 1 || boundaries[0].EndCh != 20 {
		t.Fatalf("unexpected first boundary: %#v", boundaries[0])
	}
	if boundaries[1].Name != "第二卷：时间线上的猎手" || boundaries[1].StartCh != 21 || boundaries[1].EndCh != 60 {
		t.Fatalf("unexpected second boundary: %#v", boundaries[1])
	}
}

func stringsJoinLines(lines ...string) string {
	return strings.Join(lines, "\n")
}
