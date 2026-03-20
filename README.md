# IRacing-Display

An IRacing display over local network

- Ingest takes the data out of Iracing and sends it over the local network to the telemtry service
- The telemetry then formats this data and saves it
- The dashboard then displays the data that is stored in the telemtry service

## Infra

- Golang or C# data ingest
- RabbitMQ pub/sub
- C# telemetry service
- QuestDB database
- NextJS dashboard

### Sources

[IRacing data ingest](https://github.com/hiimkyle/vr2c20)

## Current stats
- Can ingest 1.3 million telemetry points in 9s with the golang ingest

## Next step

- [x] Add all the telemetry data to one bucket per track
- [x] look at better running in parallel
- [] Handle session num 0 meaning practice
- [] Create a store on the device to know what files have already been sent
- [] Better filter non ibt files

