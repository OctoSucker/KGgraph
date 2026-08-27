package source

import "testing"

func TestSplitYhgyChapters(t *testing.T) {
	in := `前言一段
01章 卜筮格言
原文一
（以下是小雅译）
译文一
02章 启蒙节要
原文二
（以下是小雅译）
译文二`
	chapters, preface := splitYhgyChapters(in, true, reYhgyChapter)
	if preface != "前言一段" {
		t.Fatalf("preface = %q", preface)
	}
	if len(chapters) != 2 {
		t.Fatalf("chapters = %d, want 2", len(chapters))
	}
	if chapters[0].Text != "原文一" || chapters[1].Text != "原文二" {
		t.Fatalf("translation not dropped: %+v", chapters)
	}
	if chapters[0].Num != "01" || chapters[0].Name != "卜筮格言" {
		t.Fatalf("chapter meta wrong: %+v", chapters[0])
	}
}

func TestSplitYhgyChaptersKeepTranslation(t *testing.T) {
	in := `01章 卜筮格言
原文一
（以下是小雅译）
译文一`
	chapters, _ := splitYhgyChapters(in, false, reYhgyChapter)
	if len(chapters) != 1 {
		t.Fatalf("chapters = %d", len(chapters))
	}
	if chapters[0].Text != "原文一\n【译文】（以下是小雅译）\n译文一" {
		t.Fatalf("keep-translation output wrong: %q", chapters[0].Text)
	}
}
