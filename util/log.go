package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type PoolLogger struct {
	l        *log.Logger
	logLevel int
}

type dailyRotateWriter struct {
	basePath string
	mu       sync.Mutex
	file     *os.File
	day      string
}

const (
	DEBUG = 10
	INFO  = 20
	WARN  = 30
	ERROR = 40
	SHARE = 100
	BLOCK = 101
)

var (
	logSetLevel = 10

	Debug *PoolLogger
	Info  *PoolLogger
	Warn  *PoolLogger
	Error *PoolLogger

	ShareLog *PoolLogger
	BlockLog *PoolLogger
)

func InitLog(infoFile, errorFile, shareFile, blockFile string, setLevel int) {
	logSetLevel = setLevel
	log.Println("logSetLevel:", setLevel)

	log.Println("infoFile:", infoFile)
	log.Println("errorFile:", errorFile)
	log.Println("shareFile:", shareFile)
	log.Println("blockFile:", blockFile)
	infoFd, err := newDailyRotateWriter(infoFile)
	if err != nil {
		log.Fatalln("Failed to open info log file:", err)
	}

	errorFd, err := newDailyRotateWriter(errorFile)
	if err != nil {
		log.Fatalln("Failed to open error log file:", err)
	}

	shareFd, err := newDailyRotateWriter(shareFile)
	if err != nil {
		log.Fatalln("Failed to open share log file:", err)
	}

	blockFd, err := newDailyRotateWriter(blockFile)
	if err != nil {
		log.Fatalln("Failed to open block log file:", err)
	}

	Debug = &PoolLogger{log.New(io.MultiWriter(os.Stdout, infoFd), "[D] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), DEBUG}
	Info = &PoolLogger{log.New(io.MultiWriter(os.Stdout, infoFd), "[I] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), INFO}
	Warn = &PoolLogger{log.New(io.MultiWriter(os.Stderr, infoFd), "[W] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), WARN}
	Error = &PoolLogger{log.New(io.MultiWriter(os.Stderr, errorFd), "[E] ", log.Ldate|log.Lmicroseconds|log.Lshortfile), ERROR}

	ShareLog = &PoolLogger{log.New(io.MultiWriter(shareFd, os.Stdout), "[S]", log.Ldate|log.Lmicroseconds), SHARE}
	BlockLog = &PoolLogger{log.New(io.MultiWriter(blockFd, os.Stdout), "[B]", log.Ldate|log.Lmicroseconds), BLOCK}
}

func newDailyRotateWriter(basePath string) (*dailyRotateWriter, error) {
	w := &dailyRotateWriter{basePath: basePath}
	if err := w.rotate(time.Now().Format("2006-01-02")); err != nil {
		return nil, err
	}
	return w, nil
}

func (w *dailyRotateWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	day := time.Now().Format("2006-01-02")
	if day != w.day {
		if err := w.rotate(day); err != nil {
			return 0, err
		}
	}
	return w.file.Write(p)
}

func (w *dailyRotateWriter) rotate(day string) error {
	if err := os.MkdirAll(filepath.Dir(w.basePath), 0755); err != nil {
		return err
	}

	file, err := os.OpenFile(dailyLogPath(w.basePath, day), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	if w.file != nil {
		_ = w.file.Close()
	}
	w.file = file
	w.day = day
	return nil
}

func dailyLogPath(basePath, day string) string {
	ext := filepath.Ext(basePath)
	name := strings.TrimSuffix(basePath, ext)
	return name + "-" + day + ext
}

func (l *PoolLogger) Print(v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprint(v...))
	}
}

func (l *PoolLogger) Println(v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprintln(v...))
	}
}

func (l *PoolLogger) Printf(format string, v ...interface{}) {
	if logSetLevel <= l.logLevel {
		_ = l.l.Output(2, fmt.Sprintf(format, v...))
	}
}

func (l *PoolLogger) Fatal(v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprint(v...))
	os.Exit(1)
}

func (l *PoolLogger) Fatalln(v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprintln(v...))
	os.Exit(1)
}

func (l *PoolLogger) Fatalf(format string, v ...interface{}) {
	_ = l.l.Output(2, fmt.Sprintf(format, v...))
	os.Exit(1)
}

func (l *PoolLogger) Panic(v ...interface{}) {
	s := fmt.Sprint(v...)
	_ = l.l.Output(2, s)
	panic(s)
}

func (l *PoolLogger) Panicln(v ...interface{}) {
	s := fmt.Sprintln(v...)
	_ = l.l.Output(2, s)
	panic(s)
}

func (l *PoolLogger) Panicf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	_ = l.l.Output(2, s)
	panic(s)
}
