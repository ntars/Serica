// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	imageBase   = uint64(0x100000000)
	pageSize    = uint64(0x4000)
	codeFileOff = 2048
	dataFileOff = 0x4000
	linkFileOff = 0x8000
)

func writeMachO(path, title, text string) error {
	if runtime.GOOS != "darwin" || runtime.GOARCH != "arm64" {
		return fmt.Errorf("当前后端只支持 macOS arm64")
	}
	if strings.IndexByte(title, 0) >= 0 || strings.IndexByte(text, 0) >= 0 {
		return fmt.Errorf("窗口文本暂不支持 U+0000")
	}
	image, err := makeWindowMachO(title, text)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, image, 0755); err != nil {
		return err
	}
	if err := os.Chmod(path, 0755); err != nil {
		return err
	}
	output, err := exec.Command("codesign", "-s", "-", "--force", "--timestamp=none", path).CombinedOutput()
	if err != nil {
		return fmt.Errorf("系统 ad-hoc 签名失败：%w：%s", err, output)
	}
	return nil
}

func writeSegment(image []byte, at int, name string, commandSize uint32, vmAddress, vmSize, fileOffset, fileSize uint64, maxProtection, protection, sections uint32) {
	put32(image, at, 0x19)
	put32(image, at+4, commandSize)
	putName(image, at+8, name)
	put64(image, at+24, vmAddress)
	put64(image, at+32, vmSize)
	put64(image, at+40, fileOffset)
	put64(image, at+48, fileSize)
	put32(image, at+56, maxProtection)
	put32(image, at+60, protection)
	put32(image, at+64, sections)
}

func writeSection(image []byte, at int, section, segment string, address, size uint64, offset, alignment, flags uint32) {
	putName(image, at, section)
	putName(image, at+16, segment)
	put64(image, at+32, address)
	put64(image, at+40, size)
	put32(image, at+48, offset)
	put32(image, at+52, alignment)
	put32(image, at+64, flags)
}

func writeDyldInfo(image []byte, at, bindOffset, bindSize int) {
	put32(image, at, 0x80000022)
	put32(image, at+4, 48)
	put32(image, at+16, uint32(bindOffset))
	put32(image, at+20, uint32(bindSize))
}

func writeDylinker(image []byte, at int) {
	put32(image, at, 0xe)
	put32(image, at+4, 32)
	put32(image, at+8, 12)
	copy(image[at+12:], "/usr/lib/dyld\x00")
}

func writeBuildVersion(image []byte, at int) {
	put32(image, at, 0x32)
	put32(image, at+4, 24)
	put32(image, at+8, 1)
	put32(image, at+12, 0x000b0000)
	put32(image, at+16, 0x001b0000)
}

func writeMain(image []byte, at int, entryOffset uint64) {
	put32(image, at, 0x80000028)
	put32(image, at+4, 24)
	put64(image, at+8, entryOffset)
}

func writeUUID(image []byte, at int, input []byte) {
	put32(image, at, 0x1b)
	put32(image, at+4, 24)
	sum := sha256.Sum256(input)
	copy(image[at+8:at+24], sum[:16])
}

func writeNlist(image []byte, at int, stringIndex uint32) {
	put32(image, at, stringIndex)
	image[at+4] = 1
}

func movz(register uint32, immediate uint16) uint32 {
	return 0xd2800000 | uint32(immediate)<<5 | register
}

func adr(register uint32, displacement int64) uint32 {
	imm := uint32(displacement) & 0x1fffff
	return 0x10000000 | (imm&3)<<29 | ((imm>>2)&0x7ffff)<<5 | register
}

func adrp(register uint32, target, pc uint64) uint32 {
	delta := int64((target&^0xfff)-(pc&^0xfff)) / 4096
	imm := uint32(delta) & 0x1fffff
	return 0x90000000 | (imm&3)<<29 | ((imm>>2)&0x7ffff)<<5 | register
}

func loadPointer(destination, base, offset uint32) uint32 {
	return 0xf9400000 | (offset/8)<<10 | base<<5 | destination
}

func alignOffset(value, alignment int) int { return (value + alignment - 1) &^ (alignment - 1) }

func put32(image []byte, at int, value uint32)   { binary.LittleEndian.PutUint32(image[at:], value) }
func put64(image []byte, at int, value uint64)   { binary.LittleEndian.PutUint64(image[at:], value) }
func putName(image []byte, at int, value string) { copy(image[at:at+16], value) }
