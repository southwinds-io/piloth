/*
   Pilot Host Controller
   Copyright (C) 2022-Present SouthWinds Tech Ltd - www.southwinds.io

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package core

import (
	"encoding/json"
	"os"
	"path/filepath"
	ctl "southwinds.dev/pilotctl/types"
)

// getEvents retrieve event log entries
func getEvents(max int) (*ctl.Events, error) {
	defer TRA(CE())
	dir := submitDir("")
	files, err := lsJobs(dir)
	if err != nil {
		return nil, err
	}
	// collect event file names up to a max number
	var (
		names  []string
		events = &ctl.Events{Events: []ctl.Event{}}
	)
	// loop through the files in submit directory
	for _, file := range files {
		// if the file is an event (*.ev)
		if !file.IsDir() && filepath.Ext(file.Name()) == ".ev" {
			// append its name to the event list
			names = append(names, file.Name())
			// read the event bytes
			bytes, err := os.ReadFile(submitDir(file.Name()))
			if err != nil {
				return nil, err
			}
			// unmarshal the event bytes
			var entry ctl.Event
			err = json.Unmarshal(bytes, &entry)
			if err != nil {
				return nil, err
			}
			// append the event to the event list
			events.Events = append(events.Events, entry)
			if len(names) >= max {
				break
			}
		}
	}
	// if there are no events
	if names == nil {
		// return
		return nil, nil
	}
	// otherwise, save the list of events being processed
	bytes, err := json.Marshal(names)
	if err != nil {
		return nil, err
	}
	// write to file
	err = os.WriteFile(dataDir("events.json"), bytes, os.ModePerm)
	if err != nil {
		return nil, err
	}
	return events, nil
}

// remove events that have been submitted
func removeEvents() error {
	defer TRA(CE())
	// work out the file path where events
	dir := dataDir("events.json")
	bytes, err := os.ReadFile(dir)
	if err != nil {
		return nil
	}
	var names []string
	err = json.Unmarshal(bytes, &names)
	if err != nil {
		return nil
	}
	// remove the respective event files
	for i := 0; i < len(names); i++ {
		err = os.Remove(submitDir(names[i]))
		if err != nil {
			return err
		}
	}
	// remove the events.json file marker
	return os.Remove(dataDir("events.json"))
}
