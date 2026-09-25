package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mirzakhany/yoga/icons"
)

const fileDialogRecentLimit = 10

type fileEntry struct {
	Path    string
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

type filePlace struct {
	ID    string
	Label string
	Icon  icons.Icon
	Path  string // empty for the Recent virtual place
}

func defaultFileDialogDir() string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return home
	}
	cwd, err := os.Getwd()
	if err != nil || cwd == "" {
		return "."
	}
	return cwd
}

func listFilePlaces(recent []string) []filePlace {
	var out []filePlace
	if len(recent) > 0 {
		out = append(out, filePlace{ID: "recent", Label: "Recent", Icon: icons.Clock})
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return out
	}
	out = append(out, filePlace{ID: "home", Label: "Home", Icon: icons.House, Path: home})
	for _, p := range []filePlace{
		{ID: "desktop", Label: "Desktop", Icon: icons.Folder, Path: filepath.Join(home, "Desktop")},
		{ID: "documents", Label: "Documents", Icon: icons.Folder, Path: filepath.Join(home, "Documents")},
		{ID: "downloads", Label: "Downloads", Icon: icons.Download, Path: filepath.Join(home, "Downloads")},
		{ID: "pictures", Label: "Pictures", Icon: icons.Folder, Path: filepath.Join(home, "Pictures")},
		{ID: "music", Label: "Music", Icon: icons.Folder, Path: filepath.Join(home, "Music")},
		{ID: "videos", Label: "Videos", Icon: icons.Folder, Path: filepath.Join(home, "Videos")},
		{ID: "movies", Label: "Movies", Icon: icons.Folder, Path: filepath.Join(home, "Movies")},
	} {
		if st, err := os.Stat(p.Path); err == nil && st.IsDir() {
			out = append(out, p)
		}
	}
	return out
}

func readFileEntries(dir string, showHidden bool) []fileEntry {
	list, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]fileEntry, 0, len(list))
	for _, de := range list {
		name := de.Name()
		if !showHidden && len(name) > 0 && name[0] == '.' {
			continue
		}
		e := fileEntry{
			Path:  filepath.Join(dir, name),
			Name:  name,
			IsDir: de.IsDir(),
		}
		if info, err := de.Info(); err == nil {
			e.Size = info.Size()
			e.ModTime = info.ModTime()
		}
		out = append(out, e)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
	return out
}

func filterFileEntries(entries []fileEntry, f FileFilter) []fileEntry {
	if len(f.Exts) == 0 {
		return entries
	}
	out := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		if e.IsDir || matchFileExt(e.Name, f.Exts) {
			out = append(out, e)
		}
	}
	return out
}

func matchFileExt(name string, exts []string) bool {
	got := strings.ToLower(filepath.Ext(name))
	for _, raw := range exts {
		want := strings.ToLower(strings.TrimSpace(raw))
		if want == "" {
			continue
		}
		if !strings.HasPrefix(want, ".") {
			want = "." + want
		}
		if got == want {
			return true
		}
	}
	return false
}

func statFileEntry(path string) (fileEntry, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return fileEntry{}, false
	}
	return fileEntry{
		Path:    path,
		Name:    filepath.Base(path),
		IsDir:   info.IsDir(),
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}, true
}

func formatFileSize(n int64, isDir bool) string {
	if isDir {
		return "—"
	}
	const kb = 1024.0
	v := float64(n)
	switch {
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case v < kb*kb:
		return fmt.Sprintf("%.1f KB", v/kb)
	case v < kb*kb*kb:
		return fmt.Sprintf("%.1f MB", v/(kb*kb))
	default:
		return fmt.Sprintf("%.1f GB", v/(kb*kb*kb))
	}
}

func formatFileType(name string, isDir bool) string {
	if isDir {
		return "Folder"
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(name)), ".")
	if ext == "" {
		return "File"
	}
	return strings.ToUpper(ext)
}

func formatFileMod(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	now := time.Now()
	if t.Year() == now.Year() && t.YearDay() == now.YearDay() {
		return t.Format("15:04")
	}
	return t.Format("2 Jan 2006")
}

func pathCrumbs(dir string) []string {
	dir = filepath.Clean(dir)
	var segs []string
	for {
		segs = append([]string{dir}, segs...)
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return segs
}

func crumbLabel(p string) string {
	if filepath.Dir(p) == p {
		if p == string(filepath.Separator) {
			return string(filepath.Separator)
		}
		return p
	}
	base := filepath.Base(p)
	if base == "" || base == "." {
		return p
	}
	return base
}

func rememberRecent(recent []string, paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, fileDialogRecentLimit)
	add := func(p string) {
		p = filepath.Clean(p)
		if p == "" || p == "." {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	for _, p := range paths {
		if len(out) >= fileDialogRecentLimit {
			return out
		}
		add(p)
	}
	for _, p := range recent {
		if len(out) >= fileDialogRecentLimit {
			break
		}
		add(p)
	}
	return out
}

func sameFilePath(a, b string) bool {
	return filepath.Clean(a) == filepath.Clean(b)
}
