# Initial bitmap uploads

The painter may reconcile a caller's pre-painted canvas and then paint again
with the platform text engine before creating the native window. These are
preparation passes, not frames consumed by the native renderer. A change to text
or layout between them must not suppress the first upload of an unchanged bitmap.

After preparing the final initial frame, the painter marks every bitmap label
for upload. Only subsequent frames may omit pixels for retained bitmaps. This
applies to both `RunCanvas` and application-managed windows through their shared
painter initialization. No additional repaint, timer, worker, or image ID change
is needed, and steady-state bitmap reuse is unchanged.

`app.Label.BitmapUpload` in a driver snapshot exposes the most recently prepared
frame's native upload decision. It is diagnostic metadata, not an instruction to
omit pixels when replaying a snapshot into a different renderer. Headless
screenshots intentionally upload snapshot pixels to their own offscreen renderer;
the macOS `vivahead` regression test instead sends the actual prepared window's
labels to Metal to exercise the native staging path.

The acceptance case is Motty's sidebar lockup on a direct home-page launch.
Motty owns the embedded artwork and its layout; Viva owns renderer resource
lifetimes and first-frame uploads. No application-side redraw workaround belongs
in this fix.
