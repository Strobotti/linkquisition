#import <Cocoa/Cocoa.h>
#import <Carbon/Carbon.h>

extern void HandleURL(char*);

@interface GoPasser : NSObject
+ (void)handleGetURLEvent:(NSAppleEventDescriptor *)event withReplyEvent:(NSAppleEventDescriptor *)replyEvent;
@end

void StartURLHandler(void);
void PumpEvents(double seconds);
