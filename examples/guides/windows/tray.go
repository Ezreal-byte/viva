package main

import "github.com/vivaui/viva/app"

// addTray keeps an icon in the menu bar, notification area or tray. The icon is
// square RGBA; a template icon has only alpha and is tinted to suit the bar.
func addTray(application *app.Application, window *app.Window, icon app.StatusIcon) (*app.StatusItem, error) {
	return application.AddStatusItem(app.StatusItemConfig{
		Tooltip: "Notes",
		Icon:    icon,
		Menu: func() []app.MenuItem {
			// Built each time the menu is about to open, so it shows current state.
			return []app.MenuItem{
				{Title: "Show Notes", OnSelect: window.Show},
				{Title: "Hide Notes", OnSelect: window.Hide, Disabled: window.Hidden()},
				{Separator: true},
				{Title: "Quit", OnSelect: application.Quit},
			}
		},
	})
}
