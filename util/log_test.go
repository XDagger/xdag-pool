package util

import (
	"path/filepath"
	"testing"
)

func TestLog(t *testing.T) {
	dir := t.TempDir()

	InitLog(
		filepath.Join(dir, "info.log"),
		filepath.Join(dir, "error.log"),
		filepath.Join(dir, "share.log"),
		filepath.Join(dir, "block.log"),
		40,
	)
	Debug.Println("debug")
	Error.Println("error")
	ShareLog.Println("share")
	BlockLog.Println("block")
}

func TestDailyLogPath(t *testing.T) {
	got := dailyLogPath(filepath.Join("logs", "info.log"), "2026-05-25")
	want := filepath.Join("logs", "info-2026-05-25.log")
	if got != want {
		t.Fatalf("dailyLogPath() = %q, want %q", got, want)
	}
}
