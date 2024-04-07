// SPDX-License-Identifier: Unlicense OR MIT

//go:build ios
// +build ios

#include <UIKit/UIKit.h>
#include <stdint.h>
#include <UniformTypeIdentifiers/UniformTypeIdentifiers.h>
#include "_cgo_export.h"

@implementation explorer_picker
- (void)documentPicker:(UIDocumentPickerViewController *)controller didPickDocumentsAtURLs:(NSArray<NSURL *> *)urls {
    NSURL *url = [urls objectAtIndex:0];

    switch (self.mode) {
    case EXPORT_MODE:
        exportCallback((__bridge_retained CFTypeRef)url, self.id);
        return;
    case IMPORT_MODE:
        importCallback((__bridge_retained CFTypeRef)url, self.id);
        return;
    }
}
- (void)documentPickerWasCancelled:(UIDocumentPickerViewController *)controller {
    switch (self.mode) {
    case EXPORT_MODE:
        exportCallback(0, self.id);
        return;
    case IMPORT_MODE:
        importCallback(0, self.id);
        return;
    }
}
@end

CFTypeRef createPicker(CFTypeRef controllerRef, int32_t id) {
	explorer_picker *e = [[explorer_picker alloc] init];
	e.controller = (__bridge UIViewController *)controllerRef;
	e.id = id;
	return (__bridge_retained CFTypeRef)e;
}

bool exportFile(CFTypeRef expl, char * name) {
   if (@available(iOS 14, *)) {
        explorer_picker *explorer = (__bridge explorer_picker *)expl;
        explorer.picker = [[UIDocumentPickerViewController alloc] initForExportingURLs:@[[NSURL URLWithString:@(name)]] asCopy:true];
        explorer.picker.delegate = explorer;
        explorer.mode = EXPORT_MODE;

        [explorer.controller presentViewController:explorer.picker animated:YES completion:nil];
        return YES;
    }
    return NO;
}

bool importFile(CFTypeRef expl, char * ext) {
  if (@available(iOS 14, *)) {
        explorer_picker *explorer = (__bridge explorer_picker *)expl;

        NSMutableArray<NSString*> *exts = [[@(ext) componentsSeparatedByString:@","] mutableCopy];
        NSMutableArray<UTType*> *contentTypes = [[NSMutableArray alloc]init];

        int i;
        for (i = 0; i < [exts count]; i++) {
            UTType *utt = [UTType typeWithFilenameExtension:exts[i]];
            if (utt != nil) {
                [contentTypes addObject:utt];
            }
        }

        explorer.picker = [[UIDocumentPickerViewController alloc] initForOpeningContentTypes:contentTypes asCopy:true];
        explorer.picker.delegate = explorer;
        explorer.mode = IMPORT_MODE;

        [explorer.controller presentViewController:explorer.picker animated:YES completion:nil];
        return YES;
    }
    return NO;
}
