# End to end

## Intro

I want to be able to test and benchmark the whole racing telemetry system from the breaking down of files to checking at the data has been stored without any regression. This allows me to make changes knowing that there hasn't been any regression and I can see what speeds to expect.

## Challenges

Currently, the ingest system works as a CLI and isn't containerised yet. This isn't a big challenge but would be good to have for testing. Next we need to setup testing containers either hand rolled or through TestContainers, to be reviewed. Then it would be nice to have this in a make file. Running in the pipeline would be overkill for this project but it would be perfect at a bigger scale.
