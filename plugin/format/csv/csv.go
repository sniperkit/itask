package gocsv

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	// sync "github.com/sniperkit/xutil/plugin/concurrency/sync/debug"
)

type App struct {
	File  string
	Keys  []string
	Items []map[string]string
}

func New(filepath string) (*App, error) {
	if filepath == "" {
		return nil, errors.New("please provide a  filepath")
	}

	app := &App{
		File:  filepath,
		Keys:  make([]string, 0),
		Items: make([]map[string]string, 0),
	}

	return app, nil
}

func (a *App) parseCSVData(parseItems chan []string) {
	csvfile, err := os.Open(a.File)

	if err != nil {
		fmt.Println("error while opening file: ", err)
		return
	}
	defer csvfile.Close()

	fmt.Println("opening file: ", a.File)

	reader := csv.NewReader(csvfile)
	reader.FieldsPerRecord = -1
	rowCount := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if rowCount == 0 {
			a.Keys = record
		}
		rowCount++
		// Stop at EOF.
		go func(pi []string) {
			parseItems <- pi
		}(record)
	}
}

func (a *App) moldObject(parseItems <-chan []string, wg *sync.WaitGroup, lineItems chan<- map[string]string) {
	defer wg.Done()

	for pi := range parseItems {
		l := make(map[string]string, len(a.Keys))
		for i, key := range a.Keys {
			fmt.Println(l[key])
			fmt.Println(pi[i])
			fmt.Println(i)
			l[key] = pi[i]
		}
		lineItems <- l
	}
}

func (a *App) Run() {
	parseItems := make(chan []string)
	lineItems := make(chan map[string]string)

	go a.parseCSVData(parseItems)

	wg := new(sync.WaitGroup)
	for i := 0; i <= 3; i++ {
		wg.Add(1)
		go a.moldObject(parseItems, wg, lineItems)
	}

	go func() {
		wg.Wait()
		close(lineItems)
	}()

	for li := range lineItems {
		a.Items = append(a.Items, li)
	}
}
