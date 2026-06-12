package logging

import (
    "os"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"
)

func Init() {
    zerolog.TimeFieldFormat = time.RFC3339Nano

    log.Logger = zerolog.New(os.Stdout).
        With().
        Timestamp().
        Caller().
        Logger()
}
