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
	} else if *helpFlag || len(flag.Args()) < 1 {
		showUsage(0)
	} else {
		for i := 0; i < len(flag.Args()); i++ {
			printStatsForFile(flag.Args()[i])
		}
	}
}

func printStatsForFile(filePath string) {
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

	s := stats.CalculateStats(ps)

	fileName := filepath.Base(f.Name())
	switch *statTypeFlag {
	case "all":
		fmt.Printf("Found %d track points in '%s', after cleanup %d points left.\n",
			pointsNo, fileName, pointsCleanedNo)
		fmt.Print(s.TxtStats())
	case "2s":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat2s), fileName)
	case "10sAvg":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10sAvg), fileName)
	case "10s1":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10s1), fileName)
	case "10s2":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10s2), fileName)
	case "10s3":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10s3), fileName)
	case "10s4":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10s4), fileName)
	case "10s5":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat10s5), fileName)
	case "15m":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat15m), fileName)
	case "1h":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat1h), fileName)
	case "100m":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat100m), fileName)
	case "1nm":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.Stat1nm), fileName)
	case "alpha":
		fmt.Printf("%s (%s)", s.TxtSingleStat(stats.StatAlpha), fileName)
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
	fmt.Println("  -h Show usage")
	fmt.Println("  -v Show version")
	fmt.Println("Single stat value flags (only one allowed):")
	fmt.Println("  -2s     Print the 2 second peak speed")
	fmt.Println("  -10sAvg Print the 5x10 second average speed")
	fmt.Println("  -10s1   Print the first 10 second peak speed")
	fmt.Println("  -10s2   Print the second 10 second peak speed")
	fmt.Println("  -10s3   Print the third 10 second peak speed")
	fmt.Println("  -10s4   Print the fourth 10 second peak speed")
	fmt.Println("  -10s5   Print the fifth 10 second peak speed")
	fmt.Println("  -15m    Print the 15 minute speed")
	fmt.Println("  -1h     Print the 1 hour speed")
	fmt.Println("  -100m   Print the 100 meter peak speed")
	fmt.Println("  -1nm    Print the nautical mile (1582 meter) peak speed")
	fmt.Println("  -alpha  Print the alpha (max 500 m) peak speed")
	fmt.Println("")
	fmt.Println("Example:")
	fmt.Printf(" %s my_gps_data.SBN\n", os.Args[0])
	fmt.Println("   - runs analysis of the SBN data")

	os.Exit(exitStatus)
}
