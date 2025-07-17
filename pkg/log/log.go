package log

import (
	"fmt"
	"log"
	"os"
)

type Logger interface {
	Debug(msg string, kv ...any)
	Info(msg string, kv ...any)
	Warn(msg string, kv ...any)
	Error(msg string, kv ...any)
}

type std struct {
	lvl string
	l   *log.Logger
}

func New(level string) Logger {
	if level == "" {
		level = "info"
	}
	return &std{lvl: level, l: log.New(os.Stderr, "", log.LstdFlags|log.LUTC|log.Lmicroseconds)}
}

func (s *std) Debug(msg string, kv ...any) {
	if s.lvl == "debug" {
		s.log("DEBUG", msg, kv...)
	}
}
func (s *std) Info(msg string, kv ...any)  { s.log("INFO", msg, kv...) }
func (s *std) Warn(msg string, kv ...any)  { s.log("WARN", msg, kv...) }
func (s *std) Error(msg string, kv ...any) { s.log("ERROR", msg, kv...) }

func (s *std) log(level, msg string, kv ...any) {
	if len(kv)%2 != 0 {
		kv = append(kv, "<odd>")
	}
	pairs := ""
	for i := 0; i < len(kv); i += 2 {
		pairs += fmt.Sprintf(" %s=%v", kv[i], kv[i+1])
	}
	s.l.Printf("%s %s%s", level, msg, pairs)
}
