package api

type StationStatus struct {
	Username     string `json:"username"`
	IsLive       bool   `json:"isLive"`
	SegmentCount int    `json:"segmentCount"`
	IsRelaying   bool   `json:"isRelaying"`
}

type PublicStation struct {
	Username     string `json:"username"`
	Name         string `json:"name"`
	Summary      string `json:"summary"`
	IsLive       bool   `json:"isLive"`
	SegmentCount int    `json:"segmentCount"`
	HLSURL       string `json:"hlsUrl"`
}
