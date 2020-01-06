/*
Copyright © 2020 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"bufio"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var searchString string
var multiline bool
var pathstart string
var startpath string
var searchfrom string
var searchduration string
var terse bool
var context int

const (
	forward  = 1
	backward = -1
)

// searchCmd represents the search command
var searchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search for a term or terms in the directory history files.",
	Long: `Specify search terms, can use | and & between terms.  Order of
operation is to split the ORs and process all of the ANDs between the ORs.
For example:

search-directory-history search "kubectl & namespace | kubectl & statefulset"
  - this will search for both sides of the | separately.
  - it will NOT process "kubectl & (namespace | statefulset)" correctly.
  - currently this is a very basic searching feature.  I haven't found a need
  - to be more concise than what the basic order of operations handles.

search-directory-history search "statefulset | configmap | deployment"
`,
	Aliases: []string{"s"},
	Run: func(cmd *cobra.Command, args []string) {

		var baseDir string
		multiline, _ = cmd.Flags().GetBool("multiline")
		terse, _ = cmd.Flags().GetBool("terse")
		context, _ = cmd.Flags().GetInt("context")
		if len(os.Getenv("HISTORY_BASE")) > 0 {
			baseDir = os.Getenv("HISTORY_BASE")
		} else {
			usr, err := user.Current()
			if err != nil {
				fmt.Println("Erorr fetching the user's home directory")
				os.Exit(1)
			}
			baseDir = fmt.Sprintf("%s/.directory_history", usr.HomeDir)
		}
		searchPath := fmt.Sprintf("%s/", baseDir)
		startpath, _ = cmd.Flags().GetString("startpath")
		pathstart, _ = cmd.Flags().GetString("pathstart")
		startpath = addLeadingSlash(startpath)
		pathstart = addLeadingSlash(pathstart)
		if startpath != "/" {
			searchPath = fmt.Sprintf("%s%s", baseDir, startpath)
		}
		if pathstart != "/" {
			searchPath = fmt.Sprintf("%s%s", baseDir, pathstart)
		}
		searchfrom, _ = cmd.Flags().GetString("searchfrom")
		searchduration, _ = cmd.Flags().GetString("searchduration")
		searchString = args[0]
		filepath.Walk(searchPath, parseDirectoryHistory)
	},
}

func addLeadingSlash(s string) string {
	if s[0] != '/' {
		return fmt.Sprintf("/%s", s)
	} else {
		return s
	}
}

// leadingInt consumes the leading [0-9]* from s.
func leadingInt(s string) (x int, rem string, err error) {
	i := 0
	for ; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			break
		}
		if x > (1<<63-1)/10 {
			// overflow
			return 0, "", errors.New("time: bad [0-9]*")
		}
		x = x*10 + int(c) - '0'
		if x < 0 {
			// overflow
			return 0, "", errors.New("time: bad [0-9]*")
		}
	}
	return x, s[i:], nil
}

func getDurationDate(duration string, timeStart time.Time, direction int) time.Time {
	var durationDate time.Time
	var yearAdjust int
	var monthAdjust int
	var dayAdjust int

	yearAdjust = 0
	monthAdjust = 0
	dayAdjust = 0
	for duration != "" {

		var value int
		var err error
		// The next character must be [0-9.]
		if !(duration[0] == '.' || '0' <= duration[0] && duration[0] <= '9') {
			// return our default of -5 years
			return time.Now().AddDate(-5, 0, 0)
		}
		// Consume [0-9]*
		value, duration, err = leadingInt(duration)
		if err != nil {
			// return our default of -5 years
			return time.Now().AddDate(-5, 0, 0)
		}

		// Consume unit.
		i := 0
		for ; i < len(duration); i++ {
			c := duration[i]
			if c == '.' || '0' <= c && c <= '9' {
				break
			}
		}
		if i == 0 {
			// return our default of -5 years
			return time.Now().AddDate(-5, 0, 0)
		}
		u := duration[:i]
		duration = duration[i:]
		switch u {
		case "y":
			yearAdjust = value
		case "m":
			monthAdjust = value
		case "w":
			dayAdjust = dayAdjust + value*7
		case "d":
			dayAdjust = dayAdjust + value
		}
	}

	durationDate = timeStart.AddDate((yearAdjust * direction), (monthAdjust * direction), (dayAdjust * direction))

	return durationDate
}

func parseDirectoryHistory(path string, f os.FileInfo, err error) error {
	var histLines []string
	var contextLookup []string
	if strings.HasSuffix(path, "history") {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		// Splits on newlines by default.
		scanner := bufio.NewScanner(f)

		afterDate := getDurationDate(searchfrom, time.Now(), backward)
		beforeDate := getDurationDate(searchduration, afterDate, forward)
		// fmt.Printf("> %v > %v\n", afterDate, beforeDate)
		line := 1
		var cmdLines []string
		var strDate string
		var cmdDate time.Time
		var multilineAdded bool
		var displayCmd string
		var plainCmd string
		var haveMatch bool
		var afterMatchCount int
		hashes := make(map[string]string)
		haveMatch = false
		for scanner.Scan() {
			var cmdstart = strings.Index(scanner.Text(), ":")
			if cmdstart == 0 {
				multilineAdded = false
				cmdLines = cmdLines[:0]
				// cmdLines = append(cmdLines, scanner.Text())
				i, _ := strconv.ParseInt((strings.TrimSpace(strings.Split(scanner.Text(), ":")[1])), 10, 64)
				t := time.Unix(i, 0)
				cmdDate = t
				strDate = t.Format("2006-01-02")
			}
			var index = strings.Index(scanner.Text(), ";")
			if index > 0 && cmdstart == 0 {
				displayCmd = fmt.Sprintf("%s: %s", strDate, scanner.Text()[(index+1):])
				plainCmd = fmt.Sprintf("%s", scanner.Text()[(index+1):])
			} else {
				displayCmd = fmt.Sprintf("%s: %s", strDate, scanner.Text())
				plainCmd = fmt.Sprintf("%s", scanner.Text())
			}
			cmdLines = append(cmdLines, displayCmd)
			if cmdDate.After(afterDate) && cmdDate.Before(beforeDate) {
				hash := sha256.New()
				hash.Write([]byte(plainCmd))
				if _, prs := hashes[fmt.Sprintf("%x", hash.Sum(nil))]; prs == false {
					hashes[fmt.Sprintf("%x", hash.Sum(nil))] = fmt.Sprintf("%x", hash.Sum(nil))
					//if strings.Contains(scanner.Text(), searchString) {
					if basicSearch(searchString, scanner.Text()) {
						if context > 0 {
							haveMatch = true
							afterMatchCount = 0
							contextLookup = append(contextLookup, fmt.Sprintf("=%s", displayCmd))
						} else {
							if multiline && !multilineAdded {
								for _, cmdline := range cmdLines {
									histLines = append(histLines, cmdline)
								}
								multilineAdded = true
							} else {
								histLines = append(histLines, displayCmd)
							}
						}
					} else {
						if context > 0 {
							if haveMatch {
								if afterMatchCount >= context {
									for _, cmdline := range contextLookup {
										histLines = append(histLines, cmdline)
									}
									haveMatch = false
									afterMatchCount = 0
								} else {
									contextLookup = append(contextLookup, fmt.Sprintf("-%s", displayCmd))
									afterMatchCount++
								}
							} else {
								contextLookup = append(contextLookup, fmt.Sprintf("+%s", displayCmd))
								if len(contextLookup) > context {
									contextLookup = contextLookup[2:]
								}
							}
						}
					}
					line++
				}
			}
		}
		if haveMatch && len(contextLookup) > 0 {
			for _, cmdline := range contextLookup {
				histLines = append(histLines, cmdline)
			}
		}

		if err := scanner.Err(); err != nil {
			// Handle the error
		}
	}
	if len(histLines) > 0 {
		if !terse {
			fmt.Printf("\n%s:\n", path)
			fmt.Printf("-----------------\n")
		}
		for _, cmd := range histLines {
			fmt.Printf("%s\n", cmd)
		}
	}
	return nil
}

func basicSearch(searchTerms string, cmdString string) bool {

	matched := false
	if strings.Contains(searchTerms, "|") {
		// Loop through all the "Ored" terms
		orTerms := strings.Split(searchTerms, "|")
		for _, orTerm := range orTerms {
			// Handle "Anded" terms for each "Ored" section
			andTerms := strings.Split(orTerm, "&")
			matched = matched || andSearchCmd(andTerms, cmdString)
		}
	} else {
		// Handle JUST "Anded" terms
		andTerms := strings.Split(searchString, "&")
		matched = andSearchCmd(andTerms, cmdString)
	}
	return matched
}

func andSearchCmd(searchTerms []string, cmdString string) bool {

	allMatch := true

	for _, term := range searchTerms {
		allMatch = allMatch && strings.Contains(cmdString, strings.TrimSpace(term))
		if !allMatch {
			break
		}
	}

	return allMatch
}
func init() {
	rootCmd.AddCommand(searchCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// searchCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// searchCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	searchCmd.Flags().BoolP("multiline", "m", false, "Show full multiline commands with match")
	searchCmd.Flags().BoolP("terse", "t", false, "Suppress the directory name output")
	searchCmd.Flags().String("pathstart", "/", "Specify a starting relative path to start the search under")
	searchCmd.Flags().String("startpath", "/", "Specify a starting relative path to start the search under")
	searchCmd.Flags().String("searchfrom", "5y", "Specify how far back to search, in y,m,w,d")
	searchCmd.Flags().String("searchduration", "5y", "Specify duration to search from 'searchfrom', in y,m,w,d.  Defaults to 'searchfrom' value.")
	searchCmd.Flags().Int("context", 0, "How many lines on each side of match to display")
}
