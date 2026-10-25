# Renders strings with a real GtkLabel (GTK 3 on Pango and Cairo, with the font settings
# of the running X session), the way a GTK application draws text, for the fidelity test.
#
#   gtkref.py cases.json outdir
#
# cases.json is a list of {name, text, size (px), weight (css), fg, bg, width, height, x,
# family}; every case is written to outdir/name.png and the file settings.json
# records what GTK used.
#
# Metrics are not hinted and glyph positions are not rounded to the device pixel, as in GTK 4,
# Qt 6 and browsers, and as Viva draws: a line then has the same width at every
# display scale. GTK 3 itself hints metrics, which is why the labels are set to do without.
import json, os, sys, gi, cairo
gi.require_version('Gtk', '3.0')
gi.require_version('Gdk', '3.0')
from gi.repository import Gtk, Gdk

cases = json.load(open(sys.argv[1]))
out = sys.argv[2]
screen = Gdk.Screen.get_default()
settings = Gtk.Settings.get_default()
record = {name: settings.get_property(name) for name in
          ('gtk-xft-antialias', 'gtk-xft-hinting', 'gtk-xft-hintstyle', 'gtk-xft-rgba', 'gtk-xft-dpi', 'gtk-font-name')}
options = screen.get_font_options()
if options is None:
    options = cairo.FontOptions()
else:
    options = options.copy()
options.set_hint_metrics(cairo.HINT_METRICS_OFF)
screen.set_font_options(options)
if options is not None:
    record['cairo'] = {'antialias': int(options.get_antialias()), 'hint_style': int(options.get_hint_style()),
                       'hint_metrics': int(options.get_hint_metrics()), 'subpixel_order': int(options.get_subpixel_order())}
json.dump(record, open(os.path.join(out, 'settings.json'), 'w'))
provider = None
for case in cases:
    if provider is not None:
        Gtk.StyleContext.remove_provider_for_screen(screen, provider)
    # Without a family the label takes the desktop's (gtk-font-name), as Viva's engine does.
    family = 'font-family: %s; ' % case['family'] if case.get('family') else ''
    css = ('window, label { background-color: %s; color: %s; %sfont-size: %spx; font-weight: %s; '
           'padding: 0; margin: 0; border: none; }' % (case['bg'], case['fg'], family, case['size'], case['weight']))
    provider = Gtk.CssProvider()
    provider.load_from_data(css.encode())
    Gtk.StyleContext.add_provider_for_screen(screen, provider, Gtk.STYLE_PROVIDER_PRIORITY_USER)
    window = Gtk.OffscreenWindow()
    window.set_default_size(case['width'], case['height'])
    box = Gtk.Fixed()
    label = Gtk.Label(label=case['text'])
    label.get_pango_context().set_round_glyph_positions(False)
    label.set_xalign(0)
    label.set_yalign(0)
    box.put(label, case.get('x', 0), 0)
    window.add(box)
    window.show_all()
    for _ in range(20):
        while Gtk.events_pending():
            Gtk.main_iteration()
    window.get_pixbuf().savev(os.path.join(out, case['name'] + '.png'), 'png', [], [])
    window.destroy()
