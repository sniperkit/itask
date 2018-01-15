package main

import (
	"fmt"

	// "github.com/Kesci/lazyskiplist"
	// "github.com/hypersolid/duckmap"

	"github.com/sniperkit/xtask/pkg"
	"github.com/sniperkit/xtask/util/runtime"
	"github.com/sniperkit/xutil/plugin/format/convert/json2csv"

	tablib "github.com/sniperkit/xutil/plugin/format/convert/tabular"
	jsoniter "github.com/sniperkit/xutil/plugin/format/json"
	cmap "github.com/sniperkit/xutil/plugin/map/multi"
)

var (
	json                                    = jsoniter.ConfigCompatibleWithStandardLibrary
	writers  map[string]*json2csv.CSVWriter = make(map[string]*json2csv.CSVWriter, 0)
	sheets   map[string][]interface{}       = make(map[string][]interface{}, 0)
	datasets map[string]*tablib.Dataset     = make(map[string]*tablib.Dataset, 0) // := NewDataset([]string{"firstName", "lastName"})
	cds                                     = cmap.NewConcurrentMultiMap()
)

var headerStyleTable = map[string]json2csv.KeyStyle{
	"jsonpointer": json2csv.JSONPointerStyle,
	"slash":       json2csv.SlashStyle,
	"dot":         json2csv.DotNotationStyle,
	"dot-bracket": json2csv.DotBracketStyle,
}

var encoderCsv = func(result xtask.TaskResult) {
	log.Println("exportCSV")
}

var processor = func(result xtask.TaskResult) {
	if result.Error != nil {
		log.Println("error: ", result.Error.Error(), "debug=", runtime.WhereAmI())
	}
	log.Println("response:", result.Result)
}

func exportCSV(eg string, input interface{}) xtask.Tsk {
	if writers[eg] == nil {
		writers[eg] = newWriterJSON2CSV(eg)
	}

	// t1.Interface().(time.Time)
	// cds.Append("repos", repo)
	return func() *xtask.TaskResult {
		return &xtask.TaskResult{}
	}
}

func flushAllWriters() {
	for k, w := range writers {
		if w != nil {
			data, _ := cds.Get(k)
			results, err := json2csv.JSON2CSV(data)
			if err != nil {
				log.Fatalln("JSON2CSV error:", err)
			}
			w.WriteCSV(results)
			w.Flush()
			if err := w.Error(); err != nil {
				log.Fatalln("Error: ", err)
			}
		}
	}
	cds.Clear()
}

// add prefixPath, headerStyleTable, transpose
func newWriterJSON2CSV(basename string) *json2csv.CSVWriter {
	outputFile := fmt.Sprintf("./shared/data/export/%s.csv", basename)
	log.Debugln("instanciate new concurrent writer to output file=", outputFile)
	w, err := json2csv.NewCSVWriterToFile(outputFile)
	if err != nil {
		log.Fatalf("Could not open `%s` for writing", outputFile)
	}
	w.HeaderStyle = headerStyleTable["dot"]
	w.NoHeaders(true)
	return w
}
