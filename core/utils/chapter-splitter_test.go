package utils

import "testing"

func TestSplitChapters_ChineseHeadings(t *testing.T) {
	input := "第一回：桃园结义\n\n群雄并起。\n\n第二回：怒鞭督邮\n\n风云再起。"
	chapters := SplitChapters(input)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "桃园结义" || chapters[0].Content != "群雄并起。" {
		t.Fatalf("unexpected first chapter: %#v", chapters[0])
	}
	if chapters[1].Title != "怒鞭督邮" || chapters[1].Content != "风云再起。" {
		t.Fatalf("unexpected second chapter: %#v", chapters[1])
	}
}

func TestSplitChapters_FallbackChineseTitle(t *testing.T) {
	input := "第一回\n\n天下大势，分久必合。"
	chapters := SplitChapters(input)
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	if chapters[0].Title != "第1回" {
		t.Fatalf("expected fallback title 第1回, got %q", chapters[0].Title)
	}
}

func TestSplitChapters_RoundZeroNumeral(t *testing.T) {
	input := "第九〇九回：秋雨退兵\n\n且看下文分解。\n\n第一〇〇回：漫兵攻城\n\n另有安排。"
	chapters := SplitChapters(input)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "秋雨退兵" || chapters[1].Title != "漫兵攻城" {
		t.Fatalf("unexpected titles: %#v", chapters)
	}
}

func TestSplitChapters_EnglishAndRomanHeadings(t *testing.T) {
	english := "Chapter 1: Prelude\n\nThe harbor bells rang before dawn.\n\nChapter 2: Into the Fog\n\nMara followed the lantern."
	chapters := SplitChapters(english)
	if len(chapters) != 2 {
		t.Fatalf("expected 2 chapters, got %d", len(chapters))
	}
	if chapters[0].Title != "Prelude" || chapters[1].Title != "Into the Fog" {
		t.Fatalf("unexpected chapter titles: %#v", chapters)
	}

	roman := "CHAPTER I.\n\nThe harbor bells rang before dawn.\n\nCHAPTER II.\n\nMara followed the lantern."
	romanChapters := SplitChapters(roman)
	if len(romanChapters) != 2 {
		t.Fatalf("expected 2 roman chapters, got %d", len(romanChapters))
	}
	if romanChapters[0].Title != "Chapter 1" || romanChapters[1].Title != "Chapter 2" {
		t.Fatalf("expected fallback roman titles, got %#v", romanChapters)
	}
}

func TestSplitChapters_CustomRegexFallbackAndLicenseStrip(t *testing.T) {
	roman := "CHAPTER I.\n\nThe harbor bells rang before dawn."
	chapters := SplitChapters(roman, `^CHAPTER\s+[IVXLCDM]+\.$`)
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	if chapters[0].Title != "Chapter 1" {
		t.Fatalf("expected fallback title Chapter 1, got %q", chapters[0].Title)
	}

	withTrailer := "Chapter 1: Finale\n\nThe harbor bells rang once and went silent.\n\nProject Gutenberg™ depends upon support.\npublic donations continue."
	stripped := SplitChapters(withTrailer)
	if len(stripped) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(stripped))
	}
	if stripped[0].Content != "The harbor bells rang once and went silent." {
		t.Fatalf("expected trailer stripped, got %q", stripped[0].Content)
	}
}
