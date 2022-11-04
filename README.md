# gps-stats

`gps-stats` is a command-line tool that can read and analyze GPS data in a
`SBN` format.

Multiple files can be analyzed at once.

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

## Example usage

Here are few example runs of the gps-stats app:
```
$ gps-stats ../gps-data/VVidovic_113200915_20221014_140124.SBN
Found 9341 track points in 'VVidovic_113200915_20221014_140124.SBN', after cleanup 9110 points left.
Total Distance:     48.589 km
2 Second Peak:      17.665 kts (2 sec, 18.176 m, 2022-10-14 14:40:37 +0000 UTC)
5x10 Average:       16.689 kts
  Top 1 5x10 speed: 17.144 kts (10 sec, 88.194 m, 2022-10-14 14:40:35 +0000 UTC)
  Top 2 5x10 speed: 16.715 kts (10 sec, 85.989 m, 2022-10-14 14:36:22 +0000 UTC)
  Top 3 5x10 speed: 16.679 kts (10 sec, 85.802 m, 2022-10-14 14:48:27 +0000 UTC)
  Top 4 5x10 speed: 16.621 kts (10 sec, 85.506 m, 2022-10-14 14:41:48 +0000 UTC)
  Top 5 5x10 speed: 16.285 kts (10 sec, 83.776 m, 2022-10-14 14:33:12 +0000 UTC)
15 Min:             12.522 kts (900 sec, 5797.652 m, 2022-10-14 14:34:22 +0000 UTC)
1 Hr:               11.404 kts (3600 sec, 21120.870 m, 2022-10-14 14:17:55 +0000 UTC)
100m peak:          16.984 kts (12 sec, 104.850 m, 2022-10-14 14:40:34 +0000 UTC)
Nautical Mile:      13.801 kts (261 sec, 1853.104 m, 2022-10-14 14:35:47 +0000 UTC)
Alpha 500:          14.378 kts (29 sec, 214.498 m, 2022-10-14 14:48:26 +0000 UTC)

$ go run gps-stats.go -t=alpha ../gps-data/VVidovic_113200915_20221014_140124.SBN
14.381 (VVidovic_113200915_20221014_140124.SBN)

$ go run gps-stats.go -t=alpha ../gps-data/*.SBN | sort
...
14.509 (VVidovic_113200915_20220912_125830.SBN)
14.572 (VVidovic_113200915_20220905_111830.SBN)
14.772 (VVidovic_113200915_20220806_112219.SBN)
15.638 (VVidovic_113200915_20220902_151000.SBN)
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
