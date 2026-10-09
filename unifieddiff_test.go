package tuikit

import "testing"

func TestParseUnifiedDiff(t *testing.T) {
	diff := "commit abc\n" +
		"diff --git a/old.txt b/new.txt\nrename from old.txt\nrename to new.txt\n" +
		"--- a/old.txt\n+++ b/new.txt\n@@ -1 +1 @@\n-old\n+new\n" +
		"@@ -10 +10 @@\n context\n-next\n+after\n" +
		"diff --git a/gone.txt b/gone.txt\ndeleted file mode 100644\n" +
		"--- a/gone.txt\n+++ /dev/null\n@@ -1 +0,0 @@\n-gone\n"
	files := ParseUnifiedDiff(diff)
	if len(files) != 2 {
		t.Fatalf("files = %d", len(files))
	}
	if files[0].Path != "new.txt" || files[0].Before != "old\n⋯\ncontext\nnext\n" || files[0].After != "new\n⋯\ncontext\nafter\n" {
		t.Fatalf("rename = %+v", files[0])
	}
	if files[1].Path != "gone.txt" || !files[1].Deleted || files[1].Before != "gone\n" || files[1].After != "" {
		t.Fatalf("delete = %+v", files[1])
	}
	if got := ParseUnifiedDiff("not a diff"); len(got) != 0 {
		t.Fatalf("plain text yielded %+v", got)
	}
}

func TestParseUnifiedDiffQuotedPath(t *testing.T) {
	files := ParseUnifiedDiff("diff --git \"a/name with space.go\" \"b/name with space.go\"\n--- \"a/name with space.go\"\n+++ \"b/name with space.go\"\n@@ -1 +1 @@\n-old\n+new\n")
	if len(files) != 1 || files[0].Path != "name with space.go" {
		t.Fatalf("quoted path = %+v", files)
	}
}
