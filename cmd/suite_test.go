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

package cmd

import (
	"southwinds.dev/artisan/core"
	ctl "southwinds.dev/pilotctl/types"
	pilotCore "southwinds.dev/piloth/core"
	"testing"
)

func TestStart(t *testing.T) {
	// collects device/host information
	hostInfo, err := ctl.NewHostInfo()
	if err != nil {
		core.RaiseErr("cannot collect host information")
	}
	// creates pilot instance
	p, err := pilotCore.NewPilot(pilotCore.PilotOptions{
		UseHwId:            false,
		Telemetry:          true,
		Tracing:            false,
		Info:               hostInfo,
		CPU:                false,
		MEM:                false,
		InsecureSkipVerify: true,
		CVEPath:            "cve",
	})
	core.CheckErr(err, "cannot start pilot")
	// start the pilot
	p.Start()
}
