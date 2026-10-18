//go:build webview && darwin

package launcher

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Cocoa
#include <stdlib.h>
#import <Cocoa/Cocoa.h>

static NSMenuItem *gokiItem(NSMenu *menu, NSString *title, SEL action, NSString *key, NSEventModifierFlags mods) {
	NSMenuItem *item = [menu addItemWithTitle:title action:action keyEquivalent:key];
	if (mods) [item setKeyEquivalentModifierMask:mods];
	return item;
}

// gokiSetupApp installs a standard menu bar. Without an Edit menu, WebKit
// text fields on macOS ignore Cmd+C, Cmd+V, Cmd+Z and friends.
static void gokiSetupApp(const char *ctitle, const void *icon, int iconLen) {
	NSString *name = [NSString stringWithUTF8String:ctitle];
	NSEventModifierFlags cmd = NSEventModifierFlagCommand;
	NSEventModifierFlags cmdShift = NSEventModifierFlagCommand | NSEventModifierFlagShift;
	NSEventModifierFlags cmdAlt = NSEventModifierFlagCommand | NSEventModifierFlagOption;

	NSMenu *bar = [[NSMenu alloc] init];

	NSMenu *app = [[NSMenu alloc] initWithTitle:name];
	gokiItem(app, [@"Hide " stringByAppendingString:name], @selector(hide:), @"h", 0);
	gokiItem(app, @"Hide Others", @selector(hideOtherApplications:), @"h", cmdAlt);
	gokiItem(app, @"Show All", @selector(unhideAllApplications:), @"", 0);
	[app addItem:[NSMenuItem separatorItem]];
	gokiItem(app, [@"Quit " stringByAppendingString:name], @selector(terminate:), @"q", 0);
	[[bar addItemWithTitle:name action:nil keyEquivalent:@""] setSubmenu:app];

	NSMenu *edit = [[NSMenu alloc] initWithTitle:@"Edit"];
	gokiItem(edit, @"Undo", @selector(undo:), @"z", 0);
	gokiItem(edit, @"Redo", @selector(redo:), @"z", cmdShift);
	[edit addItem:[NSMenuItem separatorItem]];
	gokiItem(edit, @"Cut", @selector(cut:), @"x", 0);
	gokiItem(edit, @"Copy", @selector(copy:), @"c", 0);
	gokiItem(edit, @"Paste", @selector(paste:), @"v", 0);
	gokiItem(edit, @"Paste and Match Style", @selector(pasteAsPlainText:), @"v", cmdAlt | NSEventModifierFlagShift);
	gokiItem(edit, @"Select All", @selector(selectAll:), @"a", 0);
	[[bar addItemWithTitle:@"Edit" action:nil keyEquivalent:@""] setSubmenu:edit];

	NSMenu *view = [[NSMenu alloc] initWithTitle:@"View"];
	gokiItem(view, @"Reload Page", @selector(reload:), @"r", 0);
	gokiItem(view, @"Enter Full Screen", @selector(toggleFullScreen:), @"f", cmd | NSEventModifierFlagControl);
	[[bar addItemWithTitle:@"View" action:nil keyEquivalent:@""] setSubmenu:view];

	NSMenu *window = [[NSMenu alloc] initWithTitle:@"Window"];
	gokiItem(window, @"Minimize", @selector(performMiniaturize:), @"m", 0);
	gokiItem(window, @"Zoom", @selector(performZoom:), @"", 0);
	gokiItem(window, @"Close", @selector(performClose:), @"w", 0);
	[[bar addItemWithTitle:@"Window" action:nil keyEquivalent:@""] setSubmenu:window];
	[NSApp setWindowsMenu:window];

	[NSApp setMainMenu:bar];

	if (icon && iconLen > 0) {
		NSData *data = [NSData dataWithBytes:icon length:iconLen];
		NSImage *img = [[NSImage alloc] initWithData:data];
		if (img) [NSApp setApplicationIconImage:img];
	}
}
*/
import "C"

import (
	"unsafe"

	"github.com/jdtobe/goki/web"
)

func platformSetup(title string) {
	ctitle := C.CString(title)
	defer C.free(unsafe.Pointer(ctitle))
	icon, _ := web.FS.ReadFile("img/icon.svg")
	var ptr unsafe.Pointer
	if len(icon) > 0 {
		ptr = C.CBytes(icon)
		defer C.free(ptr)
	}
	C.gokiSetupApp(ctitle, ptr, C.int(len(icon)))
}
