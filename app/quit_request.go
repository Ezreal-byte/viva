package app

// OnQuitRequest lets an application confirm a system Quit action before it
// closes document windows. Call it on the application thread during startup.
// The callback may retain allow while presenting unsaved-content confirmation.
// It currently receives macOS menu/Dock Quit requests. Application.Quit remains
// the explicit force-quit path and never asks for confirmation a second time.
func (application *Application) OnQuitRequest(handler func(allow func())) {
	if application == nil || application.host == nil {
		return
	}
	if host, ok := application.host.(interface{ SetQuitRequestHandler(func()) }); ok {
		if handler == nil {
			host.SetQuitRequestHandler(nil)
		} else {
			host.SetQuitRequestHandler(func() { handler(application.Quit) })
		}
	}
}
