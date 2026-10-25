package components

import "github.com/vivaui/viva/ui"

// nonBlockHits drops the inert surface block that a popup paints inside its
// dismiss backdrop, so a test can talk about its controls and rows.
func nonBlockHits(hits []ui.Hit) []ui.Hit {
	out := make([]ui.Hit, 0, len(hits))
	for _, hit := range hits {
		if !hit.Block && !hit.Container {
			out = append(out, hit)
		}
	}
	return out
}
