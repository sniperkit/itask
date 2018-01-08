package main

import (
	lal "github.com/sniperkit/xlogger/pkg"        // logger abstraction layer
	lcf "github.com/sniperkit/xlogger/pkg/config" // logger configuration (engine, output, ...)
	lfi "github.com/sniperkit/xlogger/pkg/fields" // logger initial fields
)

var (
	log       lal.Logger
	logConf   *lcf.Config
	logFields *lfi.Fields
)

func newLogger() {}

func setLoggerConfig() {}

func setLoggerInitialFields() {}
