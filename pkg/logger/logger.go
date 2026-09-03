package logger

import (
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
)

func New(level, format string) (*logrus.Logger, error) {
	parsed, err := logrus.ParseLevel(level)
	if err != nil {
		return nil, fmt.Errorf("invalid log level %q: %w", level, err)
	}
	log := logrus.New()
	log.SetOutput(os.Stdout)
	log.SetLevel(parsed)
	switch format {
	case "json":
		log.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		log.SetFormatter(&logrus.TextFormatter{DisableColors: true, FullTimestamp: true})
	case "colored_text":
		log.SetFormatter(&logrus.TextFormatter{ForceColors: true, FullTimestamp: true})
	default:
		return nil, fmt.Errorf("invalid log format %q", format)
	}
	return log, nil
}
