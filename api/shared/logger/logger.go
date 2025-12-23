package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	output := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func Debug(msg string) {
	// log.Debug().Msg(msg)
}

func Debugf(format string, args ...any) {

	// log.Debug().Msg(fmt.Sprintf(format+"\n", args...))
}

func Info(msg string) {
	// log.Info().Msg(msg)
}

func Infof(format string, args ...any) {
	// log.Info().Msg(fmt.Sprintf(format+"\n", args...))
}

func Warning(msg string) {
	// log.Warn().Msg(msg)
}

func Warningf(format string, args ...any) {
	// log.Warn().Msg(fmt.Sprintf(format+"\n", args...))
}

func Critical(msg string) {
	// log.Error().Msg(msg)
}

func Criticalf(format string, args ...any) {
	// log.Error().Msg(fmt.Sprintf(format+"\n", args...))
}

func Fatal(msg string) {
	// log.Fatal().Msg(msg)
	// os.Exit(1)
}

func Fatalf(format string, args ...any) {
	// log.Fatal().Msg(fmt.Sprintf(format+"\n", args...))
	// os.Exit(1)
}
