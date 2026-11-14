package endpoints

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// moduleRoot 是模块根目录：测试跑在 internal/endpoints 下。
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("解析模块根目录: %v", err)
	}
	return root
}

// TestDocsConstantsAreInSync 校验 docs/endpoints.md 的端点常量表与源码一致。
func TestDocsConstantsAreInSync(t *testing.T) {
	root := moduleRoot(t)
	idx, err := Scan(root)
	if err != nil {
		t.Fatalf("扫描源码: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "docs", "endpoints.md"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("docs/ 未纳入版本控制，跳过漂移校验")
		}
		t.Fatalf("读取 docs/endpoints.md: %v", err)
	}
	got, ok := extract(string(body), "<!-- BEGIN GENERATED: constants -->", "<!-- END GENERATED: constants -->")
	if !ok {
		t.Fatal("docs/endpoints.md 缺少端点常量表的生成标记")
	}
	want := strings.TrimSpace(idx.ConstantsTable())
	if got != want {
		t.Errorf("docs/endpoints.md 的端点常量表与源码不一致：\n--- 文档 ---\n%s\n--- 源码 ---\n%s", got, want)
	}
}

// TestDocsFamiliesAreInSync 校验 docs/caching.md 的键族表与源码一致。
func TestDocsFamiliesAreInSync(t *testing.T) {
	root := moduleRoot(t)
	idx, err := Scan(root)
	if err != nil {
		t.Fatalf("扫描源码: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "docs", "caching.md"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("docs/ 未纳入版本控制，跳过漂移校验")
		}
		t.Fatalf("读取 docs/caching.md: %v", err)
	}
	got, ok := extract(string(body), "<!-- BEGIN GENERATED: families -->", "<!-- END GENERATED: families -->")
	if !ok {
		t.Fatal("docs/caching.md 缺少键族表的生成标记")
	}
	want := strings.TrimSpace(idx.FamiliesTable())
	if got != want {
		t.Errorf("docs/caching.md 的键族表与源码不一致：\n--- 文档 ---\n%s\n--- 源码 ---\n%s", got, want)
	}
}

// TestMethodsTableCoversEveryEndpoint 校验方法总表覆盖了全部端点常量。
//
// 方法总表是手工维护的：位置形态这一列要靠人判断，扫不出来。端点常量
// 那一列可以校验 —— 源码里有、文档里没有的端点，读者就查不到。
func TestMethodsTableCoversEveryEndpoint(t *testing.T) {
	root := moduleRoot(t)
	idx, err := Scan(root)
	if err != nil {
		t.Fatalf("扫描源码: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "docs", "endpoints.md"))
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("docs/ 未纳入版本控制，跳过漂移校验")
		}
		t.Fatalf("读取 docs/endpoints.md: %v", err)
	}
	methods, ok := extract(string(body), "<!-- BEGIN TABLE: methods -->", "<!-- END TABLE: methods -->")
	if !ok {
		t.Fatal("docs/endpoints.md 缺少方法总表")
	}
	known := map[string]bool{}
	for _, c := range idx.Constants {
		known[c.Const] = true
	}
	mentioned := map[string]bool{}
	for _, line := range strings.Split(methods, "\n") {
		if !strings.HasPrefix(line, "| ") {
			continue
		}
		cols := strings.Split(line, "|")
		if len(cols) < 3 {
			continue
		}
		name := strings.TrimSpace(cols[len(cols)-2])
		if name == "端点常量" || name == "" || name == "---" {
			continue
		}
		for _, part := range strings.Split(name, " / ") {
			part = strings.Trim(strings.TrimSpace(part), "\x60")
			if part == "" {
				continue
			}
			if !known[part] {
				t.Errorf("方法总表里的端点常量 %q 在源码里不存在", part)
			}
			mentioned[part] = true
		}
	}
	for _, c := range idx.Constants {
		if !mentioned[c.Const] {
			t.Errorf("端点常量 %q 没有出现在 docs/endpoints.md 的方法总表里", c.Const)
		}
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
