# Racing line

Displays the line the driver took around the track. There can be different colour overlays, the top ones are, speed, throttle and brake. This clearly shows what inputs the driver made. 

Although the racing line is a key visual part it's only one piece of the puzzle for allowing a race engineer to analyse the lap data. So to advance the user experience, the user should be able to hover over any part of the racing line and it will put a point on the racing line and it will save the the point in a hook which the charts on the page will use and show a line and dot where the racing line point is on the chart. This allows users to see the values at each and every point. 

## Implementation

As part of the map libre migration I want to review how this is done, mainly doing a lot more of the calculations on the golang backend. To do that I need to sure up the interfaces for the endpoints. Currently, I have the racing line being shown on the map. So in theory it just needs colour.


## Future prospects

In the future I want to be able to compare different laps on the same page, so it will require two racing lines and two lines on each chart, when you hover it will show the point and the values on each chart so it's easy to compare for the race engineer.

I also want to look at giving the user the option of selecting a sector, so s1, s2, s3. that then allows the user to first see the sector time difference and then focus in one where their driver was either faster or slower. Selecting a sector will zoom the user slightly closer and reframe to having a centre of the middle of the sector, this can be done through getting the coordinates of that sector and getting the cords from the middle. The chart should also focus in on just the data from that sector.

## Tech notes

The frontend must be smooth to use when transitioning. It's important to do heavy computation on the BE/DB as they are more optimised for doing it. Make sure that we aren't sending data that is not needed to the frontend.