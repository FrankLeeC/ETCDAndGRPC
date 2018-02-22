// Package config config util
// Author: Frank Lee
// Date: 2017-08-07 10:43:26
// Last Modified by:   Frank Lee
// Last Modified time: 2017-08-07 10:43:26
package config

import (
	"bufio"
	"container/list"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var (
	runMode    string
	runModeSet bool
	findNode   bool
	properties = make(map[string]string)
	global     = make(map[string]string)
	keyStack   = list.New()
	stackDepth = 0

	commentSegmentRegex = regexp.MustCompile(".*\\s+#.*")
	commentStartRegex   = regexp.MustCompile("\\s+#")

	nodeRegex = regexp.MustCompile("^\\[+\\w*\\]+$")
)

// func main() {
//     fmt.Println(properties)
//     fmt.Println("===================")
//     fmt.Println(global)
//     fmt.Println("===================")
//     fmt.Println(GetString("mysql>default>port"))
//     fmt.Println(GetString("a"))
// }

func init() {
	runModeSet = false
	findNode = false
	parseConfigFile()
}

// GetString get string value by key
func GetString(key string) string {
	s, contains := getFromMode(key)
	if !contains {
		s, contains = getFromGlobal(key)
	}
	if contains {
		return s
	}
	fmt.Println("value of key", key, "is not set!")
	return ""
}

// GetInt get int value by key. If key is not set, return 0. strconv.Atoi error, return 0.
func GetInt(key string) int {
	s, contains := getFromMode(key)
	if !contains {
		s, contains = getFromGlobal(key)
	}
	if contains {
		i, err := strconv.Atoi(s)
		if err != nil {
			fmt.Println("GetInt error:", err)
			return 0
		}
		return i
	}
	fmt.Println("value of", key, "is not set!")
	return 0
}

// GetInt64 get int64 value by key. If key is not set, return 0. strconv.ParseInt error, return 0.
func GetInt64(key string) int64 {
	s, contains := getFromMode(key)
	if !contains {
		s, contains = getFromGlobal(key)
	}
	if contains {
		i, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			fmt.Println("GetInt64 error:", err)
			return 0
		}
		return i
	}
	fmt.Println("values of", key, "is not set!")
	return 0
}

// GetBool get bool value by key. If key is not set, return false. strconv.ParseBool error, return false.
func GetBool(key string) bool {
	s, contains := getFromMode(key)
	if !contains {
		s, contains = getFromGlobal(key)
	}
	if contains {
		b, err := strconv.ParseBool(s)
		if err != nil {
			fmt.Println("GetBool error:", err)
			return false
		}
		return b
	}
	fmt.Println("value of", key, "is not set!")
	return false
}

// Contains contains key or not
func Contains(key string) bool {
	_, contains := getFromMode(key)
	if !contains {
		_, contains = getFromGlobal(key)
	}
	return contains
}

func getFromMode(key string) (string, bool) {
	if v, contains := properties[runMode+">"+key]; contains {
		return v, true
	}
	return "", false
}

func getFromGlobal(key string) (string, bool) {
	if v, contains := global[key]; contains {
		return v, true
	}
	return "", false
}

// GetRunMode get current run mode
func GetRunMode() string {
	return runMode
}

func parseConfigFile() {
	file, err := os.Open("./app.conf")
	if err != nil {
		fmt.Println("config file not found!")
		os.Exit(2)
	}
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if isValidLine(line) {
			line = processIfContainsComment(line)
			if !findNode { // before first node
				if isEntryLine(line) {
					_key, value := getKeyValue(line)
					if _key == "RunMode" {
						runMode = value
						runModeSet = true
					} else {
						global[_key] = value
					}
				} else if isNode(line) {
					findNode = true
					parseOneLine(line)
				}
			} else {
				if !runModeSet {
					fmt.Println("RunMode must set before any node!")
					os.Exit(2)
				}
				parseOneLine(line)
			}
		}

	}
}

func processIfContainsComment(line string) string {
	if containsComment(line) {
		i := commentStartIndex(line)
		if i > 0 { // TODO if # follows =, there mustn't be space between = and #
			line = line[0:i]
		}
	}
	return line
}

func parseOneLine(line string) {
	nodeDepth := getNodeDepth(line)
	if nodeDepth > 0 { // this line is a node
		if nodeDepth <= stackDepth {
			refreshStack(nodeDepth, getNodeContent(line, nodeDepth))
		} else {
			pushStack(getNodeContent(line, nodeDepth))
		}
	} else { // this line is an entry
		_key, value := getKeyValue(line)
		baseKey := getBaseKey()
		properties[baseKey+_key] = value
	}
}

func pushStack(content string) {
	keyStack.PushBack(content)
	stackDepth = keyStack.Len()
}

func refreshStack(depth int, content string) {
	if keyStack.Len() == 0 {
		keyStack.PushFront(content)
	} else {
		var removeKey *list.Element
		for i := 0; i < depth; i++ {
			if removeKey == nil {
				removeKey = keyStack.Front()
			} else {
				removeKey = removeKey.Next()
			}
		}
		keyStack.InsertBefore(content, removeKey)
		for keyStack.Len() > depth {
			keyStack.Remove(keyStack.Back())
		}
	}

	stackDepth = keyStack.Len()
}

func getBaseKey() string {
	key := ""
	for e := keyStack.Front(); e != nil; e = e.Next() {
		key += e.Value.(string) + ">"
	}
	return key
}

func getKeyValue(line string) (string, string) {
	entries := strings.Split(line, "=")
	return strings.TrimSpace(entries[0]), strings.TrimSpace(entries[1])
}

func getNodeContent(line string, depth int) string {
	return line[depth : len(line)-depth]
}

// not comment line and contains 'key = value'
func isValidLine(line string) bool {
	return !isCommentLine(line) && (isEntryLine(line) || isNode(line))
}

// contains 'key = value'
func isEntryLine(line string) bool {
	return strings.Contains(line, "=") && strings.Index(line, "=") > 0 && strings.Index(line, "=") < len(line)-1
}

// line starting with # is comment
func isCommentLine(line string) bool {
	return strings.HasPrefix(line, "#")
}

// comment after properties
func containsComment(line string) bool {
	return commentSegmentRegex.MatchString(line)
}

// get index of comment segment
func commentStartIndex(line string) int {
	return commentStartRegex.FindStringIndex(line)[0]
}

// must start with [, end with ] and contains content. if line == '[]', it's not a node
func isNode(line string) bool {
	return nodeRegex.MatchString(line) && len(line) > 2
}

// if line is a node, return depth, return 0 otherwise
func getNodeDepth(line string) int {
	depth := 0
	for {
		if isNode(line) {
			depth++
			line = line[1 : len(line)-1]
		} else {
			break
		}
	}
	return depth
}
