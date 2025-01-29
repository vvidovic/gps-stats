package stats

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
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
}

func (p Point) String() string {
	return fmt.Sprintf("{%v/%v (%v)}", p.lat, p.lon, p.ts)
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

// TxtLine display human-readable entry for each track.
func (t Track) TxtLine() string {
	var timestamp time.Time
	if len(t.ps) > 0 {
		timestamp = t.ps[0].ts
	}
	return fmt.Sprintf("%06.3f %s (%0.0f sec, %06.3f m, %v)",
		t.speed, t.speedUnits, t.duration, t.distance, timestamp)
}
func (t Track) String() string {
	return fmt.Sprintf("dur: %v, dist: %v, speed: %v, ps[0]: %v\n",
		t.duration, t.distance, t.speed, t.ps[0])
}

// reCalculate sums durations and distanes from points and calculates
//
//	speed from those.
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

// addPointAlpha500
//   - add a new Point to the end of the Track for Alpha 500 m calculation
//   - ensures the Track is as close but no longer than 500 m
//   - try to find the subtrack that contains alpha for entry/exit gate max 50 m
//   - return two Tracks: "this" Track and subtrack containing best alpha
//     (as described above)
func (t Track) addPointAlpha500(p Point) (Track, Track) {
	return t.addPointAlphaMaxDistance(p, 500, 100, 50)
}

// addPointAlphaMaxDistance
//   - add a new Point to the end of the Track for Alpha calculation
//   - ensures the Track is as close but no longer than maxDistance (removing
//     Points from the beginning of the Track if possible)
//   - try to find the subtrack that is no shorter than minDistance (to ensure
//     this is alpha and no riding straight) and that the first and the last point
//     are ate most gateSize away
//   - return two Tracks: "this" Track and subtrack containing best alpha
//     (as described above)
func (t Track) addPointAlphaMaxDistance(p Point,
	maxDistance, minDistance, gateSize float64) (Track, Track) {
	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])

		// 1. Do we need to remove some points from the start of this track?
		//    - find a track with length most close to the maxDistance
		if t.distance > maxDistance && l > 2 {
			distTest := t.distance - distance(t.ps[0], t.ps[1])
			for distTest > maxDistance && l > 2 {
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

		// 2. Can we find a gate, maybe by removing some points from the start?
		// Distance between the first and the last point must be max gateSize (50m).
		subtrackDistance := t.distance
		for i := 0; i < l-2; i++ {
			gateDistance := distance(t.ps[i], t.ps[l-1])
			if subtrackDistance < minDistance {
				break
			}
			if gateDistance <= gateSize && subtrackDistance >= minDistance {
				subtrack := Track{ps: t.ps[i:], valid: true, speedUnits: t.speedUnits}.reCalculate()
				return t, subtrack
			}
			subtrackDistance = subtrackDistance - distance(t.ps[i], t.ps[i+1])
		}
	}

	return t, Track{speedUnits: t.speedUnits}
}

// Stats constains calculated statistics.
type Stats struct {
	totalDistance float64
	totalDuration float64
	speed2s       Track
	speed5x10s    []Track
	speed15m      Track
	speed1h       Track
	speed100m     Track
	speed1NM      Track
	alpha500m     Track
	speedUnits    UnitsFlag
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
		return fmt.Sprintf("%06.3f", s.Calc5x10sAvg())
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
	return fmt.Sprintf(
		`Total Distance:     %06.3f km
Total Duration:     %06.3f h
2 Second Peak:      %s
5x10 Average:       %06.3f %s
  Top 1 5x10 speed: %s
  Top 2 5x10 speed: %s
  Top 3 5x10 speed: %s
  Top 4 5x10 speed: %s
  Top 5 5x10 speed: %s
15 Min:             %s
1 Hr:               %s
100m peak:          %s
Nautical Mile:      %s
Alpha 500:          %s
`,
		s.totalDistance/1000,
		s.totalDuration,
		s.speed2s.TxtLine(),
		s.Calc5x10sAvg(),
		s.speedUnits,
		s.speed5x10s[0].TxtLine(), s.speed5x10s[1].TxtLine(),
		s.speed5x10s[2].TxtLine(), s.speed5x10s[3].TxtLine(),
		s.speed5x10s[4].TxtLine(),
		s.speed15m.TxtLine(), s.speed1h.TxtLine(),
		s.speed100m.TxtLine(), s.speed1NM.TxtLine(),
		s.alpha500m.TxtLine())
}
func (s Stats) String() string {
	return fmt.Sprintf(
		"dist: %v\n  2s: %v\n  5x10s: %v\n  %v\n  15m: %v\n  1h: %v\n  100m: %v\n  1NM: %v\n  alpha: %v\n",
		s.totalDistance, s.speed2s, s.Calc5x10sAvg(), s.speed5x10s,
		s.speed15m, s.speed1h,
		s.speed100m, s.speed1NM, s.alpha500m)
}

// Calc5x10sAvg calculate average from 5 10s speed records.
func (s Stats) Calc5x10sAvg() float64 {
	res := 0.0
	for i := 0; i < len(s.speed5x10s); i++ {
		res += s.speed5x10s[i].speed
	}
	res = res / float64(len(s.speed5x10s))

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

// dist calculate a distance between two points by their lattitudes and
// longitudes.
func dist(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLon := (lon2 - lon1) * math.Pi / 180
	a := sq(math.Sin(dLat/2)) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*sq(math.Sin(dLon/2))
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// distSimple calculate a distance between two points by their lattitudes and
// longitudes, ignoring curvature of the earth surface (small distances).
func distSimple(lat1, lon1, lat2, lon2 float64) float64 {
	dLatM := (lat2 - lat1) / 360 * earthCircPoles
	dLonM := (lon2 - lon1) / 360 * earthCircEquator * math.Cos((lat1+lat2)/2*math.Pi/180)

	return math.Sqrt(sq(dLatM) + sq(dLonM))
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

// CalculateStats calculate statistics from cleaned up points.
func CalculateStats(ps []Point, statType StatFlag, speedUnits UnitsFlag) Stats {
	switch speedUnits {
	case UnitsMs:
	}
	res := Stats{speedUnits: speedUnits}
	res.speed5x10s = append(res.speed5x10s,
		Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits}, Track{speedUnits: speedUnits})
	if len(ps) > 1 {
		track2s := Track{speedUnits: speedUnits}
		track15m := Track{speedUnits: speedUnits}
		track1h := Track{speedUnits: speedUnits}
		track100m := Track{speedUnits: speedUnits}
		track1NM := Track{speedUnits: speedUnits}
		trackAlpha500m := Track{speedUnits: speedUnits}
		subtrackAlpha500m := Track{speedUnits: speedUnits}

		switch statType {
		case StatAll:
			track2s = track2s.addPointMinDuration(ps[0], 2)
			track15m = track15m.addPointMinDuration(ps[0], 900)
			track1h = track1h.addPointMinDuration(ps[0], 3600)
			track100m = track100m.addPointMinDistance(ps[0], 100)
			track1NM = track1NM.addPointMinDistance(ps[0], 1852)
			trackAlpha500m, subtrackAlpha500m =
				trackAlpha500m.addPointAlpha500(ps[0])
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
		case StatAlpha:
			trackAlpha500m, subtrackAlpha500m =
				trackAlpha500m.addPointAlpha500(ps[0])
		}
		for i := 1; i < len(ps); i++ {
			res.totalDistance = res.totalDistance + distance(ps[i-1], ps[i])
			switch statType {
			case StatAll:
				track2s = track2s.addPointMinDuration(ps[i], 2)
				track15m = track15m.addPointMinDuration(ps[i], 900)
				track1h = track1h.addPointMinDuration(ps[i], 3600)
				track100m = track100m.addPointMinDistance(ps[i], 100)
				track1NM = track1NM.addPointMinDistance(ps[i], 1852)
				trackAlpha500m, subtrackAlpha500m =
					trackAlpha500m.addPointAlpha500(ps[i])
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
			case StatAlpha:
				trackAlpha500m, subtrackAlpha500m =
					trackAlpha500m.addPointAlpha500(ps[i])
			}
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
			if subtrackAlpha500m.valid && res.alpha500m.speed < subtrackAlpha500m.speed {
				res.alpha500m = subtrackAlpha500m
			}
		}

		res.totalDuration = ps[len(ps)-1].ts.Sub(ps[0].ts).Hours()

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
		}

	}

	return res
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
