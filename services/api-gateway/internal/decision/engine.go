package decision

import (
	"errors"
	"fmt"
	"math"
	"sort"
)

const (
	RipenessHarvestable     = "harvestable"
	RipenessNotReady        = "not_ready"
	RipenessOccludedUnclear = "occluded_unclear"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Recommend(decisionID string, createdAt string, req SnapshotRequest) (RecommendationResponse, error) {
	if req.FrameWidth <= 0 || req.FrameHeight <= 0 {
		return RecommendationResponse{}, errors.New("frame_width and frame_height must be greater than 0")
	}

	startPoint := point{
		x: float64(req.FrameWidth) / 2,
		y: float64(req.FrameHeight),
	}

	candidates := make([]candidate, 0, len(req.Detections))
	skips := make([]SkipItem, 0, len(req.Detections))

	for index, detection := range req.Detections {
		if err := validateDetection(detection); err != nil {
			return RecommendationResponse{}, fmt.Errorf("detection %d: %w", index, err)
		}

		center := bboxCenter(detection.BBox)
		zone := zoneFromCenter(center, req.FrameWidth, req.FrameHeight)
		ripeness := normalizeRipeness(detection.Ripeness)

		switch ripeness {
		case RipenessHarvestable:
			candidates = append(candidates, candidate{
				detectionIndex: index,
				detection:      detection,
				zone:           zone,
				center:         center,
			})
		case RipenessNotReady, RipenessOccludedUnclear:
			skips = append(skips, buildSkipItem(index, detection, zone, ripeness))
		default:
			skips = append(skips, buildSkipItem(index, detection, zone, RipenessOccludedUnclear))
		}
	}

	zonePriorities := buildZonePriorities(candidates, startPoint)
	pickSequence, pathLength := buildPickSequence(candidates, startPoint)

	var mainZone *string
	if len(zonePriorities) > 0 {
		mainZone = &zonePriorities[0].Zone
	}

	return RecommendationResponse{
		DecisionID: decisionID,
		CreatedAt:  createdAt,
		Summary: Summary{
			TotalDetections:     len(req.Detections),
			HarvestableCount:    len(candidates),
			SkippedCount:        len(skips),
			MainPriorityZone:    mainZone,
			EstimatedPathLength: roundFloat(pathLength),
		},
		ZonePriorities: zonePriorities,
		PickSequence:   pickSequence,
		SkipItems:      skips,
	}, nil
}

type candidate struct {
	detectionIndex int
	detection      Detection
	zone           string
	center         point
}

type point struct {
	x float64
	y float64
}

func validateDetection(d Detection) error {
	if d.ClassName != "camellia_oleifera_fruit" {
		return fmt.Errorf("class_name must be camellia_oleifera_fruit, got %q", d.ClassName)
	}
	if d.Confidence < 0 || d.Confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1, got %f", d.Confidence)
	}
	x1, y1, x2, y2 := d.BBox[0], d.BBox[1], d.BBox[2], d.BBox[3]
	if x2 < x1 || y2 < y1 {
		return errors.New("bbox must be ordered as [x1, y1, x2, y2]")
	}
	return nil
}

func normalizeRipeness(ripeness *string) string {
	if ripeness == nil {
		return ""
	}
	switch *ripeness {
	case RipenessHarvestable, RipenessNotReady, RipenessOccludedUnclear:
		return *ripeness
	default:
		return ""
	}
}

func buildSkipItem(index int, detection Detection, zone string, reason string) SkipItem {
	message := "成熟度缺失，按遮挡不清保守跳过。"
	switch reason {
	case RipenessNotReady:
		message = "成熟度为暂不可采，当前轮次建议跳过。"
	case RipenessOccludedUnclear:
		message = "视野遮挡或成熟度不清晰，当前轮次建议跳过。"
	}

	return SkipItem{
		DetectionIndex: index,
		TrackID:        detection.TrackID,
		Zone:           zone,
		SkipReason:     reason,
		Confidence:     roundFloat(detection.Confidence),
		Reason:         message,
	}
}

func buildZonePriorities(candidates []candidate, start point) []ZonePriority {
	type zoneStat struct {
		zone            string
		count           int
		distanceToStart float64
	}

	statsByZone := map[string]*zoneStat{}
	for _, item := range candidates {
		distance := pointDistance(start, item.center)
		stat, ok := statsByZone[item.zone]
		if !ok {
			statsByZone[item.zone] = &zoneStat{
				zone:            item.zone,
				count:           1,
				distanceToStart: distance,
			}
			continue
		}
		stat.count++
		if distance < stat.distanceToStart {
			stat.distanceToStart = distance
		}
	}

	stats := make([]zoneStat, 0, len(statsByZone))
	for _, stat := range statsByZone {
		stats = append(stats, *stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].count != stats[j].count {
			return stats[i].count > stats[j].count
		}
		if stats[i].distanceToStart != stats[j].distanceToStart {
			return stats[i].distanceToStart < stats[j].distanceToStart
		}
		return stats[i].zone < stats[j].zone
	})

	items := make([]ZonePriority, 0, len(stats))
	for index, stat := range stats {
		items = append(items, ZonePriority{
			Order:            index + 1,
			Zone:             stat.zone,
			HarvestableCount: stat.count,
			DistanceToStart:  roundFloat(stat.distanceToStart),
			Note:             fmt.Sprintf("该区域有 %d 枚可采目标，且距离起始点较近。", stat.count),
		})
	}
	return items
}

func buildPickSequence(candidates []candidate, start point) ([]PickSequenceItem, float64) {
	if len(candidates) == 0 {
		return []PickSequenceItem{}, 0
	}

	remaining := append([]candidate(nil), candidates...)
	current := start
	totalDistance := 0.0
	items := make([]PickSequenceItem, 0, len(candidates))

	for len(remaining) > 0 {
		sort.Slice(remaining, func(i, j int) bool {
			leftDistance := pointDistance(current, remaining[i].center)
			rightDistance := pointDistance(current, remaining[j].center)
			if leftDistance != rightDistance {
				return leftDistance < rightDistance
			}
			if remaining[i].detection.Confidence != remaining[j].detection.Confidence {
				return remaining[i].detection.Confidence > remaining[j].detection.Confidence
			}
			return remaining[i].detectionIndex < remaining[j].detectionIndex
		})

		next := remaining[0]
		segmentDistance := pointDistance(current, next.center)
		totalDistance += segmentDistance
		items = append(items, PickSequenceItem{
			Order:          len(items) + 1,
			DetectionIndex: next.detectionIndex,
			TrackID:        next.detection.TrackID,
			Zone:           next.zone,
			Confidence:     roundFloat(next.detection.Confidence),
			Reason:         fmt.Sprintf("成熟度为可采，位于 %s，且相对当前位置更近。", next.zone),
		})

		current = next.center
		remaining = remaining[1:]
	}

	return items, totalDistance
}

func bboxCenter(bbox [4]float64) point {
	return point{
		x: (bbox[0] + bbox[2]) / 2,
		y: (bbox[1] + bbox[3]) / 2,
	}
}

func zoneFromCenter(center point, frameWidth int, frameHeight int) string {
	x := clamp(center.x, 0, float64(frameWidth))
	y := clamp(center.y, 0, float64(frameHeight))

	column := 0
	switch {
	case x >= float64(frameWidth)*2/3:
		column = 2
	case x >= float64(frameWidth)/3:
		column = 1
	}

	row := 0
	switch {
	case y >= float64(frameHeight)*2/3:
		row = 2
	case y >= float64(frameHeight)/3:
		row = 1
	}

	rows := []string{"upper", "middle", "lower"}
	cols := []string{"left", "center", "right"}
	return rows[row] + "_" + cols[column]
}

func clamp(value float64, min float64, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func pointDistance(a point, b point) float64 {
	return math.Hypot(b.x-a.x, b.y-a.y)
}

func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}
