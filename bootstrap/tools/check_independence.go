// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] != "source" {
		fmt.Fprintln(os.Stderr, "用法：tools/check-independence source")
		os.Exit(2)
	}
	root, err := os.Getwd()
	if err != nil {
		fail(err.Error())
	}
	manifests := map[string]bool{
		"go.mod": true, "go.sum": true, "Cargo.toml": true,
		"package.json": true, "requirements.txt": true,
		"CMakeLists.txt": true,
	}
	dependencies := map[string]bool{
		"vendor": true, "node_modules": true,
		".toolchain": true, "toolchains": true, "sdk": true, ".venv": true,
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, _ := filepath.Rel(root, path)
		if rel == ".git" {
			return filepath.SkipDir
		}
		insideBootstrap := rel == "bootstrap" || strings.HasPrefix(rel, "bootstrap"+string(filepath.Separator))
		if entry.IsDir() && dependencies[entry.Name()] {
			return fmt.Errorf("发现依赖副本或工具链目录：%s", rel)
		}
		if entry.IsDir() && !insideBootstrap && (entry.Name() == "vendor" || entry.Name() == "node_modules" || entry.Name() == "third_party") {
			return fmt.Errorf("发现第三方源码目录：%s", rel)
		}
		if !entry.IsDir() && !insideBootstrap && strings.HasSuffix(entry.Name(), ".vcxproj") {
			return fmt.Errorf("发现跨语言构建描述：%s", rel)
		}
		if !entry.IsDir() && !insideBootstrap && manifests[entry.Name()] {
			return fmt.Errorf("发现跨语言构建描述：%s", rel)
		}
		if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), ".serica")) {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			for _, line := range strings.Split(string(data), "\n") {
				if strings.Contains(line, "Copyright (c)") && !strings.Contains(line, "行简（Serica）贡献者") {
					return fmt.Errorf("版权归属需核对：%s", rel)
				}
			}
		}
		return nil
	})
	if err != nil {
		fail(err.Error())
	}
	if err := checkCaches(root); err != nil {
		fail(err.Error())
	}
	fmt.Println("源码级：通过")
}

func checkCaches(root string) error {
	goBin := os.Getenv("GO")
	if goBin == "" {
		goBin = "go"
	}
	output, err := exec.Command(goBin, "env", "GOMODCACHE", "GOCACHE", "GOBIN").Output()
	if err != nil {
		return fmt.Errorf("读取 Go 缓存路径：%w", err)
	}
	for _, value := range strings.Fields(string(output)) {
		path, err := filepath.Abs(value)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			return fmt.Errorf("Go 缓存落在仓库内：%s", path)
		}
	}
	return nil
}

func fail(message string) {
	fmt.Fprintln(os.Stderr, "源码级：失败（"+message+"）")
	os.Exit(1)
}
