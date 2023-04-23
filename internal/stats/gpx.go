package stats

import (
	"encoding/xml"
	"io"
	"io/ioutil"
	"time"
)

// Gpx contains all tracks from a GPX file.
type Gpx struct {
	XMLName xml.Name `xml:"gpx"`
	Trks   []Trk   `xml:"trk"`
}

// Trk contains a single track from a GPX file
// with multiple segments.
type Trk struct {
	XMLName xml.Name `xml:"trk"`
	Trksegs   []Trkseg   `xml:"trkseg"`
}

// Trkseg contains a single track segment from a GPX file
// track with multiple points.
type Trkseg struct {
	XMLName xml.Name `xml:"trkseg"`
	Trkpts   []Trkpt   `xml:"trkpt"`
}

// Trkseg contains a single track segment from a GPX file
// track segment with multiple points.
type Trkpt struct {
	XMLName xml.Name `xml:"trkpt"`
	Lat    float64   `xml:"lat,attr"`
	Lon    float64   `xml:"lon,attr"`
	Time   time.Time    `xml:"time"`
}

// ReadPointsGpx reads all available SBN Points from the Reader.
func ReadPointsGpx(r io.Reader) ([]Point, error) {
	res := []Point{}

	byteValue, err := ioutil.ReadAll(r)
	if err != nil {
		return res, err
	}

	var gpx Gpx
	// we unmarshal our byteArray which contains our
	err = xml.Unmarshal(byteValue, &gpx)
	if err != nil {
		return res, err
	}

	for trkIdx := 0; trkIdx < len(gpx.Trks); trkIdx++ {
		for segIdx := 0; segIdx < len(gpx.Trks[trkIdx].Trksegs); segIdx++ {
			points := gpx.Trks[trkIdx].Trksegs[segIdx].Trkpts
			for ptIdx := 0; ptIdx < len(points); ptIdx++ {
				p, err := readPointGpx(points[ptIdx])
				if err != nil {
					return res, err
				}

				if p.isPoint {
					p.globalIdx = len(res)
					res = append(res, p)
				}
			}
		}
	}

	return res, err
}

// readPointGpx transforms a track point from a GPX file
// to internal Point structure.
func readPointGpx(trkpt Trkpt) (Point, error) {
	return Point{isPoint: true, lat: trkpt.Lat, lon: trkpt.Lon, ts: trkpt.Time}, nil
}