// Package log log util
// Author: Frank Lee
// Date: 2017-08-06 14:15:48
// Last Modified by: Frank Lee
// Last Modified time: 2017-09-08 10:44:35
package log

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (

	// Trace trace level
	Trace = iota

	// Info info level
	Info

	// Warn warn level
	Warn

	// Error error level
	Error

	// Fatal fatal level
	Fatal
)

const (

	// Day 2006-01-02
	Day = 1 << iota
	// Time 15:04:05
	Time
	// Lfile full path of file
	Lfile
	// Sfile only file name
	Sfile
	// Std Day|Time|Sfile
	Std
)

const (
	red = uint8(iota + 91)
	green
	yellow
	blue
	magenta
)

// Logger logger
type Logger struct {
	path    string // log file path
	level   int
	maxLine int // max lines
	maxSize int // max size in by
	cLine   int
	cSize   int
	mode    int
	creTime time.Time
	mutex   *sync.Mutex
	file    *os.File // log file
}

// LoggerOption logger options
type LoggerOption struct {
	Level    int
	MaxLine  int
	MaxSize  int
	Mode     int
	Compress bool
}

// NewLogger new logger
func NewLogger(path string, option *LoggerOption) *Logger {
	filePath, err := filepath.Abs(path)
	if err != nil {
		fmt.Println("init logger error:", err)
		return nil
	}
	fi, exists := fileExists(filePath)
	if exists { // file exists, rename it
		modDay := fi.ModTime()
		d := time.Now().Format("20060102")
		point, _ := time.ParseInLocation("20060102", d, time.Local)
		if modDay.Before(point) {
			mod := modDay.Format("20060102")
			rename(filePath, true, mod)
		} else {
			rename(filePath, false, "")
		}
	} else {
		i := strings.LastIndex(filePath, string(filepath.Separator))
		dir := filePath[0:i]
		_, e := fileExists(dir)
		if !e {
			os.MkdirAll(dir, 0666)
		}
	}
	file, err := os.Create(path)
	panicErr(err)
	logger := new(Logger)
	logger.file = file
	logger.path = filePath
	logger.cLine = 0
	logger.cSize = 0
	logger.creTime = time.Now()
	logger.maxLine = 0
	logger.maxSize = 0
	logger.level = Trace
	logger.mode = Std
	logger.mutex = new(sync.Mutex)
	if option != nil {
		if option.MaxLine > 0 {
			logger.maxLine = option.MaxLine
		}
		if option.MaxSize > 0 {
			logger.maxSize = option.MaxSize
		}
		if option.Level > 0 {
			logger.level = option.Level
		}
		if option.Mode > 0 {
			logger.mode = option.Mode
		}

		if option.Compress {
			go func() {
				t := time.Tick(time.Hour * 24)
				for _ = range t {
					logger.compressLog()
				}
			}()
		}
	}
	return logger
}

func (logger *Logger) compressLog() {
	var err error
	deleteFile := make([]string, 0)
	files := logger.getEarlyFile()
	if files != nil {
		for k, v := range files {
			deleteFile = append(deleteFile, v...)
			fileName := filepath.Base(logger.path)
			dir := filepath.Dir(logger.path)
			files := make([]*os.File, 0)
			for _, s := range v {
				f, err1 := os.Open(s)
				err = err1
				files = append(files, f)
			}
			namePrefix := fileName
			if strings.Contains(fileName, ".") {
				i := strings.Index(fileName, ".")
				namePrefix = fileName[0:i]
			}
			zipName := dir + string(filepath.Separator) + namePrefix + k + ".zip"
			d, err2 := os.Create(zipName)
			err = err2
			defer d.Close()
			w := zip.NewWriter(d)
			defer w.Close()
			for _, file := range files {
				err = compress(file, "", w)
			}
		}
		if err == nil {
			deleteLog(deleteFile)
		}
	}
}

func compress(file *os.File, prefix string, zw *zip.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, zw)
			if err != nil {
				return err
			}
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		if prefix != "" {
			header.Name = prefix + "/" + header.Name
		}
		if err != nil {
			return err
		}
		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, file)
		file.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (logger *Logger) getEarlyFile() map[string][]string {
	result := make(map[string][]string)
	todayStr := time.Now().AddDate(0, 0, -2).Format("20060102")
	pointTime, _ := time.ParseInLocation("20060102", todayStr, time.Local)
	point := pointTime.Unix()
	dir := filepath.Dir(logger.path)
	fileName := filepath.Base(logger.path)
	filePrefix := fileName
	if strings.Contains(fileName, ".") {
		i := strings.Index(fileName, ".")
		filePrefix = fileName[0:i]
	}
	fl, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil
	}
	logRegex := regexp.MustCompile(`_\\d{8}_`)
	for _, f := range fl {
		if strings.HasPrefix(f.Name(), filePrefix) && logRegex.MatchString(f.Name()) {
			loc := logRegex.FindIndex([]byte(f.Name()))
			day := f.Name()[loc[0]+1 : loc[1]-1]
			fileTime, _ := time.ParseInLocation("20060102", day, time.Local)
			t := fileTime.Unix()
			if t <= point {
				if v, contains := result[day]; contains {
					v = append(v, dir+string(filepath.Separator)+f.Name())
				} else {
					s := make([]string, 0)
					s = append(s, dir+string(filepath.Separator)+f.Name())
					result[day] = s
				}
			}
		}
	}
	return result
}

func deleteLog(file []string) {
	for _, s := range file {
		os.Remove(s)
	}
}

func fileExists(path string) (os.FileInfo, bool) {
	fi, err := os.Stat(path)
	if err == nil {
		return fi, true
	} else if os.IsNotExist(err) {
		return nil, false
	}
	return nil, false
}

func rename(path string, isRotate bool, modDay string) {
	dir := filepath.Dir(path)   // /data/log/xxx.log
	name := filepath.Base(path) // xxx.log
	namePrefix := name          // xxx
	suffix := ""                // .log
	containDot := false
	if strings.Contains(name, ".") {
		containDot = true
		i := strings.Index(name, ".")
		namePrefix = name[0:i]
		suffix = name[i:]
	}
	fl, err := ioutil.ReadDir(dir)
	if err != nil && os.IsNotExist(err) { // dir not exists
		err = os.MkdirAll(dir, 666)
		panicErr(err)
	}
	names := make(map[string]string) // names: xxx_20170908_1.log   xxx_20170908_2.log   xxx.log
	for _, f := range fl {
		names[f.Name()] = ""
	}

	var day string
	if isRotate {
		if modDay != "" {
			day = modDay
		} else {
			day = time.Now().AddDate(0, 0, -1).Format("20060102")
		}
	} else {
		day = time.Now().Format("20060102")
	}
	r := 1
	for i := 1; i <= len(fl); i++ {
		if _, contains := names[namePrefix+"_"+day+"_"+strconv.Itoa(i)+suffix]; contains {
			if i >= r {
				r = i + 1
			}
		}
	}
	// rename old file with suffix
	var newName string
	if containDot {
		newName = dir + string(filepath.Separator) + namePrefix + "_" + day + "_" + strconv.Itoa(r) + suffix
	} else {
		newName = dir + string(filepath.Separator) + name + "_" + day + "_" + strconv.Itoa(r)
	}
	os.Rename(path, newName)
}

// Traceln trace level console logs
func Traceln(format string, a ...interface{}) {
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", green, "[TRACE]"), nil)
	s := fmt.Sprintf(header+format, a...)
	fmt.Println(s)
}

// Infoln info level console logs
func Infoln(format string, a ...interface{}) {
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", blue, "[INFO]"), nil)
	s := fmt.Sprintf(header+format, a...)
	fmt.Println(s)
}

// Warnln warn level console logs
func Warnln(format string, a ...interface{}) {
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", yellow, "[WARN]"), nil)
	s := fmt.Sprintf(header+format, a...)
	fmt.Println(s)
}

// Errorln error level console logs
func Errorln(format string, a ...interface{}) {
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", magenta, "[ERROR]"), nil)
	s := fmt.Sprintf(header+format, a...)
	fmt.Println(s)
}

// Fatalln fatal level console logs
func Fatalln(format string, a ...interface{}) {
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", red, "[FATAL]"), nil)
	s := fmt.Sprintf(header+format, a...)
	fmt.Println(s)
}

// Trace trace level logs
func (logger *Logger) Trace(format string, a ...interface{}) {
	logger.check()
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", green, "[TRACE]"), logger)
	s := fmt.Sprintf(format, a...)
	logger.output(header + s)
	logger.cLine++
	logger.cSize += len(header + s)
}

// Info info level logs
func (logger *Logger) Info(format string, a ...interface{}) {
	logger.check()
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", blue, "[INFO]"), logger)
	s := fmt.Sprintf(format, a...)
	logger.output(header + s)
	logger.cLine++
	logger.cSize += len(header + s)
}

// Warn warn level logs
func (logger *Logger) Warn(format string, a ...interface{}) {
	logger.check()
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", yellow, "[WARN]"), logger)
	s := fmt.Sprintf(format, a...)
	logger.output(header + s)
	logger.cLine++
	logger.cSize += len(header + s)
}

// Error error level logs
func (logger *Logger) Error(format string, a ...interface{}) {
	logger.check()
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", magenta, "[ERROR]"), logger)
	s := fmt.Sprintf(format, a...)
	logger.output(header + s)
	logger.cLine++
	logger.cSize += len(header + s)
}

// Fatal fatal level logs
func (logger *Logger) Fatal(format string, a ...interface{}) {
	logger.check()
	header := prepareHeader(fmt.Sprintf("\x1b[%dm%s\x1b[0m", red, "[FATAL]"), logger)
	s := fmt.Sprintf(format, a...)
	logger.output(header + s)
	logger.cLine++
	logger.cSize += len(header + s)
}

func (logger *Logger) check() {
	_, exists := fileExists(logger.path)
	if !exists {
		logger.mutex.Lock()
		defer logger.mutex.Unlock()
		_, ex := fileExists(logger.path)
		if !ex {
			i := strings.LastIndex(logger.path, string(filepath.Separator))
			dir := logger.path[0:i]
			_, e := fileExists(dir)
			if !e {
				os.MkdirAll(dir, 0666)
			}
			file, _ := os.Create(logger.path)
			logger.file = file
			logger.cLine = 0
			logger.cSize = 0
			logger.creTime = time.Now()
		}

	} else {
		if logger.needSplit() || logger.needRotate() {
			logger.change()
		}
	}
}

func prepareHeader(prefix string, logger *Logger) string {
	s := prefix + " "
	t := time.Now()
	if logger == nil || logger.mode == Std {
		s += t.Format("2006-01-02 15:04:05.999") + " "
		_, file, line, ok := runtime.Caller(2)
		if ok {
			f := filepath.Base(file)
			s += f + ":" + strconv.Itoa(line) + ":"
		}
	} else {
		if logger.mode&Day == Day {
			s += t.Format("2006-01-02") + " "
		}
		if logger.mode&Time == Time {
			s += t.Format("15:04:05") + " "
		}
		if logger.mode&Lfile == Lfile || logger.mode&Sfile == Sfile {
			_, file, line, ok := runtime.Caller(2)
			f := "unkonow file"
			l := 0
			if ok {
				if logger.mode&Lfile == Lfile {
					f = file
				} else {
					f = filepath.Base(file)
				}
				l = line
				s += f + ":" + strconv.Itoa(l) + ":"
			}
		}
	}
	return s

}

func (logger *Logger) output(s string) {
	s += "\n"
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	logger.file.Write([]byte(s))
}

func (logger *Logger) validLevel(level int) bool {
	return level >= logger.level
}

func (logger *Logger) needSplit() bool {
	if logger.maxLine <= 0 && logger.maxSize <= 0 {
		return false
	}
	return logger.cSize >= logger.maxSize || logger.cLine >= logger.maxLine
}

func (logger *Logger) change() {
	logger.mutex.Lock()
	defer logger.mutex.Unlock()
	if logger.needSplit() || logger.needRotate() {
		isRotate := logger.needRotate()
		logger.file.Close() //  important  save current file
		rename(logger.path, isRotate, "")
		f, _ := os.Create(logger.path)
		logger.file = f
		logger.cSize = 0            // important
		logger.cLine = 0            // important
		logger.creTime = time.Now() // important
	}
}

func (logger *Logger) needRotate() bool {
	nextDay := logger.creTime.AddDate(0, 0, 1).Format("20060102")
	splitPoint, _ := time.ParseInLocation("20060102 15:04:05", nextDay+" 00:00:00", time.Local)
	return time.Now().After(splitPoint)
}

func panicErr(err error) {
	if err != nil {
		panic(err)
	}
}
