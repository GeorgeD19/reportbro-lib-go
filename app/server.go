package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/GeorgeD19/reportbro-lib-go"
	"github.com/google/uuid"
	"github.com/patrickmn/go-cache"

	"github.com/go-chi/chi"
	"github.com/shomali11/util/xconditions"
)

type ReportBro struct {
	OutputFormat string                 `json:"outputFormat"`
	IsTestData   bool                   `json:"isTestData"`
	Report       map[string]interface{} `json:"report"`
	Data         map[string]interface{} `json:"data"`
}

func main() {

	c := cache.New(5*time.Minute, 10*time.Minute)
	router := chi.NewRouter()

	router.Handle("/report/run", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With")

		key := r.URL.Query().Get("key")
		if key != "" {
			if report, ok := c.Get(key); ok {
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, post-check=0, pre-check=0")
				w.Header().Set("Connection", "keep-alive")
				w.Header().Set("Content-Disposition", "inline; filename=\"report.pdf\"")
				w.Header().Set("Content-Type", "application/pdf")
				w.Header().Set("Date", time.Now().String())
				w.Header().Set("Expires", time.Now().String())
				w.Header().Set("Pragma", "no-cache")
				w.Write(report.([]byte))
			}
		} else {
			decoder := json.NewDecoder(r.Body)
			var data ReportBro
			err := decoder.Decode(&data)

			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				fmt.Println(err)
				json.NewEncoder(w).Encode(err)
				return
			}

			report := reportbro.NewReport(data.Report, data.Data, data.IsTestData, "", nil)
			generated, err := report.GeneratePDF(true)
			if err != nil {
				w.Header().Set("Content-Type", "application/json")
				fmt.Println(err)
				json.NewEncoder(w).Encode(err)
				return
			}

			key := uuid.New()
			c.Set(key.String(), generated, cache.DefaultExpiration)

			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("key:" + key.String()))
		}
	}))

	log.Fatal(http.ListenAndServe(":"+xconditions.IfThenElse(os.Getenv("HTTP_PLATFORM_PORT") != "", os.Getenv("HTTP_PLATFORM_PORT"), "3001").(string), router))
}
