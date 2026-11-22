// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type moduleLoader struct {
	rootDir         string
	state           map[string]uint8
	bindings        []binding
	origins         map[string]string
	functions       []functionDecl
	functionOrigins map[string]string
}

func setTokenFile(tokens []token, file string) {
	for index := range tokens {
		tokens[index].pos.file = file
	}
}

func loadModules(entryPath string, root *program) (*program, *issue) {
	absolute, err := filepath.Abs(entryPath)
	if err != nil {
		return nil, moduleIssue("S-4001", "有效的入口路径", entryPath, "", "使用项目内的 .serica 源文件", sourcePosition{file: filepath.Base(entryPath)})
	}
	loader := moduleLoader{
		rootDir:         filepath.Dir(absolute),
		state:           map[string]uint8{},
		origins:         map[string]string{},
		functionOrigins: map[string]string{},
	}
	rootInfo, err := os.Lstat(loader.rootDir)
	if err != nil || rootInfo.Mode()&os.ModeSymlink != 0 {
		return nil, moduleIssue("S-4001", "不经由符号链接的项目目录", loader.rootDir, "入口目录不可用或是符号链接", "从普通目录打开入口源文件", sourcePosition{file: filepath.Base(entryPath)})
	}
	loader.state[absolute] = 1
	if problem := loader.loadImports(absolute, filepath.Base(absolute), root.imports); problem != nil {
		return nil, problem
	}
	loader.state[absolute] = 2
	rootRel, err := filepath.Rel(loader.rootDir, absolute)
	if err != nil {
		return nil, moduleIssue("S-4001", "项目内入口路径", entryPath, "", "使用项目目录中的源文件", sourcePosition{file: filepath.Base(entryPath)})
	}
	rootRel = filepath.ToSlash(rootRel)
	if problem := loader.addBindings(root.bindings, rootRel); problem != nil {
		return nil, problem
	}
	if problem := loader.addFunctions(root.functions, rootRel); problem != nil {
		return nil, problem
	}
	return &program{
		rootName:   root.rootName,
		root:       root.root,
		bindings:   loader.bindings,
		functions:  loader.functions,
		statements: root.statements,
	}, nil
}

func (loader *moduleLoader) loadImports(parentPath, parentRel string, imports []importDecl) *issue {
	ordered := append([]importDecl(nil), imports...)
	sort.Slice(ordered, func(left, right int) bool {
		leftPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(ordered[left].path)))
		rightPath := filepath.ToSlash(filepath.Clean(filepath.FromSlash(ordered[right].path)))
		return leftPath < rightPath
	})
	for _, declaration := range ordered {
		target, relative, problem := loader.resolve(parentPath, declaration)
		if problem != nil {
			return problem
		}
		if loader.state[target] == 1 {
			return moduleIssue("S-4003", "无环模块依赖", relative, "检测到依赖回边", "移除循环导入", declaration.pos)
		}
		if loader.state[target] == 2 {
			continue
		}
		loader.state[target] = 1
		info, err := os.Lstat(target)
		if err != nil {
			return moduleIssue("S-4002", "可读取的 UTF-8 模块文件", relative, err.Error(), "确认文件存在且当前用户可读取", declaration.pos)
		}
		if !info.Mode().IsRegular() {
			return moduleIssue("S-4002", "普通 UTF-8 模块文件", relative, "目标不是普通文件", "使用普通 .serica 文件", declaration.pos)
		}
		data, err := os.ReadFile(target)
		if err != nil {
			return moduleIssue("S-4002", "可读取的 UTF-8 模块文件", relative, err.Error(), "确认文件存在且当前用户可读取", declaration.pos)
		}
		if !validUTF8(data) {
			return moduleIssue("S-4002", "有效 UTF-8 模块文件", relative, "模块包含无效 UTF-8", "将模块保存为有效 UTF-8", declaration.pos)
		}
		tokens, problem := lex(string(data))
		if problem != nil {
			problem.pos.file = relative
			return problem
		}
		setTokenFile(tokens, relative)
		module, problem := parseModule(tokens, string(data))
		if problem != nil {
			return problem
		}
		if problem := loader.loadImports(target, relative, module.imports); problem != nil {
			return problem
		}
		if problem := loader.addBindings(module.bindings, relative); problem != nil {
			return problem
		}
		if problem := loader.addFunctions(module.functions, relative); problem != nil {
			return problem
		}
		loader.state[target] = 2
	}
	return nil
}

func (loader *moduleLoader) resolve(parentPath string, declaration importDecl) (string, string, *issue) {
	if !validModulePath(declaration.path) {
		return "", "", moduleIssue("S-4001", "项目内相对 .serica 路径", declaration.path, "路径包含非法字符、绝对路径或错误扩展名", "使用以 .serica 结尾的 ASCII 相对路径", declaration.pos)
	}
	target := filepath.Clean(filepath.Join(filepath.Dir(parentPath), filepath.FromSlash(declaration.path)))
	relative, err := filepath.Rel(loader.rootDir, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(os.PathSeparator)) || filepath.IsAbs(relative) {
		return "", "", moduleIssue("S-4001", "留在入口项目目录内的模块路径", declaration.path, "路径越出项目目录", "将模块放在入口目录内", declaration.pos)
	}
	if symlink, err := containsSymlink(loader.rootDir, target); err != nil {
		return "", "", moduleIssue("S-4001", "不经过符号链接的模块路径", declaration.path, err.Error(), "改用项目内普通文件路径", declaration.pos)
	} else if symlink {
		return "", "", moduleIssue("S-4001", "不经过符号链接的模块路径", declaration.path, "路径包含符号链接", "改用项目内普通文件路径", declaration.pos)
	}
	if caseMismatch(loader.rootDir, relative) {
		return "", "", moduleIssue("S-4001", "大小写完全匹配的模块路径", declaration.path, "路径大小写与文件系统条目不一致", "使用磁盘上完全相同的路径大小写", declaration.pos)
	}
	return target, filepath.ToSlash(relative), nil
}

func validModulePath(path string) bool {
	if path == "" || !strings.HasSuffix(path, ".serica") || strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\\:") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" {
			return false
		}
		for _, character := range segment {
			if !(character >= 'A' && character <= 'Z' || character >= 'a' && character <= 'z' || character >= '0' && character <= '9' || character == '.' || character == '_' || character == '-') {
				return false
			}
		}
	}
	return true
}

func containsSymlink(root, target string) (bool, error) {
	relative, err := filepath.Rel(root, target)
	if err != nil {
		return false, err
	}
	current := root
	for _, segment := range strings.Split(relative, string(os.PathSeparator)) {
		if segment == "." || segment == "" {
			continue
		}
		current = filepath.Join(current, segment)
		info, err := os.Lstat(current)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return true, nil
		}
	}
	return false, nil
}

func caseMismatch(root, relative string) bool {
	current := root
	for _, segment := range strings.Split(relative, string(os.PathSeparator)) {
		entries, err := os.ReadDir(current)
		if err != nil {
			return false
		}
		exact, variant := false, false
		for _, entry := range entries {
			if entry.Name() == segment {
				exact = true
			} else if strings.EqualFold(entry.Name(), segment) {
				variant = true
			}
		}
		if !exact && variant {
			return true
		}
		if !exact {
			return false
		}
		current = filepath.Join(current, segment)
	}
	return false
}

func (loader *moduleLoader) addBindings(bindings []binding, source string) *issue {
	for _, item := range bindings {
		name := item.name.text
		if origin, exists := loader.origins[name]; exists && origin != source {
			return moduleIssue("S-4005", "模块闭包内唯一的常量名", name, "名称同时由 "+origin+" 与 "+source+" 导出", "重命名其中一个常量", item.name.pos)
		}
		if _, exists := loader.origins[name]; !exists {
			loader.origins[name] = source
		}
		loader.bindings = append(loader.bindings, item)
	}
	return nil
}

func (loader *moduleLoader) addFunctions(functions []functionDecl, source string) *issue {
	for _, function := range functions {
		name := function.name.text
		if origin, exists := loader.functionOrigins[name]; exists && origin != source {
			return moduleIssue("S-4006", "模块闭包中唯一的函数名", name, "名称同时由 "+origin+" 与 "+source+" 导出", "重命名其中一个函数", function.name.pos)
		}
		if _, exists := loader.functionOrigins[name]; !exists {
			loader.functionOrigins[name] = source
		}
		loader.functions = append(loader.functions, function)
	}
	return nil
}

func moduleIssue(code, expected, actual, reason, suggestion string, pos sourcePosition) *issue {
	if reason != "" {
		actual += "（" + reason + "）"
	}
	return &issue{code: code, expected: expected, actual: actual, suggestion: suggestion, pos: pos}
}
