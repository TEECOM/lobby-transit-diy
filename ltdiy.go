/*
 * Copyright (c) 2016, TEECOM
 *
 * This code is provided for free, as is, under the MIT license
 * (see LICENSE.md).
 */

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

// Strucures; the actual information is separated from updates
type line struct {
	Name  string
	ID    string
	Times []int
	Color string
}

type coordinates struct {
	Lat float64
	Lon float64
}

type station struct {
	Name  string
	ID    string
	Coord coordinates

	Directions [2]string

	Lines [2]map[string]*line
}

type system struct {
	Name    string
	Tagline string
	Stops   []station
	TimeMax int
	stopMap map[string]*station
}

// Update structures (externally generated)
type lineUpdate struct {
	LineID string
	Index  int
	Times  []int
}

type stationUpdate struct {
	StationID string
	Lines     []lineUpdate
}

type update struct {
	Stops []stationUpdate
}

// This is the main system information; fill this as appropriate
var mainSystem system = system{
	Name:    "TEECOM Shuttle Service",
	Tagline: "San Francisco Bay Area",
	Stops: []station{
		station{
			Name: "TEECOM",
			ID:   "tee",
			Coord: coordinates{
				Lat: 10.5,
				Lon: 10.5,
			},
			Directions: [2]string{"Northbound", "Southbound"},
			Lines: [2]map[string]*line{
				map[string]*line{
					"sh": &line{
						Name:  "Shuttle",
						ID:    "sh",
						Color: "#ff0000",
					},
				},
			},
		},
	},

	TimeMax: 45,

	stopMap: make(map[string]*station),
}

// Where static files will be found
const staticDirectory string = "static"

func prepareSystem() {
	// Cache system IDs for future lookup
	for i := 0; i < len(mainSystem.Stops); i++ {
		stop := &mainSystem.Stops[i]
		mainSystem.stopMap[stop.ID] = stop
	}
}

func main() {
	log.Println("Starting server")
	prepareSystem()

	// Setup routing
	http.HandleFunc("/info", handleInfo)
	http.HandleFunc("/update", handleUpdate)
	http.HandleFunc("/stop", handleStopInfo)

	// Run server on port 8080
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
}

// JSON encode all of the information
func handleInfo(w http.ResponseWriter, r *http.Request) {
	if err := json.NewEncoder(w).Encode(mainSystem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Internal Server Error")
	}
}

func handleStopInfo(w http.ResponseWriter, r *http.Request) {
	// Check for valid GET parameters
	stopID := r.URL.Query()["id"]
	if stopID == nil || len(stopID) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "400 Bad Request: Missing stop ID")
		return
	}

	// Try to find the correct stop
	stop := mainSystem.stopMap[stopID[0]]
	if stop == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "400 Bad Request: Invalid stop id (%s)", stopID[0])
	}

	// Send the response
	if err := json.NewEncoder(w).Encode(stop); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "Internal Server Error")
	}
}

// Handle update request
func handleUpdate(w http.ResponseWriter, r *http.Request) {
	// Ensure we are dealing with a POST request
	if r.Method != "POST" {
		serve(w, "update.html", http.StatusBadRequest)
		return
	}

	// Decode the JSON
	var new update
	if err := json.NewDecoder(r.Body).Decode(&new); err != nil {
		serve(w, "badupdate.html", http.StatusBadRequest)
		return
	}

	// Try to apply the updates
	if err := processUpdates(&new); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "400 Bad Request: %s", err.Error())
		return
	}

	log.Println(new)
}

func processUpdates(u *update) error {
	for _, su := range u.Stops {
		stop := mainSystem.stopMap[su.StationID]
		if stop == nil {
			return errors.New("Invalid station ID")
		}

		for _, lu := range su.Lines {
			if lu.Index > 1 {
				return errors.New("Line index out of bounds")
			}

			ln := stop.Lines[lu.Index][lu.LineID]
			if ln == nil {
				return errors.New("Invalid line ID")
			}

			ln.Times = lu.Times
			log.Println(ln.Times)
		}
	}

	return nil
}

func serve(w http.ResponseWriter, f string, code int) {
	text, err := ioutil.ReadFile(staticDirectory + "/" + f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "500 Internal Server Error\n")
		return
	}

	w.WriteHeader(code)
	fmt.Fprintf(w, "%s", text)
}
