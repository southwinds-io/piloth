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
	"southwinds.dev/artisan/data"
	"southwinds.dev/pilotctl/types"
	"testing"
	"time"
)

// test the worker
func TestWorker(t *testing.T) {
	os.Setenv("PILOT_HOME", "../")
	// create a new job processing worker
	w := NewWorker(
		// define the processing logic
		func(data interface{}) (string, error) {
			// unbox the data
			c, ok := data.(types.CmdInfo)
			if !ok {
				panic("CmdInfo type casting failed")
			}
			// print start of job message
			log.Printf("processing job %d, %s -> %s\n", c.JobId, c.Package, c.Function)
			// simulate process with delay
			time.Sleep(1 * time.Second)
			// return the job result
			return fmt.Sprintf("JOB %d => complete\n", c.JobId), nil
		})
	// start the worker loop
	w.Start()
	// add a couple of jobs
	w.AddJob(types.CmdInfo{
		JobId:    1010,
		Package:  "list",
		Function: "list2",
		Input:    &data.Input{},
	})
	w.AddJob(types.CmdInfo{
		JobId:    1020,
		Package:  "list",
		Function: "list2",
		Input:    &data.Input{},
	})
	// wait until no more jobs to process
	for w.Jobs() > 0 {
		time.Sleep(1 * time.Second)
	}

	// starts a loop to process job results
	var (
		results = 2
		count   = 0
	)
	for count < results {
		// attempt to retrieve the next result
		r, _ := peekJobResult()
		if r != nil {
			count++
			status := func() string {
				if r.Success {
					return "successful"
				}
				return "failed"
			}()
			if status == "successful" {
				removeJobResult(*r)
			}
			log.Printf("result for job %d: %s\n", r.JobId, status)
		} else {
			fmt.Printf(".")
			time.Sleep(time.Second)
		}
	}
}
