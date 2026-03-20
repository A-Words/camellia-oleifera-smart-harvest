package decision

type Detection struct {
	BBox       [4]float64 `json:"bbox"`
	ClassName  string     `json:"class_name"`
	Confidence float64    `json:"confidence"`
	TrackID    *int       `json:"track_id"`
	Ripeness   *string    `json:"ripeness"`
}

type SnapshotRequest struct {
	FrameIndex  int         `json:"frame_index"`
	TimestampMS int64       `json:"timestamp_ms"`
	FrameWidth  int         `json:"frame_width"`
	FrameHeight int         `json:"frame_height"`
	Detections  []Detection `json:"detections"`
}

type Summary struct {
	TotalDetections     int     `json:"total_detections"`
	HarvestableCount    int     `json:"harvestable_count"`
	SkippedCount        int     `json:"skipped_count"`
	MainPriorityZone    *string `json:"main_priority_zone"`
	EstimatedPathLength float64 `json:"estimated_path_length"`
}

type ZonePriority struct {
	Order            int     `json:"order"`
	Zone             string  `json:"zone"`
	HarvestableCount int     `json:"harvestable_count"`
	DistanceToStart  float64 `json:"distance_to_start"`
	Note             string  `json:"note"`
}

type PickSequenceItem struct {
	Order          int     `json:"order"`
	DetectionIndex int     `json:"detection_index"`
	TrackID        *int    `json:"track_id"`
	Zone           string  `json:"zone"`
	Confidence     float64 `json:"confidence"`
	Reason         string  `json:"reason"`
}

type SkipItem struct {
	DetectionIndex int     `json:"detection_index"`
	TrackID        *int    `json:"track_id"`
	Zone           string  `json:"zone"`
	SkipReason     string  `json:"skip_reason"`
	Confidence     float64 `json:"confidence"`
	Reason         string  `json:"reason"`
}

type RecommendationResponse struct {
	DecisionID     string             `json:"decision_id"`
	CreatedAt      string             `json:"created_at"`
	Summary        Summary            `json:"summary"`
	ZonePriorities []ZonePriority     `json:"zone_priorities"`
	PickSequence   []PickSequenceItem `json:"pick_sequence"`
	SkipItems      []SkipItem         `json:"skip_items"`
}

type HistoryItem struct {
	DecisionID     string             `json:"decision_id"`
	CreatedAt      string             `json:"created_at"`
	Request        SnapshotRequest    `json:"request"`
	Summary        Summary            `json:"summary"`
	ZonePriorities []ZonePriority     `json:"zone_priorities"`
	PickSequence   []PickSequenceItem `json:"pick_sequence"`
	SkipItems      []SkipItem         `json:"skip_items"`
}

type HistoryResponse struct {
	Items []HistoryItem `json:"items"`
}

type ObservationRequest struct {
	TreeID      string      `json:"tree_id"`
	CapturedAt  string      `json:"captured_at"`
	FrameIndex  int         `json:"frame_index"`
	TimestampMS int64       `json:"timestamp_ms"`
	FrameWidth  int         `json:"frame_width"`
	FrameHeight int         `json:"frame_height"`
	Detections  []Detection `json:"detections"`
}

type Observation struct {
	ObservationID string      `json:"observation_id"`
	TreeID        string      `json:"tree_id"`
	CapturedAt    string      `json:"captured_at"`
	FrameIndex    int         `json:"frame_index"`
	TimestampMS   int64       `json:"timestamp_ms"`
	FrameWidth    int         `json:"frame_width"`
	FrameHeight   int         `json:"frame_height"`
	Detections    []Detection `json:"detections"`
}

type ObservationListResponse struct {
	Items []Observation `json:"items"`
}

type PlanRequest struct {
	PlotID string `json:"plot_id"`
}

type PlanSummary struct {
	TotalTrees              int `json:"total_trees"`
	ReadyTrees              int `json:"ready_trees"`
	PendingObservationTrees int `json:"pending_observation_trees"`
	TotalHarvestableCount   int `json:"total_harvestable_count"`
}

type PlanTreeSequenceItem struct {
	TreeID                             string  `json:"tree_id"`
	TreeCode                           string  `json:"tree_code"`
	PriorityOrder                      int     `json:"priority_order"`
	HarvestableCount                   int     `json:"harvestable_count"`
	HarvestableConfidenceWeightedScore float64 `json:"harvestable_confidence_weighted_score"`
	DistanceFromPrevious               float64 `json:"distance_from_previous"`
	Status                             string  `json:"status"`
}

type TreeRecommendation struct {
	TreeID         string             `json:"tree_id"`
	TreeCode       string             `json:"tree_code"`
	ObservationID  *string            `json:"observation_id"`
	Status         string             `json:"status"`
	Summary        Summary            `json:"summary"`
	ZonePriorities []ZonePriority     `json:"zone_priorities"`
	PickSequence   []PickSequenceItem `json:"pick_sequence"`
	SkipItems      []SkipItem         `json:"skip_items"`
}

type ManualOverride struct {
	TreeSequence  []string       `json:"tree_sequence"`
	TreeOverrides []TreeOverride `json:"tree_overrides"`
}

type DecisionPlan struct {
	PlanID              string                 `json:"plan_id"`
	PlotID              string                 `json:"plot_id"`
	GeneratedAt         string                 `json:"generated_at"`
	ManualOverride      bool                   `json:"manual_override"`
	ManualOverrideState ManualOverride         `json:"manual_override_state"`
	Summary             PlanSummary            `json:"summary"`
	TreeSequence        []PlanTreeSequenceItem `json:"tree_sequence"`
	TreeRecommendations []TreeRecommendation   `json:"tree_recommendations"`
}

type DecisionPlanListResponse struct {
	Items []DecisionPlan `json:"items"`
}

type TreeOverride struct {
	TreeID                        string   `json:"tree_id"`
	ZoneOrder                     []string `json:"zone_order"`
	PickSequenceDetectionIndices  []int    `json:"pick_sequence_detection_indices"`
	ManualSkippedDetectionIndices []int    `json:"manual_skipped_detection_indices"`
}

type UpdatePlanRequest struct {
	TreeSequence  []string       `json:"tree_sequence"`
	TreeOverrides []TreeOverride `json:"tree_overrides"`
}

type TreeMeta struct {
	TreeID   string
	TreeCode string
	PlotID   string
	RowIndex int
	ColIndex int
	X        float64
	Y        float64
	Status   string
}

type latestObservation struct {
	Tree        TreeMeta
	Observation *Observation
}
