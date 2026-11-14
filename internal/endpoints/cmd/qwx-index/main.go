// Command qwx-index 从源码扫出端点清单并刷新文档里的表体。
//
// 用法：
//
//	go run ./internal/endpoints/cmd/qwx-index -write
//
// 不带 -write 时只报告文档与源码的差异，不改文件。
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/KanoCifer/qwx/internal/endpoints"
)

// sections 是要刷新的文档段落：文件路径、起止标记、表体。
type section struct {
	path   string
	begin  string
	end    string
	render func(*endpoints.Index) string
}

func main() {
	write := flag.Bool("write", false, "把扫出的表体写回文档")
	root := flag.String("root", ".", "模块根目录")
	flag.Parse()

	idx, err := endpoints.Scan(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, "qwx-index:", err)
		os.Exit(1)
	}

	sections := []section{
		{
			path:   filepath.Join("docs", "endpoints.md"),
			begin:  "<!-- BEGIN GENERATED: constants -->",
			end:    "<!-- END GENERATED: constants -->",
			render: (*endpoints.Index).ConstantsTable,
		},
		{
			path:   filepath.Join("docs", "caching.md"),
			begin:  "<!-- BEGIN GENERATED: families -->",
			end:    "<!-- END GENERATED: families -->",
			render: (*endpoints.Index).FamiliesTable,
		},
	}

	drift := false
	for _, s := range sections {
		body, err := os.ReadFile(filepath.Join(*root, s.path))
		if err != nil {
			fmt.Fprintln(os.Stderr, "qwx-index:", err)
			os.Exit(1)
		}
		want := s.render(idx)
		got, ok := extract(string(body), s.begin, s.end)
		if !ok {
			fmt.Fprintf(os.Stderr, "qwx-index: %s 缺少标记 %s\n", s.path, s.begin)
			os.Exit(1)
		}
		if got == want {
			continue
		}
		drift = true
		if !*write {
			fmt.Fprintf(os.Stderr, "qwx-index: %s 与源码不一致，用 -write 刷新\n", s.path)
			continue
		}
		updated := replace(string(body), s.begin, s.end, want)
		if err := os.WriteFile(filepath.Join(*root, s.path), []byte(updated), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "qwx-index:", err)
			os.Exit(1)
		}
		fmt.Printf("qwx-index: 已刷新 %s\n", s.path)
	}
	if drift && !*write {
		os.Exit(1)
	}
}

// extract 取两个标记之间的内容。
func extract(body, begin, end string) (string, bool) {
	i := strings.Index(body, begin)
	if i < 0 {
		return "", false
	}
	i += len(begin)
	j := strings.Index(body[i:], end)
	if j < 0 {
		return "", false
	}
	return strings.TrimSpace(body[i : i+j]), true
}

// replace 替换两个标记之间的内容。
func replace(body, begin, end, body2 string) string {
	i := strings.Index(body, begin)
	if i < 0 {
		return body
	}
	i += len(begin)
	j := strings.Index(body[i:], end)
	if j < 0 {
		return body
	}
	return body[:i] + "\n" + strings.TrimRight(body2, "\n") + "\n" + body[i+j:]
}
