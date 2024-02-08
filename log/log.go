package log

import (
	"log/slog"
	"os"
)

// New creates a new logger with the given level and output.
func New(level Level, output Output) *slog.Logger {
	var h slog.Handler
	switch output {
	case OutputJson:
		h = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level.ToSlog()})
	case OutputText:
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level.ToSlog()})
	default:
		h = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level.ToSlog()})
	}
	return slog.New(h)
}

// Level represents the log level.
type Level string

// ToSlog converts the level to slog.Level.
func (l Level) ToSlog() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

const (
	// LogDebug is the debug log level.
	LevelDebug Level = "debug"
	// LogInfo is the info log level.
	LevelInfo Level = "info"
	// LogWarn is the warn log level.
	LevelWarn Level = "warn"
	// LogError is the error log level.
	LevelError Level = "error"
)

// ToLevel converts the level to Level.
func ToLevel(level string) Level {
	switch level {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warn":
		return LevelWarn
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// GetLevel returns the log level from the environment variable.
func GetLevel() Level {
	level := os.Getenv("LOG_LEVEL")
	return ToLevel(level)
}

// Output represents the log output.
type Output string

const (
	// OutputJson is the JSON log output.
	OutputJson Output = "json"
	// OutputText is the text log output.
	OutputText Output = "text"
)

// ToOutput converts the output to Output.
func ToOutput(output string) Output {
	switch output {
	case "json":
		return OutputJson
	case "text":
		return OutputText
	default:
		return OutputText
	}
}

// GetOutput returns the log output from the environment variable.
func GetOutput() Output {
	output := os.Getenv("LOG_OUTPUT")
	return ToOutput(output)
}
