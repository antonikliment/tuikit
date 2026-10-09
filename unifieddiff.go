package tuikit

import (
	"strconv"
	"strings"
)

// DiffFile is one file in a unified git diff. Before and After contain the
// changed hunks and their context, not the full files.
type DiffFile struct {
	Path, Before, After string
	Deleted             bool
}

// ParseUnifiedDiff reconstructs per-file fragments for DiffView. It accepts
// git diff and git show output, skipping commit and file metadata.
func ParseUnifiedDiff(diff string) []DiffFile {
	var files []DiffFile
	current := -1
	inHunk := false
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			header := strings.TrimPrefix(line, "diff --git ")
			_, path, _ := strings.Cut(header, " b/")
			if path == "" {
				if i := strings.LastIndex(header, "\"b/"); i >= 0 {
					path = diffPath(header[i:])
				}
			}
			files = append(files, DiffFile{Path: path})
			current, inHunk = len(files)-1, false
		case current < 0:
			continue
		case strings.HasPrefix(line, "deleted file mode"):
			files[current].Deleted = true
		case strings.HasPrefix(line, "+++ "):
			if path := diffPath(strings.TrimPrefix(line, "+++ ")); path != "" {
				files[current].Path = path
			}
		case strings.HasPrefix(line, "--- ") && files[current].Deleted:
			if path := diffPath(strings.TrimPrefix(line, "--- ")); path != "" {
				files[current].Path = path
			}
		case strings.HasPrefix(line, "@@"):
			if inHunk {
				files[current].Before += "⋯\n"
				files[current].After += "⋯\n"
			}
			inHunk = true
		case inHunk && len(line) > 0:
			switch line[0] {
			case '+':
				files[current].After += line[1:] + "\n"
			case '-':
				files[current].Before += line[1:] + "\n"
			case ' ':
				files[current].Before += line[1:] + "\n"
				files[current].After += line[1:] + "\n"
			}
		}
	}
	return files
}

func diffPath(path string) string {
	path = strings.TrimSpace(path)
	if decoded, err := strconv.Unquote(path); err == nil {
		path = decoded
	}
	if path == "/dev/null" {
		return ""
	}
	if strings.HasPrefix(path, "a/") || strings.HasPrefix(path, "b/") {
		return path[2:]
	}
	return path
}
