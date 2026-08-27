package source

import (
	"strings"
	"testing"
)

func TestCleanWikitext(t *testing.T) {
	in := `-{T|周易/乾}-
{{header2
|section=-{乾}-
}}
'''易經：'''
*'''乾'''：元亨。利貞。
*#初九：潛龍勿用。
<ref>note</ref>
[[周易|链接]] &amp; &nbsp; &#39;x&#39;`
	got := CleanWikitext(in)
	for _, want := range []string{
		"周易/乾", "易經：", "乾：元亨。利貞。", "初九：潛龍勿用。",
		"链接 &   x",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("CleanWikitext missing %q in output:\n%s", want, got)
		}
	}
	for _, bad := range []string{"{{header2", "'''", "<ref>", "&#39;"} {
		if strings.Contains(got, bad) {
			t.Fatalf("CleanWikitext left markup %q in output:\n%s", bad, got)
		}
	}
}

func TestVerifyText(t *testing.T) {
	clean := "木主於東；應春。陽氣觸動，冒地而生也。"
	if issues := VerifyText(clean); len(issues) != 0 {
		t.Fatalf("clean text should have no issues: %+v", issues)
	}
	dirty := "鐙髖愉鋼𔂻瘢瓤𔁯𔕥𔑴𔓧�" + clean
	if issues := VerifyText(dirty); len(issues) == 0 {
		t.Fatal("dirty text should be flagged")
	}
}
