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

		namespace := strings.Replace(filepath.Base(path), ".proto", "", 1)
		protoEnums := parseEnums(path, protoEnumRegexp, protoNameRegexp)
		allProtoEnums[namespace] = protoEnums

		return nil
	})
	csEnumRegexp := regexp.MustCompile(`public enum (\S+)`)
	csNameRegexp := regexp.MustCompile(`\[OriginalName\("([^"]+)"\)\]`)
	csEnums := parseEnums("dump.cs", csEnumRegexp, csNameRegexp)

	outsideMatches := make(map[string]string)
	containedMatches := make(map[string]string)
	potentialMatches := make(map[string][]string)
	for csEnumName, csValues := range csEnums {
		for namespace, protoEnums := range allProtoEnums {
			for protoEnumName, protoValues := range protoEnums {
				if len(csValues) != len(protoValues) {
					continue
				}

				isMatch := true

				for i, protoValue := range protoValues {
					if protoValue != csValues[i] {
						isMatch = false
					}
				}

				if !isMatch {
					continue
				}

				name := fmt.Sprintf("%s/%s", namespace, protoEnumName)
				if _, ok := containedMatches[name]; ok {
					potentialMatches[name] = append(potentialMatches[name], csEnumName)
					delete(containedMatches, name)
				} else if _, ok := potentialMatches[name]; !ok {
					if strings.Contains(csEnumName, ".") {
						containedMatches[name] = csEnumName
					} else {
						outsideMatches[name] = csEnumName
					}
				}
			}
		}
	}

	bindMessagesWithContainedEnum(containedMatches)
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

// func bindMessagesWithOutsideEnum(matches map[string]string) {
// 	messageNameRegexp := regexp.MustCompile(`message\s+(\w+)\s+{`)

// 	for match := range matches {
// 		found := 0
// 		lastMessage := ""
// 		filepath.WalkDir("proto", func(path string, d fs.DirEntry, err error) error {
// 			if err != nil {
// 				return err
// 			}

// 			if d.IsDir() || !strings.HasSuffix(path, ".proto") {
// 				return nil
// 			}

// 			file, err := os.Open(path)
// 			if err != nil {
// 				return err
// 			}
// 			defer file.Close()

// 			enumFieldRegexp := regexp.MustCompile(match)
// 			scanner := bufio.NewScanner(file)
// 			for scanner.Scan() {
// 				line := scanner.Text()
// 				switch true {
// 				case messageNameRegexp.Match([]byte(line)):
// 					if found == 0 {
// 						lastMessage = messageNameRegexp.FindStringSubmatch(line)[1]
// 					}
// 				case enumFieldRegexp.Match([]byte(line)):
// 					found += 1
// 				}
// 			}

// 			return nil
// 		})

// 		if found == 1 {
// 			fmt.Printf("%s", fmt.Sprintf("\t\"type.ankama.com/%s\": (&%s.%s{}).ProtoReflect().Type(),\n", strings.Split(csEnumName, ".")[0], namespace, messageName))
// 		}
// 	}
// }

func bindMessagesWithContainedEnum(matches map[string]string) {
	messageNameRegexp := regexp.MustCompile(`message\s+(\w+)\s+{`)
	protoEnumRegexp := regexp.MustCompile(`enum\s+(\w+)\s+{`)
	filepath.WalkDir("proto", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
			return nil
		}

		namespace := strings.Replace(filepath.Base(path), ".proto", "", 1)
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		messageStack := []string{}
		skip := 0
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			switch true {
			case messageNameRegexp.Match([]byte(line)):
				messageStack = append(messageStack, messageNameRegexp.FindStringSubmatch(line)[1])
			case protoEnumRegexp.Match([]byte(line)):
				skip += 1
				csEnumName, ok := matches[fmt.Sprintf("%s/%s", namespace, protoEnumRegexp.FindStringSubmatch(line)[1])]
				if !ok {
					break
				}
				csEnumNameCount := strings.Count(csEnumName, ".")
				if csEnumNameCount < 2 || len(messageStack) < csEnumNameCount/2 {
					break
				}
				messageName := messageStack[len(messageStack)-(csEnumNameCount/2)]
				fmt.Printf("%s", fmt.Sprintf("\t\"type.ankama.com/%s\": (&%s.%s{}).ProtoReflect().Type(),\n", strings.Split(csEnumName, ".")[0], namespace, messageName))
			case strings.Contains(line, "{"):
				skip += 1
			case strings.Contains(line, "}"):
				if skip > 0 {
					skip -= 1
				} else {
					messageStack = messageStack[:len(messageStack)-1]
				}
			}
		}

		return nil
	})
}
