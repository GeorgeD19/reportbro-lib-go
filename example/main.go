package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	reportbro "github.com/GeorgeD19/reportbro-lib-go"
)

func main() {
	var files []string

	err := filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if filepath.Ext(path) == ".json" && !strings.HasPrefix(path, ".") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	for _, fileName := range files {
		file, err := ioutil.ReadFile(fileName)
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
