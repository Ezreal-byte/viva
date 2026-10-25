//go:build linux

package linux

import (
	"slices"

	"github.com/vivaui/viva/internal/platform"
	"github.com/vivaui/viva/internal/render"
	"github.com/vivaui/viva/internal/scene"
)

type compositingBatchKind uint8

const (
	compositeRectangles compositingBatchKind = iota + 1
	compositeText
)

type compositingBatch struct {
	Kind         compositingBatchKind
	Layer        uint16
	Clipped      bool
	Clip         scene.Rect
	First, Count int
}

func groupTextLayers(labels []vulkanLabelTexture, order []int) []int {
	order = order[:0]
	for index := range labels {
		order = append(order, index)
	}
	slices.SortStableFunc(order, func(left, right int) int {
		return int(labels[left].label.Layer) - int(labels[right].label.Layer)
	})
	return order
}

func groupCompositingLayers(rectangles []render.Batch, labels []vulkanLabelTexture, labelOrder []int, batches []compositingBatch) []compositingBatch {
	batches = batches[:0]
	for rectangleIndex, labelIndex := 0, 0; rectangleIndex < len(rectangles) || labelIndex < len(labelOrder); {
		layer := ^uint16(0)
		if rectangleIndex < len(rectangles) {
			layer = rectangles[rectangleIndex].Layer
		}
		if labelIndex < len(labelOrder) && labels[labelOrder[labelIndex]].label.Layer < layer {
			layer = labels[labelOrder[labelIndex]].label.Layer
		}
		for rectangleIndex < len(rectangles) && rectangles[rectangleIndex].Layer == layer {
			rectangle := rectangles[rectangleIndex]
			batches = append(batches, compositingBatch{Kind: compositeRectangles, Layer: layer, Clipped: rectangle.Clipped, Clip: rectangle.Clip, First: rectangle.First, Count: rectangle.Count})
			rectangleIndex++
		}
		firstLabel := labelIndex
		for labelIndex < len(labelOrder) && labels[labelOrder[labelIndex]].label.Layer == layer {
			labelIndex++
		}
		if labelIndex != firstLabel {
			batches = append(batches, compositingBatch{Kind: compositeText, Layer: layer, First: firstLabel, Count: labelIndex - firstLabel})
		}
	}
	return batches
}

func validateInitialTextLabels(labels []platform.TextLabel) error {
	for index, label := range labels {
		if err := validateTextLabel(index, label); err != nil {
			return err
		}
	}
	return nil
}
