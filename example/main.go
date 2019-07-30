package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	reportbro "github.com/GeorgeD19/reportbro-lib-go"
)

func main() {
	files := []string{
		"blank",
		"certificate",
		"contract",
		"deliveryslip",
		"invoice",
	}
	for _, fileName := range files {
		file, err := ioutil.ReadFile(fileName + ".json")
		if err != nil {
			panic(err)
		}

		var contents map[string]interface{}
		json.Unmarshal([]byte(file), &contents)
		definition := contents["report"].(map[string]interface{})
		data := contents["data"].(map[string]interface{})

		// Pass in image data as raw bytes with key references using a json path (see https://jsonpath.com/) data[0].image
		report := reportbro.NewReport(definition, data, false, "", nil)
		generated, err := report.GeneratePDF(false)

		if err != nil {
			fmt.Println(err)
		}

		pdfFile, err := os.Create(fileName + "-golang.pdf")
		if err == nil {
			pdfFile.Write(generated)
			pdfFile.Close()
		}
	}

}
