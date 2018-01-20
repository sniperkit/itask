package main

import (
	"encoding/csv"
	"fmt"
	"strings"
)

const Buffer = 20000

/*
	Refs:
	- https://github.com/Gujarats/csv-reader/blob/master/app.go
*/

type CsvLine struct {
	Header []string
	Line   []string
}

// streamCsv
//  Streams a CSV Reader into a returned channel.  Each CSV row is streamed along with the header.
//  "true" is sent to the `done` channel when the file is finished.
//
// Args
//  csv    - The csv.Reader that will be read from.
//  buffer - The "lines" buffer factor.  Send "0" for an unbuffered channel.
func streamCsv(csv *csv.Reader, buffer int) (lines chan *CsvLine) {
	lines = make(chan *CsvLine, buffer)

	go func() {
		// get Header
		header, err := csv.Read()
		if err != nil {
			close(lines)
			return
		}

		i := 0

		for {
			line, err := csv.Read()

			if len(line) > 0 {
				i++
				lines <- &CsvLine{Header: header, Line: line}
			}

			if err != nil {
				fmt.Printf("Sent %d lines\n", i)
				close(lines)
				return
			}
		}
	}()

	return
}

/*
func convertLine(csvLines chan *CsvLine) (lines chan *FlowLine) {
	lines = make(chan *FlowLine, Buffer)

	go func() {
		var flowLine *FlowLine

		for line := range csvLines {
			flowLine, _ = NewFlowLine(line)
			lines <- flowLine
		}
		close(lines)
	}()

	return
}

type FlowLine struct {
	created_at    string
	duration      string
	duration_time string
	finished_at   string
	service       string
	topic         string
}

type FlowTable []*FlowLine

func NewFlowLine(csv *CsvLine) (*FlowLine, error) {
	self := FlowLine{}

	// TODO: Find a better way
	self.created_at = csv.Get("created_at")
	self.duration = csv.Get("duration")
	self.duration_time = csv.Get("duration_time")
	self.finished_at = csv.Get("finished_at")
	self.service = csv.Get("service")
	self.topic = csv.Get("topic")

	return &self, nil
}

func (self *FlowTable) Send() {
	// code to send to the database here.
	fmt.Printf("----\nSending %d lines\n%s", len(*self), *self)
}

func printStream(lines chan *FlowLine) (done chan int) {
	done = make(chan int)

	go func() {
		table := FlowTable{}
		i := 0

		for line := range lines {
			i++
			table = append(table, line)

			if len(table) >= 1000 {
				table.Send()
				table = FlowTable{}
			}
		}

		if len(table) > 0 {
			table.Send()
		}

		done <- i
	}()

	return
}
*/

func (self *CsvLine) Get(key string) (value string) {
	x := -1
	for i, value := range self.Header {
		if value == key {
			x = i
			break
		}
	}

	if x == -1 {
		return ""
	}

	return strings.TrimSpace(self.Line[x])
}
