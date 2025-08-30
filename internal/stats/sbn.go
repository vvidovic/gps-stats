package stats

import (
	"bytes"
	"io"
	"time"

	"github.com/vvidovic/gps-stats/internal/errs"
)

// ReadPointsSbn reads all available SBN Points from the Reader.
func ReadPointsSbn(r io.Reader) (Points, error) {
	ps := []Point{}
	res := Points{Name: "SBN track", Ps: ps}

	p, err := readPointSbn(r)
	for err == nil {
		if err != nil {
			res.Ps = ps
			return res, err
		}

		if p.isPoint {
			p.globalIdx = len(ps)
			ps = append(ps, p)
		}

		p, err = readPointSbn(r)
	}

	res.Ps = ps
	return res, err
}

// readPointSbn reads a next potential SBN Point from the Reader.
// If no point is found, return Point with isPoint set to false.
func readPointSbn(r io.Reader) (Point, error) {
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

	return Point{isPoint: true, lat: lat, lon: lon, ts: ts, heading: -1, tackType: TackUnknown}, nil
}
