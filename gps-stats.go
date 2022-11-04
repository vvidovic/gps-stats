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
	helpFlag     *bool
	versionFlag  *bool
	statTypeFlag *string
)

func main() {
	helpFlag = flag.Bool("h", false, "Show gps-stats usage with examples")
	versionFlag = flag.Bool("v", false, "Show gps-stats version")
	statTypeFlag = flag.String("t", "all",
		"Set the statistics type to print (all, 2s, 10sAvg, 10s1, 10s2, 10s3, 10s4, 10s5, 15m, 1h, 100m, 1nm, alpha)")

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

	r := bufio.NewReader(f)

	ps, err := stats.ReadPoints(r)

	if err != nil {
		fmt.Printf("Error reading points: %v\n", err)
		return
	}

	pointsNo := len(ps)
	ps = stats.CleanUp(ps)
	pointsCleanedNo := len(ps)

	s := stats.CalculateStats(ps, statType)

	fileName := filepath.Base(f.Name())

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
	fmt.Println("Flags:")
	fmt.Println("  -h Show usage (optional)")
	fmt.Println("  -v Show version (optional)")
	fmt.Println("  -t Set the statistics type to print (optional)")
	fmt.Println("    (all, 2s, 10sAvg, 10s1, 10s2, 10s3, 10s4, 10s5, 15m, 1h, 100m, 1nm, alpha)")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Printf(" %s my_gps_data.SBN\n", os.Args[0])
	fmt.Println("   - runs analysis of the SBN data")
	fmt.Println("")
	fmt.Printf(" %s -t=1nm *.SBN\n", os.Args[0])
	fmt.Println("   - runs analysis of multiple SBN data only for 1 NM statistics")

	os.Exit(exitStatus)
}
