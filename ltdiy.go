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
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"
)

// Strucures; the actual information is separated from updates
type line struct {
	Name  string `json:"name"`
	ID    string `json:"id"`
	Times []int  `json:"times"`
	Color string `json:"color"`
}

type coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type station struct {
	Name  string      `json:"name"`
	ID    string      `json:"id"`
	Coord coordinates `json:"coord"`

	Directions [2]string `json:"directions"`

	Lines [2]map[string]*line `json:"lines"`
}

type system struct {
	sync.RWMutex           // Protects everything below
	Name         string    `json:"name"`
	Tagline      string    `json:"tagline"`
	Stops        []station `json:"stops"`
	TimeMax      int       `json:"timeMax"`
	stopMap      map[string]*station
}

// Update structures (externally generated)
type lineUpdate struct {
	LineID string `json:"lineID"`
	Index  int    `json:"index"`
	Times  []int  `json:"times"`
}

type stationUpdate struct {
	StationID string       `json:"stationID"`
	Lines     []lineUpdate `json:"lines"`
}

type update struct {
	Stops []stationUpdate `json:"stops"`
}

// This is the main system information; at runtime this is filled
// in by the supplied configuration file
var mainSystem system = system{
	stopMap: make(map[string]*station),
}

// Where static files will be found
const staticDirectory string = "static"

func main() {
	log.Println("Starting server")

	// Setup command line flags
	configPtr := flag.String("config", "", "Configuration file")
	flag.Parse()
	if *configPtr == "" {
		log.Fatal("No configuration provided. Use '-config=<config filename>'")
	}

	// Build the server configuration
	readConfig(*configPtr)

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
	// Obtain a read lock for the system
	mainSystem.RLock()
	defer mainSystem.RUnlock()

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
		fmt.Fprintln(w, "400 Bad Request: Missing stop ID")
		return
	}

	// Obtain a read lock for the system
	mainSystem.RLock()
	defer mainSystem.RUnlock()

	// Try to find the correct stop
	stop := mainSystem.stopMap[stopID[0]]
	if stop == nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "400 Bad Request: Invalid stop id (%s)\n", stopID[0])
		return
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
		fmt.Fprintf(w, "400 Bad Request: %s\n", err.Error())
		return
	}
}

func processUpdates(u *update) error {
	// Obtain a writer lock
	mainSystem.Lock()
	defer mainSystem.Unlock()

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
		}
	}

	return nil
}

func serve(w http.ResponseWriter, f string, code int) {
	text, err := ioutil.ReadFile(staticDirectory + "/" + f)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "500 Internal Server Error")
		return
	}

	w.WriteHeader(code)
	fmt.Fprintf(w, "%s", text)
}

func readConfig(filename string) {
	f, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Unable to open configuration file (%s)", filename)
	}

	log.Printf("Using configuration file (%s)", filename)

	if jserr := json.NewDecoder(f).Decode(&mainSystem); jserr != nil {
		log.Fatal("Malformed json configuration")
	}

	// Cache system IDs for future lookup.
	// No need to do any locking as the server hasn't
	// started up yet.
	for i := 0; i < len(mainSystem.Stops); i++ {
		stop := &mainSystem.Stops[i]
		mainSystem.stopMap[stop.ID] = stop
	}
}
