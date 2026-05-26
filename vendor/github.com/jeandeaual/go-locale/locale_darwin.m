//go:build darwin && !ios

#import <Foundation/Foundation.h>

bool hasAppTranslations() {
    return [[[NSBundle mainBundle] localizations] count] > 1;
}

const char *preferredLocalization()
{
    if (!hasAppTranslations()) {
        return "";
    }

    NSString *locale = [[[NSBundle mainBundle] preferredLocalizations] firstObject];

    return [locale UTF8String];
}

const char *preferredLocalizations()
{
    if (!hasAppTranslations()) {
        return "";
    }

    NSString *locales = [[[NSBundle mainBundle] preferredLocalizations] componentsJoinedByString:@","];

    return [locales UTF8String];
}
