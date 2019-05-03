package main

import (
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"
)

func csvWriter(w io.Writer, results []Result) error {
	if len(results) == 0 {
		return nil
	}

	// Deduplicate all times from all results by passing them as key into a map.
	timesMap := make(map[string]bool)
	for _, result := range results {
		for tm := range result.Values {
			timesMap[tm] = true
		}
	}

	// Create a sorted slice of all times to iterate over later.
	var times []string
	for tm := range timesMap {
		times = append(times, tm)
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	// Iterate over all times and find the belonging values for each result.
	for _, tm := range times {
		i64, _ := strconv.ParseInt(tm, 10, 64)
		uxdate := time.Unix(i64, 0).String()
		fmt.Fprint(w, uxdate)
		for _, result := range results {
			fmt.Fprint(w, ","+result.Values[tm])
		}
		fmt.Fprintln(w)
	}

	return nil
}

func csvHeaderWriter(w io.Writer, results []Result) error {
	if len(results) == 0 {
		return nil
	}

	header := []string{"Time"}
	for _, result := range results {
		header = append(header, result.Metric)
	}

	fmt.Fprintln(w, strings.Join(header, ","))
	return nil
}

func matplotlibWriter(w io.Writer, results []Result) error {
	if len(results) == 0 {
		return nil
	}

	// Deduplicate all times from all results by passing them as key into a map.
	timesMap := make(map[string]bool)
	for _, result := range results {
		for tm := range result.Values {
			timesMap[tm] = true
		}
	}

	// Create a sorted slice of all times to iterate over later.
	var times []string
	for tm := range timesMap {
		times = append(times, tm)
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i] < times[j]
	})

	fmt.Fprintf(w, "t = [%s]\n", strings.Join(times, ", "))

	for i, result := range results {
		var vals []string
		for _, tm := range times {
			if val, ok := result.Values[tm]; ok {
				vals = append(vals, val)
			} else {
				vals = append(vals, "None")
			}
		}
		fmt.Fprintf(w, "s%d = [%s]\n", i, strings.Join(vals, ", "))
		fmt.Fprintf(w, "plot.plot(t, s%d)\n", i)
	}

	return nil
}

func matplotlibLegendWriter(w io.Writer, results []Result) error {
	labels := []string{}
	for _, result := range results {
		labels = append(labels, fmt.Sprintf("'%s'", result.Metric))
	}

	fmt.Fprintf(w, "plot.legend([%s], loc='upper left')\n", strings.Join(labels, ", "))

	return nil
}
