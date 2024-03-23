package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gocarina/gocsv"
)

const TIMELAYOUT = "2006-01-02"

type WhatCMSResponse struct {
	Result  Result
	Results []Technology
	Meta    Meta
}

type Result struct {
	Code int
	Msg  string
}

type Technology struct {
	Name       string   `csv:"Name"`
	ID         int      `csv:"Id"`
	Version    string   `csv:"Version"`
	Categories []string `csv:"Categories"`
	Url        string   `csv:"URL"`
}

type Meta struct {
	Socials []Social
}

type Social struct {
	Network string `csv:"Network"`
	Url     string `csv:"URL"`
	Profile string `csv:"Profile"`
}

type urlList []string

func main() {
	apiKey := flag.String("key", "", "your api key")
	delay := flag.Int("delay", 10, "your required delay, in seconds")
	flag.Parse()

	urlList := "imports/urlList.csv"
	technologyExport := "export/technology.csv"
	socialsExport := "export/socials.csv"

	// open file
	f, err := os.Open(urlList)
	if err != nil {
		log.Fatal(err)
	}

	// remember to close the file at the end of the program
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}

	// convert records to array of structs
	importedList := createUrlList(data)

	technology, social, unretrieved := queryWhatCMS(importedList, *apiKey, *delay)

	// write technology records to csv
	err = writeListToCsv(technologyExport, technology)
	if err != nil {
		panic(err)
	}

	// write social records to csv
	err = writeListToCsv(socialsExport, social)
	if err != nil {
		panic(err)
	}

	// print unretrieved
	fmt.Printf("unretrieved urls:\n")
	for _, item := range unretrieved {
		fmt.Printf(item + "\n")
	}
}

func writeListToCsv[name string, items []Technology | []Social](filename string, list items) error {
	csvFile, err := os.Create(filename)
	if err != nil {
		log.Fatalf("failed creating file: %s", err)
		return err
	}

	err = gocsv.MarshalFile(&list, csvFile)
	if err != nil {
		return err
	}

	csvFile.Close()

	return nil
}

func createUrlList(data [][]string) []string {
	var urls []string
	for _, line := range data {
		urls = append(urls, line[0])
	}
	return urls
}

func queryWhatCMS(urlList []string, apiKey string, delay int) ([]Technology, []Social, []string) {
	var unretrieved []string
	var technology []Technology
	var social []Social

	for _, url := range urlList {
		requestUrl := "https://whatcms.org/API/Tech?key=" + apiKey + "&url=" + url

		fmt.Printf("requesting " + url + ":\n")
		resp, err := http.Get(requestUrl)

		fmt.Printf("response: ")
		fmt.Print(resp)
		fmt.Printf("\n")
		if err != nil {
			unretrieved = append(unretrieved, url)
			continue
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			unretrieved = append(unretrieved, url)
			continue
		}

		whatCmsResponse := WhatCMSResponse{}
		err = json.Unmarshal(body, &whatCmsResponse)

		if err != nil {
			unretrieved = append(unretrieved, url)
			continue
		}

		if whatCmsResponse.Result.Code != 200 {
			unretrieved = append(unretrieved, url)
			continue
		}

		technology = append(technology, whatCmsResponse.Results...)
		social = append(social, whatCmsResponse.Meta.Socials...)

		fmt.Println("Delay started, waiting for 10 seconds.")
		time.Sleep(time.Duration(delay) * time.Second)
	}

	return technology, social, unretrieved
}
