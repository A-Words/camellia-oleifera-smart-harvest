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
	SkipReasonManual        = "manual_skip"

	TreeRecommendationReady              = "ready"
	TreeRecommendationPendingObservation = "pending_observation"
)

type Engine struct{}

func NewEngine() *Engine {
	return &Engine{}
}

func (e *Engine) Recommend(decisionID string, createdAt string, req SnapshotRequest) (RecommendationResponse, error) {
	treeRecommendation, err := e.analyzeTree("", "", nil, req)
	if err != nil {
		return RecommendationResponse{}, err
	}

	return RecommendationResponse{
		DecisionID:     decisionID,
		CreatedAt:      createdAt,
		Summary:        treeRecommendation.Summary,
		ZonePriorities: treeRecommendation.ZonePriorities,
		PickSequence:   treeRecommendation.PickSequence,
		SkipItems:      treeRecommendation.SkipItems,
	}, nil
}

func (e *Engine) BuildPlan(planID string, generatedAt string, plotID string, inputs []latestObservation) (DecisionPlan, error) {
	ready := make([]treePlanCandidate, 0, len(inputs))
	recommendations := make([]TreeRecommendation, 0, len(inputs))
	totalHarvestable := 0

	for _, input := range inputs {
		if input.Tree.Status != "active" {
			continue
		}

		if input.Observation == nil {
			recommendations = append(recommendations, TreeRecommendation{
				TreeID:         input.Tree.TreeID,
				TreeCode:       input.Tree.TreeCode,
				ObservationID:  nil,
				Status:         TreeRecommendationPendingObservation,
				Summary:        Summary{},
				ZonePriorities: []ZonePriority{},
				PickSequence:   []PickSequenceItem{},
				SkipItems:      []SkipItem{},
			})
			continue
		}

		recommendation, err := e.analyzeTree(input.Tree.TreeID, input.Tree.TreeCode, &input.Observation.ObservationID, SnapshotRequest{
			FrameIndex:  input.Observation.FrameIndex,
			TimestampMS: input.Observation.TimestampMS,
			FrameWidth:  input.Observation.FrameWidth,
			FrameHeight: input.Observation.FrameHeight,
			Detections:  input.Observation.Detections,
		})
		if err != nil {
			return DecisionPlan{}, fmt.Errorf("analyze tree %s: %w", input.Tree.TreeID, err)
		}

		recommendations = append(recommendations, recommendation)
		totalHarvestable += recommendation.Summary.HarvestableCount
		ready = append(ready, treePlanCandidate{
			tree:             input.Tree,
			recommendation:   recommendation,
			weightedScore:    harvestableConfidenceScore(recommendation),
			harvestableCount: recommendation.Summary.HarvestableCount,
		})
	}

	sequence := buildTreeSequence(ready)

	return DecisionPlan{
		PlanID:         planID,
		PlotID:         plotID,
		GeneratedAt:    generatedAt,
		ManualOverride: false,
		ManualOverrideState: ManualOverride{
			TreeSequence:  []string{},
			TreeOverrides: []TreeOverride{},
		},
		Summary: PlanSummary{
			TotalTrees:              len(inputs),
			ReadyTrees:              len(sequence),
			PendingObservationTrees: countPendingRecommendations(recommendations),
			TotalHarvestableCount:   totalHarvestable,
		},
		TreeSequence:        sequence,
		TreeRecommendations: recommendations,
	}, nil
}

func (e *Engine) ApplyManualOverride(plan DecisionPlan, trees map[string]TreeMeta, request UpdatePlanRequest) (DecisionPlan, error) {
	next := plan
	next.ManualOverride = true
	next.ManualOverrideState = ManualOverride{
		TreeSequence:  append([]string(nil), request.TreeSequence...),
		TreeOverrides: append([]TreeOverride(nil), request.TreeOverrides...),
	}

	if len(request.TreeSequence) > 0 {
		sequenceMap := map[string]PlanTreeSequenceItem{}
		for _, item := range next.TreeSequence {
			sequenceMap[item.TreeID] = item
		}

		reordered := make([]PlanTreeSequenceItem, 0, len(next.TreeSequence))
		used := map[string]bool{}
		for _, treeID := range request.TreeSequence {
			item, ok := sequenceMap[treeID]
			if !ok {
				return DecisionPlan{}, fmt.Errorf("tree_sequence contains unknown tree_id %q", treeID)
			}
			reordered = append(reordered, item)
			used[treeID] = true
		}
		for _, item := range next.TreeSequence {
			if !used[item.TreeID] {
				reordered = append(reordered, item)
			}
		}
		recomputeTreeSequenceDistances(reordered, trees)
		next.TreeSequence = reordered
	}

	recommendationIndex := map[string]int{}
	for index, item := range next.TreeRecommendations {
		recommendationIndex[item.TreeID] = index
	}

	for _, override := range request.TreeOverrides {
		index, ok := recommendationIndex[override.TreeID]
		if !ok {
			return DecisionPlan{}, fmt.Errorf("tree_overrides contains unknown tree_id %q", override.TreeID)
		}

		recommendation := next.TreeRecommendations[index]
		recommendation.ZonePriorities = reorderZonePriorities(recommendation.ZonePriorities, override.ZoneOrder)
		recommendation.PickSequence, recommendation.SkipItems = reorderAndSkipPickSequence(
			recommendation.PickSequence,
			recommendation.SkipItems,
			override.PickSequenceDetectionIndices,
			override.ManualSkippedDetectionIndices,
		)
		recommendation.Summary.HarvestableCount = len(recommendation.PickSequence)
		recommendation.Summary.SkippedCount = len(recommendation.SkipItems)
		if len(recommendation.ZonePriorities) > 0 {
			recommendation.Summary.MainPriorityZone = &recommendation.ZonePriorities[0].Zone
		}
		next.TreeRecommendations[index] = recommendation
	}

	next.Summary.TotalHarvestableCount = 0
	next.Summary.PendingObservationTrees = countPendingRecommendations(next.TreeRecommendations)
	next.Summary.ReadyTrees = 0
	for _, item := range next.TreeRecommendations {
		if item.Status == TreeRecommendationReady {
			next.Summary.ReadyTrees++
		}
		next.Summary.TotalHarvestableCount += item.Summary.HarvestableCount
	}

	return next, nil
}

type treePlanCandidate struct {
	tree             TreeMeta
	recommendation   TreeRecommendation
	weightedScore    float64
	harvestableCount int
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

func (e *Engine) analyzeTree(treeID string, treeCode string, observationID *string, req SnapshotRequest) (TreeRecommendation, error) {
	if req.FrameWidth <= 0 || req.FrameHeight <= 0 {
		return TreeRecommendation{}, errors.New("frame_width and frame_height must be greater than 0")
	}

	startPoint := point{
		x: float64(req.FrameWidth) / 2,
		y: float64(req.FrameHeight),
	}

	candidates := make([]candidate, 0, len(req.Detections))
	skips := make([]SkipItem, 0, len(req.Detections))

	for index, detection := range req.Detections {
		if err := validateDetection(detection); err != nil {
			return TreeRecommendation{}, fmt.Errorf("detection %d: %w", index, err)
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

	return TreeRecommendation{
		TreeID:        treeID,
		TreeCode:      treeCode,
		ObservationID: observationID,
		Status:        TreeRecommendationReady,
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
	case SkipReasonManual:
		message = "该目标被人工调整为跳过。"
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
			Note:             fmt.Sprintf("该区域有 %d 枚可采目标，且距离当前动作起点较近。", stat.count),
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

func buildTreeSequence(candidates []treePlanCandidate) []PlanTreeSequenceItem {
	if len(candidates) == 0 {
		return []PlanTreeSequenceItem{}
	}

	remaining := append([]treePlanCandidate(nil), candidates...)
	current := point{x: 0, y: 0}
	sequence := make([]PlanTreeSequenceItem, 0, len(candidates))

	for len(remaining) > 0 {
		sort.Slice(remaining, func(i, j int) bool {
			if remaining[i].harvestableCount != remaining[j].harvestableCount {
				return remaining[i].harvestableCount > remaining[j].harvestableCount
			}
			if remaining[i].weightedScore != remaining[j].weightedScore {
				return remaining[i].weightedScore > remaining[j].weightedScore
			}
			leftDistance := pointDistance(current, point{x: remaining[i].tree.X, y: remaining[i].tree.Y})
			rightDistance := pointDistance(current, point{x: remaining[j].tree.X, y: remaining[j].tree.Y})
			if leftDistance != rightDistance {
				return leftDistance < rightDistance
			}
			return remaining[i].tree.TreeCode < remaining[j].tree.TreeCode
		})

		next := remaining[0]
		distance := pointDistance(current, point{x: next.tree.X, y: next.tree.Y})
		sequence = append(sequence, PlanTreeSequenceItem{
			TreeID:                             next.tree.TreeID,
			TreeCode:                           next.tree.TreeCode,
			PriorityOrder:                      len(sequence) + 1,
			HarvestableCount:                   next.harvestableCount,
			HarvestableConfidenceWeightedScore: roundFloat(next.weightedScore),
			DistanceFromPrevious:               roundFloat(distance),
			Status:                             TreeRecommendationReady,
		})
		current = point{x: next.tree.X, y: next.tree.Y}
		remaining = remaining[1:]
	}

	return sequence
}

func recomputeTreeSequenceDistances(sequence []PlanTreeSequenceItem, trees map[string]TreeMeta) {
	current := point{x: 0, y: 0}
	for index := range sequence {
		sequence[index].PriorityOrder = index + 1
		tree, ok := trees[sequence[index].TreeID]
		if !ok {
			sequence[index].DistanceFromPrevious = 0
			continue
		}
		distance := pointDistance(current, point{x: tree.X, y: tree.Y})
		sequence[index].DistanceFromPrevious = roundFloat(distance)
		current = point{x: tree.X, y: tree.Y}
	}
}

func reorderZonePriorities(items []ZonePriority, zoneOrder []string) []ZonePriority {
	if len(zoneOrder) == 0 {
		return items
	}

	ordered := make([]ZonePriority, 0, len(items))
	indexByZone := map[string]ZonePriority{}
	for _, item := range items {
		indexByZone[item.Zone] = item
	}

	used := map[string]bool{}
	for _, zone := range zoneOrder {
		item, ok := indexByZone[zone]
		if !ok {
			continue
		}
		ordered = append(ordered, item)
		used[zone] = true
	}
	for _, item := range items {
		if !used[item.Zone] {
			ordered = append(ordered, item)
		}
	}
	for index := range ordered {
		ordered[index].Order = index + 1
	}
	return ordered
}

func reorderAndSkipPickSequence(items []PickSequenceItem, skipItems []SkipItem, order []int, manualSkip []int) ([]PickSequenceItem, []SkipItem) {
	if len(order) == 0 && len(manualSkip) == 0 {
		return items, skipItems
	}

	itemByDetectionIndex := map[int]PickSequenceItem{}
	for _, item := range items {
		itemByDetectionIndex[item.DetectionIndex] = item
	}

	manualSkipSet := map[int]bool{}
	for _, index := range manualSkip {
		manualSkipSet[index] = true
	}

	reordered := make([]PickSequenceItem, 0, len(items))
	used := map[int]bool{}
	for _, detectionIndex := range order {
		item, ok := itemByDetectionIndex[detectionIndex]
		if !ok || manualSkipSet[detectionIndex] {
			continue
		}
		reordered = append(reordered, item)
		used[detectionIndex] = true
	}
	for _, item := range items {
		if used[item.DetectionIndex] || manualSkipSet[item.DetectionIndex] {
			continue
		}
		reordered = append(reordered, item)
	}
	for index := range reordered {
		reordered[index].Order = index + 1
	}

	nextSkipItems := append([]SkipItem(nil), skipItems...)
	for _, item := range items {
		if !manualSkipSet[item.DetectionIndex] {
			continue
		}
		nextSkipItems = append(nextSkipItems, SkipItem{
			DetectionIndex: item.DetectionIndex,
			TrackID:        item.TrackID,
			Zone:           item.Zone,
			SkipReason:     SkipReasonManual,
			Confidence:     item.Confidence,
			Reason:         "该目标被人工调整为跳过。",
		})
	}

	return reordered, nextSkipItems
}

func harvestableConfidenceScore(recommendation TreeRecommendation) float64 {
	score := 0.0
	for _, item := range recommendation.PickSequence {
		score += item.Confidence
	}
	return roundFloat(score)
}

func countPendingRecommendations(items []TreeRecommendation) int {
	count := 0
	for _, item := range items {
		if item.Status == TreeRecommendationPendingObservation {
			count++
		}
	}
	return count
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
