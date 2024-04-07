#include "_cgo_export.h"

@implementation explorer_file
@end

CFTypeRef newFile(CFTypeRef url) {
    if (@available(iOS 13, macOS 10.15, *)) {
        explorer_file *f = [[explorer_file alloc] init];
        f.url = (__bridge NSURL *)url;
        [f.url startAccessingSecurityScopedResource];

        NSError *err = nil;
        f.handler = [NSFileHandle fileHandleForUpdatingURL:f.url error:&err];
        f.err = err;
        return (__bridge_retained CFTypeRef)f;
    }
    return 0;
}

uint64_t fileRead(CFTypeRef file, uint8_t *b, uint64_t len) {
    explorer_file *f = (__bridge explorer_file *)file;
    if (@available(iOS 13, macOS 10.15, *)) {
        NSError *err = nil;
        NSData *data = [f.handler readDataUpToLength:len error:&err];
        if (err != nil) {
            f.err = err;
            return 0;
        }

        [data getBytes:b length:data.length];
        return data.length;
    }
    return 0; // Impossible condition since newFileReader will return 0.
}

bool fileWrite(CFTypeRef file, uint8_t *b, uint64_t len) {
    explorer_file *f = (__bridge explorer_file *)file;
    if (@available(iOS 13, macOS 10.15, *)) {
        NSError *err = nil;
        [f.handler writeData:[NSData dataWithBytes:b length:len] error:&err];
        if (err != nil) {
            f.err = err;
            return NO;
        }

        return YES;
    }
    return NO; // Impossible condition since newFileWriter will return 0.
}

bool fileClose(CFTypeRef file) {
    explorer_file *f = (__bridge explorer_file *)file;
    if (@available(iOS 13, macOS 10.15, *)) {
        [f.url stopAccessingSecurityScopedResource];

        NSError *err = nil;
        [f.handler closeAndReturnError:&err];
        if (err != nil) {
            f.err = err;
            return NO;
        }
        return YES;
    }
    return NO; // Impossible condition since newFileWriter will return 0.
}

char* getError(CFTypeRef file) {
    explorer_file *f = (__bridge explorer_file *)file;
    if (f.err == nil) {
        return 0;
    }
    return (char*)([[f.err localizedDescription] UTF8String]);
}

const char* getURL(CFTypeRef url_ref) {
    NSURL *url = (__bridge NSURL *)url_ref;
    NSString *str = [url absoluteString];

    const char *unsafe_cstr = [str UTF8String];
    char *safe_cstr = strdup(unsafe_cstr);
    return safe_cstr;
}
