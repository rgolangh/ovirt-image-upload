package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type RoyRoundTripper struct {
}

var uploadUrl = "http://localhost:9000/upload"

func foomain() {
	if len(os.Args) < 2 {
		panic("pass a file to upload")
	}

	pr, pw := io.Pipe()
	go func() {
		fmt.Fprintln(pw, "writing to pipe 1")
		fmt.Fprintln(pw, "writing to pipe 1")
		pw.Close()
	}()
	_, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	d := make([]byte, 0)
	_, err = pr.Read(d)
	if err != nil {
		panic("error not eof" + err.Error())
	}
	log.Printf("got this data %s", d)

	//putRequest, err := http.NewRequest(http.MethodPut, uploadUrl, f)
	//if err != nil {
	//	panic(fmt.Sprintf("[DEBUG] Failed writing to create a PUT request %s", err))
	//}
	//stats, err := f.Stat()
	//start := 0
	//length := stats.Size()
	//putRequest.Header.Add("content-type", "application/octet-stream")
	//putRequest.Header.Add("content-length", string(stats.Size()))
	//putRequest.Header.Add("content-range", string(start, start + length - 1))
	//
	////http.Client{Transport: RoyRoundTripper{}}

}

func (r *RoyRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	panic("e")
}
