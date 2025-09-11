package stats

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
	"time"

	"github.com/vvidovic/gps-stats/internal/errs"
)

const (
	// Earth radius from Wikipedia
	// SI base unit	   6.3781×106 m[1]
	// Metric system	   6,357 to 6,378 km
	earthRadius      = 6370000  // Earth Radius in meters
	mPerSecToKts     = 1.94384  // Number of kts in 1 m/s
	mPerSecToKmh     = 3.6      // Number of km/h in 1 m/s
	earthCircPoles   = 40007863 // Earth Circumference around poles
	earthCircEquator = 40075017 // Earth Circumference around equator
)

// StatFlag shows which statistics are we calculating/printing.
type StatFlag int64

// StatFlag shows which statistics are we calculating/printing.
const (
	StatNone StatFlag = iota
	StatAll
	StatDistance
	StatDuration
	Stat2s
	Stat10sAvg
	Stat10s1
	Stat10s2
	Stat10s3
	Stat10s4
	Stat10s5
	Stat15m
	Stat1h
	Stat100m
	Stat1nm
	StatAlpha
)

// UnitsFlag shows which speed units are we printing.
type UnitsFlag int64

// UnitsFlag shows which speed units are we printing.
const (
	UnitsMs UnitsFlag = iota
	UnitsKmh
	UnitsKts
)

func (u UnitsFlag) String() string {
	unitsName := "ms"
	switch u {
	case UnitsMs:
		unitsName = "ms"
	case UnitsKmh:
		unitsName = "kmh"
	case UnitsKts:
		unitsName = "kts"
	}

	return unitsName
}

// TrackType defines type of track file.
type TrackType int64

const (
	TrackSbn TrackType = iota
	TrackGpx
	TrackUnknown
)

// Points represent all GPS points from our GPS data
type Points struct {
	Creator string
	Name    string
	Type    string
	Ps      []Point
}

// Point represent one GPS point with timestamp.
type Point struct {
	isPoint    bool
	valid      bool
	validCheck bool
	ele        float64
	lat        float64
	lon        float64
	ts         time.Time
	usedFor10s bool
	globalIdx  int
	speed      *float64 // MetersPerSecond_t: This type contains a speed measured in meters per second.
	hr         *int16   // BeatsPerMinute_t: This type contains a heart rate measured in beats per minute.
	tackSide   TackSide // Starboard / Port: if wind direction is known
	heading    float64  // Angle from the North (North == 0, East == 90)
}

func (p Point) String() string {
	return fmt.Sprintf("{%v/%v %.1f° (%v)}", p.lat, p.lon, p.heading, p.ts)
}

// TurnType is used to distinguish between a turn type, can be jibe or tack.
type TurnType int64

// TurnType is used to distinguish between a turn type, can be jibe or tack.
const (
	TurnUnknown TurnType = iota
	TurnJibe
	TurnTack
)

func (t TurnType) String() string {
	turnName := "unknown"
	switch t {
	case TurnJibe:
		turnName = "jibe"
	case TurnTack:
		turnName = "tack"
	}

	return turnName
}

// TackSide is used to distinguis between a tack side (riding side),
// can be starboard or port tack.
type TackSide int64

const (
	TackUnknown TackSide = iota
	TackStarboard
	TackPort
)

func (t TackSide) String() string {
	switch t {
	case TackStarboard:
		return "starboard"
	case TackPort:
		return "port"
	default:
		return "unknown"
	}
}

// Track is a collection of points and can contain sum of durations,
//
//	sum of calculated distances and calculated speed.
//
// valid field can be used to mark that the Track is valid for the statistic
//
//	we are currently preparing.
type Track struct {
	ps         []Point
	duration   float64
	distance   float64
	speed      float64
	speedUnits UnitsFlag
	valid      bool
}

// TackSide returns TackSide based on the TackSide of the first point
func (t Track) TackSide() TackSide {
	if len(t.ps) > 0 {
		return t.ps[0].tackSide
	}
	return TackUnknown
}

// TxtLine display human-readable entry for each track.
func (t Track) TxtLine() string {
	var timestamp time.Time
	if len(t.ps) > 0 {
		timestamp = t.ps[0].ts
	}
	tackSideString := ""
	if t.TackSide() != TackUnknown {
		tackSideString = fmt.Sprintf(", %v", t.TackSide())
	}
	return fmt.Sprintf("%06.3f %s (%0.0f sec, %06.3f m, %v%s)",
		t.speed, t.speedUnits, t.duration, t.distance, timestamp, tackSideString)
}
func (t Track) String() string {
	l := len(t.ps)
	if l > 0 {
		return fmt.Sprintf("dur: %v, dist: %.3f, speed: %.3f, ps: %v - %v\n",
			t.duration, t.distance, t.speed, t.ps[0], t.ps[l-1])
	}
	return fmt.Sprintf("dur: %v, dist: %.3f, speed: %.3f, ps: -\n",
		t.duration, t.distance, t.speed)
}

// reCalculate sums durations and distanes from points and calculates
// speed from those.
func (t Track) reCalculate() Track {
	t.duration = 0
	t.distance = 0
	t.speed = 0
	for i := 0; i < len(t.ps)-1; i++ {
		t.duration += t.ps[i+1].ts.Sub(t.ps[i].ts).Seconds()
		t.distance += distance(t.ps[i], t.ps[i+1])
	}
	if t.duration > 0 {
		t.speed = MsToUnits(t.distance/t.duration, t.speedUnits)
	}

	return t
}

// addPointMinDuration
//   - add a new Point to the end of the Track
//   - ensures the Track is no shorter than minDuration (removing Points from the
//     beginning of the Track if possible)
func (t Track) addPointMinDuration(p Point, minDuration float64) Track {
	return t.addPointMinDurationUnused10s(p, minDuration, false)
}

// addPointMinDurationUnused10s
//   - start a new track if unused10sOnly is true and the current point is used or
//   - add a new Point to the end of the Track
//   - ensures the Track is no shorter than minDuration (removing Points from the
//     beginning of the Track if possible)
func (t Track) addPointMinDurationUnused10s(
	p Point, minDuration float64, unused10sOnly bool) Track {
	if unused10sOnly && p.usedFor10s {
		return Track{speedUnits: t.speedUnits}
	}
	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])
		t.speed = MsToUnits(t.distance/t.duration, t.speedUnits)
		t.valid = t.duration >= minDuration

		// Let's check if we can remove some points from the start of this track.
		// If duration is not at minimum and we have some points to remove...
		if t.duration > minDuration && len(t.ps) > 2 {
			durTest := t.duration - t.ps[1].ts.Sub(t.ps[0].ts).Seconds()
			for durTest >= minDuration && len(t.ps) > 2 {
				t.duration = durTest
				t.distance = t.distance - distance(t.ps[0], t.ps[1])
				t.ps = t.ps[1:]
				durTest = t.duration - t.ps[1].ts.Sub(t.ps[0].ts).Seconds()
			}
			t.speed = MsToUnits(t.distance/t.duration, t.speedUnits)
		}
	}

	return t
}

// addPointMinDistance
//   - add a new Point to the end of the Track
//   - ensures the Track is no shorter than minDistance (removing Points from the
//     beginning of the Track if possible)
func (t Track) addPointMinDistance(p Point, minDistance float64) Track {
	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])
		t.speed = MsToUnits(t.distance/t.duration, t.speedUnits)
		t.valid = t.distance >= minDistance

		// Let's check if we can remove some points from the start of this track.
		// If duration is not at minimum and we have some points to remove...
		if t.distance > minDistance && len(t.ps) > 2 {
			distTest := t.distance - distance(t.ps[0], t.ps[1])
			for distTest >= minDistance && len(t.ps) > 2 {
				t.distance = distTest
				t.duration = t.duration - t.ps[1].ts.Sub(t.ps[0].ts).Seconds()
				t.ps = t.ps[1:]
				distTest = t.distance - distance(t.ps[0], t.ps[1])
			}
			t.speed = MsToUnits(t.distance/t.duration, t.speedUnits)
		}
	}

	return t
}

// addPointTurn500
//   - add a new Point to the end of the Track for Alpha 500 m calculation
//   - ensures the Track is as close but no longer than 500 m
//   - try to find the subtrack that contains alpha for entry/exit gate max 50 m
//   - return two Tracks: "this" Track and subtrack containing best alpha
//     (as described above)
func (t Track) addPointTurn500(p Point) (Track, Track) {
	return t.addPointTurnMaxDistance(p, 500, 10, 50)
}

// addPointTurnMaxDistance
//   - add a new Point to the end of the Track for Turn calculation
//   - ensures the Track is as close but no longer than maxDistance (removing
//     Points from the beginning of the Track if possible)
//   - ensures the Track doesn't contain multiple turns (heading changes)
//   - try to find the subtrack that is no shorter than minDistance (to ensure
//     this is alpha and no riding straight) and that the first and the last point
//     are ate most gateSize away
//   - return two Tracks: "this" Track and subtrack containing best alpha
//     (as described above)
func (t Track) addPointTurnMaxDistance(p Point,
	maxDistance, minDistance, gateSize float64) (Track, Track) {
	// Don't add static or near upwind/downwind points (no heading)
	// to the beginning of track.
	if len(t.ps) == 0 && p.heading < 0 {
		return t, Track{speedUnits: t.speedUnits}
	}

	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		firstPointTackSide := t.ps[0].tackSide
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])

		// 1. Do we need to remove some points from the start of this track
		//    because the track is too long?
		//    - find a track with length most close to the maxDistance
		//      and ensure the first point has a known tack side
		if t.distance > maxDistance && l > 2 {
			distTest := t.distance - distance(t.ps[0], t.ps[1])
			for distTest > maxDistance && l > 2 && t.ps[0].tackSide != TackUnknown {
				t.distance = distTest
				t.duration = t.duration - t.ps[1].ts.Sub(t.ps[0].ts).Seconds()
				t.ps = t.ps[1:]
				l = len(t.ps)
				distTest = t.distance - distance(t.ps[0], t.ps[1])
			}
			t.distance = distTest
			t.duration = t.duration - t.ps[1].ts.Sub(t.ps[0].ts).Seconds()
			t.ps = t.ps[1:]
			l = len(t.ps)
		}

		// 2. The last point must be the opposite tack side
		// from the first point tack side.
		lastPointTackSide := t.ps[l-1].tackSide
		for lastPointTackSide == TackUnknown && l > 2 {
			t.ps = t.ps[0 : l-1]
			l--
			lastPointTackSide = t.ps[l-1].tackSide
		}

		// 3. Ensure that we did change direction
		if lastPointTackSide != TackUnknown && lastPointTackSide != firstPointTackSide {
			// fmt.Printf("  ====> whole track (firstPointTackSide: %v): %v", firstPointTackSide, t)
			// 4. Can we find a gate, maybe by removing some points from the start?
			// Distance between the first and the last point must be max gateSize (50m).
			subtrackDistance := t.distance
			for i := 0; i < l-2; i++ {
				// fmt.Printf("  ====> subt dist: %.2f (min: %.2f), p: %v\n", subtrackDistance, minDistance, t.ps[i])
				if subtrackDistance < minDistance {
					break
				}
				// The first point of the subrack must be on the same tack as the first point of the whole track.
				if t.ps[i].tackSide != firstPointTackSide {
					break
				}
				// The first point must be on the clear tack (not near upwind/downwind).
				if t.ps[i].tackSide == TackUnknown {
					subtrackDistance = subtrackDistance - distance(t.ps[i], t.ps[i+1])
					continue
				}
				gateDistance := distance(t.ps[i], t.ps[l-1])
				// fmt.Printf("    ====> ? gate dist: %.2f (max: %.2f), subtrackDistance: %.2f (min: %.2f), %s - %s\n", gateDistance, gateSize, subtrackDistance, minDistance, t.ps[i].ts, t.ps[l-1].ts)
				if gateDistance <= gateSize && subtrackDistance >= minDistance {
					// fmt.Printf("    ====> ! gate dist: %.2f (max: %.2f), subtrackDistance: %.2f (min: %.2f), %s - %s\n", gateDistance, gateSize, subtrackDistance, minDistance, t.ps[i].ts, t.ps[l-1].ts)
					// fmt.Printf("    ====> gate dist: %.2f (max: %.2f), subtrackDistance: %.2f (min: %.2f), %s - %s\n", gateDistance, gateSize, subtrackDistance, minDistance, t.ps[i].ts, t.ps[l-1].ts)
					subtrack := Track{ps: t.ps[i:], valid: true, speedUnits: t.speedUnits}.reCalculate()
					// fmt.Printf("    ====> real - gate dist: %.2f, subtrackDistance: %.2f\n", distance(subtrack.ps[0], subtrack.ps[len(subtrack.ps)-1]), subtrack.distance)
					return t, subtrack
				}
				subtrackDistance = subtrackDistance - distance(t.ps[i], t.ps[i+1])
			}
		}
	}

	return t, Track{speedUnits: t.speedUnits}
}

// WindDirectionStats contains statistics for a specific wind direction
type WindDirectionStats struct {
	windDirection       float64
	jibesCount          int
	tacksCount          int
	alpha500m           Track
	delta500m           Track
	starboardSpeed2s    Track
	starboardSpeed5x10s []Track
	starboardSpeed100m  Track
	starboardAlpha500m  Track
	starboardDelta500m  Track
	portSpeed2s         Track
	portSpeed5x10s      []Track
	portSpeed100m       Track
	portAlpha500m       Track
	portDelta500m       Track
}

// Stats constains calculated statistics.
type Stats struct {
	totalDistance  float64
	totalDuration  float64
	unknTurnsCount int
	speed2s        Track
	speed5x10s     []Track
	speed15m       Track
	speed1h        Track
	speed100m      Track
	speed1NM       Track
	alpha500m      Track
	speedUnits     UnitsFlag
	wDirKnown      bool
	wDirStats      WindDirectionStats
}

func (s Stats) AllTurnsCount() int {
	return s.unknTurnsCount + s.wDirStats.jibesCount + s.wDirStats.tacksCount
}

// TxtSingleStat returns a single statistic.
func (s Stats) TxtSingleStat(statType StatFlag) string {
	switch statType {
	case Stat2s:
		return s.speed2s.TxtLine()
	case StatDistance:
		return fmt.Sprintf("%06.3f km", s.totalDistance/1000)
	case StatDuration:
		return fmt.Sprintf("%06.3f h", s.totalDuration)
	case Stat10sAvg:
		return fmt.Sprintf("%06.3f", CalcTracksAvg(s.speed5x10s))
	case Stat10s1:
		return s.speed5x10s[0].TxtLine()
	case Stat10s2:
		return s.speed5x10s[1].TxtLine()
	case Stat10s3:
		return s.speed5x10s[2].TxtLine()
	case Stat10s4:
		return s.speed5x10s[3].TxtLine()
	case Stat10s5:
		return s.speed5x10s[4].TxtLine()
	case Stat15m:
		return s.speed15m.TxtLine()
	case Stat1h:
		return s.speed1h.TxtLine()
	case Stat100m:
		return s.speed100m.TxtLine()
	case Stat1nm:
		return s.speed1NM.TxtLine()
	case StatAlpha:
		return s.alpha500m.TxtLine()
	}
	return ""
}

// TxtStats formats statistics as a human-readable text.
func (s Stats) TxtStats() string {
	var b strings.Builder
	fmt.Fprintf(&b, "Total Distance:     %06.3f km\n", s.totalDistance/1000)
	fmt.Fprintf(&b, "Total Duration:     %06.3f h\n", s.totalDuration)

	if s.wDirKnown {
		fmt.Fprintf(&b, "Wind Direction:     %06.3f\n", s.wDirStats.windDirection)
	} else {
		fmt.Fprintf(&b, "Wind Dir +/- 180°:  %06.3f\n", s.wDirStats.windDirection)
	}

	if s.wDirKnown {
		fmt.Fprintf(&b, "Unkn turns Count:   %d\n", s.unknTurnsCount)
		fmt.Fprintf(&b, "Jibes Count:        %d\n", s.wDirStats.jibesCount)
		fmt.Fprintf(&b, "Tacks Count:        %d\n", s.wDirStats.tacksCount)
	} else {
		fmt.Fprintf(&b, "Turns Count:        %d\n", s.unknTurnsCount+s.wDirStats.jibesCount+s.wDirStats.tacksCount)
	}

	fmt.Fprintf(&b, "2 Second Peak:      %s\n", s.speed2s.TxtLine())
	fmt.Fprintf(&b, "5x10 Average:       %06.3f %s\n", CalcTracksAvg(s.speed5x10s), s.speedUnits)
	fmt.Fprintf(&b, "  Top 1 5x10 speed: %s\n", s.speed5x10s[0].TxtLine())
	fmt.Fprintf(&b, "  Top 2 5x10 speed: %s\n", s.speed5x10s[1].TxtLine())
	fmt.Fprintf(&b, "  Top 3 5x10 speed: %s\n", s.speed5x10s[2].TxtLine())
	fmt.Fprintf(&b, "  Top 4 5x10 speed: %s\n", s.speed5x10s[3].TxtLine())
	fmt.Fprintf(&b, "  Top 5 5x10 speed: %s\n", s.speed5x10s[4].TxtLine())
	fmt.Fprintf(&b, "15 Min:             %s\n", s.speed15m.TxtLine())
	fmt.Fprintf(&b, "1 Hr:               %s\n", s.speed1h.TxtLine())
	fmt.Fprintf(&b, "100m peak:          %s\n", s.speed100m.TxtLine())
	fmt.Fprintf(&b, "Nautical Mile:      %s\n", s.speed1NM.TxtLine())
	fmt.Fprintf(&b, "Alpha 500:          %s\n", s.alpha500m.TxtLine())

	if s.wDirKnown {
		fmt.Fprintf(&b, "Delta 500:          %s\n", s.wDirStats.delta500m.TxtLine())
		fmt.Fprintf(&b, "\n")
		fmt.Fprintf(&b, "Starboard 2s:       %s\n", s.wDirStats.starboardSpeed2s.TxtLine())
		fmt.Fprintf(&b, "Starboard 5x10s:    %06.3f %s\n", CalcTracksAvg(s.wDirStats.starboardSpeed5x10s), s.speedUnits)
		fmt.Fprintf(&b, "  Top 1 5x10 speed: %s\n", s.wDirStats.starboardSpeed5x10s[0].TxtLine())
		fmt.Fprintf(&b, "  Top 2 5x10 speed: %s\n", s.wDirStats.starboardSpeed5x10s[1].TxtLine())
		fmt.Fprintf(&b, "  Top 3 5x10 speed: %s\n", s.wDirStats.starboardSpeed5x10s[2].TxtLine())
		fmt.Fprintf(&b, "  Top 4 5x10 speed: %s\n", s.wDirStats.starboardSpeed5x10s[3].TxtLine())
		fmt.Fprintf(&b, "  Top 5 5x10 speed: %s\n", s.wDirStats.starboardSpeed5x10s[4].TxtLine())
		fmt.Fprintf(&b, "Starboard 100m:     %s\n", s.wDirStats.starboardSpeed100m.TxtLine())
		fmt.Fprintf(&b, "Starboard Alpha:    %s\n", s.wDirStats.starboardAlpha500m.TxtLine())
		fmt.Fprintf(&b, "Starboard Delta:    %s\n", s.wDirStats.starboardDelta500m.TxtLine())
		fmt.Fprintf(&b, "\n")
		fmt.Fprintf(&b, "Port 2s:            %s\n", s.wDirStats.portSpeed2s.TxtLine())
		fmt.Fprintf(&b, "Port 5x10s:         %06.3f %s\n", CalcTracksAvg(s.wDirStats.portSpeed5x10s), s.speedUnits)
		fmt.Fprintf(&b, "  Top 1 5x10 speed: %s\n", s.wDirStats.portSpeed5x10s[0].TxtLine())
		fmt.Fprintf(&b, "  Top 2 5x10 speed: %s\n", s.wDirStats.portSpeed5x10s[1].TxtLine())
		fmt.Fprintf(&b, "  Top 3 5x10 speed: %s\n", s.wDirStats.portSpeed5x10s[2].TxtLine())
		fmt.Fprintf(&b, "  Top 4 5x10 speed: %s\n", s.wDirStats.portSpeed5x10s[3].TxtLine())
		fmt.Fprintf(&b, "  Top 5 5x10 speed: %s\n", s.wDirStats.portSpeed5x10s[4].TxtLine())
		fmt.Fprintf(&b, "Port 100m:          %s\n", s.wDirStats.portSpeed100m.TxtLine())
		fmt.Fprintf(&b, "Port Alpha:         %s\n", s.wDirStats.portAlpha500m.TxtLine())
		fmt.Fprintf(&b, "Port Delta:         %s\n", s.wDirStats.portDelta500m.TxtLine())
	}
	return b.String()
}

func (s Stats) String() string {
	return fmt.Sprintf(
		"dist: %v\n  2s: %v\n  5x10s: %v\n  %v\n  15m: %v\n  1h: %v\n  100m: %v\n  1NM: %v\n  alpha: %v\n",
		s.totalDistance, s.speed2s, CalcTracksAvg(s.speed5x10s), s.speed5x10s,
		s.speed15m, s.speed1h,
		s.speed100m, s.speed1NM, s.alpha500m)
}

// Calculate []Track average speed.
func CalcTracksAvg(tracks []Track) float64 {
	res := 0.0
	for i := 0; i < len(tracks); i++ {
		res += tracks[i].speed
	}
	res = res / float64(len(tracks))
	return res
}

// intFrom2ub converts 2 unsigned bytes to int.
func intFrom2ub(b2 []byte) int {
	return int(b2[0])*256 + int(b2[1])
}

// intFrom4sb converts 4 signed bytes to int.
func intFrom4sb(b4 []byte) int {
	if b4[0]&0x80 != 0 {
		return int(b4[0]&0x7F)*256*256*256 + int(b4[1])*256*256 + int(b4[2])*256 + int(b4[3])
	}
	return int(b4[0])*256*256*256 + int(b4[1])*256*256 + int(b4[2])*256 + int(b4[3])
}

// ReadPoints read all Points from the Reader.
func ReadPoints(r io.Reader) (Points, error) {
	tt := determineType(r)

	switch tt {
	case TrackSbn:
		return ReadPointsSbn(r)
	case TrackGpx:
		return ReadPointsGpx(r)
	default:
		return Points{Ps: []Point{}}, errs.Errorf("Unknown track type (%v).", tt)
	}
}

func determineType(r io.Reader) TrackType {
	br := bufio.NewReaderSize(r, 100)
	startBytes, _ := br.Peek(100)

	if len(startBytes) >= 4 {
		// 160 162 0 34 253 86 86 105 100 111 118
		if bytes.Equal(startBytes[0:4], []byte{160, 162, 0, 34}) {
			return TrackSbn
		}
		// 60 63 120 109 108 32 118 101 114 115 105
		if bytes.Equal(startBytes[0:6], []byte("<?xml ")) {
			return TrackGpx
		}
	}

	return TrackUnknown
}

// speed calculate speed as a result of moving between two Points.
func speed(p1, p2 Point, speedUnits UnitsFlag) float64 {
	d := distance(p1, p2)
	dt := p2.ts.Sub(p1.ts)

	speed := MsToUnits(d/dt.Seconds(), speedUnits)

	return speed
}

// distance calculates a distance between two Points.
func distance(p1, p2 Point) float64 {
	return distSimple(p1.lat, p1.lon, p2.lat, p2.lon)
}

// sq calculate square of a float64 number.
func sq(n float64) float64 {
	return n * n
}

// // dist calculate a distance between two points by their lattitudes and
// // longitudes.
// func dist(lat1, lon1, lat2, lon2 float64) float64 {
// 	dLat := (lat2 - lat1) * math.Pi / 180
// 	dLon := (lon2 - lon1) * math.Pi / 180
// 	a := sq(math.Sin(dLat/2)) +
// 		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*sq(math.Sin(dLon/2))
// 	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
// 	return earthRadius * c
// }

// distSimple calculate a distance between two points by their lattitudes and
// longitudes, ignoring curvature of the earth surface (small distances).
func distSimple(lat1, lon1, lat2, lon2 float64) float64 {
	dLatM := (lat2 - lat1) / 360 * earthCircPoles
	dLonM := (lon2 - lon1) / 360 * earthCircEquator * math.Cos((lat1+lat2)/2*math.Pi/180)

	return math.Sqrt(sq(dLatM) + sq(dLonM))
}

// headiheadingSimpleng returns heading in degrees from p1 to p2 (0 = North, 90 = East)
// ignoring curvature of the earth surface (small distances).
func headingSimple(lat1, lon1, lat2, lon2 float64) float64 {
	dLatM := (lat2 - lat1) / 360 * earthCircPoles
	dLonM := (lon2 - lon1) / 360 * earthCircEquator * math.Cos((lat1+lat2)/2*math.Pi/180)

	dist := math.Sqrt(sq(dLatM) + sq(dLonM))

	heading := -1.0
	// If distance between 2 points is small, heading is unknown.
	if dist > 0.5 {
		// angle in the normal coordinate system (0 = East, 90 = North)
		angleNormalCoordSys := math.Atan2(dLatM, dLonM) * 180 / math.Pi
		angle := -angleNormalCoordSys + 90
		// fmt.Printf("dLonM: %.3f, dLatM: %.3f, an: %.3f, ah: %.3f\n", dLonM, dLatM, angleNormalCoordSys, angle)

		heading = math.Mod(angle+360, 360)
	}
	// fmt.Printf("  ====> dist: %.2f (%.2f, %.2f), heading: %.2f\n", d, dLatM, dLonM, heading)

	return heading
}

// CleanUp removes points that seems not valid.
func CleanUp(points Points, deltaSpeedMax float64, speedUnits UnitsFlag) []Point {
	psCurr := points.Ps
	res := []Point{}
	if len(psCurr) > 1 {
		// Simple cleanup strategies working great for Amazfit T-Rex Pro:
		// - if points have same timestamp, remove both points
		// - removing points "around" missing points (1 before, 3 after)
		//
		// When we find missing point(s):
		// - remove 1 point before the first missing point
		// - remove 3 points after the last missing point
		//
		// For example, we should have seconds:
		// - 43, 44, 45, 46, 47. 48, 49, 50, 51, 52, 53, 54
		// There are only:
		// - 43, 44, 45, 46,     48,     50, 51, 52, 53, 54
		// We need to produce:
		// - 43, 44, 45,         48,                 53, 54
		psCleaned := []Point{}
		psCleaned = append(psCleaned, psCurr[0])
		psLen := len(psCurr)
		for idxPs := 1; idxPs < psLen; idxPs++ {
			pCurr := psCurr[idxPs]

			if idxPs < psLen-1 {
				pNext := psCurr[idxPs+1]
				// fmt.Printf("curr / next ts: %v / %v, next - curr: %v\n", pCurr.ts, pNext.ts, pNext.ts.Sub(pCurr.ts).Seconds())
				if pCurr.ts == pNext.ts {
					// Skip both points if times are equal.
					idxPs++
					// fmt.Printf("====> skipping curr & next: %v & %v\n", pCurr, pNext)
				} else {
					// Remove points "around" missing points.
					// Missing point is point more than 1 second after previous point.
					dt := pNext.ts.Sub(pCurr.ts).Seconds()
					if dt > 1 {
						idxNext := idxPs + 1
						idxLast := idxNext
						// fmt.Printf("====> dt > 1, idxPs, idxNext, idxLast, pNext: %v, %v, %v, %v\n", idxPs, idxNext, idxLast, pNext)
						for idxNext < psLen-1 && dt > 1 {
							p1 := psCurr[idxNext]
							p2 := psCurr[idxNext+1]
							dt = p2.ts.Sub(p1.ts).Seconds()
							idxLast = idxNext
							idxNext++
							// fmt.Printf("====> dt: %v, idxPs, idxNext, idxLast: %v, %v, %v\n", dt, idxPs, idxNext, idxLast)
						}
						// Skip points from the pCurr (first before first missing) to pLast + 2 (third after last missing)
						idxPs += idxLast - idxPs + 2
						// fmt.Printf("====> skipping from %v to %v\n", pCurr, psCurr[idxLast])
					} else {
						// fmt.Printf("adding %v\n", pCurr)
						psCleaned = append(psCleaned, pCurr)
						pCurr = pNext
					}
				}
			} else {
				psCleaned = append(psCleaned, pCurr)
			}
		}
		psCurr = psCleaned
		psCleaned = nil
		// res = psCurr

		// Cleanup speeds - remove outlier points:
		// - fast stops are permitted - crashes or near stops
		// - fast speedups are not permitted - errors
		// - filter out series of points where the speed increases, decreases
		//   and again increases in a short time period
		res = append(res, psCurr[0], psCurr[1])
		speedPrev := speed(psCurr[0], psCurr[1], speedUnits)
		idxRes := 1
		for idxPs := 2; idxPs < len(psCurr)-1; idxPs++ {
			// Compare speed changes between 3 points
			// (previous, current & next point).
			// 3 speeds: 2 speeds between 3 points + previous speed.
			speedCur := speed(res[idxRes], psCurr[idxPs], speedUnits)
			speedNext1 := speed(psCurr[idxPs], psCurr[idxPs+1], speedUnits)
			// 2 speed changes
			speed0Delta := speedCur - speedPrev
			speed1Delta := speedNext1 - speedCur
			// 1 differences between speed changes
			diffDelta1 := speed0Delta - speed1Delta

			// Ignore points where the speed difference between last two points
			//   increases more than given params.
			// if (diffDelta1 < deltaKtsMax && diffDelta2 < deltaKtsMax) || speed0DeltaKts < 0 {
			if (diffDelta1 < deltaSpeedMax) || speed0Delta < 0 {
				// fmt.Printf("OK  idxPs: %v, idxRes: %v, speedCur/n1/n2: %v/%v/%v, sd0: %v, sd1: %v, dd1: %v (%v)\n", idxPs, idxRes, speedCur, speedNext1, speedNext2, speed0DeltaKts, speed1DeltaKts, diffDelta1, psCurr[idxPs].ts)
				speedPrev = speedCur
				res = append(res, psCurr[idxPs])
				idxRes++
				res[idxRes].globalIdx = idxRes
			} else {
				// fmt.Printf("==== NOK idxPs: %v, idxRes: %v, speedCur/n1/n2: %v/%v/%v, sd0: %v, sd1: %v, dd1: %v (%v)\n", idxPs, idxRes, speedCur, speedNext1, speedNext2, speed0DeltaKts, speed1DeltaKts, diffDelta1, psCurr[idxPs].ts)
			}
		}
	}

	return res
}

// UpdateHeadings sets headings for all points. Headings are used in many other calculations.
func UpdateHeadings(ps []Point) {
	for i := 1; i < len(ps); i++ {
		ps[i].heading = heading(ps[i-1], ps[i])
	}
}

// CalculateStats calculate statistics from cleaned up points.
func CalculateStats(ps []Point, statType StatFlag, speedUnits UnitsFlag, preferedTurn TurnType, windDir float64, debug bool) Stats {
	// Calculate heading for each point.
	for i := 1; i < len(ps); i++ {
		ps[i].heading = heading(ps[i-1], ps[i])
		// fmt.Printf("====> p[%d] (%s), h: %.3f, speed: %.2f\n", i, ps[i].ts, ps[i].heading, *ps[i].speed)
	}

	// Auto-detect wind direction if requested
	if preferedTurn != TurnUnknown {
		windDir = autoDetectWindDir(ps, preferedTurn)
	}

	windDirKnown := windDir >= 0

	// If wind directio is knot known, calculate it based on assumption that favorite turn is jibe.
	// Makes turn detection more precise.
	if !windDirKnown {
		windDir = autoDetectWindDir(ps, TurnJibe)
	}

	// Calculate tackSide for each point.
	// fmt.Printf("wind dir: %.3f\n", windDir)
	for i := 1; i < len(ps); i++ {
		ps[i].tackSide = detectTackSideFromHeading(ps[i].heading, windDir)
		// fmt.Printf("====> p[%d] (%s), h: %.3f, ts: %s, speed: %.2f\n", i, ps[i].ts, ps[i].heading, ps[i].tackSide, *ps[i].speed)
	}

	res := Stats{speedUnits: speedUnits, wDirKnown: windDirKnown}
	res.speed5x10s = append(res.speed5x10s,
		Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits})
	res.unknTurnsCount = 0

	// Initialize wind dir statistics, even for assumed wind direction.
	res.wDirStats.windDirection = windDir
	res.wDirStats.tacksCount = 0
	res.wDirStats.starboardSpeed5x10s = append(res.wDirStats.starboardSpeed5x10s,
		Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits})
	res.wDirStats.portSpeed5x10s = append(res.wDirStats.portSpeed5x10s,
		Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits})

	if len(ps) > 1 {
		res.totalDuration = ps[len(ps)-1].ts.Sub(ps[0].ts).Hours()

		track2s := Track{speedUnits: speedUnits}
		track15m := Track{speedUnits: speedUnits}
		track1h := Track{speedUnits: speedUnits}
		track100m := Track{speedUnits: speedUnits}
		track1NM := Track{speedUnits: speedUnits}

		switch statType {
		case StatAll:
			track2s = track2s.addPointMinDuration(ps[0], 2)
			track15m = track15m.addPointMinDuration(ps[0], 900)
			track1h = track1h.addPointMinDuration(ps[0], 3600)
			track100m = track100m.addPointMinDistance(ps[0], 100)
			track1NM = track1NM.addPointMinDistance(ps[0], 1852)
		case Stat2s:
			track2s = track2s.addPointMinDuration(ps[0], 2)
		case Stat15m:
			track15m = track15m.addPointMinDuration(ps[0], 900)
		case Stat1h:
			track1h = track1h.addPointMinDuration(ps[0], 3600)
		case Stat100m:
			track100m = track100m.addPointMinDistance(ps[0], 100)
		case Stat1nm:
			track1NM = track1NM.addPointMinDistance(ps[0], 1852)
		}

		// Calculate statistics for all values except turns and 5x10.
		for i := 1; i < len(ps); i++ {
			res.totalDistance = res.totalDistance + distance(ps[i-1], ps[i])
			switch statType {
			case StatAll:
				track2s = track2s.addPointMinDuration(ps[i], 2)
				track15m = track15m.addPointMinDuration(ps[i], 900)
				track1h = track1h.addPointMinDuration(ps[i], 3600)
				track100m = track100m.addPointMinDistance(ps[i], 100)
				track1NM = track1NM.addPointMinDistance(ps[i], 1852)
			case Stat2s:
				track2s = track2s.addPointMinDuration(ps[i], 2)
			case Stat15m:
				track15m = track15m.addPointMinDuration(ps[i], 900)
			case Stat1h:
				track1h = track1h.addPointMinDuration(ps[i], 3600)
			case Stat100m:
				track100m = track100m.addPointMinDistance(ps[i], 100)
			case Stat1nm:
				track1NM = track1NM.addPointMinDistance(ps[i], 1852)
			}
			// fmt.Printf(" ===> %4d (%s): t: %v, st: %v, valid: %v\n", i, ps[i].ts, trackTurn500m, subtrackTurn500m, subtrackTurn500m.valid)

			// If any of calculated statistics is prepared (valid) and the statistic
			//   is a highest one, save it.
			if track2s.valid && res.speed2s.speed < track2s.speed {
				res.speed2s = track2s
			}
			if track15m.valid && res.speed15m.speed < track15m.speed {
				res.speed15m = track15m
			}
			if track1h.valid && res.speed1h.speed < track1h.speed {
				res.speed1h = track1h
			}
			if track100m.valid && res.speed100m.speed < track100m.speed {
				res.speed100m = track100m
			}
			if track1NM.valid && res.speed1NM.speed < track1NM.speed {
				res.speed1NM = track1NM
			}

			// Save the best starboard stats
			if track2s.TackSide() == TackStarboard && track2s.valid && res.wDirStats.starboardSpeed2s.speed < track2s.speed {
				res.wDirStats.starboardSpeed2s = track2s
			}
			if track100m.TackSide() == TackStarboard && track100m.valid && res.wDirStats.starboardSpeed100m.speed < track100m.speed {
				res.wDirStats.starboardSpeed100m = track100m
			}
			// Save the best port stats
			if track2s.TackSide() == TackPort && track2s.valid && res.wDirStats.portSpeed2s.speed < track2s.speed {
				res.wDirStats.portSpeed2s = track2s
			}
			if track100m.TackSide() == TackPort && track100m.valid && res.wDirStats.portSpeed100m.speed < track100m.speed {
				res.wDirStats.portSpeed100m = track100m
			}
		}

		// Calculate turn statistics
		switch statType {
		case StatAll, StatAlpha:
			// Get all track segments with exactly 1 turn (2 tack, 2 riding sides)
			turnTracks := collectTurnTracks(ps, speedUnits)
			// fmt.Printf("====> turnTracks len: %d\n\n", len(turnTracks))

			for ttIdx := 0; ttIdx < len(turnTracks); ttIdx++ {
				turnTrack := turnTracks[ttIdx]
				// fmt.Printf("====> turnTracks[%d]: %s", ttIdx, turnTrack)
				turnSubtrack := findMaxTurnSubtrack(turnTrack, speedUnits)
				// fmt.Printf("====> turnTracks[%d] valid: %v: tt: %s st: %v", ttIdx, subtrackTurn500m.valid, turnTrack, subtrackTurn500m)
				if turnSubtrack.valid {
					// fmt.Printf("====> turnTracks[%d] valid: %v: tt: %s st: %v", ttIdx, subtrackTurn500m.valid, turnTrack, subtrackTurn500m)
					turnType := detectTurnType(turnSubtrack.ps, windDir)

					if debug {
						fmt.Printf("%s turn (%-9s): %s", turnType, turnSubtrack.TackSide(), turnSubtrack)
					}

					switch turnType {
					case TurnUnknown:
						if res.alpha500m.speed < turnSubtrack.speed {
							res.alpha500m = turnSubtrack
							res.unknTurnsCount++
						}
					case TurnJibe:
						if res.wDirStats.alpha500m.speed < turnSubtrack.speed {
							res.wDirStats.alpha500m = turnSubtrack
						}
					case TurnTack:
						if res.wDirStats.delta500m.speed < turnSubtrack.speed {
							res.wDirStats.delta500m = turnSubtrack
						}
					}

					switch turnSubtrack.TackSide() {
					case TackStarboard:
						// Save the best starboard stats
						if turnType == TurnJibe && res.wDirStats.starboardAlpha500m.speed < turnSubtrack.speed {
							res.wDirStats.starboardAlpha500m = turnSubtrack
						}
						if turnType == TurnTack && res.wDirStats.starboardDelta500m.speed < turnSubtrack.speed {
							res.wDirStats.starboardDelta500m = turnSubtrack
						}
					case TackPort:
						// Save the best port stats
						if turnType == TurnJibe && res.wDirStats.portAlpha500m.speed < turnSubtrack.speed {
							res.wDirStats.portAlpha500m = turnSubtrack
						}
						if turnType == TurnTack && res.wDirStats.portDelta500m.speed < turnSubtrack.speed {
							res.wDirStats.portDelta500m = turnSubtrack
						}
					}

					switch turnType {
					case TurnJibe:
						res.wDirStats.jibesCount++
					case TurnTack:
						res.wDirStats.tacksCount++
					}
				} else {
					turnType := detectTurnType(turnSubtrack.ps, windDir)
					if debug {
						fmt.Printf("%s turn (%-9s): %s, full track: %s", turnType, turnSubtrack.TackSide(), turnSubtrack, turnTrack)
					}
				}
			}
		}

		switch statType {
		case StatAll, Stat10sAvg, Stat10s1, Stat10s2, Stat10s3, Stat10s4, Stat10s5:
			// 5 x 10 secs need to gather 5 different, non-overlapping tracks.
			for track5x10sIdx := 0; track5x10sIdx < 5; track5x10sIdx++ {
				track5x10s := Track{speedUnits: speedUnits}
				track5x10s = track5x10s.addPointMinDurationUnused10s(ps[0], 10, true)
				for i := 1; i < len(ps); i++ {
					track5x10s = track5x10s.addPointMinDurationUnused10s(ps[i], 10, true)
					if track5x10s.valid && res.speed5x10s[track5x10sIdx].speed < track5x10s.speed {
						res.speed5x10s[track5x10sIdx] = track5x10s
					}
				}

				track5x10s = res.speed5x10s[track5x10sIdx]
				for i := 0; i < len(track5x10s.ps); i++ {
					ps[track5x10s.ps[i].globalIdx].usedFor10s = true
				}
			}

			// Fill starboardSpeed5x10s and portSpeed5x10s with best 5x10s tracks for each tack side
			if res.wDirKnown {
				// Reset usedFor10s for all points before per-tack search
				for i := range ps {
					ps[i].usedFor10s = false
				}

				// TackStarboard
				for trackIdx := 0; trackIdx < 5; trackIdx++ {
					bestTrack := Track{speedUnits: speedUnits}
					track := Track{speedUnits: speedUnits}
					track = track.addPointMinDurationUnused10s(ps[0], 10, true)
					for i := 1; i < len(ps); i++ {
						track = track.addPointMinDurationUnused10s(ps[i], 10, true)
						if track.valid && track.TackSide() == TackStarboard && bestTrack.speed < track.speed {
							bestTrack = track
						}
					}
					res.wDirStats.starboardSpeed5x10s[trackIdx] = bestTrack
					for i := 0; i < len(bestTrack.ps); i++ {
						ps[bestTrack.ps[i].globalIdx].usedFor10s = true
					}
				}

				// Reset usedFor10s for all points before port search
				for i := range ps {
					ps[i].usedFor10s = false
				}

				// TackPort
				for trackIdx := 0; trackIdx < 5; trackIdx++ {
					bestTrack := Track{speedUnits: speedUnits}
					track := Track{speedUnits: speedUnits}
					track = track.addPointMinDurationUnused10s(ps[0], 10, true)
					for i := 1; i < len(ps); i++ {
						track = track.addPointMinDurationUnused10s(ps[i], 10, true)
						if track.valid && track.TackSide() == TackPort && bestTrack.speed < track.speed {
							bestTrack = track
						}
					}
					res.wDirStats.portSpeed5x10s[trackIdx] = bestTrack
					for i := 0; i < len(bestTrack.ps); i++ {
						ps[bestTrack.ps[i].globalIdx].usedFor10s = true
					}
				}
			}
		}

	}
	return res
}

// collectTurnTracks gets all track segments with exactly 1 tack change
// (2 tacks, 2 riding sides) without calculating any statistics.
func collectTurnTracks(ps []Point, speedUnits UnitsFlag) []Track {
	turnTracks := []Track{}

	currTrack := Track{speedUnits: speedUnits}
	nextTrack := Track{speedUnits: speedUnits}
	currSide := TackUnknown
	currTrackTacksCount := 0
	// A total number of tacks (riding side)
	totalTacksNo := 0

	for psIdx := 0; psIdx < len(ps); psIdx++ {
		p := ps[psIdx]

		if p.tackSide != TackUnknown && p.tackSide != currSide {
			// fmt.Printf("====> [%4d, %s] - ch - totalTacksNo: %d, currTrackTacksCount: %d, currSide/newSide: %v/%v\n", psIdx, p.ts, totalTacksNo, currTrackTacksCount, currSide, p.tackSide)
			currSide = p.tackSide
			currTrackTacksCount++
			totalTacksNo++
		}

		// fmt.Printf("====> [%4d, %s] - totalTacksNo: %d, currTrackTacksCount: %d, currSide: %v\n", psIdx, p.ts, totalTacksNo, currTrackTacksCount, currSide)

		if totalTacksNo > 0 {
			if currTrackTacksCount <= 2 {
				currTrack.ps = append(currTrack.ps, p)
				if totalTacksNo > 1 {
					// The first point must have known tack side
					if len(nextTrack.ps) > 0 || p.tackSide != TackUnknown {
						nextTrack.ps = append(nextTrack.ps, p)
					}
				}
			} else {
				nextTrack.ps = append(nextTrack.ps, p)
				// fmt.Printf("====> [%4d, %s] - totalTacksNo: %d, curr: %v, next: %v\n", psIdx, p.ts, totalTacksNo, currTrack, nextTrack)
				turnTracks = append(turnTracks, currTrack)

				currTrack = nextTrack
				nextTrack = Track{speedUnits: speedUnits}
				currTrackTacksCount = 2
			}
		}
	}

	// Add the end segment if it contains 2 tacks (riding sides)
	if currTrackTacksCount > 1 {
		turnTracks = append(turnTracks, currTrack)
	}

	return turnTracks
}

// findMaxTurnSubtrack finds fastest subtrack from a single-turn track.
func findMaxTurnSubtrack(turnTrack Track, speedUnits UnitsFlag) Track {
	trackTurn500m := Track{speedUnits: speedUnits}
	subtrackTurn500m := Track{speedUnits: speedUnits}
	subtrackTurn500mMax := Track{speedUnits: speedUnits}

	for psIdx := 0; psIdx < len(turnTrack.ps); psIdx++ {
		trackTurn500m, subtrackTurn500m = trackTurn500m.addPointTurn500(turnTrack.ps[psIdx])

		// fmt.Printf("====> turnTracks[%d] valid: %v: tt: %s st: %v", ttIdx, subtrackTurn500m.valid, turnTrack, subtrackTurn500m)
		if subtrackTurn500m.valid && subtrackTurn500m.speed > subtrackTurn500mMax.speed {
			// fmt.Printf("====> turnTracks[%d] valid: %v: tt: %s st: %v", ttIdx, subtrackTurn500m.valid, turnTrack, subtrackTurn500m)
			subtrackTurn500mMax = subtrackTurn500m
			// fmt.Printf("====> turnTracks[%d] valid: %v: st sides: %v - %v, tt: %s", ttIdx, subtrackTurn500m.valid, subtrackTurn500m.ps[0].tackSide, subtrackTurn500m.ps[len(subtrackTurn500m.ps)-1].tackSide, turnTrack)
			// Determine turn type (jibe or tack), use known or assumed wind direction.
		}
	}

	return subtrackTurn500mMax
}

// KtsToMs converts kts to m/s.
func KtsToMs(speedKts float64) float64 {
	return speedKts / mPerSecToKts
}

// MsToUnits converts m/s to specified units.
func MsToUnits(speedMs float64, speedUnits UnitsFlag) float64 {
	switch speedUnits {
	case UnitsMs:
		return speedMs
	case UnitsKmh:
		return speedMs * mPerSecToKmh
	case UnitsKts:
		return speedMs * mPerSecToKts
	default:
		return speedMs
	}
}

// overlapByGlobalIdx: checks if two slices of points overlap by globalIdx
func overlapByGlobalIdx(a, b []Point) bool {
	for _, p1 := range a {
		for _, p2 := range b {
			if p1.globalIdx == p2.globalIdx {
				return true
			}
		}
	}
	return false
}

// heading returns heading in degrees from p1 to p2 (0 = North, 90 = East)
func heading(p1, p2 Point) float64 {
	return headingSimple(p1.lat, p1.lon, p2.lat, p2.lon)
}

// autoDetectWindDir automatically estimates the wind direction based on heading and the preferred maneuver ("jibe" or "tack").
// The 'prefer' parameter determines the preferred maneuver ("jibe" or "tack"); if not specified or unknown, "jibe" is used as the default value.
// Returns the estimated wind direction (in degrees, where the wind is coming from)
func autoDetectWindDir(ps []Point, prefer TurnType) float64 {
	if len(ps) < 2 {
		return -1
	}
	// 2. Histogram of headings (bins of 10°)
	bins := make([]int, 36)
	for _, p := range ps {
		bin := int(p.heading/10) % 36
		bins[bin]++
	}

	// fmt.Printf("====> awd - bins: %v\n", bins)

	// 3. Find the most populated bin, then find the most populated bin that is 180° apart
	primaryBin := -1
	primaryCount := 0
	for i, count := range bins {
		if count > primaryCount {
			primaryCount = count
			primaryBin = i
		}
	}
	if primaryBin == -1 {
		return -1
	}
	// Find the bin that is 180° apart (opposite direction) with the highest count
	oppositeBin := (primaryBin + 18) % 36
	secondaryBin := -1
	secondaryCount := 0
	// Search for the most populated bin among bins that are within ±1 bin of the exact opposite
	for offset := -1; offset <= 1; offset++ {
		bin := (oppositeBin + offset + 36) % 36
		if bins[bin] > secondaryCount {
			secondaryCount = bins[bin]
			secondaryBin = bin
		}
	}
	if secondaryBin == -1 || secondaryCount == 0 {
		return -1
	}

	// 4. Collect all headings from both bins (primary and secondary/opposite), rotate secondary by 180°
	selectedHeadings := []float64{}
	for _, p := range ps {
		h := p.heading
		bin := int(h/10) % 36
		// does the bin belong to the primary bin or its neighbors
		if bin == primaryBin || bin == (primaryBin+1)%36 || bin == (primaryBin+35)%36 {
			selectedHeadings = append(selectedHeadings, h)
		} else if bin == secondaryBin || bin == (secondaryBin+1)%36 || bin == (secondaryBin+35)%36 {
			// rotate by 180°
			rotated := math.Mod(h+180, 360)
			selectedHeadings = append(selectedHeadings, rotated)
		}
	}
	if len(selectedHeadings) == 0 {
		return -1
	}

	// 5. Calculate the mean heading (circular mean)
	sumSin, sumCos := 0.0, 0.0
	for _, h := range selectedHeadings {
		rad := h * math.Pi / 180
		sumSin += math.Sin(rad)
		sumCos += math.Cos(rad)
	}
	avgHeading := math.Atan2(sumSin, sumCos) * 180 / math.Pi
	if avgHeading < 0 {
		avgHeading += 360
	}

	// 6. Wind direction is perpendicular to the mean heading, assume one of 2 opposite directions.
	wdAssumed := math.Mod(avgHeading+90, 360)
	wdOpposite := math.Mod(avgHeading-90+360, 360)
	pTurnBegin := Point{heading: -1}
	pTurnEnd := Point{heading: -1}
	tackSidePrev := TackUnknown
	tackSideCurr := TackUnknown
	jibeCnt := 0
	tackCnt := 0
	for _, p := range ps {
		t := detectTackSideFromHeading(p.heading, wdAssumed)
		// Ignore unknown tack sides and find 2 points for each turn (start & end of the turn).
		if t != TackUnknown {
			if tackSidePrev == TackUnknown {
				// Initial known tack
				tackSidePrev = t
				tackSideCurr = t
			} else if t == tackSideCurr {
				// Set the begin point of the turn to the last point on the same tack side.
				pTurnBegin = p
			} else if t != tackSideCurr {
				// Set the end point of the turn to the first point on the new tack side.
				pTurnEnd = p
				tackSideCurr = t
				// fmt.Printf("====> awd - tack prev -> curr: %v -> %v, pStart: %v, pEnd: %v, dist: %.2f\n", tackSidePrev, tackSideCurr, pTurnBegin, pTurnEnd, dist)
				turnType := detectTurnTypeFromTurnPoints(pTurnBegin, pTurnEnd, wdAssumed)
				switch turnType {
				case TurnJibe:
					jibeCnt++
				case TurnTack:
					tackCnt++
					// case TurnUnknown:
					// 	fmt.Printf("====> awd - unknown turn, pStart: %v, pEnd: %v, dist: %.2f, h: %.2f\n", pTurnBegin, pTurnEnd, dist, h)
				}

				tackSidePrev = tackSideCurr
			}
		}
	}
	// Determine wd based on preferred maneuver
	var result float64
	if prefer == TurnTack {
		if tackCnt >= jibeCnt {
			result = wdAssumed
		} else {
			result = wdOpposite
		}
	} else {
		if jibeCnt >= tackCnt {
			result = wdAssumed
		} else {
			result = wdOpposite
		}
	}

	// fmt.Printf("====> awd - assumed wd: %.2f, j/t: %d/%d, pref: %v, res: %v\n", wdAssumed, jibeCnt, tackCnt, prefer, result)

	return result
}

// angleDiff finds a smaller agle between two angles (350, 10 -> 20; 150, 30 -> 180)
func angleDiff(angle1, angle2 float64) float64 {
	diff := math.Mod(angle2-angle1+360, 360)
	if diff > 180 {
		diff = 360 - diff
	}
	return diff
}

// detectTurnType determines the type of sailing maneuver (jibe or tack) based on the track points and wind direction.
func detectTurnType(ps []Point, windDir float64) TurnType {
	if len(ps) < 2 {
		return TurnUnknown
	}

	// Find last point on initial tack side and the first point on the oposite side tack.
	pTurnBegin := Point{}
	pTurnEnd := Point{}
	for i := 0; i < len(ps); i++ {
		p := ps[i]
		if p.tackSide != TackUnknown {
			if pTurnBegin.tackSide == TackUnknown {
				// Set the start point to the first one with defined tack.
				pTurnBegin = p
			} else if pTurnBegin.tackSide == p.tackSide {
				// Update the start point to the last one with the same tack.
				pTurnBegin = p
			} else if pTurnEnd.tackSide == TackUnknown {
				// Set the end point to the first one with the opposite tack.
				pTurnEnd = p
				break
			}
		}
	}

	// fmt.Printf("====> awd - pStart: %v, pEnd: %v, dist: %.2f\n", pTurnBegin, pTurnEnd, dist)
	return detectTurnTypeFromTurnPoints(pTurnBegin, pTurnEnd, windDir)
}

// detectTurnTypeFromTurnPoints: detects the type of turn (jibe/tack) from turn begin and enc points and wind direction
// Tack: heading is near windDir
// Jibe: heading is near windDir+180
// Unknown: otherwise
func detectTurnTypeFromTurnPoints(pTurnBegin, pTurnEnd Point, windDir float64) TurnType {
	dist := distSimple(pTurnBegin.lat, pTurnBegin.lon, pTurnEnd.lat, pTurnEnd.lon)
	// fmt.Printf("====> awd - pStart: %v, pEnd: %v, dist: %.2f\n", pTurnBegin, pTurnEnd, dist)
	// IF distance is too small, can't know the turn type.
	if dist >= 1.0 {
		h := heading(pTurnBegin, pTurnEnd)
		turnType := detectTurnTypeFromTurnHeading(h, windDir)
		// fmt.Printf("  ==> awd - pStart: %v, pEnd: %v, dist: %.2f, h: %.2f, wd: %.2f, turn: %v\n", pTurnBegin, pTurnEnd, dist, h, windDir, turnType)
		return turnType
	}

	return TurnUnknown
}

// detectTurnTypeFromTurnHeading: detects the type of turn (jibe/tack) from a single heading and wind direction
// Tack: heading is near windDir
// Jibe: heading is near windDir+180
// Unknown: otherwise
func detectTurnTypeFromTurnHeading(heading float64, windDir float64) TurnType {
	diff := angleDiff(heading, windDir)

	// minHeadingDiff: a minimum difference between exact upwind or downwind to recognize a turn type.
	minHeadingDiff := 60.0
	if diff < minHeadingDiff {
		return TurnTack
	} else if diff > (180 - minHeadingDiff) {
		return TurnJibe
	}

	return TurnUnknown
}

// detectTackSideFromHeading determines a tack side
// (TackStarboard, TackPort, or TackUnknown) relative to the wind direction.
// If the wind direction or heading is unknown, it returns TackUnknown.
//
// Parameters:
//   - heading: The ride direction heading
//   - windDir: The wind direction in degrees (from where the wind is coming).
//
// Returns:
//   - A TackSide indicating the tack side: TackStarboard, TackPort, or TackUnknown.
func detectTackSideFromHeading(heading, windDir float64) TackSide {
	// If we don't know wind direction or heading (ride direction), return TackUnknown.
	if windDir < 0 || heading < 0 {
		return TackUnknown
	}

	// Heading relative to wind direction (where the wind is coming from)
	relHeading := math.Mod(windDir-heading+360, 360)
	// fmt.Printf("  ====> dist: %.2f, speeds: (%.2f, %.2f, %.2f), wd: %.2f, heading: %.2f, rel h: %.2f\n",
	// 	distance(pPrev, p), *pPrev.speed, *p.speed, distance(pPrev, p)/p.ts.Sub(pPrev.ts).Seconds(),
	// 	windDir, h, relHeading)

	// minHeadingDiff: a minimum difference between exact upwind or downwind to recognize a tack side.
	minHeadingDiff := 30.0
	if relHeading >= (0+minHeadingDiff) && relHeading <= (180-minHeadingDiff) {
		return TackStarboard
	}
	if relHeading >= (180+minHeadingDiff) && relHeading <= (360-minHeadingDiff) {
		return TackPort
	}
	return TackUnknown
}
