// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
package main

/*
#cgo LDFLAGS: -framework CoreFoundation
#include <CoreFoundation/CoreFoundation.h>
#include <stdlib.h>
#include <string.h>
static int nfc(const unsigned char *src, long size) {
  CFStringRef s = CFStringCreateWithBytes(kCFAllocatorDefault, src, size, kCFStringEncodingUTF8, false);
  if (!s) return 0;
  CFMutableStringRef n = CFStringCreateMutableCopy(kCFAllocatorDefault, 0, s);
  CFRelease(s);
  if (!n) return 0;
  CFStringNormalize(n, kCFStringNormalizationFormC);
  CFIndex cap = CFStringGetMaximumSizeForEncoding(CFStringGetLength(n), kCFStringEncodingUTF8);
  char *out = malloc((size_t)cap + 1);
  if (!out) { CFRelease(n); return 0; }
  CFIndex used = 0;
  CFStringGetBytes(n, CFRangeMake(0, CFStringGetLength(n)), kCFStringEncodingUTF8, 0, false, (UInt8 *)out, cap, &used);
  int ok = used == size && memcmp(src, out, (size_t)size) == 0;
  free(out); CFRelease(n); return ok;
}
*/
import "C"

import (
  "fmt"
  "os"
  "strings"
  "unicode/utf8"
)

func main() {
  failed := false
  for _, path := range os.Args[1:] {
    data, err := os.ReadFile(path)
    if err != nil { fmt.Fprintln(os.Stderr, err); failed = true; continue }
    ptr := C.CBytes(data)
    validNFC := C.nfc((*C.uchar)(ptr), C.long(len(data))) != 0
    C.free(ptr)
    half, disabled := scan(string(data))
    fmt.Printf("%s：NFC=%t 半角标点=%d 禁用字符=%d\n", path, validNFC, half, disabled)
    if !validNFC || half != 0 || disabled != 0 { failed = true }
  }
  if len(os.Args) < 2 { fmt.Fprintln(os.Stderr, "用法：punct-scan <file.serica>"); os.Exit(2) }
  if failed { os.Exit(1) }
}

func scan(src string) (int, int) {
  half, disabled := 0, 0
  quote, comment := rune(0), false
  for _, r := range src {
    if comment { if r == '\n' { comment = false }; continue }
    if quote != 0 {
      if r == quote { quote = 0 }
      continue
    }
    if r == '#' { comment = true; continue }
    if r == '『' { quote = '』'; continue }
    if r == '「' { quote = '」'; continue }
    if strings.ContainsRune(",;():", r) { half++ }
    if r == 0x3000 || r >= 0x200b && r <= 0x200d || r == 0xfeff || r >= 0xff10 && r <= 0xff19 || r >= 0xff21 && r <= 0xff3a || r >= 0xff41 && r <= 0xff5a { disabled++ }
  }
  if !utf8.ValidString(src) { disabled++ }
  return half, disabled
}
