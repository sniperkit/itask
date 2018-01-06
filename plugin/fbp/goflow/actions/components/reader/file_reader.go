package reader

import (
	"bufio"
	"log"
	"os"

	"github.com/sniperkit/xtask/plugin/fbp/goflow/actions/configuration"
)

type fileReader struct {
	inputFile *os.File
}

func newFileReader(configuration *configuration.Configuration) *fileReader {
	inputFile := configuration.FuzzSetFile()
	return &fileReader{inputFile: inputFile}
}

func (f *fileReader) pipe(out chan<- string, done chan<- bool) {
	scanner := bufio.NewScanner(f.inputFile)
	f.pipeFileContent(scanner, out)
	f.closeFile()
	done <- true
}

func (f *fileReader) pipeFileContent(scanner *bufio.Scanner, out chan<- string) {
	for scanner.Scan() {
		// loop over suffixes
		line := scanner.Text()
		out <- line
	}
}

func (f *fileReader) closeFile() {
	if error := f.inputFile.Close(); nil != error {
		log.Fatalf("Error occured while closing a file \"%v\": %v", f.inputFile.Name(), error)
	}
}
