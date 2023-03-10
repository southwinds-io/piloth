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
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type Processor struct {
	path      string
	api       *PilotCtl
	telemType string
}

func NewProcessor(path string, api *PilotCtl, telemType string) (*Processor, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return &Processor{
		path:      absPath,
		api:       api,
		telemType: telemType,
	}, nil
}

func (p *Processor) Start() {
	go p.run()
}

func (p *Processor) run() {
	var count = 0
	// working loop
	for {
		files, err := getFiles(p.path)
		if err != nil {
			log.Fatalf("cannot read files in path '%s': %s", p.path, err)
		}
		// if there are no files
		if len(files) == 0 {
			// sleeps a bit
			time.Sleep(30 * time.Second)
			// then restart the loop
			continue
		}
		// picks the oldest file
		file := filepath.Join(p.path, files[0].Name())
		if err != nil {
			log.Fatalf("cannot figure absolute path for '%s': %s", files[0].Name(), err)
		}
		c, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("cannot read file '%s': %s", files[0].Name(), err)
		}
		result, err := p.api.SubmitTelemetry(filepath.Base(p.path), c, p.telemType)
		if err != nil {
			waitTime := backoffTime(count)
			log.Printf("ERROR: cannot submit %s: %s; waiting %v...\n", p.telemType, err, waitTime)
			count++
			time.Sleep(waitTime)
		} else if len(result.Error) > 0 {
			waitTime := backoffTime(count)
			log.Printf("ERROR: cannot submit %s: %s; waiting %v...\n", p.telemType, result.Error, waitTime)
			count++
			time.Sleep(waitTime)
		} else {
			count = 0
			if err = os.Remove(file); err != nil {
				log.Printf("ERROR: cannot delete %s file after submition: %s\n", p.telemType, err)
			}
		}
	}
}

func getFiles(path string) ([]os.DirEntry, error) {
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		fmt.Errorf("cannot open path: %s", err)
	}
	dirs, err := f.ReadDir(-1)
	if err != nil {
		fmt.Errorf("cannot read path: %s", err)
	}
	// filter hidden files
	var files []os.DirEntry
	for _, dir := range dirs {
		ff, _ := dir.Info()
		if ff.Name()[0] != '.' {
			files = append(files, dir)
		}
	}
	// sort the directory entries by modification time
	sort.SliceStable(files, func(i, j int) bool {
		iInfo, _ := files[i].Info()
		jInfo, _ := files[j].Info()
		return iInfo.ModTime().Before(jInfo.ModTime())
	})
	return files, err
}

// backoffTime exponentially increase backoff time until reaching 1 hour
func backoffTime(attempts int) time.Duration {
	var exponentialBackoffCeilingSecs int64 = 3600 // 1 hour
	delaySecs := int64(math.Floor((math.Pow(2, float64(attempts)) - 1) * 0.5))
	if delaySecs > exponentialBackoffCeilingSecs {
		delaySecs = exponentialBackoffCeilingSecs
	}
	return time.Duration(delaySecs) * time.Second
}
