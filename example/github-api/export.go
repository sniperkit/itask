package main

import (
	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/util/runtime"
)

var processor = func(result xtask.TaskResult) {
	if result.Error != nil {
		log.Println("error: ", result.Error.Error(), "debug=", runtime.WhereAmI())
	}
	log.Println("response:", result.Result)
	/*
		for _, val := range result.Result {
			log.Println("response:", val.Interface())
		}
	*/
}

var encoderCsv = func(result xtask.TaskResult) { log.Println("exportCSV") }
