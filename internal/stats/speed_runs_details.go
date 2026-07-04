package stats

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"
)

// WindDirection returns the wind direction used for statistics.
func (s Stats) WindDirection() float64 {
	return s.wDirStats.windDirection
}

// WindDirectionKnown returns true if wind direction was explicitly known.
func (s Stats) WindDirectionKnown() bool {
	return s.wDirKnown
}

type speedRunDetails struct {
	peakTrack      Track
	windowTrack    Track
	start          Point
	end            Point
	headingAvg     float64
	headingStd     float64
	headingMin     float64
	headingMax     float64
	windRelative   float64
	windDirKnown   bool
	windDir        float64
	accelFromSpeed float64
	accelToSpeed   float64
	accelDuration  float64
	accelDistance  float64
	accelMean      float64
	thresholds     []speedRunThresholdDetails
	stability      speedRunStabilityScore
}

type speedRunThresholdDetails struct {
	threshold float64
	duration  float64
	distance  float64
}

type speedRunStabilityScore struct {
	heading      int
	retention    int
	acceleration int
	total        int
}

// SpeedRunsDetails returns an additional human-readable analysis of the top speed runs.
func SpeedRunsDetails(ps []Point, numRuns int, windowSecs float64, speedUnits UnitsFlag, windDirKnown bool, windDir float64) string {
	if numRuns <= 0 || windowSecs < 2 || len(ps) < 2 {
		return ""
	}

	runs := findSpeedRuns(ps, numRuns, windowSecs, speedUnits, windDirKnown, windDir)
	if len(runs) == 0 {
		return "\nSpeed Runs Details:\n  No valid speed runs found.\n"
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\nSpeed Runs Details:\n")
	fmt.Fprintf(&b, "  Settings:\n")
	fmt.Fprintf(&b, "    Runs shown:       %d\n", len(runs))
	fmt.Fprintf(&b, "    Analysis window:  %.0f sec\n", windowSecs)
	fmt.Fprintf(&b, "    Thresholds:       peak-1 / peak-2 / peak-3\n")
	fmt.Fprintf(&b, "\n")

	for i, run := range runs {
		fmt.Fprintf(&b, "  Run %d:\n", i+1)
		fmt.Fprintf(&b, "    Peak:              %s\n", run.peakTrack.TxtLine())
		fmt.Fprintf(&b, "    %-18s %s\n", fmt.Sprintf("%.0fs around peak:", windowSecs), run.windowTrack.TxtLine())
		fmt.Fprintf(&b, "    Position:          %.6f, %.6f → %.6f, %.6f\n", run.start.lat, run.start.lon, run.end.lat, run.end.lon)
		fmt.Fprintf(&b, "    Heading:           %05.1f° avg, ±%.1f° stddev, %05.1f°–%05.1f° range\n", run.headingAvg, run.headingStd, run.headingMin, run.headingMax)
		if run.windDirKnown {
			fmt.Fprintf(&b, "    Wind-relative:     %.1f° off wind (wind %.1f°)\n", run.windRelative, run.windDir)
		}
		fmt.Fprintf(&b, "    Acceleration:      %.1f→%.1f %s in %.1f sec / %.1f m (%.2f %s/s)\n",
			run.accelFromSpeed, run.accelToSpeed, speedUnits, run.accelDuration, run.accelDistance, run.accelMean, speedUnits)
		fmt.Fprintf(&b, "    Above thresholds:  ")
		for tIdx, threshold := range run.thresholds {
			if tIdx > 0 {
				fmt.Fprintf(&b, ", ")
			}
			fmt.Fprintf(&b, ">%.1f %s: %.1f sec / %.1f m", threshold.threshold, speedUnits, threshold.duration, threshold.distance)
		}
		fmt.Fprintf(&b, "\n")
		fmt.Fprintf(&b, "    Stability Score:   %d/100 (heading %d/40, retention %d/35, acceleration %d/25)\n",
			run.stability.total, run.stability.heading, run.stability.retention, run.stability.acceleration)
		if i < len(runs)-1 {
			fmt.Fprintf(&b, "\n")
		}
	}

	return b.String()
}

func findSpeedRuns(ps []Point, numRuns int, windowSecs float64, speedUnits UnitsFlag, windDirKnown bool, windDir float64) []speedRunDetails {
	candidates := collectAll2sTracks(ps, speedUnits)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].speed > candidates[j].speed
	})

	selected := []Track{}
	for _, candidate := range candidates {
		if !candidate.valid || len(candidate.ps) == 0 {
			continue
		}
		candidateTime := peakTime(candidate)
		duplicate := false
		for _, existing := range selected {
			if math.Abs(candidateTime.Sub(peakTime(existing)).Seconds()) <= windowSecs {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		selected = append(selected, candidate)
		if len(selected) >= numRuns {
			break
		}
	}

	runs := []speedRunDetails{}
	for _, peak := range selected {
		windowTrack := findWindowTrackContainingPeak(ps, peak, windowSecs, speedUnits)
		if !windowTrack.valid {
			windowTrack = peak
		}

		start := windowTrack.ps[0]
		end := windowTrack.ps[len(windowTrack.ps)-1]
		headingAvg, headingStd, headingMin, headingMax := headingStats(windowTrack.ps)

		accelFromSpeed, accelDuration, accelDistance, accelMean := accelerationDetails(ps, peak, speedUnits)
		thresholds := thresholdDetails(ps, peak, []float64{peak.speed - 1, peak.speed - 2, peak.speed - 3}, speedUnits)
		stability := stabilityScore(ps, peak, windowTrack, headingStd, windowSecs, speedUnits)

		run := speedRunDetails{
			peakTrack:      peak,
			windowTrack:    windowTrack,
			start:          start,
			end:            end,
			headingAvg:     headingAvg,
			headingStd:     headingStd,
			headingMin:     headingMin,
			headingMax:     headingMax,
			windDirKnown:   windDirKnown,
			windDir:        windDir,
			accelFromSpeed: accelFromSpeed,
			accelToSpeed:   peak.speed,
			accelDuration:  accelDuration,
			accelDistance:  accelDistance,
			accelMean:      accelMean,
			thresholds:     thresholds,
			stability:      stability,
		}
		if windDirKnown && headingAvg >= 0 {
			run.windRelative = absAngleDiff(headingAvg, windDir)
		}
		runs = append(runs, run)
	}

	return runs
}

func collectAll2sTracks(ps []Point, speedUnits UnitsFlag) []Track {
	tracks := []Track{}
	if len(ps) < 2 {
		return tracks
	}

	track2s := Track{speedUnits: speedUnits}
	track2s = track2s.addPointMinDuration(ps[0], 2)
	for i := 1; i < len(ps); i++ {
		track2s = track2s.addPointMinDuration(ps[i], 2)
		if track2s.valid {
			tracks = append(tracks, track2s)
		}
	}
	return tracks
}

func findWindowTrackContainingPeak(ps []Point, peak Track, windowSecs float64, speedUnits UnitsFlag) Track {
	if len(ps) < 2 || len(peak.ps) == 0 {
		return Track{speedUnits: speedUnits}
	}

	peakStart := peak.ps[0].ts
	peakEnd := peak.ps[len(peak.ps)-1].ts
	best := Track{speedUnits: speedUnits}
	track := Track{speedUnits: speedUnits}
	track = track.addPointMinDuration(ps[0], windowSecs)
	for i := 1; i < len(ps); i++ {
		track = track.addPointMinDuration(ps[i], windowSecs)
		if track.valid && track.ps[0].ts.Before(peakStart.Add(time.Nanosecond)) && track.ps[len(track.ps)-1].ts.After(peakEnd.Add(-time.Nanosecond)) && best.speed < track.speed {
			best = track
		}
	}
	return best
}

func peakTime(track Track) time.Time {
	if len(track.ps) == 0 {
		return time.Time{}
	}
	return track.ps[0].ts
}

func headingStats(ps []Point) (float64, float64, float64, float64) {
	headings := []float64{}
	for _, p := range ps {
		if p.heading >= 0 {
			headings = append(headings, p.heading)
		}
	}
	if len(headings) == 0 {
		return -1, 0, -1, -1
	}

	sinSum := 0.0
	cosSum := 0.0
	for _, h := range headings {
		rad := h * math.Pi / 180
		sinSum += math.Sin(rad)
		cosSum += math.Cos(rad)
	}
	avg := math.Mod(math.Atan2(sinSum, cosSum)*180/math.Pi+360, 360)

	variance := 0.0
	minDelta := 0.0
	maxDelta := 0.0
	for i, h := range headings {
		delta := signedAngleDiff(h, avg)
		variance += delta * delta
		if i == 0 || delta < minDelta {
			minDelta = delta
		}
		if i == 0 || delta > maxDelta {
			maxDelta = delta
		}
	}
	std := math.Sqrt(variance / float64(len(headings)))
	minHeading := normalizeAngle(avg + minDelta)
	maxHeading := normalizeAngle(avg + maxDelta)

	return avg, std, minHeading, maxHeading
}

func accelerationDetails(ps []Point, peak Track, speedUnits UnitsFlag) (float64, float64, float64, float64) {
	if len(peak.ps) == 0 {
		return 0, 0, 0, 0
	}

	peakStartIdx := findPointIndexByTime(ps, peak.ps[0].ts)
	peakEndIdx := findPointIndexByTime(ps, peak.ps[len(peak.ps)-1].ts)
	if peakStartIdx <= 0 || peakEndIdx <= peakStartIdx {
		return 0, 0, 0, 0
	}

	threshold := MsToUnits(KtsToMs(20), speedUnits)
	startIdx := findAccelerationStart(ps, peakStartIdx, threshold, speedUnits)
	if startIdx == peakStartIdx {
		threshold = peak.speed * 0.9
		startIdx = findAccelerationStart(ps, peakStartIdx, threshold, speedUnits)
	}

	duration, dist := sumSegments(ps, startIdx, peakEndIdx)
	meanAccel := 0.0
	if duration > 0 {
		meanAccel = (peak.speed - threshold) / duration
	}
	return threshold, duration, dist, meanAccel
}

func findAccelerationStart(ps []Point, peakStartIdx int, threshold float64, speedUnits UnitsFlag) int {
	startIdx := peakStartIdx
	for i := peakStartIdx; i > 0; i-- {
		segSpeed := speed(ps[i-1], ps[i], speedUnits)
		if segSpeed < threshold {
			break
		}
		startIdx = i - 1
	}
	return startIdx
}

func thresholdDetails(ps []Point, peak Track, thresholds []float64, speedUnits UnitsFlag) []speedRunThresholdDetails {
	res := []speedRunThresholdDetails{}
	if len(peak.ps) == 0 {
		return res
	}

	peakStartIdx := findPointIndexByTime(ps, peak.ps[0].ts)
	peakEndIdx := findPointIndexByTime(ps, peak.ps[len(peak.ps)-1].ts)
	if peakStartIdx <= 0 || peakEndIdx <= peakStartIdx {
		for _, threshold := range thresholds {
			res = append(res, speedRunThresholdDetails{threshold: threshold})
		}
		return res
	}

	for _, threshold := range thresholds {
		startIdx := peakStartIdx
		for startIdx > 0 && speed(ps[startIdx-1], ps[startIdx], speedUnits) >= threshold {
			startIdx--
		}

		endIdx := peakEndIdx
		for endIdx < len(ps)-1 && speed(ps[endIdx], ps[endIdx+1], speedUnits) >= threshold {
			endIdx++
		}

		duration, dist := sumSegments(ps, startIdx, endIdx)
		res = append(res, speedRunThresholdDetails{threshold: threshold, duration: duration, distance: dist})
	}
	return res
}

func stabilityScore(ps []Point, peak Track, windowTrack Track, headingStd float64, windowSecs float64, speedUnits UnitsFlag) speedRunStabilityScore {
	headingScore := scoreLinearDescending(headingStd, 1, 8, 40)

	retentionScore := 0
	if peak.speed > 0 {
		retention := windowTrack.speed / peak.speed
		retentionScore = scoreLinearAscending(retention, 0.88, 0.97, 35)
	}

	accelScore := accelerationSmoothnessScore(ps, peak, windowSecs, speedUnits)
	total := headingScore + retentionScore + accelScore
	return speedRunStabilityScore{heading: headingScore, retention: retentionScore, acceleration: accelScore, total: total}
}

func accelerationSmoothnessScore(ps []Point, peak Track, windowSecs float64, speedUnits UnitsFlag) int {
	if len(peak.ps) == 0 {
		return 0
	}
	peakEndIdx := findPointIndexByTime(ps, peak.ps[len(peak.ps)-1].ts)
	if peakEndIdx <= 2 {
		return 0
	}

	startTime := ps[peakEndIdx].ts.Add(-time.Duration(windowSecs * float64(time.Second)))
	startIdx := 1
	for i := peakEndIdx; i > 1; i-- {
		if ps[i].ts.Before(startTime) {
			startIdx = i
			break
		}
	}

	totalSteps := 0
	smoothSteps := 0
	prevSpeed := speed(ps[startIdx-1], ps[startIdx], speedUnits)
	for i := startIdx + 1; i <= peakEndIdx; i++ {
		currSpeed := speed(ps[i-1], ps[i], speedUnits)
		if currSpeed >= prevSpeed-0.1 {
			smoothSteps++
		}
		totalSteps++
		prevSpeed = currSpeed
	}
	if totalSteps == 0 {
		return 0
	}
	ratio := float64(smoothSteps) / float64(totalSteps)
	return int(math.Round(ratio * 25))
}

func scoreLinearDescending(value, best, worst float64, maxScore int) int {
	if value <= best {
		return maxScore
	}
	if value >= worst {
		return 0
	}
	score := float64(maxScore) * (worst - value) / (worst - best)
	return int(math.Round(score))
}

func scoreLinearAscending(value, worst, best float64, maxScore int) int {
	if value >= best {
		return maxScore
	}
	if value <= worst {
		return 0
	}
	score := float64(maxScore) * (value - worst) / (best - worst)
	return int(math.Round(score))
}

func findPointIndexByTime(ps []Point, ts time.Time) int {
	for i, p := range ps {
		if p.ts.Equal(ts) {
			return i
		}
	}
	return -1
}

func sumSegments(ps []Point, startIdx, endIdx int) (float64, float64) {
	if startIdx < 0 || endIdx <= startIdx || endIdx >= len(ps) {
		return 0, 0
	}
	duration := 0.0
	dist := 0.0
	for i := startIdx + 1; i <= endIdx; i++ {
		duration += ps[i].ts.Sub(ps[i-1].ts).Seconds()
		dist += distance(ps[i-1], ps[i])
	}
	return duration, dist
}

func normalizeAngle(angle float64) float64 {
	return math.Mod(angle+360, 360)
}

func signedAngleDiff(a, b float64) float64 {
	return math.Mod(a-b+540, 360) - 180
}

func absAngleDiff(a, b float64) float64 {
	return math.Abs(signedAngleDiff(a, b))
}
