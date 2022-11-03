package stats

import (
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
	earthRadius  = 6370000 // Earth Radius in meters
	mPerSecToKts = 1.94384 // Number of KTS in 1 m/s
)

// Point represent one GPS point with timestamp.
type Point struct {
	isPoint    bool
	valid      bool
	validCheck bool
	lat        float64
	lon        float64
	ts         time.Time
	usedFor10s bool
	globalIdx  int
}

func (p Point) String() string {
	return fmt.Sprintf("{%v/%v (%v)}", p.lat, p.lon, p.ts)
}

// Track is a collection of points and can contain sum of durations,
//   sum of calculated distances and calculated speed.
// valid field can be used to mark that the Track is valid for the statistic
//   we are currently preparing.
type Track struct {
	ps       []Point
	duration float64
	distance float64
	speed    float64
	valid    bool
}

// TxtLine display human-readable entry for each track.
func (t Track) TxtLine() string {
	var timestamp time.Time
	if len(t.ps) > 0 {
		timestamp = t.ps[0].ts
	}
	return fmt.Sprintf("%0.3f kts (%0.0f sec, %0.3f m, %v)",
		t.speed, t.duration, t.distance, timestamp)
}
func (t Track) String() string {
	return fmt.Sprintf("dur: %v, dist: %v, speed: %v, ps[0]: %v\n",
		t.duration, t.distance, t.speed, t.ps[0])
}

// reCalculate sums durations and distanes from points and calculates
//   speed from those.
func (t Track) reCalculate() Track {
	t.duration = 0
	t.distance = 0
	t.speed = 0
	for i := 0; i < len(t.ps)-1; i++ {
		t.duration += t.ps[i+1].ts.Sub(t.ps[i].ts).Seconds()
		t.distance += distance(t.ps[i], t.ps[i+1])
	}
	if t.duration > 0 {
		t.speed = t.distance / t.duration * mPerSecToKts
	}

	return t
}

// addPointMinDuration
// - add a new Point to the end of the Track
// - ensures the Track is no shorter than minDuration (removing Points from the
//   beginning of the Track if possible)
func (t Track) addPointMinDuration(p Point, minDuration float64) Track {
	return t.addPointMinDurationUnused10s(p, minDuration, false)
}

// addPointMinDurationUnused10s
// - start a new track if unused10sOnly is true and the current point is used or
// - add a new Point to the end of the Track
// - ensures the Track is no shorter than minDuration (removing Points from the
//   beginning of the Track if possible)
func (t Track) addPointMinDurationUnused10s(
	p Point, minDuration float64, unused10sOnly bool) Track {
	if unused10sOnly && p.usedFor10s {
		return Track{}
	}
	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])
		t.speed = t.distance / t.duration * mPerSecToKts
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
			t.speed = t.distance / t.duration * mPerSecToKts
		}
	}

	return t
}

// addPointMinDistance
// - add a new Point to the end of the Track
// - ensures the Track is no shorter than minDistance (removing Points from the
//   beginning of the Track if possible)
func (t Track) addPointMinDistance(p Point, minDistance float64) Track {
	t.ps = append(t.ps, p)
	l := len(t.ps)
	if l > 1 {
		t.duration = t.duration + t.ps[l-1].ts.Sub(t.ps[l-2].ts).Seconds()
		t.distance = t.distance + distance(t.ps[l-2], t.ps[l-1])
		t.speed = t.distance / t.duration * mPerSecToKts
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
			t.speed = t.distance / t.duration * mPerSecToKts
		}
	}

	return t
}

// addPointAlphaMaxDistance
// - add a new Point to the end of the Track
// - ensures the Track is as close but no longer than maxDistance (removing
//   Points from the beginning of the Track if possible)
// - try to find the subtrack that is no shorter than minDistance (to ensure
//   this is alpha and no riding straight) and that the first and the last point
//   are ate most gateSize away
// - return two Tracks: "this" Track and subtrack (as described above)
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
			if gateDistance <= gateSize {
				subtrack := Track{ps: t.ps[i:], valid: true}.reCalculate()
				return t, subtrack
			}
			subtrackDistance = subtrackDistance - distance(t.ps[i], t.ps[i+1])
		}
	}

	return t, Track{}
}

// Stats constains calculated statistics.
type Stats struct {
	totalDistance float64
	speed2s       Track
	speed5x10s    []Track
	speed15m      Track
	speed1h       Track
	speed100m     Track
	speed1NM      Track
	alpha500m     Track
}

// TxtStats formats statistics as a human-readable text.
func (s Stats) TxtStats() string {
	return fmt.Sprintf(
		`Total Distance:     %0.3f km
2 Second Peak:      %s
5x10 Average:       %0.3f kts
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
		s.speed2s.TxtLine(),
		s.Calc5x10sAvg(),
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
func ReadPoints(r io.Reader) ([]Point, error) {
	res := []Point{}

	p, err := ReadPoint(r)
	for err == nil {
		if err != nil {
			return res, err
		}

		if p.isPoint {
			p.globalIdx = len(res)
			res = append(res, p)
		}

		p, err = ReadPoint(r)
	}
	return res, nil
}

// ReadPoint reads a next potential Point from the Reader.
// If no point is found, return Point with isPoint set to false.
func ReadPoint(r io.Reader) (Point, error) {
	h := make([]byte, 4)
	numBytes, err := io.ReadFull(r, h)
	if err != nil {
		return Point{}, err
	}
	if numBytes != 4 {
		return Point{}, errs.Errorf("Invalid number of header bytes read: %d.", numBytes)
	}

	bodyLen := int(h[3])
	body := make([]byte, h[3])
	numBytes, err = io.ReadFull(r, body)
	if err != nil {
		return Point{}, err
	}
	if numBytes != bodyLen {
		return Point{}, errs.Errorf("Invalid number of body bytes read: %d.", numBytes)
	}

	checksum := make([]byte, 2)
	numBytes, err = io.ReadFull(r, checksum)
	if err != nil {
		return Point{}, err
	}
	if numBytes != 2 {
		return Point{}, errs.Errorf("Invalid number of checksum bytes read: %d.", numBytes)
	}
	checksumInt := intFrom2ub(checksum)

	endSequence := make([]byte, 2)
	numBytes, err = io.ReadFull(r, endSequence)
	if err != nil {
		return Point{}, err
	}
	if numBytes != 2 {
		return Point{}, errs.Errorf("Invalid number of end sequence bytes read: %d.", numBytes)
	}
	if bytes.Compare(endSequence, []byte("\xb0\xb3")) != 0 {
		return Point{}, errs.Errorf("Invalid end sequence of bytes: %v.", endSequence)
	}

	csCalc := 0
	for i := 0; i < bodyLen; i++ {
		csCalc = csCalc + int(body[i])
		csCalc = csCalc & 0x7FFF
	}

	if body[0] != 0x29 {
		return Point{}, nil
	}

	if checksumInt != csCalc {
		return Point{}, errs.Errorf("Invalid checksum: %d (%04x), should be %d (%04x).",
			checksumInt, checksum, csCalc, csCalc)
	}

	navValid := body[1:3]
	msecs := intFrom2ub(body[17:19])
	ts := time.Date(
		intFrom2ub(body[11:13]), time.Month(body[13]), int(body[14]),
		int(body[15]), int(body[16]), msecs/1000,
		msecs%1000*1000000, time.UTC)
	lat := float64(intFrom4sb(body[23:27])) / 10000000
	lon := float64(intFrom4sb(body[27:31])) / 10000000
	if navValid[0] != 0 || navValid[1] != 0 {
		return Point{}, errs.Errorf("Nav Valid != 0: %x.", navValid)
	}

	return Point{isPoint: true, lat: lat, lon: lon, ts: ts}, nil
}

// speed calculate speed as a result of moving between two Points.
func speed(p1, p2 Point) float64 {
	d := distance(p1, p2)
	dt := p2.ts.Sub(p1.ts)

	speed := d / dt.Seconds() * mPerSecToKts

	return speed
}

// distance calculates a distance between two Points.
func distance(p1, p2 Point) float64 {
	return dist(p1.lat, p1.lon, p2.lat, p2.lon)
}

// dist calculate a distance between two points by their lattitudes and
// longitudes.
func dist(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := lat2*math.Pi/180 - lat1*math.Pi/180
	dLon := lon2*math.Pi/180 - lon1*math.Pi/180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadius * c
}

// CleanUp removes points that seems not valid.
func CleanUp(ps []Point) []Point {
	res := []Point{}
	if len(ps) > 1 {
		res = append(res, ps[0], ps[1])
		speedPrev := speed(ps[0], ps[1])
		idxRes := 1
		for idxPs := 2; idxPs < len(ps); idxPs++ {
			speedCur := speed(res[idxRes], ps[idxPs])
			speedDeltaPerc := math.Abs(speedCur / speedPrev)
			speedDeltaKts := math.Abs(speedCur - speedPrev)
			// Ignore points where the speed is above 3 kts and the speed difference
			//   from the last two points increased or decreased more than 50%.
			if speedDeltaPerc <= 1.5 || speedDeltaKts < 3 {
				speedPrev = speedCur
				res = append(res, ps[idxPs])
				idxRes++
				res[idxRes].globalIdx = idxRes
			}
		}
	}

	return res
}

// CalculateStats calculate statistics from cleaned up points.
func CalculateStats(ps []Point) Stats {
	res := Stats{}
	res.speed5x10s = append(res.speed5x10s,
		Track{}, Track{}, Track{}, Track{}, Track{})
	if len(ps) > 1 {
		track2s := Track{}
		track15m := Track{}
		track1h := Track{}
		track100m := Track{}
		track1NM := Track{}
		trackAlpha500m := Track{}
		subtrackAlpha500m := Track{}

		track2s = track2s.addPointMinDuration(ps[0], 2)
		track15m = track15m.addPointMinDuration(ps[0], 900)
		track1h = track1h.addPointMinDuration(ps[0], 3600)
		track100m = track100m.addPointMinDistance(ps[0], 100)
		track1NM = track1NM.addPointMinDistance(ps[0], 1852)
		trackAlpha500m, subtrackAlpha500m =
			trackAlpha500m.addPointAlphaMaxDistance(ps[0], 500, 100, 50)
		for i := 1; i < len(ps); i++ {
			res.totalDistance = res.totalDistance + distance(ps[i-1], ps[i])
			track2s = track2s.addPointMinDuration(ps[i], 2)
			track15m = track15m.addPointMinDuration(ps[i], 900)
			track1h = track1h.addPointMinDuration(ps[i], 3600)
			track100m = track100m.addPointMinDistance(ps[i], 100)
			track1NM = track1NM.addPointMinDistance(ps[i], 1852)
			trackAlpha500m, subtrackAlpha500m =
				trackAlpha500m.addPointAlphaMaxDistance(ps[i], 500, 100, 50)
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

		// 5 x 10 secs need to gather 5 different, non-overlapping tracks.
		for track5x10sIdx := 0; track5x10sIdx < 5; track5x10sIdx++ {
			track5x10s := Track{}
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

	return res
}