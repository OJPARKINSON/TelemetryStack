# Racing line

Displays the line the driver took around the track. There can be different colour overlays, the top ones are, speed, throttle and brake. This clearly shows what inputs the driver made.

## Implementation

As part of the map libre migration I want to review how this is done, mainly doing a lot more of the calculations on the golang backend. To do that I need to sure up the interfaces for the endpoints. Currently, I have the racing line being shown on the map. So in theory it just needs colour.

## Racing line with colour (1st phase)

Look at options for adding colour to the racing line, there should be a way through geojsonData. Spike it out more:

So we need a new endpoint that creates a to spec geoJSON, as a start we can just return the colour based on speed

## Racing line hover with charts (2nd phase)
