// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

/*
#cgo LDFLAGS: -framework Cocoa -framework CoreText -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
static int serica_is_nfc(const unsigned char *src, long size) {
  CFStringRef input = CFStringCreateWithBytes(kCFAllocatorDefault, src, size, kCFStringEncodingUTF8, false);
  if (input == NULL) return 0;
  CFMutableStringRef copy = CFStringCreateMutableCopy(kCFAllocatorDefault, 0, input);
  CFRelease(input);
  if (copy == NULL) return 0;
  CFStringNormalize(copy, kCFStringNormalizationFormC);
  CFIndex cap = CFStringGetMaximumSizeForEncoding(CFStringGetLength(copy), kCFStringEncodingUTF8);
  char *out = malloc((size_t)cap + 1);
  if (out == NULL) { CFRelease(copy); return 0; }
  CFIndex used = 0;
  CFStringGetBytes(copy, CFRangeMake(0, CFStringGetLength(copy)), kCFStringEncodingUTF8, 0, false, (UInt8 *)out, cap, &used);
  int same = used == size && memcmp(src, out, (size_t)size) == 0;
  free(out);
  CFRelease(copy);
  return same;
}
void serica_show_window(void);
*/
import "C"

import (
  "encoding/binary"
  "fmt"
  "os"
  "os/exec"
  "path/filepath"
  "runtime"
  "strings"
  "unicode"
)

const fixture = "bootstrap/prototype/lexical.serica"

func main() {
  runtime.LockOSThread()
  if err := checkLexical(); err != nil { fail(err) }
  fmt.Println("3.8.1 中文词法：通过")
  if len(os.Args) > 1 && os.Args[1] == "--lexical-only" { return }
  if len(os.Args) > 1 && os.Args[1] == "--window-only" {
    C.serica_show_window()
    return
  }
  if err := checkCodegen(); err != nil { fail(err) }
  fmt.Println("3.8.2 直接代码生成：通过")
  if len(os.Args) > 1 && os.Args[1] == "--window" {
    C.serica_show_window()
  }
}

func fail(err error) {
  fmt.Fprintln(os.Stderr, "原型失败：", err)
  os.Exit(1)
}

func checkLexical() error {
  data, err := os.ReadFile(fixture)
  if err != nil { return err }
  ptr := C.CBytes(data)
  if C.serica_is_nfc((*C.uchar)(ptr), C.long(len(data))) == 0 {
    C.free(ptr)
    return fmt.Errorf("源码不是 NFC")
  }
  C.free(ptr)
  tokens, comments, err := tokenize(string(data))
  if err != nil { return err }
  joined := "|" + strings.Join(tokens, "|") + "|"
  for _, want := range []string{"计算总额", "订单数", "，", "。", "（", "）", "『", "』", "「", "」", "【", "】", "+", "-", "*", "/", "=", "<", ">", "\\"} {
    if !strings.Contains(joined, "|"+want+"|") { return fmt.Errorf("未切出符号 %q", want) }
  }
  if !comments { return fmt.Errorf("未识别 # 注释") }
  _, _, err = tokenize("计算总额,订单数")
  if err == nil || !strings.Contains(err.Error(), "码位列 5／显示列 9") {
    return fmt.Errorf("半角逗号诊断未给出双列号：%v", err)
  }
  return nil
}

func tokenize(src string) ([]string, bool, error) {
  runes := []rune(src)
  tokens := []string{}
  commentSeen := false
  for i := 0; i < len(runes); {
    r := runes[i]
    if unicode.IsSpace(r) { i++; continue }
    if r == '#' {
      commentSeen = true
      for i < len(runes) && runes[i] != '\n' { i++ }
      continue
    }
    if r == '『' || r == '「' {
      end := '』'
      if r == '「' { end = '」' }
      i++
      for i < len(runes) && runes[i] != end {
        if r == '『' && runes[i] == '\\' && i+1 < len(runes) {
          tokens = append(tokens, "\\")
          i++
        }
        i++
      }
      if i == len(runes) { return nil, commentSeen, fmt.Errorf("字符串未闭合") }
      tokens = append(tokens, string(r), string(end))
      i++
      continue
    }
    repair := map[rune]string{',': "，", ';': "。", ':': "：", '(': "（", ')': "）"}
    if want, ok := repair[r]; ok {
      line, codeCol, displayCol := columns(runes, i)
      return nil, commentSeen, fmt.Errorf("第 %d 行码位列 %d／显示列 %d：应为「%s」", line, codeCol, displayCol, want)
    }
    if strings.ContainsRune("，。（）『』「」【】\\+-*/=<>", r) {
      tokens = append(tokens, string(r))
      i++
      continue
    }
    if unicode.IsLetter(r) || unicode.IsDigit(r) {
      start := i
      for i < len(runes) && (unicode.IsLetter(runes[i]) || unicode.IsDigit(runes[i])) { i++ }
      tokens = append(tokens, string(runes[start:i]))
      continue
    }
    return nil, commentSeen, fmt.Errorf("不支持的字符 U+%04X", r)
  }
  return tokens, commentSeen, nil
}

func columns(runes []rune, at int) (int, int, int) {
  line, codeCol, displayCol := 1, 1, 1
  for i := 0; i < at; i++ {
    if runes[i] == '\n' { line++; codeCol = 1; displayCol = 1; continue }
    codeCol++
    if unicode.Is(unicode.Han, runes[i]) { displayCol += 2 } else { displayCol++ }
  }
  return line, codeCol, displayCol
}

func checkCodegen() error {
  root, err := os.Getwd()
  if err != nil { return err }
  path := filepath.Join(root, "build", "prototype-hello")
  if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil { return err }
  message := []byte("行简原型：代码生成成功\n")
  if err = os.WriteFile(path, machoHello(message), 0755); err != nil { return err }
  if err = os.Chmod(path, 0755); err != nil { return err }
	if output, err := exec.Command("codesign", "-s", "-", "--force", "--timestamp=none", path).CombinedOutput(); err != nil { return fmt.Errorf("签名生成文件：%w：%s", err, output) }
  output, err := exec.Command(path).Output()
  if err != nil { return fmt.Errorf("运行生成文件：%w", err) }
  if string(output) != string(message) { return fmt.Errorf("生成文件输出不符：%q", output) }
  return nil
}

func machoHello(message []byte) []byte {
	const codeOffset, bindOffset = 816, 32768
	words := []uint32{0xd2800020, 0x10000141, movz(2, uint16(len(message))), 0x90000030, 0xf9400210, 0xd63f0200, 0xd2800000, 0x90000030, 0xf9400610, 0xd63f0200, 0xd4200000}
	bind := []byte{0x11, 0x40, '_', 'w', 'r', 'i', 't', 'e', 0, 0x51, 0x72, 0, 0x90, 0x40, '_', 'e', 'x', 'i', 't', 0, 0x51, 0x72, 8, 0x90, 0}
	symOffset := uint32((bindOffset + len(bind) + 7) &^ 7); strOffset := symOffset + 32; symbols := []byte{0, '_', 'w', 'r', 'i', 't', 'e', 0, '_', 'e', 'x', 'i', 't', 0}
	out := make([]byte, int(strOffset)+len(symbols))
	put32(out, 0, 0xfeedfacf); put32(out, 4, 0x0100000c); put32(out, 12, 2); put32(out, 16, 12); put32(out, 20, 760); put32(out, 24, 0x200084)
	putSegment(out, 32, "__PAGEZERO", 72, 0, 0x100000000, 0, 0, 0, 0, 0)
	putSegment(out, 104, "__TEXT", 152, 0x100000000, 16384, 0, 16384, 5, 5, 1)
	putName(out, 176, "__text"); putName(out, 192, "__TEXT"); put64(out, 208, 0x100000000+codeOffset); put64(out, 216, uint64(len(words)*4+len(message))); put32(out, 224, codeOffset); put32(out, 228, 2); put32(out, 240, 0x80000400)
	putSegment(out, 256, "__DATA", 152, 0x100004000, 16384, 16384, 16384, 3, 3, 1)
	putName(out, 328, "__data"); putName(out, 344, "__DATA"); put64(out, 360, 0x100004000); put64(out, 368, 16); put32(out, 376, 16384); put32(out, 380, 3)
	putSegment(out, 408, "__LINKEDIT", 72, 0x100008000, 16384, bindOffset, uint64(len(out)-bindOffset), 1, 1, 0)
	put32(out, 480, 0x80000022); put32(out, 484, 48); put32(out, 496, bindOffset); put32(out, 500, uint32(len(bind)))
	put32(out, 528, 0xe); put32(out, 532, 32); put32(out, 536, 12); copy(out[540:], "/usr/lib/dyld\x00")
	put32(out, 560, 0xc); put32(out, 564, 56); put32(out, 568, 24); copy(out[584:], "/usr/lib/libSystem.B.dylib\x00"); put32(out, 576, 0); put32(out, 580, 0x00010000)
	put32(out, 616, 0x32); put32(out, 620, 24); put32(out, 624, 1); put32(out, 628, 0x000b0000); put32(out, 632, 0x001b0000)
	put32(out, 640, 0x80000028); put32(out, 644, 24); put64(out, 648, codeOffset)
	put32(out, 664, 0x1b); put32(out, 668, 24); copy(out[672:688], "SERICA0000000001")
	put32(out, 688, 2); put32(out, 692, 24); put32(out, 696, symOffset); put32(out, 700, 2); put32(out, 704, strOffset); put32(out, 708, uint32(len(symbols)))
	put32(out, 712, 0xb); put32(out, 716, 80); put32(out, 740, 2)
	out[symOffset+4] = 1; out[symOffset+7] = 1; put32(out, int(symOffset)+16, 8); out[symOffset+20] = 1; out[symOffset+23] = 1
	for i, word := range words { binary.LittleEndian.PutUint32(out[codeOffset+i*4:], word) }
	copy(out[codeOffset+len(words)*4:], message); copy(out[bindOffset:], bind); copy(out[strOffset:], symbols[1:])
  return out
}

func putSegment(b []byte, at int, name string, size uint32, vm, vmSize, file, fileSize uint64, max, init, sections uint32) {
	put32(b, at, 0x19); put32(b, at+4, size); putName(b, at+8, name); put64(b, at+24, vm); put64(b, at+32, vmSize); put64(b, at+40, file); put64(b, at+48, fileSize); put32(b, at+56, max); put32(b, at+60, init); put32(b, at+64, sections)
}

func movz(reg uint32, value uint16) uint32 { return 0xd2800000 | uint32(value)<<5 | reg }
func put32(b []byte, at int, value uint32) { binary.LittleEndian.PutUint32(b[at:], value) }
func put64(b []byte, at int, value uint64) { binary.LittleEndian.PutUint64(b[at:], value) }
func putName(b []byte, at int, value string) { copy(b[at:at+16], value) }
