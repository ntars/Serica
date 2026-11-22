// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import "fmt"

func isModuleRead(input value) bool {
	return input.kind == "调用" && (input.text == "readModule" || input.text == "readModuleResult") && len(input.args) == 4
}

func isModuleReadResult(input value) bool {
	return input.text == "readModuleResult" && isModuleRead(input)
}

func validateModuleRead(input value, environment map[string]constantValue) *issue {
	diagnostic := &issue{code: "S-3073", expected: input.text + "（项目目录，整数路径数组，起点，字节数）", actual: "模块读取参数无效", suggestion: "向项目目录传入静态路径或 argumentText，提供规范化路径字节的数组范围", pos: input.pos}
	if !isModuleRead(input) {
		return diagnostic
	}
	root, problem := constantType(input.args[0], environment)
	if problem != nil {
		return problem
	}
	if root == "字符串" {
		path, problem := evaluateString(input.args[0], 0, environment)
		if problem != nil {
			return problem
		}
		if path == "" {
			return diagnostic
		}
		for _, character := range path {
			if character == 0 {
				return diagnostic
			}
		}
	} else if !isRuntimeArgumentText(input.args[0]) {
		return diagnostic
	}
	if input.args[1].kind != "引用" || environment[input.args[1].text].kind != "运行时整数数组" {
		return diagnostic
	}
	for _, index := range input.args[2:] {
		kind, problem := constantType(index, environment)
		if problem != nil {
			return problem
		}
		if kind != "整数" && kind != "运行时整数" {
			return diagnostic
		}
	}
	return nil
}

// Darwin arm64: two F_GETPATH buffers, one NUL-terminated path, and struct stat.
const (
	modulePathBuffer  = 32
	moduleRootBuffer  = 1056
	moduleFileBuffer  = 2080
	moduleStatBuffer  = 3104
	moduleRootFD      = 16
	moduleFileFD      = 24
	moduleMappingSlot = 3264
	moduleScratchSize = 3280
	moduleContentSize = 16 * 1024 * 1024
	modulePathFailure = 41
	moduleReadFailure = 42
)

// Attach this part of the checked runtime to a specific failure status.
func (c *windowCode) emitModuleFailure(start int, status uint16, failures []int) []int {
	resume := c.unconditionalPlaceholder()
	for _, patch := range c.failurePatches[start:] {
		c.patchConditional(patch, len(c.words))
	}
	c.failurePatches = c.failurePatches[:start]
	c.movImmediate(0, status)
	failures = append(failures, c.unconditionalPlaceholder())
	c.words[resume] = branch(len(c.words) - resume)
	return failures
}

func (c *windowCode) emitModuleRead(input value, constants map[string]constantValue, slots map[string]uint32) error {
	array, exists := c.arraySlots[input.args[1].text]
	if !exists || array.text {
		return fmt.Errorf("readModule 需要运行时整数路径数组")
	}
	pathFailures := len(c.failurePatches)
	failures := []int{}
	c.words = append(c.words, subImmediate(31, 31, moduleScratchSize), storeIntegerSlot(31, 31, moduleMappingSlot))
	if err := c.emitRegisterImmediate(0, -1); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, moduleRootFD), storeIntegerSlot(0, 31, moduleFileFD))
	if err := c.emitRuntimeFilePath(input.args[0], constants, slots, 26, 27); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(27, 31, 0))
	if err := c.emitRuntimeInteger(input.args[2], constants, slots); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(0, 31, 8))
	if err := c.emitRuntimeInteger(input.args[3], constants, slots); err != nil {
		return err
	}
	c.movRegister(23, 0)
	c.words = append(c.words, loadIntegerSlot(24, 31, 8), loadIntegerSlot(25, 19, array.length), compareImmediate(24, 0))
	c.branchToRuntimeFailure(11) // signed lt: negative start
	c.words = append(c.words, compareRegister(24, 25))
	c.branchToRuntimeFailure(12) // signed gt: start beyond the array
	c.words = append(c.words, subtractRegister(25, 25, 24), compareImmediate(23, 7))
	c.branchToRuntimeFailure(11) // at least ".serica"
	c.words = append(c.words, compareImmediate(23, 1024))
	c.branchToRuntimeFailure(10) // include room for the trailing NUL
	c.words = append(c.words, compareRegister(23, 25))
	c.branchToRuntimeFailure(12)
	c.words = append(c.words, loadIntegerSlot(21, 19, array.data), addRegisterShifted(21, 21, 24, 3))
	c.emitModulePathBytes()
	c.words = append(c.words, loadIntegerSlot(0, 31, 0))
	// O_DIRECTORY | O_NOFOLLOW_ANY, including all root path components.
	if err := c.emitRegisterImmediate(1, 0x20100000); err != nil {
		return err
	}
	c.call(c.fileOpenIndex)
	c.words = append(c.words, compareWordImmediate(0, 0))
	c.branchToRuntimeFailure(11)
	c.movRegister(25, 0) // project descriptor
	c.words = append(c.words, storeIntegerSlot(25, 31, moduleRootFD))
	c.emitModuleGetPath(25, moduleRootBuffer)
	failures = c.emitModuleFailure(pathFailures, modulePathFailure, failures)

	c.movRegister(0, 25)
	c.words = append(c.words, addImmediate(1, 31, modulePathBuffer))
	// O_NOFOLLOW_ANY | O_NONBLOCK on openat, relative to the held directory.
	if err := c.emitRegisterImmediate(2, 0x20000004); err != nil {
		return err
	}
	c.call(c.moduleOpenIndex)
	c.words = append(c.words, compareWordImmediate(0, 0))
	opened := c.conditionalPlaceholder(10)
	c.call(c.moduleErrorIndex)
	c.words = append(c.words, loadByte(0, 0), compareImmediate(0, 62)) // Darwin ELOOP
	c.movImmediate(0, moduleReadFailure)
	unreadable := c.conditionalPlaceholder(1)
	c.movImmediate(0, modulePathFailure)
	c.patchConditional(unreadable, len(c.words))
	failures = append(failures, c.unconditionalPlaceholder())
	c.patchConditional(opened, len(c.words))
	c.movRegister(22, 0) // module descriptor
	c.words = append(c.words, storeIntegerSlot(22, 31, moduleFileFD))

	readFailures := len(c.failurePatches)
	c.words = append(c.words, addImmediate(1, 31, moduleStatBuffer))
	c.call(c.moduleStatIndex)
	c.words = append(c.words, compareWordImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	// Darwin arm64 struct stat has st_mode at offset 4. Check S_IFREG.
	c.words = append(c.words, addImmediate(4, 31, moduleStatBuffer+5), loadByte(4, 4), movz(5, 240), andRegister(4, 4, 5), compareImmediate(4, 128))
	c.branchToRuntimeFailure(1)
	failures = c.emitModuleFailure(readFailures, moduleReadFailure, failures)

	c.emitModuleGetPath(22, moduleFileBuffer)
	c.emitModulePathIdentity()
	failures = c.emitModuleFailure(pathFailures, modulePathFailure, failures)
	c.movRegister(0, 25)
	c.call(c.fileCloseIndex)
	c.words = append(c.words, compareWordImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	if err := c.emitRegisterImmediate(1, -1); err != nil {
		return err
	}
	c.words = append(c.words, storeIntegerSlot(1, 31, moduleRootFD))
	if err := c.emitRuntimeFileContents(true); err != nil {
		return err
	}
	failures = c.emitModuleFailure(readFailures, moduleReadFailure, failures)
	c.words = append(c.words, addImmediate(31, 31, moduleScratchSize))
	if isModuleReadResult(input) {
		c.movImmediate(0, 0)
		if err := c.emitRegisterImmediate(1, moduleContentSize); err != nil {
			return err
		}
		c.movRegister(2, 21)
		c.movRegister(3, 28)
	}
	done := c.unconditionalPlaceholder()
	for _, failure := range failures {
		c.words[failure] = branch(len(c.words) - failure)
	}
	if isModuleReadResult(input) {
		if err := c.emitModuleReadRecovery(); err != nil {
			return err
		}
	} else {
		c.call(c.exitIndex)
	}
	c.words[done] = branch(len(c.words) - done)
	return nil
}

func (c *windowCode) emitModuleReadRecovery() error {
	c.movRegister(27, 0)
	for _, slot := range []uint32{moduleFileFD, moduleRootFD} {
		c.words = append(c.words, loadIntegerSlot(0, 31, slot), compareWordImmediate(0, 0))
		absent := c.conditionalPlaceholder(11)
		c.call(c.fileCloseIndex)
		c.patchConditional(absent, len(c.words))
	}
	c.words = append(c.words, loadIntegerSlot(0, 31, moduleMappingSlot), compareImmediate(0, 0))
	unmapped := c.conditionalPlaceholder(0)
	if err := c.emitRegisterImmediate(1, moduleContentSize); err != nil {
		return err
	}
	c.call(c.arrayUnmapIndex)
	c.patchConditional(unmapped, len(c.words))
	c.words = append(c.words, addImmediate(31, 31, moduleScratchSize), movz(0, 1), movz(2, 0), movz(3, 0))
	c.movRegister(1, 27)
	return nil
}

func (c *windowCode) emitModuleReadResult(input value, constants map[string]constantValue, slots map[string]uint32, result resultSlot) error {
	if !result.runtimeText {
		return fmt.Errorf("readModuleResult 需要运行时文本结果变量")
	}
	if err := c.emitModuleRead(input, constants, slots); err != nil {
		return err
	}
	// Release the previous owned mapping after argument evaluation and the read.
	c.words = append(c.words, subImmediate(31, 31, 32), storeIntegerSlot(0, 31, 0), storeIntegerSlot(1, 31, 8), storeIntegerSlot(2, 31, 16), storeIntegerSlot(3, 31, 24))
	c.words = append(c.words, loadIntegerSlot(0, 19, result.pointer), compareImmediate(0, 0))
	absent := c.conditionalPlaceholder(0)
	if err := c.emitRegisterImmediate(1, moduleContentSize); err != nil {
		return err
	}
	c.call(c.arrayUnmapIndex)
	c.words = append(c.words, compareWordImmediate(0, 0))
	c.branchToRuntimeFailure(1)
	c.patchConditional(absent, len(c.words))
	c.words = append(c.words, loadIntegerSlot(0, 31, 0), loadIntegerSlot(1, 31, 8), loadIntegerSlot(2, 31, 16), loadIntegerSlot(3, 31, 24), addImmediate(31, 31, 32))
	return nil
}

// Copy the checked integer range into a C path; accept only canonical ASCII components.
func (c *windowCode) emitModulePathBytes() {
	if _, exists := c.dataLabels["modulePathClasses"]; !exists {
		classes := make([]byte, 128)
		for _, item := range "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789_-" {
			classes[item] = 1
		}
		classes['.'], classes['/'] = 2, 3
		c.addData("modulePathClasses", classes, 1)
		c.addData("modulePathSuffix", []byte(".serica"), 1)
	}
	c.address("modulePathClasses", 8)
	c.words = append(c.words, addImmediate(22, 31, modulePathBuffer), movz(24, 0), movz(25, 0))
	loop := len(c.words)
	c.words = append(c.words, compareRegister(24, 23))
	finished := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadIndexedRegister(4, 21, 24), compareImmediate(4, 128))
	c.branchToRuntimeFailure(2) // unsigned hs also rejects negative byte values
	c.words = append(c.words, addRegister(5, 8, 4), loadByte(5, 5), compareImmediate(5, 0))
	c.branchToRuntimeFailure(0)
	c.words = append(c.words, compareImmediate(5, 3))
	notSlash := c.conditionalPlaceholder(1)
	c.words = append(c.words, compareImmediate(25, 3))
	c.branchToRuntimeFailure(1)
	c.movImmediate(25, 0)
	saveSlash := c.unconditionalPlaceholder()
	c.patchConditional(notSlash, len(c.words))
	c.words = append(c.words, compareImmediate(25, 3))
	saveValid := c.conditionalPlaceholder(0)
	c.words = append(c.words, compareImmediate(5, 2))
	dot := c.conditionalPlaceholder(0)
	c.movImmediate(25, 3)
	saveName := c.unconditionalPlaceholder()
	c.patchConditional(dot, len(c.words))
	c.words = append(c.words, addImmediate(25, 25, 1))
	for _, patch := range []int{saveSlash, saveName} {
		c.words[patch] = branch(len(c.words) - patch)
	}
	c.patchConditional(saveValid, len(c.words))
	c.words = append(c.words, addRegister(6, 22, 24), storeByte(4, 6), addImmediate(24, 24, 1))
	c.words = append(c.words, branch(loop-len(c.words)))
	c.patchConditional(finished, len(c.words))
	c.words = append(c.words, compareImmediate(25, 3))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, addRegister(6, 22, 23), storeByte(31, 6), subImmediate(6, 6, 7))
	c.address("modulePathSuffix", 7)
	for range ".serica" {
		c.words = append(c.words, loadByte(4, 6), loadByte(5, 7), compareRegister(4, 5))
		c.branchToRuntimeFailure(1)
		c.words = append(c.words, addImmediate(6, 6, 1), addImmediate(7, 7, 1))
	}
}

func (c *windowCode) emitModuleGetPath(descriptor, buffer uint32) {
	// Darwin passes fcntl's variadic third argument on the stack.
	c.words = append(c.words, addImmediate(2, 31, buffer), subImmediate(31, 31, 16), storeIntegerSlot(2, 31, 0))
	c.movRegister(0, descriptor)
	c.movImmediate(1, 50) // F_GETPATH
	c.call(c.moduleControlIndex)
	c.words = append(c.words, addImmediate(31, 31, 16), compareWordImmediate(0, 0))
	c.branchToRuntimeFailure(1)
}

// Require the opened file's canonical spelling to equal root + "/" + the request.
func (c *windowCode) emitModulePathIdentity() {
	c.words = append(c.words, addImmediate(4, 31, moduleRootBuffer), addImmediate(5, 31, moduleFileBuffer))
	root := len(c.words)
	c.words = append(c.words, loadByte(6, 4), compareImmediate(6, 0))
	rootDone := c.conditionalPlaceholder(0)
	c.words = append(c.words, loadByte(7, 5), compareRegister(6, 7))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, addImmediate(4, 4, 1), addImmediate(5, 5, 1))
	c.words = append(c.words, branch(root-len(c.words)))
	c.patchConditional(rootDone, len(c.words))
	c.words = append(c.words, subImmediate(4, 4, 1), loadByte(6, 4), compareImmediate(6, 47))
	slashPresent := c.conditionalPlaceholder(0) // project directory itself is "/"
	c.words = append(c.words, loadByte(6, 5), compareImmediate(6, 47))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, addImmediate(5, 5, 1))
	c.patchConditional(slashPresent, len(c.words))
	c.words = append(c.words, addImmediate(4, 31, modulePathBuffer))
	relative := len(c.words)
	c.words = append(c.words, loadByte(6, 4), loadByte(7, 5), compareRegister(6, 7))
	c.branchToRuntimeFailure(1)
	c.words = append(c.words, addImmediate(4, 4, 1), addImmediate(5, 5, 1), compareImmediate(6, 0))
	c.patchConditional(c.conditionalPlaceholder(1), relative)
}
