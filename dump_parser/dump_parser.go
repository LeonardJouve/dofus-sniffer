package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func main() {
	allProtoEnums := make(map[string]map[string][]string)
	protoEnumRegexp := regexp.MustCompile(`enum (\S+) \{`)
	protoNameRegexp := regexp.MustCompile(`\s*(\S+)\s*=\s*(\d+);`)
	filepath.WalkDir("proto", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}

		fileName := strings.Replace(filepath.Base(path), ".proto", "", 1)
		protoEnums := parseEnums(path, protoEnumRegexp, protoNameRegexp)
		allProtoEnums[fileName] = protoEnums

		return nil
	})
	csEnumRegexp := regexp.MustCompile(`public enum (\S+)`)
	csNameRegexp := regexp.MustCompile(`\[OriginalName\("([^"]+)"\)\]`)
	csEnums := parseEnums("dump.cs", csEnumRegexp, csNameRegexp)

	for csEnumName, csValues := range csEnums {
		for fileName, protoEnums := range allProtoEnums {
			for protoEnumName, protoValues := range protoEnums {
				if len(csValues) != len(protoValues) {
					continue
				}

				hasMatched := true

				for i, protoValue := range protoValues {
					if protoValue != csValues[i] {
						hasMatched = false
					}
				}

				if !hasMatched {
					continue
				}

				fmt.Printf("%s -> %s/%s\n", csEnumName, fileName, protoEnumName)
			}
		}
	}
}

func parseEnums(fileName string, enumRegexp *regexp.Regexp, nameRegexp *regexp.Regexp) map[string][]string {
	enumMap := make(map[string][]string)

	file, err := os.Open(fileName)
	if err != nil {
		return enumMap
	}
	defer file.Close()

	var currentEnum string
	hasCurrentEnum := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.Contains(line, "}") {
			hasCurrentEnum = false
		}

		matches := enumRegexp.FindStringSubmatch(line)
		if matches != nil {
			currentEnum = matches[1]
			hasCurrentEnum = true
			continue
		}

		matches = nameRegexp.FindStringSubmatch(line)
		if hasCurrentEnum && matches != nil {
			enumMap[currentEnum] = append(enumMap[currentEnum], matches[1])
			continue
		}
	}

	return enumMap
}
