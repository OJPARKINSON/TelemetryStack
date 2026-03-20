package geojson

import (
	"math"

	"github.com/ojparkinson/telemetryService/internal/messaging"
)

func ConvertToGeoJSON(lapData []messaging.Telemetry, options ConversionOptions) (*FeatureCollection, error) {
	minSpeed := math.MaxFloat32
	maxSpeed := 0.0
	for _, lap := range lapData {
		speed := lap.Speed
		if speed < minSpeed {
			minSpeed = speed
		}

		if speed > maxSpeed {
			maxSpeed = speed
		}
	}

	type ColourPosition struct {
		Positions [][]float64
		colour    string
	}

	coords := make([]ColourPosition, 0)
	for _, lap := range lapData {

		normalisedSpeed := (lap.Speed - minSpeed) / (maxSpeed - minSpeed)

		var colour string
		switch true {
		case normalisedSpeed < 0.3:
			colour = "#ef4444"
		case normalisedSpeed < 0.6:
			colour = "#f97316"
		case normalisedSpeed < 0.8:
			colour = "#eab308"
		default:
			colour = "#22c55e"
		}

		if len(coords) > 0 && colour == coords[len(coords)-1].colour {
			coords[len(coords)-1].Positions = append(coords[len(coords)-1].Positions, []float64{lap.Lon, lap.Lat})
		} else {
			coords = append(coords, ColourPosition{
				Positions: [][]float64{{lap.Lon, lap.Lat}},
				colour:    colour,
			})
		}
	}

	features := make([]Feature, 0, len(coords))

	for _, coord := range coords {
		if len(coord.Positions) > 0 {
			features = append(features, Feature{
				Type: "Feature",
				Geometry: Geometry{
					Type:        "LineString",
					Coordinates: coord.Positions,
				},
				Properties: map[string]interface{}{
					"color": coord.colour,
				},
			})
		}
	}

	featureCollection := &FeatureCollection{
		Type:     "FeatureCollection",
		Features: features,
		Metadata: map[string]interface{}{},
	}

	return featureCollection, nil
}
