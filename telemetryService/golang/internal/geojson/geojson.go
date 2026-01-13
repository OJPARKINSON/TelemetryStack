package geojson

import (
	"fmt"
	"math"
)

func ConvertToGeoJSON(lapData []map[string]interface{}, options ConversionOptions) (*FeatureCollection, error) {

	// currentSpeedGroup := 0.6
	coords := make([]Position, 0, len(lapData))
	for _, lap := range lapData {
		lon, _ := extractFloat64(lap, "lon")
		lat, _ := extractFloat64(lap, "lat")

		coords = append(coords, Position{lon, lat})
	}

	feature := Feature{
		Type: "Feature",
		Geometry: Geometry{
			Type:        "lineString",
			Coordinates: coords,
		},
		Properties: map[string]interface{}{
			"color": "#FF0000",
		},
	}

	featureCollection := &FeatureCollection{
		Type:     "Feature",
		Features: []Feature{feature},
		Metadata: map[string]interface{}{},
	}

	return featureCollection, nil
}

func extractFloat64(row map[string]interface{}, key string) (float64, error) {
	val, ok := row[key]
	if !ok {
		return 0, fmt.Errorf("key %s not found", key)
	}

	switch v := val.(type) {
	case float64:
		return validateFloat64(v), nil
	case float32:
		return validateFloat64(float64(v)), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", val)
	}
}

func validateFloat64(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.0
	}
	return value
}
