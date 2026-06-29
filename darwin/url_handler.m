#include "url_handler.h"

@implementation GoPasser

+ (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    NSString *urlStr = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    if (urlStr != nil) {
        HandleURL((char *)[urlStr UTF8String]);
    }
}

@end

void StartURLHandler(void) {
    NSAppleEventManager *appleEventManager = [NSAppleEventManager sharedAppleEventManager];
    [appleEventManager setEventHandler:[GoPasser class]
                           andSelector:@selector(handleGetURLEvent:withReplyEvent:)
                         forEventClass:kInternetEventClass
                            andEventID:kAEGetURL];
}

void PumpEvents(double seconds) {
    // Ensure NSApplication is initialized
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    [NSApp finishLaunching];

    // Process events for the specified duration
    NSDate *until = [NSDate dateWithTimeIntervalSinceNow:seconds];
    while (true) {
        NSEvent *event = [NSApp nextEventMatchingMask:NSEventMaskAny
                                           untilDate:until
                                              inMode:NSDefaultRunLoopMode
                                             dequeue:YES];
        if (event == nil) break;
        [NSApp sendEvent:event];
    }
}
