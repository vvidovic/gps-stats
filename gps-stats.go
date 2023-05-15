package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/vvidovic/gps-stats/internal/stats"
	"github.com/vvidovic/gps-stats/internal/version"
)

var (
	helpFlag                   *bool
	versionFlag                *bool
	statTypeFlag               *string
	cleanupDeltaPercentageFlag *int
	cleanupDeltaKnotsFlag      *float64
	saveFilteredGpxFlag        *bool
)

func main() {
	helpFlag = flag.Bool("h", false, "Show gps-stats usage with examples")
	versionFlag = flag.Bool("v", false, "Show gps-stats version")
	statTypeFlag = flag.String("t", "all",
		"Set the statistics type to print (all, 2s, 10sAvg, 10s1, 10s2, 10s3, 10s4, 10s5, 15m, 1h, 100m, 1nm, alpha)")
	cleanupDeltaPercentageFlag = flag.Int("csp", 50,
		"Clean up points where difference in speed is more than given percentage (default 50 %)")
	cleanupDeltaKnotsFlag = flag.Float64("csk", 3,
		"Clean up points where difference in speed is more than given number of knots (default 3 kts)")
	saveFilteredGpxFlag = flag.Bool("sf", false, "Save filtered track to a new GPX file")

	flag.Parse()

	if *versionFlag {
		showVersion()
	} else if *helpFlag {
		showUsage(0)
	} else if len(flag.Args()) < 1 {
		showUsage(1)
	} else {
		statType := stats.StatNone
		switch *statTypeFlag {
		case "all":
			statType = stats.StatAll
		case "2s":
			statType = stats.Stat2s
		case "10sAvg":
			statType = stats.Stat10sAvg
		case "10s1":
			statType = stats.Stat10s1
		case "10s2":
			statType = stats.Stat10s2
		case "10s3":
			statType = stats.Stat10s3
		case "10s4":
			statType = stats.Stat10s4
		case "10s5":
			statType = stats.Stat10s5
		case "15m":
			statType = stats.Stat15m
		case "1h":
			statType = stats.Stat1h
		case "100m":
			statType = stats.Stat100m
		case "1nm":
			statType = stats.Stat1nm
		case "alpha":
			statType = stats.StatAlpha
		default:
			showUsage(2)
			return
		}

		for i := 0; i < len(flag.Args()); i++ {
			printStatsForFile(flag.Args()[i], statType)
		}
	}
}

func printStatsForFile(filePath string, statType stats.StatFlag) {
	f, err := os.Open(filePath)
	if err != nil {
		return
	}

	fileName := filepath.Base(f.Name())

	r := bufio.NewReader(f)

	ps, err := stats.ReadPoints(r)

	if err != nil && err != io.EOF {
		fmt.Printf("Error reading track points from '%s': %v\n", fileName, err)
		if statType == stats.StatAll {
			fmt.Println("")
		}
		return
	}

	pointsNo := len(ps)
	ps = stats.CleanUp(ps, *cleanupDeltaPercentageFlag, *cleanupDeltaKnotsFlag)
	pointsCleanedNo := len(ps)

	if *saveFilteredGpxFlag {
		newFilePath := filePath + ".filtered.gpx"
		f, err := os.Create(newFilePath)
		if err != nil {
			fmt.Printf("Error creating new file '%s' for GPX export: %v\n", newFilePath, err)
			if statType == stats.StatAll {
				fmt.Println("")
			}
			return
		}

		err = stats.SavePointsAsGpx(ps, f)
		if err != nil {
			fmt.Printf("Error saving file '%s' for GPX export: %v\n", newFilePath, err)
			if statType == stats.StatAll {
				fmt.Println("")
			}
			return
		}

		fmt.Printf("Filtered GPX file '%s' saved.\n", newFilePath)
		if statType == stats.StatAll {
			fmt.Println("")
		}
	}

	s := stats.CalculateStats(ps, statType)

	switch statType {
	case stats.StatAll:
		fmt.Printf("Found %d track points in '%s', after cleanup %d points left.\n",
			pointsNo, fileName, pointsCleanedNo)
		fmt.Print(s.TxtStats())
	default:
		fmt.Printf("%s (%s)", s.TxtSingleStat(statType), fileName)
	}
	fmt.Println("")
}

func showVersion() {
	fmt.Printf("gps-stat version %s %s %s\n", version.Version, version.Platform, version.BuildTime)

	os.Exit(0)
}

// usage prints usage help information with examples to console.
func showUsage(exitStatus int) {
	fmt.Println("Usage:")
	fmt.Printf(" %s GPS_data_file1 [GPS_data_file2 ...]\n", os.Args[0])
	fmt.Println("")
	fmt.Println("Parses 1 or more GPS data files (SBN or GPX)")
	fmt.Println("")
	fmt.Println("Flags:")
	fmt.Println("  -h Show usage (optional)")
	fmt.Println("  -v Show version (optional)")
	fmt.Println("  -t Set the statistics type to print (optional)")
	fmt.Println("     (all, 2s, 10sAvg, 10s1, 10s2, 10s3, 10s4, 10s5, 15m, 1h, 100m, 1nm, alpha)")
	fmt.Println("  -sf Save filtered points as a new GPX file without points detected as errors")
	fmt.Println("      with suffix '.filtered.gpx' (optional)")
	fmt.Println("")
	fmt.Println("  -csp Clean up points where difference in speed is more than given percentage (default 50 %)")
	fmt.Println("  -csk Clean up points where difference in speed is more than given number of knots (default 3 kts)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf(" %s my_gps_data.SBN\n", os.Args[0])
	fmt.Println("   - runs analysis of the SBN data")
	fmt.Println("")
	fmt.Printf(" %s -csp 70 -csk 7 my_gps_data.gpx\n", os.Args[0])
	fmt.Println("   - runs analysis of the SBN data with custom clean up settings")
	fmt.Println("")
	fmt.Printf(" %s -t=1nm *.SBN *.gpx\n", os.Args[0])
	fmt.Println("   - runs analysis of multiple SBN & GPX data only for 1 NM statistics")
	fmt.Println("")
	fmt.Printf(" %s -sf my_gps_data.GPX\n", os.Args[0])
	fmt.Println("   - runs analysis of the GPX data and save a copy of track with filtered points detected as errors")

	os.Exit(exitStatus)
}
