package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vvidovic/gps-stats/internal/stats"
	"github.com/vvidovic/gps-stats/internal/version"
)

var (
	helpFlag    *bool
	versionFlag *bool
)

func main() {
	helpFlag = flag.Bool("h", false, "Show gps-stats usage with examples")
	versionFlag = flag.Bool("v", false, "Show gps-stats version")

	flag.Parse()

	if *versionFlag {
		showVersion()
	} else if *helpFlag || len(flag.Args()) != 1 {
		showUsage(0)
	} else {
		f, err := os.Open(flag.Args()[0])
		if err != nil {
			return
		}

		r := bufio.NewReader(f)

		ps, err := stats.ReadPoints(r)

		if err != nil {
			fmt.Printf("Error reading points: %v\n", err)
			return
		}

		pointsNo := len(ps)
		ps = stats.CleanUp(ps)
		pointsCleanedNo := len(ps)

		fmt.Printf("Found %d track points in '%s', after cleanup %d points left.\n",
			pointsNo, filepath.Base(f.Name()), pointsCleanedNo)

		s := stats.CalculateStats(ps)
		fmt.Print(s.TxtStats())
		fmt.Println("")
	}

}

func showVersion() {
	fmt.Printf("gps-stat version %s %s %s\n", version.Version, version.Platform, version.BuildTime)

	os.Exit(0)
}

// usage prints usage help information with examples to console.
func showUsage(exitStatus int) {
	fmt.Println("Usage:")
	fmt.Printf(" %s GPS_data_file\n", os.Args[0])
	fmt.Println("")
	fmt.Println("")
	fmt.Println("Example:")
	fmt.Printf(" %s my_gps_data.SBN\n", os.Args[0])
	fmt.Println("   - runs analysis of the SBN data")

	os.Exit(exitStatus)
}