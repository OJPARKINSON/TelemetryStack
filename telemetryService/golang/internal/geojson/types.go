package geojson

// GeoJSON Standard Types (RFC 7946 compliant)
type FeatureCollection struct {
	Type     string                 `json:"type"`
	Features []Feature              `json:"features"`
	Metadata map[string]interface{} `json:"metadata"`
}

type Feature struct {
	Type       string                 `json:"type"`
	Geometry   Geometry               `json:"geometry"`
	Properties map[string]interface{} `json:"properties"`
}

type Geometry struct {
	Type        string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

type Position []float64

// Conversion settings
type ConversionOptions struct {
	tolerance       float64
	SimplifyEnabled bool
	MinPoints       int
	MaxPoints       int

	StyleMetric  StyleMetric
	SegmentMode  SegmentMode
	ColourScheme ColourScheme
	ColourSteps  int

	IncludeRawData      bool
	PropertiesToInclude []string
}

type StyleMetric string

const (
	StyleMetricSpeed     StyleMetric = "speed"
	StyleMetricThrottle  StyleMetric = "throttle"
	StyleMetricBrake     StyleMetric = "brake"
	StyleMetricLatAccel  StyleMetric = "lat_accel"
	StyleMetricLongAccel StyleMetric = "long_accel"
	StyleMetricCombined  StyleMetric = "combined"
)

type SegmentMode string

const (
	SegmentModeGradient  SegmentMode = "gradient"
	SegmentModeThreshold SegmentMode = "threshold"
	SegmentModeQuantile  SegmentMode = "quantile"
)

type ColourScheme string

const (
	ColorSchemeSpeed    ColourScheme = "speed"    // Blue→Yellow→Red
	ColorSchemeThrottle ColourScheme = "throttle" // Red→Green
	ColorSchemeBrake    ColourScheme = "brake"    // Green→Red
	ColorSchemeGForce   ColourScheme = "gforce"   // Blue→Purple
	ColorSchemeViridis  ColourScheme = "viridis"  // Perceptually uniform
	ColorSchemeTurbo    ColourScheme = "turbo"    // High contrast
)
