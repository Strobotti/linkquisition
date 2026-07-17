#include "url_handler.h"

@implementation GoPasser

+ (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    NSString *urlStr = [[event paramDescriptorForKeyword:keyDirectObject] stringValue];
    if (urlStr != nil) {
        HandleURL((char *)[urlStr UTF8String]);
    }
}

+ (void)handleOpenDocumentEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent {
    NSAppleEventDescriptor *directObject = [event paramDescriptorForKeyword:keyDirectObject];
    if (directObject == nil) return;

    // The direct object is a list of file descriptors
    NSInteger count = [directObject numberOfItems];
    for (NSInteger i = 1; i <= count; i++) {
        NSAppleEventDescriptor *item = [directObject descriptorAtIndex:i];
        // Try to get the URL from the file descriptor
        NSString *urlStr = [item stringValue];
        if (urlStr != nil) {
            // Convert file path to file:// URL if it doesn't already have a scheme
            if (![urlStr hasPrefix:@"file://"] && ![urlStr hasPrefix:@"http://"] && ![urlStr hasPrefix:@"https://"]) {
                urlStr = [[NSURL fileURLWithPath:urlStr] absoluteString];
            }
            HandleURL((char *)[urlStr UTF8String]);
            break; // Only handle the first document
        }
    }
}

@end

void StartURLHandler(void) {
    NSAppleEventManager *appleEventManager = [NSAppleEventManager sharedAppleEventManager];
    [appleEventManager setEventHandler:[GoPasser class]
                           andSelector:@selector(handleGetURLEvent:withReplyEvent:)
                         forEventClass:kInternetEventClass
                            andEventID:kAEGetURL];
    [appleEventManager setEventHandler:[GoPasser class]
                           andSelector:@selector(handleOpenDocumentEvent:withReplyEvent:)
                         forEventClass:kCoreEventClass
                            andEventID:kAEOpenDocuments];
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
