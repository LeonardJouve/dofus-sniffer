package main

import (
	"bufio"
	"dofus-sniffer/messages"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"google.golang.org/protobuf/reflect/protoreflect"
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

				isMatch := true

				for i, protoValue := range protoValues {
					if protoValue != csValues[i] {
						isMatch = false
					}
				}

				if !isMatch {
					continue
				}

				findUsage(fileName, protoEnumName, csEnumName)
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

func findUsage(fileName string, protoName string, csName string) {
	csParts := strings.Split(csName, ".")
	if len(csParts) < 2 {
		return
	}

	csParentName := strings.Join(csParts[:len(csParts)-2], ".")
	for _, message := range messages.Messages {
		descriptor := message.New().Interface().ProtoReflect().Descriptor()
		for protoFieldIndex := range descriptor.Fields().Len() {
			protoField := descriptor.Fields().Get(protoFieldIndex)
			var fieldName string
			switch protoField.Kind() {
			case protoreflect.EnumKind:
				fieldName = string(protoField.Enum().Name())
			case protoreflect.MessageKind:
				fieldName = string(protoField.Message().Name())
			}

			if protoName != fieldName {
				continue
			}

			if strings.Contains(csParentName, ".") {
				findUsage(fileName, csParentName, string(descriptor.Name()))
			} else {
				fmt.Printf("%s", fmt.Sprintf("\t\"type.ankama.com/%s\": (&%s.%s{}).ProtoReflect().Type(),\n", csParentName, fileName, string(descriptor.Name())))
			}
		}
	}
}

// func extractProtoMessages() {
// 	var sb strings.Builder
// 	filepath.WalkDir("proto", func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			return err
// 		}

// 		if d.IsDir() || !strings.HasSuffix(path, ".proto") {
// 			return nil
// 		}

// 		fileName := strings.Replace(filepath.Base(path), ".proto", "", 1)

// 		content, err := os.ReadFile(path)
// 		if err != nil {
// 			return err
// 		}

// 		lines := strings.Split(string(content), "\n")
// 		for _, line := range lines {
// 			if strings.Index(line, "message") == 0 {
// 				sb.WriteString(fmt.Sprintf("\t(&%s.%s{}).ProtoReflect().Type(),\n", fileName, regexp.MustCompile(`message (\w+)`).FindStringSubmatch(line)[1]))
// 			}
// 		}

// 		return nil
// 	})

// 	os.WriteFile("messages.txt", []byte(sb.String()), 0644)
// }
