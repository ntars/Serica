// SPDX-License-Identifier: Apache-2.0
// Copyright (c) 2026 行简（Serica）贡献者
#import <Cocoa/Cocoa.h>
#import <CoreText/CoreText.h>
#include <stdio.h>

static void serica_draw_text(CGContextRef context, CGFloat height) {
  CFStringRef text = CFStringCreateWithCString(NULL, "中文渲染验证：计算总额、订单数", kCFStringEncodingUTF8);
  CTFontRef font = CTFontCreateWithName(CFSTR("Hiragino Sans GB"), 28, NULL);
  const void *keys[] = { kCTFontAttributeName };
  const void *values[] = { font };
  CFDictionaryRef attrs = CFDictionaryCreate(NULL, keys, values, 1, &kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
  CFAttributedStringRef string = CFAttributedStringCreate(NULL, text, attrs);
  CTLineRef line = CTLineCreateWithAttributedString(string);
  CGContextSetRGBFillColor(context, 0, 0, 0, 1);
  CGContextSetTextMatrix(context, CGAffineTransformIdentity);
  CGContextSetTextPosition(context, 32, height - 88);
  CTLineDraw(line, context);
  CFRelease(line); CFRelease(string); CFRelease(attrs); CFRelease(font); CFRelease(text);
}

static void serica_save_preview(void) {
  CGColorSpaceRef color = CGColorSpaceCreateDeviceRGB();
  CGContextRef context = CGBitmapContextCreate(NULL, 560, 180, 8, 560 * 4, color, kCGImageAlphaPremultipliedLast | kCGBitmapByteOrder32Big);
  CGContextSetRGBFillColor(context, 1, 1, 1, 1);
  CGContextFillRect(context, CGRectMake(0, 0, 560, 180));
  serica_draw_text(context, 180);
  CGImageRef rendered = CGBitmapContextCreateImage(context);
  NSBitmapImageRep *bitmap = [[[NSBitmapImageRep alloc] initWithCGImage:rendered] autorelease];
  NSData *png = [bitmap representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
  [png writeToFile:@"/private/tmp/serica-window.png" atomically:YES];
  CGImageRelease(rendered); CGContextRelease(context); CGColorSpaceRelease(color);
}

@interface SericaPrototypeView : NSView
@end

@implementation SericaPrototypeView
- (void)drawRect:(NSRect)dirtyRect {
  serica_draw_text([[NSGraphicsContext currentContext] CGContext], NSHeight(self.bounds));
}
@end

void serica_show_window(void) {
  @autoreleasepool {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    NSRect frame = NSMakeRect(0, 0, 560, 180);
    NSWindow *window = [[NSWindow alloc] initWithContentRect:frame styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable backing:NSBackingStoreBuffered defer:NO];
    [window setTitle:@"行简 P0b 原型"];
    [window setContentView:[[SericaPrototypeView alloc] initWithFrame:frame]];
    [window center];
    [window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    [window displayIfNeeded];
    serica_save_preview();
    puts("3.8.3 CoreText 中文窗口：已显示");
    fflush(stdout);
    [NSApp performSelector:@selector(terminate:) withObject:nil afterDelay:5.0];
    [NSApp run];
    [window close];
    [window release];
  }
}
