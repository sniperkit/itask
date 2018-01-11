package main

import (
	// "encoding/json"
	"flag"
	"fmt"
	"math"
	// "sort"
	"strings"

	// "github.com/yukithm/json2csv"
	// "github.com/jehiah/json2csv"
	"github.com/k0kubun/pp"
	"github.com/tsak/concurrent-csv-writer"
	"github.com/yukithm/json2csv"
	// "github.com/agrison/go-tablib"

	// "github.com/yukithm/json2csv"
	// "github.com/yukithm/json2csv/jsonpointer"

	"github.com/sniperkit/xtask/pkg"
	// "github.com/sniperkit/xtask/plugin/format/json/json2map"
	"github.com/sniperkit/xtask/util/runtime"
)

var headerStyleTable = map[string]json2csv.KeyStyle{
	"jsonpointer": json2csv.JSONPointerStyle,
	"slash":       json2csv.SlashStyle,
	"dot":         json2csv.DotNotationStyle,
	"dot-bracket": json2csv.DotBracketStyle,
}

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

func get_value(data map[string]interface{}, keyparts []string) string {
	if len(keyparts) > 1 {
		subdata, _ := data[keyparts[0]].(map[string]interface{})
		return get_value(subdata, keyparts[1:])
	} else if v, ok := data[keyparts[0]]; ok {
		switch v.(type) {
		case nil:
			return ""
		case float64:
			f, _ := v.(float64)
			if math.Mod(f, 1.0) == 0.0 {
				return fmt.Sprintf("%d", int(f))
			} else {
				return fmt.Sprintf("%f", f)
			}
		default:
			return fmt.Sprintf("%+v", v)
		}
	}
	return ""
}

type LineReader interface {
	ReadBytes(delim byte) (line []byte, err error)
}

type StringArray []string

func (a *StringArray) Set(s string) error {
	for _, ss := range strings.Split(s, ",") {
		*a = append(*a, ss)
	}
	return nil
}

func (a *StringArray) String() string {
	return fmt.Sprint(*a)
}

var paramTest = "name,owner.login,created_at"

var writers map[string]*ccsv.CsvWriter = make(map[string]*ccsv.CsvWriter, 0)

var exportInterface = func(ti xtask.TaskInfo) {
	writer := "test"
	outputFile := fmt.Sprintf("./shared/data/csv/%s.csv", writer)
	if writers[writer] == nil {
		w, err := ccsv.NewCsvWriter(outputFile)
		if err != nil {
			panic("Could not open output file for writing")
		}
		writers[writer] = w
	}
	pp.Println("Result: ", ti.Result.Result)
	/*
		pp.Println("Result: ", ti.Result.Result)

		results, err := json2csv.JSON2CSV(ti.Result.Result)
		if err != nil {
			log.Fatal(err)
		}
		if len(results) == 0 {
			return
		}
		// pp.Println("results: ", results)
	*/
}

var exportMap = func(ti xtask.TaskInfo) {
	writer := "test"
	outputFile := fmt.Sprintf("./shared/data/csv/%s.csv", writer)
	if writers[writer] == nil {
		w, err := ccsv.NewCsvWriter(outputFile)
		if err != nil {
			panic("Could not open output file for writing")
		}
		writers[writer] = w
	}
	results, err := json2csv.JSON2CSV(ti.Result.Result)
	if err != nil {
		log.Fatal(err)
	}
	if len(results) == 0 {
		return
	}
	pp.Println("results: ", results)
}

func newWriter(writer string) {
	outputFile := fmt.Sprintf("./shared/data/csv/%s.csv", writer)
	if writers == nil {
		writers = make(map[string]*ccsv.CsvWriter, 0)
	}
	if writers[writer] == nil {
		w, err := ccsv.NewCsvWriter(outputFile)
		if err != nil {
			panic("Could not open `sample.csv` for writing")
		}
		writers[writer] = w
	}
}

var (
	inputFile   = flag.String("i", "", "/path/to/input.json (optional; default is stdin)")
	outputFile  = flag.String("o", "", "/path/to/output.json (optional; default is stdout)")
	outputDelim = flag.String("d", ",", "delimiter used for output values")
	printHeader = flag.Bool("p", true, "prints header to output")
)

var encoderCsv = func(result xtask.TaskResult) { log.Println("exportCSV") }
