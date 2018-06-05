package main

import (
	"fmt"
	"io"
	"os"

	// "github.com/fanliao/go-concurrentMap"
	// "https://github.com/chonla/dbz/blob/master/db/sqlite.go"
	// "github.com/cnf/structhash"
	// "github.com/siddontang/go-mysql-elasticsearch"
	// "github.com/mandolyte/csv-utils"

	"github.com/pquerna/ffjson/ffjson"
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
	// jsonwriter map[string]*bufio.Writer       = make(map[string]*bufio.Writer, 0)
	jsonfile map[string]*os.File = make(map[string]*os.File, 0)
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

func convertInterface2(input []map[string]interface{}) []interface{} {
	results := make([]interface{}, len(input))
	for _, result := range input {
		// resultSlice := result.(map[string]interface{})
		// pp.Println("resultSlice=", resultSlice)
		results = append(results, result)
	}
	return results
}

func convertInterface(input map[string]interface{}) []interface{} {
	results := make([]interface{}, len(input))
	for _, result := range input {
		resultSlice := result.(interface{})
		results = append(results, resultSlice)
	}
	return results
}

func getHeaders(filterMap map[string]string) []string {
	var hdrs []string
	for k, _ := range filterMap {
		hdrs = append(hdrs, k)
	}
	return hdrs
}

var processor = func(result xtask.TaskResult) {
	if result.Error != nil {
		log.Println("error: ", result.Error.Error(), "debug=", runtime.WhereAmI())
	}
	log.Println("response:", result.Result)
}

func initWriters(truncate bool, groups ...string) {
	for _, group := range groups {
		if writers[group] == nil {
			writers[group] = newWriterJSON2CSV(truncate, group)
		}
	}
}

func roundU(val float64) int {
	if val > 0 {
		return int(val + 1.0)
	}
	return int(val)
}

func exportCSV(eg string, input interface{}) xtask.Tsk {
	if writers[eg] == nil {
		writers[eg] = newWriterJSON2CSV(false, eg)
	}

	return func() *xtask.TaskResult {
		return &xtask.TaskResult{}
	}
}

func Encode(item interface{}, out io.Writer) {
	buf, err := ffjson.Marshal(&item)
	if err != nil {
		log.Fatalln("Encode error:", err)
	}
	// Write the buffer
	_, _ = out.Write(buf)
	// We are now no longer need the buffer so we pool it.
	ffjson.Pool(buf)
}

func EncodeItems(items []interface{}, out io.Writer) {
	// We create an encoder.
	enc := ffjson.NewEncoder(out)
	for _, item := range items {
		if item == nil {
			continue
		}
		// Encode into the buffer
		err := enc.Encode(&item)
		enc.SetEscapeHTML(false)
		if err != nil {
			log.Fatalln("EncodeItems error:", err)
		}
		// If err is nil, the content is written to out, so we can write to it as well.
		//if i != len(items)-1 {
		//	_, _ = out.Write([]byte{""})
		//}
	}
}

// https://github.com/fanliao/go-concurrentMap#safely-use-composition-operation-to-update-the-value-from-multiple-threads
/*---- group string by first char using ConcurrentMap ----*/
//sliceAdd function returns a function that appends v into slice
var sliceAdd = func(v interface{}) func(interface{}) interface{} {
	return func(oldVal interface{}) (newVal interface{}) {
		if oldVal == nil {
			vs := make([]string, 0, 1)
			return append(vs, v.(string))
		} else {
			return append(oldVal.([]string), v.(string))
		}
	}
}

var (
	ExportPrefixPath = CacheDrive + "./shared/data/export"
)

func flushWriters() {
	for k, w := range writers {
		if w != nil {
			data, _ := cds.Get(k)

			if len(data) <= 0 {
				continue
			}

			if jsonfile[k] == nil {
				jsonOutpuFile := fmt.Sprintf(ExportPrefixPath+"/json/%s.json", k)
				var err error
				jsonfile[k], err = os.OpenFile(jsonOutpuFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Fatalln(" os.Create(jsonOutpuFile) error:", err)
				}
			}
			EncodeItems(data, jsonfile[k])

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
	// jsonfile[k].Close()
	cds.Clear()
}

// add prefixPath, headerStyleTable, transpose
func newWriterJSON2CSV(truncate bool, basename string) *json2csv.CSVWriter {
	outputFile := fmt.Sprintf(ExportPrefixPath+"/csv/%s.csv", basename)
	log.Debugln("instanciate new concurrent writer to output file=", outputFile)
	w, err := json2csv.NewCSVWriterToFile(outputFile)
	if err != nil {
		log.Fatalf("Could not open `%s` for writing", outputFile)
	}
	w.HeaderStyle = headerStyleTable["dot"]
	w.NoHeaders(true)
	return w
}
