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
