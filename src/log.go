package src

import (
	"github.com/sirupsen/logrus"
)

var Logger = logrus.New()

func EnableDebug() {
	Logger.SetLevel(logrus.DebugLevel)
}
