package stats

import (
	"encoding/xml"
	"fmt"
	"io"
	"time"

	"github.com/vvidovic/gps-stats/internal/version"
)

// Gpx contains all tracks from a GPX file.
type Gpx struct {
	XMLName  xml.Name  `xml:"gpx"`
	Creator  string    `xml:"creator,attr"`
	Version  string    `xml:"version,attr"`
	XMLNS    string    `xml:"xmlns,attr"`
	Ns3      string    `xml:"xmlns:ns3,attr,omitempty"`
	Metadata *Metadata `xml:"metadata,omitempty"`
	Trks     []Trk     `xml:"trk"`
}

// Metadata is optional element with additional info about track.
type Metadata struct {
	XMLName xml.Name  `xml:"metadata"`
	Link    *Link     `xml:"link,omitempty"`
	Time    time.Time `xml:"time,omitempty"`
}

// Link is element within metadata.
type Link struct {
	XMLName xml.Name `xml:"link"`
	Href    string   `xml:"href,attr,omitempty"`
	Text    string   `xml:"text,omitempty"`
}

// Trk contains a single track from a GPX file
// with multiple segments.
type Trk struct {
	XMLName xml.Name `xml:"trk"`
	Name    string   `xml:"name"`
	Type    string   `xml:"type,omitempty"`
	Trksegs []Trkseg `xml:"trkseg"`
}

// Trkseg contains a single track segment from a GPX file
// track with multiple points.
type Trkseg struct {
	XMLName xml.Name `xml:"trkseg"`
	Trkpts  []Trkpt  `xml:"trkpt"`
}

// Trkseg contains a single track segment from a GPX file
// track segment with multiple points.
type Trkpt struct {
	XMLName    xml.Name    `xml:"trkpt"`
	Lat        float64     `xml:"lat,attr"`
	Lon        float64     `xml:"lon,attr"`
	Ele        float64     `xml:"ele,omitempty"`
	Time       time.Time   `xml:"time"`
	Extensions *Extensions `xml:"extensions,omitempty"`
}

// Extensions contains non-Gpx namespaced extension elements.
type Extensions struct {
	XMLName             xml.Name             `xml:"extensions"`
	TrackPointExtension *TrackPointExtension `xml:"TrackPointExtension,omitempty"`
}

// TrackPointExtension contains trimmed-down combination of
// Garmin trackpoint extension v1 used by Garmin & Amazfit.
type TrackPointExtension struct {
	XMLName xml.Name `xml:"TrackPointExtension"`
	Speed   float64  `xml:"speed,omitempty"`
	Hr      int16    `xml:"hr,omitempty"`
}

// ReadPointsGpx reads all available GPX Points from the Reader.
func ReadPointsGpx(r io.Reader) (Points, error) {
	ps := []Point{}
	res := Points{Ps: ps}

	byteValue, err := io.ReadAll(r)
	if err != nil {
		return res, err
	}

	var gpx Gpx
	// we unmarshal our byteArray which contains our
	err = xml.Unmarshal(byteValue, &gpx)
	if err != nil {
		return res, err
	}

	if len(gpx.Trks) > 0 {
		res.Name = gpx.Trks[0].Name
		res.Creator = gpx.Creator
	}

	for trkIdx := 0; trkIdx < len(gpx.Trks); trkIdx++ {
		for segIdx := 0; segIdx < len(gpx.Trks[trkIdx].Trksegs); segIdx++ {
			points := gpx.Trks[trkIdx].Trksegs[segIdx].Trkpts
			for ptIdx := 0; ptIdx < len(points); ptIdx++ {
				p, err := readPointGpx(points[ptIdx])
				if err != nil {
					res.Ps = ps
					return res, err
				}

				if p.isPoint {
					p.globalIdx = len(ps)
					ps = append(ps, p)
				}
			}
		}
	}

	res.Ps = ps
	return res, err
}

// readPointGpx transforms a track point from a GPX file
// to internal Point structure.
func readPointGpx(trkpt Trkpt) (Point, error) {
	pt := Point{isPoint: true, lat: trkpt.Lat, lon: trkpt.Lon, ts: trkpt.Time, ele: trkpt.Ele}
	if trkpt.Extensions != nil && trkpt.Extensions.TrackPointExtension != nil {
		tpe := trkpt.Extensions.TrackPointExtension
		pt.speed = &tpe.Speed
		pt.hr = &tpe.Hr
	}
	return pt, nil
}

// SavePointsAsGpx save points as GPX file.
func SavePointsAsGpx(p Points, w io.Writer) error {
	gpx := Gpx{
		XMLNS:   "http://www.topografix.com/GPX/1/1",
		Ns3:     "http://www.garmin.com/xmlschemas/TrackPointExtension/v1",
		Creator: fmt.Sprintf("gps-stat version %s %s %s from %s", version.Version, version.Platform, version.BuildTime, p.Creator),
		Version: "1.1",
		Trks: []Trk{{
			Name: p.Name + " - cleaned up by gps-stat",
			Trksegs: []Trkseg{
				{Trkpts: []Trkpt{}}}}}}
	trkpts := gpx.Trks[0].Trksegs[0].Trkpts
	if p.Type != "" {
		gpx.Trks[0].Type = p.Type
	}

	ps := p.Ps
	for pIdx := 0; pIdx < len(ps); pIdx++ {
		p := ps[pIdx]
		trkpt := Trkpt{
			Lat:  p.lat,
			Lon:  p.lon,
			Time: p.ts,
			Ele:  p.ele}
		if p.speed != nil || p.hr != nil {
			trkpt.Extensions = &Extensions{TrackPointExtension: &TrackPointExtension{Speed: *p.speed, Hr: *p.hr}}
		}
		trkpts = append(trkpts, trkpt)
	}

	gpx.Trks[0].Trksegs[0].Trkpts = trkpts

	byteVal, err := xml.MarshalIndent(gpx, "", "  ")
	if err != nil {
		return err
	}
	xmlHeader := `<?xml version="1.0" encoding="UTF-8"?>` + "\n"
	_, err = w.Write([]byte(xmlHeader))
	if err != nil {
		return err
	}
	_, err = w.Write(byteVal)
	return err
}
