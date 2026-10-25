// appkit_text draws one line of text with AppKit, as a macOS application does, into an
// opaque bitmap and writes the pixels (BGRA, top row first) to a file. It is the
// reference the label rasterizer is measured against; a test compiles and
// runs it, it is not part of the product.
//
//   appkit_text OUT TEXT MODE SIZE WEIGHT SCALE FG BG WIDTH HEIGHT [BASELINE [X [ALPHA]]]
//
// MODE is "string" (NSAttributedString drawAtPoint in a flipped context, the line's baseline
// BASELINE points below the top Y, drawn at X) or "field" (an NSTextField label whose cell is WIDTH x
// HEIGHT points, drawn as AppKit draws a control). FG and BG are sRGB grays.
#import <Cocoa/Cocoa.h>

int main(int argc, char **argv) {
    @autoreleasepool {
        if (argc < 11) {
            fprintf(stderr, "usage\n");
            return 2;
        }
        const char *out = argv[1];
        NSString *text = [NSString stringWithUTF8String:argv[2]];
        NSString *mode = [NSString stringWithUTF8String:argv[3]];
        double size = atof(argv[4]), weight = atof(argv[5]), scale = atof(argv[6]);
        double fg = atof(argv[7]), bg = atof(argv[8]);
        int width = atoi(argv[9]), height = atoi(argv[10]);
        double baseline = argc > 11 ? atof(argv[11]) : height * 0.5;
        double originX = argc > 12 ? atof(argv[12]) : 0, alpha = argc > 13 ? atof(argv[13]) : 1;
        [NSApplication sharedApplication];
        NSFont *font = [NSFont systemFontOfSize:size weight:weight];
        NSColor *foreground = [NSColor colorWithSRGBRed:fg green:fg blue:fg alpha:alpha];
        NSColor *background = [NSColor colorWithSRGBRed:bg green:bg blue:bg alpha:1];
        if (height == 0) {
            // The height AppKit gives a one-line label of this font.
            NSTextField *probe = [NSTextField labelWithString:text];
            probe.font = font;
            height = (int)ceil(probe.intrinsicContentSize.height);
            baseline = height * 0.5;
        }
        NSInteger pw = (NSInteger)ceil(width * scale), ph = (NSInteger)ceil(height * scale);
        // An sRGB bitmap context of our own: a bitmap rep would convert the pixels through its
        // own (calibrated or device) color space and change the coverage of the glyphs.
        NSMutableData *data = [NSMutableData dataWithLength:pw * ph * 4];
        CGColorSpaceRef space = CGColorSpaceCreateWithName(kCGColorSpaceSRGB);
        CGContextRef bitmap = CGBitmapContextCreate(data.mutableBytes, pw, ph, 8, pw * 4, space,
                                                    kCGImageAlphaPremultipliedFirst | kCGBitmapByteOrder32Little);
        CGColorSpaceRelease(space);
        CGContextScaleCTM(bitmap, scale, scale);
        NSGraphicsContext *context = [NSGraphicsContext graphicsContextWithCGContext:bitmap flipped:NO];
        [NSGraphicsContext saveGraphicsState];
        [NSGraphicsContext setCurrentContext:context];
        if ([mode isEqualToString:@"field"]) {
            NSTextField *field = [NSTextField labelWithString:text];
            field.font = font;
            field.textColor = foreground;
            field.drawsBackground = YES;
            field.backgroundColor = background;
            field.lineBreakMode = NSLineBreakByClipping;
            field.frame = NSMakeRect(0, 0, width, height);
            field.appearance = [NSAppearance appearanceNamed:NSAppearanceNameAqua];
            [field displayRectIgnoringOpacity:field.bounds inContext:context];
        } else {
            [background setFill];
            NSRectFill(NSMakeRect(0, 0, width, height));
            NSDictionary *attributes = @{NSFontAttributeName: font, NSForegroundColorAttributeName: foreground};
            NSAttributedString *string = [[NSAttributedString alloc] initWithString:text attributes:attributes];
            // Bottom-left origin: the baseline is `baseline` below the top.
            [string drawAtPoint:NSMakePoint(originX, height - baseline - ceil(-font.descender))];
        }
        [NSGraphicsContext restoreGraphicsState];
        CGContextRelease(bitmap);
        FILE *file = fopen(out, "wb");
        fwrite(data.bytes, 1, data.length, file);
        fclose(file);
        printf("%ld %ld\n", (long)pw, (long)ph);
    }
    return 0;
}
