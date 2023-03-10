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
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"southwinds.dev/artisan/core"
)

type TelemCtl struct {
	logsChannels    []string
	metricsChannels []string
}

func NewTelemCtl() (*TelemCtl, error) {
	var err error
	path := os.Getenv("PILOT_CTL_TELEM_PATH")
	if len(path) == 0 {
		log.Printf("missing PILOT_CTL_TELEM_PATH variable, reading telemetry data from default path at ./telemetry\n")
		path = "telemetry"
	} else {
		log.Printf("reading telemetry data from %s\n", path)
	}
	path, _ = filepath.Abs(path)

	// get the logs channels
	logsPath := filepath.Join(path, "logs")
	var logsChannels []string
	if _, err = os.Stat(logsPath); os.IsNotExist(err) {
		core.InfoLogger.Printf("logs path %s not found, skipping logs publication")
	} else {
		logsChannels, err = ls(logsPath, true)
		if err != nil {
			return nil, err
		}
	}
	// get the metrics channels
	metricsPath := filepath.Join(path, "metrics")
	var metricsChannels []string
	if _, err = os.Stat(metricsPath); os.IsNotExist(err) {
		core.InfoLogger.Printf("metrics path %s not found, skipping metrics publication")
	} else {
		metricsChannels, err = ls(metricsPath, true)
		if err != nil {
			return nil, err
		}
	}

	return &TelemCtl{
		logsChannels:    logsChannels,
		metricsChannels: metricsChannels,
	}, nil
}

func (t *TelemCtl) Start(api *PilotCtl) error {
	for _, mChannel := range t.metricsChannels {
		p, _ := NewProcessor(mChannel, api, "metrics")
		p.Start()
	}
	for _, mChannel := range t.logsChannels {
		p, _ := NewProcessor(mChannel, api, "logs")
		p.Start()
	}
	return nil
}

// ls returns a list of file or folder names ordered by mod time
func ls(dirname string, isDir bool) ([]string, error) {
	// read entries from folder
	entries, err := os.ReadDir(dirname)
	if err != nil {
		return nil, fmt.Errorf("cannot read telemetry entries: %s", err)
	}
	// sort the file slice by ModTime()
	// ensuring the oldest file is processed first
	sort.Slice(entries, func(i, j int) bool {
		ii, _ := entries[i].Info()
		jj, _ := entries[j].Info()
		return ii.ModTime().UnixNano() <= jj.ModTime().UnixNano()
	})
	result := make([]string, 0)
	for _, entry := range entries {
		if isDir == entry.IsDir() {
			abs := filepath.Join(dirname, entry.Name())
			result = append(result, abs)
		}
	}
	return result, nil
}
