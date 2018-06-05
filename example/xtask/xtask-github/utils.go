package main

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// create custom client with httpcache
func downloadFromURL(url, dirName, fileName string) (err error) {
	filePath := fmt.Sprintf("%s/%s", dirName, fileName)

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	out, err := os.Create(filePath)
	if err != nil {
		fmt.Println("Error while opening file", filePath, err)
		return
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	return
}

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func shuffle(slc []interface{}) []interface{} {
	N := len(slc)
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rand.Intn(N-i)
		slc[r], slc[i] = slc[i], slc[r]
	}
	return slc
}

func shuffleInts(slc []int) []int {
	N := len(slc)
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rand.Intn(N-i)
		slc[r], slc[i] = slc[i], slc[r]
	}
	return slc
}

func shuffleStrings(slc []string) []string {
	N := len(slc)
	for i := 0; i < N; i++ {
		// choose index uniformly in [i, N-1]
		r := i + rand.Intn(N-i)
		slc[r], slc[i] = slc[i], slc[r]
	}
	return slc
}
