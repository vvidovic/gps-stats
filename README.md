# gps-stats

`gps-stats` is a command-line tool that can read, analyze and clean GPS data in a
`SBN` or `GPX` format.

Multiple files can be analyzed at once.

Units of speed is kts (default), m/s or km/h.

Results are the following stats:
- Total Distance
- 2 Second Peak (optionally, a series of 2 seconds peaks)
- 5x10 Average
- Top 5 5x10 speeds
- 15 Min
- 1 Hr
- 100m peak
- Nautical Mile
- Alpha 500
- Delta 500 (Tack 500 m, calculated only when starboard and port stats are separated)

## Usage flags

```sh
Usage:
 gps-stats [Flags] GPS_data_file1 [GPS_data_file2 ...]

Parses 1 or more GPS data files (SBN or GPX)

Flags:
  -h Show usage (optional)
  -v Show version (optional)
  -t Set the statistics type to print (optional, default all)
     (all, dist, dur, 2s, 10sAvg, 10s1, 10s2, 10s3, 10s4, 10s5, 15m, 1h, 100m, 1nm, alpha)
  -wd Set the wind direction in degrees (0-360, degree from where it comes from) (optional)
  -awd Auto-detect wind direction (optional, default jibe)
       (jibe, tack)
  -su Set the speed units to print (optional, default kts)
      (kts, kmh, ms)
  -s2s Set the number of 2 sec best speeds to print, can be used only if all speeds are calculated
      (integer number, for example: 10)
  -sf Save filtered points as a new GPX file without points detected as errors
      with suffix '.filtered.gpx' (optional)

  -cs Clean up points where speed changes are more than given number of speed units (default 5 kts)
      Calculation uses 4 points. It calculates 3 speeds based on those points.
      After that, 2 speed changes are calculated and difference between those changes is
      used to filter points.

  -amazfit Adjust algorithm for Amazfit T-Rex Pro watch tracks.
           With tracks where there are almost all points (each 1 sec) but some are missing,
           we remove points around the missing ones and it helps to improve accurracy.

  -speed-runs-details Show detailed analysis of top speed runs
  -speed-runs-details-num Set the number of top speed runs to analyze (default 5)
  -speed-runs-details-secs Set the analysis window and duplicate filtering interval in seconds (default 10)

  -d Show debug information (each detected turn details)


Examples:
 gps-stats my_gps_data.SBN
   - runs analysis of the SBN data

 gps-stats -cs 7 my_gps_data.gpx
   - runs analysis of the SBN data with custom clean up settings

 gps-stats -t=1nm *.SBN *.gpx
   - runs analysis of multiple SBN & GPX data only for 1 NM statistics

 gps-stats -sf my_gps_data.GPX
   - runs analysis of the GPX data and save a copy of track with filtered points detected as errors
```

## Example usage

Here are few example runs of the gps-stats app:
```sh
$ gps-stats -sf ../gps-data/VVidovic_113200915_20221014_140124.SBN
Filtered GPX file '../gps-data/VVidovic_113200915_20221014_140124.SBN.filtered.gpx' saved.

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

Speed Runs Details:
  Settings:
    Runs shown:       5
    Analysis window:  10 sec
    Thresholds:       peak-1 / peak-2 / peak-3

  Run 1:
    Peak:              17.663 kts (2 sec, 18.174 m, 2022-10-14 14:40:37 +0000 UTC, starboard)
    Position:          43.928492, 15.418879 → 43.927781, 15.418391
    Heading:           206.4° avg, ±1.4° stddev, 204.1°–208.5° range
    10s around peak:   17.142 kts (10 sec, 88.188 m, 2022-10-14 14:40:35 +0000 UTC, starboard)
    10s speeds:        16.8 → 17.4 → [17.7] → 17.6 → 17.3 → 17.2 → 17.1 → 16.9 → 16.7 → 16.7
    10s headings:      206° → 205° → 204° → 204° → 206° → 207° → 207° → 208° → 208° → 209°
    Heading evolution: 205.0° → 207.7° (+2.7°, +0.44°/s)
    Acceleration:      15.9 → 17.7 kts in 5.0 sec / 44.1 m (0.35 kts/s)
    Above thresholds:  >16.7 kts: 10.0 sec / 88.2 m, >15.7 kts: 17.0 sec / 146.3 m, >14.7 kts: 18.0 sec / 154.2 m
    Stability Score:   92/100 (heading 37/40, retention 35/35, acceleration 20/25)

  Run 2:
    Peak:              17.442 kts (2 sec, 17.946 m, 2022-10-14 14:36:27 +0000 UTC, port)
    Position:          43.937031, 15.417816 → 43.937411, 15.418747
    Heading:           059.7° avg, ±5.4° stddev, 052.1°–067.7° range
    10s around peak:   16.729 kts (10 sec, 86.064 m, 2022-10-14 14:36:22 +0000 UTC, port)
    10s speeds:        16.3 → 16.6 → 16.7 → 16.9 → 17.3 → [17.4] → [17.4] → 16.5 → 16.0 → 16.1
    10s headings:      54° → 55° → 54° → 55° → 60° → 64° → 66° → 64° → 65° → 68°
    Heading evolution: 55.7° → 65.3° (+9.5°, +1.65°/s)
    Acceleration:      15.7 → 17.4 kts in 13.0 sec / 110.6 m (0.13 kts/s)
    Above thresholds:  >16.4 kts: 7.0 sec / 61.2 m, >15.4 kts: 23.0 sec / 192.3 m, >14.4 kts: 25.0 sec / 207.5 m
    Stability Score:   66/100 (heading 15/40, retention 31/35, acceleration 20/25)

  Run 3:
    Peak:              17.216 kts (2 sec, 17.714 m, 2022-10-14 14:41:55 +0000 UTC, port)
    Position:          43.928004, 15.420211 → 43.928399, 15.421126
    Heading:           059.5° avg, ±2.5° stddev, 055.4°–063.8° range
    10s around peak:   16.635 kts (10 sec, 85.577 m, 2022-10-14 14:41:48 +0000 UTC, port)
    10s speeds:        15.6 → 16.1 → 16.4 → 16.6 → 16.7 → 16.9 → 17.1 → 17.2 → [17.3] → 16.6
    10s headings:      61° → 61° → 61° → 60° → 58° → 56° → 55° → 58° → 62° → 59°
    Heading evolution: 60.2° → 58.0° (-2.2°, -0.26°/s)
    Acceleration:      15.5 → 17.2 kts in 9.0 sec / 77.1 m (0.19 kts/s)
    Above thresholds:  >16.2 kts: 8.0 sec / 69.3 m, >15.2 kts: 11.0 sec / 93.5 m, >14.2 kts: 32.0 sec / 256.2 m
    Stability Score:   90/100 (heading 31/40, retention 34/35, acceleration 25/25)

  Run 4:
    Peak:              17.215 kts (2 sec, 17.712 m, 2022-10-14 14:48:32 +0000 UTC, starboard)
    Position:          43.926364, 15.430365 → 43.925693, 15.429840
    Heading:           209.5° avg, ±3.9° stddev, 198.3°–214.1° range
    10s around peak:   16.679 kts (10 sec, 85.803 m, 2022-10-14 14:48:27 +0000 UTC, starboard)
    10s speeds:        15.5 → 16.1 → 16.5 → 17.0 → [17.2] → [17.2] → [17.2] → 17.1 → 17.0 → 15.8
    10s headings:      214° → 211° → 210° → 209° → 208° → 209° → 211° → 212° → 209° → 198°
    Heading evolution: 210.7° → 208.1° (-2.7°, -0.85°/s)
    Acceleration:      15.5 → 17.2 kts in 7.0 sec / 60.1 m (0.25 kts/s)
    Above thresholds:  >16.2 kts: 7.0 sec / 61.4 m, >15.2 kts: 10.0 sec / 85.8 m, >14.2 kts: 32.0 sec / 261.7 m
    Stability Score:   77/100 (heading 24/40, retention 35/35, acceleration 18/25)

  Run 5:
    Peak:              17.107 kts (2 sec, 17.601 m, 2022-10-14 14:42:44 +0000 UTC, port)
    Position:          43.929907, 15.424378 → 43.930397, 15.425158
    Heading:           048.8° avg, ±4.3° stddev, 042.0°–056.5° range
    10s around peak:   16.175 kts (10 sec, 83.210 m, 2022-10-14 14:42:38 +0000 UTC, port)
    10s speeds:        15.5 → 15.6 → 15.5 → 15.6 → 16.2 → 16.8 → [17.1] → [17.1] → 16.4 → 15.7
    10s headings:      46° → 47° → 43° → 42° → 46° → 50° → 52° → 53° → 53° → 56°
    Heading evolution: 44.9° → 52.9° (+8.0°, +1.33°/s)
    Acceleration:      15.4 → 17.1 kts in 16.0 sec / 131.7 m (0.11 kts/s)
    Above thresholds:  >16.1 kts: 5.0 sec / 43.1 m, >15.1 kts: 20.0 sec / 163.9 m, >14.1 kts: 36.0 sec / 290.3 m
    Stability Score:   64/100 (heading 21/40, retention 25/35, acceleration 18/25)
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
