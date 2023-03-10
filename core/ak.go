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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type AK struct {
	Data      string `json:"ak"`
	Signature string `json:"s"`
}

func (ak *AK) String() string {
	b, _ := json.Marshal(ak)
	return string(b[:])
}

type AKInfo struct {
	HostUUID  string    `json:"host_uuid"`
	DeviceId  string    `json:"device_id"`
	CtlURI    string    `json:"ctl_uri"`
	Expiry    time.Time `json:"expiry"`
	VerifyKey string    `json:"verify_key"`
}

func (a AKInfo) Validate() {
	defer TRA(CE())
	// if the verification key is not provided
	if len(a.VerifyKey) == 0 {
		// cannot continue
		ErrorLogger.Printf("cannot launch pilot: activation key does not have a verification key\n")
		os.Exit(1)
	}
	if len(a.DeviceId) == 0 {
		// cannot continue
		ErrorLogger.Printf("cannot launch pilot: activation key does not have a device Id\n")
		os.Exit(1)
	}
	if len(a.HostUUID) == 0 {
		// cannot continue
		ErrorLogger.Printf("cannot launch pilot: activation key does not have a host identifier\n")
		os.Exit(1)
	}
}

func AkExist() bool {
	defer TRA(CE())
	_, err := os.Stat(AkFile())
	return err == nil
}

func UserKeyExist() bool {
	defer TRA(CE())
	_, err := os.Stat(UserKeyFile())
	return err == nil
}

func readAKey(ak AK) (*AKInfo, error) {
	defer TRA(CE())
	if valid, err := verify(ak.Data, ak.Signature); !valid {
		return nil, fmt.Errorf("signature verification failed: %s\n", err)
	}
	data, err := decrypt(sk, ak.Data, iv)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt activation key: %s\n", err)
	}
	akinfo := new(AKInfo)
	err = json.Unmarshal([]byte(data), akinfo)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal activation key: %s\n", err)
	}
	return akinfo, nil
}

func loadAKey(path string) (*AK, error) {
	defer TRA(CE())
	path = Abs(path)
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read activation key file: %s\n", err)
	}
	d, err := hex.DecodeString(string(keyBytes[:]))
	if err != nil {
		return nil, fmt.Errorf("cannot decode activation key: %s\n", err)
	}
	ak := new(AK)
	err = json.Unmarshal(d, ak)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal activation key: %s\n", err)
	}
	return ak, err
}
