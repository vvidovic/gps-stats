# gps-stats

`gps-stats` is a command-line tool that can read and analyze GPS data in a
`SBN` or `GPX` format.

Multiple files can be analyzed at once.

Units of speed is kts (default), m/s or km/h.

Results are the following stats:
- Total Distance
- 2 Second Peak
- 5x10 Average
- Top 5 5x10 speeds
- 15 Min
- 1 Hr
- 100m peak
- Nautical Mile
- Alpha 500
- Delta 500 (Tack 500 m, calculated only when starboard and port stats are separated)

## Example usage

Here are few example runs of the gps-stats app:
```
$ gps-stats ../gps-data/VVidovic_113200915_20221014_140124.SBN
Found 9341 track points in 'VVidovic_113200915_20221014_140124.SBN', after cleanup 9110 points left.
Total Distance:     48.610 km
Total Duration:     02.675 h
2 Second Peak:      17.663 kts (2 sec, 18.174 m, 2022-10-14 14:40:37 +0000 UTC)
5x10 Average:       16.693 kts
  Top 1 5x10 speed: 17.142 kts (10 sec, 88.188 m, 2022-10-14 14:40:35 +0000 UTC)
  Top 2 5x10 speed: 16.729 kts (10 sec, 86.064 m, 2022-10-14 14:36:22 +0000 UTC)
  Top 3 5x10 speed: 16.679 kts (10 sec, 85.803 m, 2022-10-14 14:48:27 +0000 UTC)
  Top 4 5x10 speed: 16.635 kts (10 sec, 85.577 m, 2022-10-14 14:41:48 +0000 UTC)
  Top 5 5x10 speed: 16.281 kts (10 sec, 83.758 m, 2022-10-14 14:33:12 +0000 UTC)
15 Min:             12.525 kts (900 sec, 5799.196 m, 2022-10-14 14:34:22 +0000 UTC)
1 Hr:               11.409 kts (3600 sec, 21130.351 m, 2022-10-14 14:17:55 +0000 UTC)
100m peak:          16.983 kts (12 sec, 104.844 m, 2022-10-14 14:40:34 +0000 UTC)
Nautical Mile:      13.804 kts (261 sec, 1853.402 m, 2022-10-14 14:35:47 +0000 UTC)
Alpha 500:          14.381 kts (29 sec, 214.553 m, 2022-10-14 14:48:26 +0000 UTC)

$ gps-stats -su kmh ../gps-data/VVidovic_113200915_20221014_140124.SBN
Found 9341 track points in 'VVidovic_113200915_20221014_140124.SBN', after cleanup 9027 points left.
Total Distance:     48.431 km
Total Duration:     02.672 h
2 Second Peak:      32.712 kmh (2 sec, 18.174 m, 2022-10-14 14:40:37 +0000 UTC)
5x10 Average:       30.916 kmh
  Top 1 5x10 speed: 31.748 kmh (10 sec, 88.188 m, 2022-10-14 14:40:35 +0000 UTC)
  Top 2 5x10 speed: 30.983 kmh (10 sec, 86.064 m, 2022-10-14 14:36:22 +0000 UTC)
  Top 3 5x10 speed: 30.889 kmh (10 sec, 85.803 m, 2022-10-14 14:48:27 +0000 UTC)
  Top 4 5x10 speed: 30.808 kmh (10 sec, 85.577 m, 2022-10-14 14:41:48 +0000 UTC)
  Top 5 5x10 speed: 30.153 kmh (10 sec, 83.758 m, 2022-10-14 14:33:12 +0000 UTC)
15 Min:             23.105 kmh (900 sec, 5776.151 m, 2022-10-14 14:34:22 +0000 UTC)
1 Hr:               21.034 kmh (3600 sec, 21034.273 m, 2022-10-14 14:17:55 +0000 UTC)
100m peak:          31.453 kmh (12 sec, 104.844 m, 2022-10-14 14:40:34 +0000 UTC)
Nautical Mile:      25.564 kmh (261 sec, 1853.402 m, 2022-10-14 14:35:47 +0000 UTC)
Alpha 500:          26.634 kmh (29 sec, 214.553 m, 2022-10-14 14:48:26 +0000 UTC)

$ gps-stats -t=alpha ../gps-data/VVidovic_113200915_20221014_140124.SBN
14.381 (VVidovic_113200915_20221014_140124.SBN)

$ gps-stats -t=alpha ../gps-data/*.SBN | sort
...
14.509 kts (64 sec, 477.689 m, 2022-09-12 13:49:12 +0000 UTC) (VVidovic_113200915_20220912_125830.SBN)
14.572 kts (21 sec, 157.422 m, 2022-09-05 12:56:40 +0000 UTC) (VVidovic_113200915_20220905_111830.SBN)
14.772 kts (41 sec, 311.573 m, 2022-08-06 13:19:12 +0000 UTC) (VVidovic_113200915_20220806_112219.SBN)
15.638 kts (62 sec, 498.796 m, 2022-09-02 15:26:05 +0000 UTC) (VVidovic_113200915_20220902_151000.SBN)

$ gps-stats ../gps-data/VVidovic_113200915_20221014_140124.SBN
Found 9341 track points in 'VVidovic_113200915_20221014_140124.SBN', after cleanup 9027 points left.
Total Distance:     48.431 km
Total Duration:     02.672 h
Wind Direction:     325.000
Jibes Count:        67
Tacks Count:        33
2 Second Peak:      17.663 kts (2 sec, 18.174 m, 2022-10-14 14:40:37 +0000 UTC, starboard)
5x10 Average:       16.693 kts
  Top 1 5x10 speed: 17.142 kts (10 sec, 88.188 m, 2022-10-14 14:40:35 +0000 UTC, starboard)
  Top 2 5x10 speed: 16.729 kts (10 sec, 86.064 m, 2022-10-14 14:36:22 +0000 UTC, port)
  Top 3 5x10 speed: 16.679 kts (10 sec, 85.803 m, 2022-10-14 14:48:27 +0000 UTC, starboard)
  Top 4 5x10 speed: 16.635 kts (10 sec, 85.577 m, 2022-10-14 14:41:48 +0000 UTC, port)
  Top 5 5x10 speed: 16.281 kts (10 sec, 83.758 m, 2022-10-14 14:33:12 +0000 UTC, starboard)
15 Min:             12.475 kts (900 sec, 5776.151 m, 2022-10-14 14:34:22 +0000 UTC)
1 Hr:               11.358 kts (3600 sec, 21034.273 m, 2022-10-14 14:17:55 +0000 UTC)
100m peak:          16.983 kts (12 sec, 104.844 m, 2022-10-14 14:40:34 +0000 UTC, starboard)
Nautical Mile:      13.804 kts (261 sec, 1853.402 m, 2022-10-14 14:35:47 +0000 UTC)
Alpha 500:          14.381 kts (29 sec, 214.553 m, 2022-10-14 14:48:26 +0000 UTC, starboard)
Delta 500:          11.623 kts (21 sec, 125.565 m, 2022-10-14 15:10:28 +0000 UTC, port)

Starboard 2s:       17.663 kts (2 sec, 18.174 m, 2022-10-14 14:40:37 +0000 UTC, starboard)
Starboard 5x10s:    16.508 kts
  Top 1 5x10 speed: 17.142 kts (10 sec, 88.188 m, 2022-10-14 14:40:35 +0000 UTC, starboard)
  Top 2 5x10 speed: 16.679 kts (10 sec, 85.803 m, 2022-10-14 14:48:27 +0000 UTC, starboard)
  Top 3 5x10 speed: 16.281 kts (10 sec, 83.758 m, 2022-10-14 14:33:12 +0000 UTC, starboard)
  Top 4 5x10 speed: 16.264 kts (10 sec, 83.669 m, 2022-10-14 14:46:46 +0000 UTC, starboard)
  Top 5 5x10 speed: 16.176 kts (10 sec, 83.218 m, 2022-10-14 15:20:41 +0000 UTC, starboard)
Starboard 100m:     16.983 kts (12 sec, 104.844 m, 2022-10-14 14:40:34 +0000 UTC, starboard)
Starboard Alpha:    14.381 kts (29 sec, 214.553 m, 2022-10-14 14:48:26 +0000 UTC, starboard)
Starboard Delta:    11.362 kts (21 sec, 122.750 m, 2022-10-14 15:11:28 +0000 UTC, starboard)

Port 2s:            17.442 kts (2 sec, 17.946 m, 2022-10-14 14:36:27 +0000 UTC, port)
Port 5x10s:         16.388 kts
  Top 1 5x10 speed: 16.729 kts (10 sec, 86.064 m, 2022-10-14 14:36:22 +0000 UTC, port)
  Top 2 5x10 speed: 16.635 kts (10 sec, 85.577 m, 2022-10-14 14:41:48 +0000 UTC, port)
  Top 3 5x10 speed: 16.233 kts (10 sec, 83.510 m, 2022-10-14 14:29:54 +0000 UTC, port)
  Top 4 5x10 speed: 16.175 kts (10 sec, 83.210 m, 2022-10-14 14:42:38 +0000 UTC, port)
  Top 5 5x10 speed: 16.167 kts (10 sec, 83.172 m, 2022-10-14 14:30:05 +0000 UTC, port)
Port 100m:          16.617 kts (12 sec, 102.582 m, 2022-10-14 14:36:21 +0000 UTC, port)
Port Alpha:         13.759 kts (40 sec, 283.123 m, 2022-10-14 15:01:23 +0000 UTC, port)
Port Delta:         11.623 kts (21 sec, 125.565 m, 2022-10-14 15:10:28 +0000 UTC, port)
```

## Build

Build should be done from project directory.

Local build:
```sh
go build gps-stats.go
```

Local cross-platform build with version flags and stripping of debug information
(should be done after properly tagging version with `git tag`):
```sh
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X 'github.com/vvidovic/gps-stats/internal/version.Version=$(git tag | tail -n1)' -X 'github.com/vvidovic/gps-stats/internal/version.Platform=windows/amd64' -X 'github.com/vvidovic/gps-stats/internal/version.BuildTime=$(git tag | tail -n1).$(date -u -Iseconds)'" -o release/gps-stats-win-amd64.exe gps-stats.go
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X 'github.com/vvidovic/gps-stats/internal/version.Version=$(git tag | tail -n1)' -X 'github.com/vvidovic/gps-stats/internal/version.Platform=darwin/amd64' -X 'github.com/vvidovic/gps-stats/internal/version.BuildTime=$(git tag | tail -n1).$(date -u -Iseconds)'" -o release/gps-stats-mac-amd64 gps-stats.go
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X 'github.com/vvidovic/gps-stats/internal/version.Version=$(git tag | tail -n1)' -X 'github.com/vvidovic/gps-stats/internal/version.Platform=linux/amd64' -X 'github.com/vvidovic/gps-stats/internal/version.BuildTime=$(git tag | tail -n1).$(date -u -Iseconds)'" -o release/gps-stats-linux-amd64 gps-stats.go
```

After building binary it can be compressed using excellent
[UPX](https://upx.github.io/) command:
```sh
upx --lzma release/*
```
