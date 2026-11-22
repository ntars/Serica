// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"encoding/binary"
	"fmt"
	"math"
)

type importedSymbol struct {
	name    string
	ordinal uint8
}
type importCallPatch struct {
	word, index int
}
type addressReference struct {
	word     int
	label    string
	register uint32
	code     bool
}
type arraySlot struct {
	data, length, capacity uint32
	text                   bool
}
type mapSlot struct{ data, length, capacity uint32 }
type resultSlot struct {
	tag, value, pointer, length uint32
	text, runtimeText           bool
}
type loopTarget struct {
	continueAt int
	breaks     []int
}
type runtimeCallPatch struct {
	position int
	name     string
}
type windowCode struct {
	words                 []uint32
	constants             []byte
	dataLabels            map[string]int
	codeLabels            map[string]int
	refs                  []addressReference
	importCallPatches     []importCallPatch
	exitIndex             int
	failurePatches        []conditionalPatch
	arraySlots            map[string]arraySlot
	mapSlots              map[string]mapSlot
	resultSlots           map[string]resultSlot
	arrayMapIndex         int
	arrayUnmapIndex       int
	writeIndex            int
	fileOpenIndex         int
	moduleOpenIndex       int
	moduleControlIndex    int
	moduleStatIndex       int
	moduleErrorIndex      int
	fileCreateIndex       int
	fileCloseIndex        int
	fileReadIndex         int
	textBuffers           map[string]uint32
	textLengths           map[string]uint32
	textCapacities        map[string]uint32
	loopTargets           []loopTarget
	usesArguments         bool
	argumentCountOffset   uint32
	argumentVectorOffset  uint32
	temporaryBufferOffset uint32
	slotCount             uint32
	runtimeFunctions      map[string]functionDecl
	runtimeFunctionOrder  []string
	runtimeFunctionAt     map[string]int
	runtimeCallPatches    []runtimeCallPatch
}

func makeWindowMachO(title, text string) ([]byte, error) {
	imports := []importedSymbol{
		{"_objc_autoreleasePoolPush", 2},
		{"_objc_autoreleasePoolPop", 2},
		{"_objc_getClass", 2},
		{"_sel_registerName", 2},
		{"_objc_msgSend", 2},
		{"_objc_allocateClassPair", 2},
		{"_class_addMethod", 2},
		{"_objc_registerClassPair", 2},
		{"_exit", 1},
	}
	c := windowCode{dataLabels: map[string]int{}, codeLabels: map[string]int{}}
	c.emitWindow(title, text)
	constOffset := alignOffset(codeFileOff+len(c.words)*4, 8)
	if constOffset+len(c.constants) > int(pageSize) {
		return nil, fmt.Errorf("标题与文本超过 P0b 窗口产物上限")
	}
	if err := c.resolve(uint64(constOffset), dataFileOff); err != nil {
		return nil, err
	}
	bind := makeWindowBinds(imports)
	symbolOffset := alignOffset(linkFileOff+len(bind), 8)
	stringTable := []byte{0}
	stringIndices := []uint32{}
	for _, item := range imports {
		stringIndices = append(stringIndices, uint32(len(stringTable)))
		stringTable = append(stringTable, []byte(item.name+"\x00")...)
	}
	stringOffset := symbolOffset + 16*len(imports)
	fileEnd := stringOffset + len(stringTable)
	image := make([]byte, fileEnd)

	commandAt, commandCount := 32, 0
	writeSegment(image, commandAt, "__PAGEZERO", 72, 0, 0x100000000, 0, 0, 0, 0, 0)
	commandAt += 72
	textCommand := commandAt
	writeSegment(image, textCommand, "__TEXT", 232, imageBase, pageSize, 0, pageSize, 5, 5, 2)
	writeSection(image, textCommand+72, "__text", "__TEXT", imageBase+uint64(codeFileOff), uint64(len(c.words)*4), uint32(codeFileOff), 2, 0x80000400)
	writeSection(image, textCommand+152, "__const", "__TEXT", imageBase+uint64(constOffset), uint64(len(c.constants)), uint32(constOffset), 3, 0)
	commandAt += 232
	dataSize := uint64(len(imports) * 8)
	dataCommand := commandAt
	writeSegment(image, dataCommand, "__DATA", 152, imageBase+pageSize, pageSize, pageSize, pageSize, 3, 3, 1)
	writeSection(image, dataCommand+72, "__data", "__DATA", imageBase+pageSize, dataSize, dataFileOff, 3, 0)
	commandAt += 152
	linkCommand := commandAt
	writeSegment(image, linkCommand, "__LINKEDIT", 72, imageBase+2*pageSize, uint64(alignOffset(fileEnd-linkFileOff, int(pageSize))), linkFileOff, uint64(fileEnd-linkFileOff), 1, 1, 0)
	commandAt += 72
	writeDyldInfo(image, commandAt, linkFileOff, len(bind))
	commandAt += 48
	writeDylinker(image, commandAt)
	commandAt += 32
	for _, library := range []string{
		"/usr/lib/libSystem.B.dylib",
		"/usr/lib/libobjc.A.dylib",
		"/System/Library/Frameworks/Foundation.framework/Versions/C/Foundation",
		"/System/Library/Frameworks/AppKit.framework/Versions/C/AppKit",
	} {
		size := writeDylibPath(image, commandAt, library)
		commandAt += size
	}
	writeBuildVersion(image, commandAt)
	commandAt += 24
	writeMain(image, commandAt, uint64(codeFileOff))
	commandAt += 24
	writeUUID(image, commandAt, []byte(title+"\x00"+text))
	commandAt += 24
	writeSymbolTable(image, commandAt, symbolOffset, stringOffset, len(imports), len(stringTable))
	commandAt += 24
	writeDynamicSymbolTable(image, commandAt, len(imports))
	commandAt += 80
	commandCount = 15
	if commandAt > codeFileOff {
		return nil, fmt.Errorf("Mach-O 加载命令区超出代码起始偏移")
	}
	writeDynamicHeader(image, commandCount, commandAt-32)
	copy(image[codeFileOff:], wordsBytes(c.words))
	copy(image[constOffset:], c.constants)
	copy(image[dataFileOff:], make([]byte, len(imports)*8))
	copy(image[linkFileOff:], bind)
	for index, stringIndex := range stringIndices {
		writeNlist(image, symbolOffset+16*index, stringIndex)
	}
	copy(image[stringOffset:], stringTable)
	return image, nil
}

func (c *windowCode) emitWindow(title, text string) {
	c.addCString("class:NSApplication", "NSApplication")
	c.call(0)
	c.movRegister(19, 0)
	c.loadClass("NSApplication", 20)
	c.message(20, "sharedApplication")
	c.call(4)
	c.movRegister(20, 0)
	c.message(20, "setActivationPolicy:")
	c.movImmediate(2, 0)
	c.call(4)

	c.loadClass("NSWindow", 21)
	c.message(21, "alloc")
	c.call(4)
	c.movRegister(21, 0)
	c.addRectangle("windowRect", 0, 0, 600, 240)
	c.message(21, "initWithContentRect:styleMask:backing:defer:")
	c.loadRectangleRegisters("windowRect")
	c.movImmediate(2, 11)
	c.movImmediate(3, 2)
	c.movImmediate(4, 0)
	c.call(4)
	c.movRegister(21, 0)

	c.loadClass("NSString", 22)
	c.message(22, "stringWithUTF8String:")
	c.loadCString("windowTitle", title)
	c.address("windowTitle", 2)
	c.call(4)
	c.movRegister(22, 0)
	c.message(21, "setTitle:")
	c.movRegister(2, 22)
	c.call(4)

	c.loadClass("NSTextView", 23)
	c.message(23, "alloc")
	c.call(4)
	c.movRegister(23, 0)
	c.addRectangle("textRect", 20, 20, 560, 190)
	c.message(23, "initWithFrame:")
	c.loadRectangleRegisters("textRect")
	c.call(4)
	c.movRegister(23, 0)
	c.loadClass("NSString", 22)
	c.message(22, "stringWithUTF8String:")
	c.loadCString("initialText", text)
	c.address("initialText", 2)
	c.call(4)
	c.movRegister(22, 0)
	c.message(23, "setString:")
	c.movRegister(2, 22)
	c.call(4)
	c.message(23, "setEditable:")
	c.movImmediate(2, 1)
	c.call(4)
	c.message(23, "setSelectable:")
	c.movImmediate(2, 1)
	c.call(4)
	c.message(21, "setContentView:")
	c.movRegister(2, 23)
	c.call(4)
	c.message(21, "center")
	c.call(4)
	c.message(21, "makeKeyAndOrderFront:")
	c.movImmediate(2, 0)
	c.call(4)
	c.message(20, "activateIgnoringOtherApps:")
	c.movImmediate(2, 1)
	c.call(4)

	c.installCloseDelegate()
	c.message(20, "run")
	c.call(4)
	c.movRegister(0, 19)
	c.call(1)
	c.movImmediate(0, 0)
	c.call(8)
	c.words = append(c.words, 0xd4200000)
	c.codeLabels["delegateCallback"] = len(c.words)
	c.words = append(c.words, movz(0, 1), 0xd65f03c0)
}

func (c *windowCode) installCloseDelegate() {
	c.loadClass("NSObject", 24)
	c.movRegister(0, 24)
	c.loadCString("delegateClassName", "SericaP0bWindowDelegate")
	c.address("delegateClassName", 1)
	c.movImmediate(2, 0)
	c.call(5)
	c.movRegister(24, 0)
	c.loadCString("closeSelector", "applicationShouldTerminateAfterLastWindowClosed:")
	c.address("closeSelector", 0)
	c.call(3)
	c.movRegister(1, 0)
	c.movRegister(0, 24)
	c.addressCode("delegateCallback", 2)
	c.loadCString("closeTypes", "B@:@")
	c.address("closeTypes", 3)
	c.call(6)
	c.movRegister(0, 24)
	c.call(7)
	c.message(24, "new")
	c.call(4)
	c.movRegister(25, 0)
	c.message(20, "setDelegate:")
	c.movRegister(2, 25)
	c.call(4)
}

func (c *windowCode) loadClass(name string, register uint32) {
	label := "class:" + name
	c.loadCString(label, name)
	c.address(label, 0)
	c.call(2)
	if register != 0 {
		c.movRegister(register, 0)
	}
}

func (c *windowCode) message(receiver uint32, selector string) {
	label := "selector:" + selector
	c.loadCString(label, selector)
	c.address(label, 0)
	c.call(3)
	c.movRegister(1, 0)
	c.movRegister(0, receiver)
}

func (c *windowCode) call(index int) {
	c.importCallPatches = append(c.importCallPatches, importCallPatch{word: len(c.words), index: index})
	c.words = append(c.words, 0, loadPointer(16, 16, uint32(index*8)), 0xd63f0200)
}

func (c *windowCode) movRegister(destination, source uint32) {
	c.words = append(c.words, 0xaa0003e0|source<<16|destination)
}

func (c *windowCode) movImmediate(register uint32, value uint16) {
	c.words = append(c.words, movz(register, value))
}

func (c *windowCode) loadCString(label, text string) {
	c.addCString(label, text)
}

func (c *windowCode) addCString(label, text string) {
	if _, exists := c.dataLabels[label]; exists {
		return
	}
	c.addData(label, append([]byte(text), 0), 1)
}

func (c *windowCode) addRectangle(label string, x, y, width, height float64) {
	data := make([]byte, 32)
	for i, value := range []float64{x, y, width, height} {
		binary.LittleEndian.PutUint64(data[i*8:], math.Float64bits(value))
	}
	c.addData(label, data, 8)
}

func (c *windowCode) addData(label string, data []byte, alignment int) {
	if _, exists := c.dataLabels[label]; exists {
		return
	}
	c.constants = append(c.constants, make([]byte, alignOffset(len(c.constants), alignment)-len(c.constants))...)
	c.dataLabels[label] = len(c.constants)
	c.constants = append(c.constants, data...)
}

func (c *windowCode) address(label string, register uint32) {
	word := len(c.words)
	c.words = append(c.words, 0x10000000|register)
	c.refs = append(c.refs, addressReference{word: word, label: label, register: register})
}

func (c *windowCode) addressCode(label string, register uint32) {
	word := len(c.words)
	c.words = append(c.words, 0x10000000|register)
	c.refs = append(c.refs, addressReference{word: word, label: label, register: register, code: true})
}

func (c *windowCode) loadRectangleRegisters(label string) {
	c.address(label, 16)
	for register := uint32(0); register < 4; register++ {
		c.words = append(c.words, loadDouble(register, 16, register*8))
	}
}

func (c *windowCode) resolve(constOffset, importOffset uint64) error {
	for _, patch := range c.importCallPatches {
		if patch.index < 0 || patch.index*8 >= 4096 {
			return fmt.Errorf("系统导入槽超出范围：%d", patch.index)
		}
		pc := imageBase + uint64(codeFileOff+patch.word*4)
		c.words[patch.word] = adrp(16, imageBase+importOffset, pc)
	}
	for _, ref := range c.refs {
		target := uint64(0)
		if ref.code {
			codeIndex, ok := c.codeLabels[ref.label]
			if !ok {
				return fmt.Errorf("代码地址未定义：%s", ref.label)
			}
			target = imageBase + uint64(codeFileOff+codeIndex*4)
		} else {
			dataIndex, ok := c.dataLabels[ref.label]
			if !ok {
				return fmt.Errorf("常量地址未定义：%s", ref.label)
			}
			target = imageBase + constOffset + uint64(dataIndex)
		}
		pc := imageBase + uint64(codeFileOff+ref.word*4)
		displacement := int64(target) - int64(pc)
		if displacement < -(1<<20) || displacement >= 1<<20 {
			return fmt.Errorf("ADR 地址偏移超出范围：%s", ref.label)
		}
		c.words[ref.word] = adr(ref.register, displacement)
	}
	return nil
}

func makeWindowBinds(imports []importedSymbol) []byte {
	program := []byte{}
	for index, item := range imports {
		program = append(program, 0x10|item.ordinal, 0x40)
		program = append(program, []byte(item.name+"\x00")...)
		program = append(program, 0x51, 0x72)
		program = appendULEB(program, uint64(index*8))
		program = append(program, 0x90)
	}
	return append(program, 0)
}

func appendULEB(output []byte, value uint64) []byte {
	for value >= 0x80 {
		output = append(output, byte(value)|0x80)
		value >>= 7
	}
	return append(output, byte(value))
}

func writeDynamicHeader(image []byte, commands, commandBytes int) {
	put32(image, 0, 0xfeedfacf)
	put32(image, 4, 0x0100000c)
	put32(image, 8, 0)
	put32(image, 12, 2)
	put32(image, 16, uint32(commands))
	put32(image, 20, uint32(commandBytes))
	put32(image, 24, 0x200084)
}

func writeDylibPath(image []byte, at int, path string) int {
	size := alignOffset(24+len(path)+1, 8)
	put32(image, at, 0xc)
	put32(image, at+4, uint32(size))
	put32(image, at+8, 24)
	put32(image, at+16, 0x00010000)
	put32(image, at+20, 0x00010000)
	copy(image[at+24:], path+"\x00")
	return size
}

func writeSymbolTable(image []byte, at, symbolOffset, stringOffset, count, stringSize int) {
	put32(image, at, 2)
	put32(image, at+4, 24)
	put32(image, at+8, uint32(symbolOffset))
	put32(image, at+12, uint32(count))
	put32(image, at+16, uint32(stringOffset))
	put32(image, at+20, uint32(stringSize))
}

func writeDynamicSymbolTable(image []byte, at, undefinedCount int) {
	put32(image, at, 0xb)
	put32(image, at+4, 80)
	put32(image, at+28, uint32(undefinedCount))
}

func wordsBytes(words []uint32) []byte {
	data := make([]byte, len(words)*4)
	for index, word := range words {
		binary.LittleEndian.PutUint32(data[index*4:], word)
	}
	return data
}

func loadDouble(destination, base, offset uint32) uint32 {
	return 0xfd400000 | (offset/8)<<10 | base<<5 | destination
}
