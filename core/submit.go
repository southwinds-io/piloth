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
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"southwinds.dev/pilotctl/types"
)

// submitJobResult persist the result of executing a Job in the file system
// fail-safe: remove start mark created by peek
//
//	create a submitted mark in case host goes before removing the job from the queue
func submitJobResult(result types.JobResult) error {
	defer TRA(CE())
	dir := submitDir(fmt.Sprintf("job_%d.result", result.JobId))
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(dir, bytes, os.ModePerm)
	if err != nil {
		return err
	}
	return submittedMarker(result.JobId)
}

func submittedMarker(jobId int64) error {
	defer TRA(CE())
	dir := dataDir(fmt.Sprintf("job_%d.submitted", jobId))
	// creates a submitted marker
	err := ioutil.WriteFile(dir, []byte{}, os.ModePerm)
	if err != nil {
		return err
	}
	// remove the started marker
	dir = dataDir(fmt.Sprintf("job_%d.started", jobId))
	err = os.Remove(dir)
	// started marker might not exist if the job failed to start
	if err != nil {
		// do nothing
	}
	return nil
}

func submittedMarkerExists(jobId int64) bool {
	defer TRA(CE())
	dir := dataDir(fmt.Sprintf("job_%d.submitted", jobId))
	_, err := os.Stat(dir)
	return err == nil
}

func peekJobResult() (jobResult *types.JobResult, err error) {
	defer TRA(CE())
	var bytes []byte
	dir := submitDir("")
	files, err := lsJobs(dir)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if !file.IsDir() && path.Ext(file.Name()) == ".result" {
			bytes, err = ioutil.ReadFile(submitDir(file.Name()))
			if err != nil {
				return nil, err
			}
			err = json.Unmarshal(bytes, &jobResult)
			if err != nil {
				return nil, err
			}
			// returns the found job and creates a started marker for the job in the file system
			return jobResult, nil
		}
	}
	// no job found
	return nil, nil
}

func removeJobResult(result types.JobResult) error {
	defer TRA(CE())
	// remove job from queue
	dir := submitDir(fmt.Sprintf("job_%d.result", result.JobId))
	return os.Remove(dir)
}
