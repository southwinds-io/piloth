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
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type AKRequestBearerToken struct {
	info     userKeyInfo
	Username string `json:"username"`
	// DeviceId is either the host primary interface MAC address or the device hardware uuid
	DeviceId  string    `json:"device_id"`
	IpAddress string    `json:"ip_address"`
	Hostname  string    `json:"hostname"`
	Time      time.Time `json:"time"`
}

func NewAKRequestBearerToken(clientInfo userKeyInfo, options PilotOptions) AKRequestBearerToken {
	defer TRA(CE())
	var deviceId string
	// if hardware id should be used to identify the device
	if options.UseHwId {
		// then set the device identifier to the hardware id
		deviceId = options.Info.HardwareId
	} else {
		// otherwise, set it to the primary mac address
		deviceId = options.Info.PrimaryMAC
	}
	return AKRequestBearerToken{
		info:      clientInfo,
		Username:  clientInfo.Username,
		DeviceId:  deviceId,
		IpAddress: options.Info.HostIP,
		Hostname:  options.Info.HostName,
		Time:      time.Now(),
	}
}

func (t AKRequestBearerToken) String() string {
	defer TRA(CE())
	b, err := json.Marshal(t)
	if err != nil {
		ErrorLogger.Printf("cannot create activation key request bearer token: %s\n", err)
		os.Exit(1)
	}
	return fmt.Sprintf("Bearer %s %s", t.Username, encrypt(t.info.SK, hex.EncodeToString(b), t.info.IV))
}

func activate(options PilotOptions) {
	defer TRA(CE())
	var (
		failures float64 = 0
		interval time.Duration
	)
	// first check for a valid activation key
	if !AkExist() {
		// if no user key exists
		if !UserKeyExist() {
			// cannot continue
			ErrorLogger.Printf("cannot launch pilot, missing user key\n")
			os.Exit(1)
		}
		// otherwise, it can start the activation process
		InfoLogger.Printf("cannot find activation key, initiating activation protocol\n")
		uKey, err := loadUserKey(UserKeyFile())
		if err != nil {
			// cannot continue
			ErrorLogger.Printf("cannot launch pilot, cannot load user key: %s\n", err)
			os.Exit(1)
		}
		tenant, err := readUserKey(*uKey)
		if err != nil {
			ErrorLogger.Printf("cannot launch pilot, cannot load user key: %s\n", err)
			os.Exit(1)
		}
		// fetch remote key
		fetched, err := requestAKey(*tenant, options)
		// if failed retry
		for !fetched {
			// calculates wait interval with exponential backoff and jitter
			interval = nextInterval(failures)
			ErrorLogger.Printf("cannot retrieve activation key, retrying in %.2f minutes: %s\n", interval.Seconds()/60, err)
			// wait interval
			time.Sleep(interval)
			failures++
			// try again
			fetched, err = requestAKey(*tenant, options)
			// if successful
			if err == nil {
				// break the loop
				break
			}
		}
		InfoLogger.Printf("activation key deployed, pilot is ready to launch\n")
	}
	// before doing anything, verify activation key
	akInfo, err := LoadActivationKey()
	if err != nil {
		log.Fatalf("cannot start pilot: %s", err)
	}
	// set the activation
	A = akInfo
	// validate the activation key
	A.Validate()
	// check expiration date
	if A.Expiry.Before(time.Now()) {
		// if activation expired the exit
		ErrorLogger.Printf("cannot launch pilot: activation key expired\n")
		os.Exit(1)
	}
	// if set to use hardware id for device identification
	if options.UseHwId {
		if A.DeviceId != options.Info.HardwareId {
			// if the device Id is not the hardware id; then exit
			ErrorLogger.Printf("cannot launch pilot: invalid host hardware id: %s\n", options.Info.HardwareId)
			os.Exit(1)
		}
	} else { // use mac address for device identification
		// check if the mac-address matches the device id in the activation key
		matchedMac := false
		for _, macAddress := range options.Info.MacAddress {
			if A.DeviceId == macAddress {
				matchedMac = true
				break
			}
		}
		// if the mac address does not match
		if !matchedMac {
			// if the device Id is not the hardware id; then exit
			ErrorLogger.Printf("cannot launch pilot: invalid host mac address: %s\n", options.Info.PrimaryMAC)
			os.Exit(1)
		}
	}
	// set host UUID
	options.Info.HostUUID = A.HostUUID
}

func LoadActivationKey() (*AKInfo, error) {
	ak, err := loadAKey(AkFile())
	if err != nil {
		// if it cannot load activation key exit
		return nil, fmt.Errorf("cannot load activation key, %s\n", err)
	}
	akInfo, err := readAKey(*ak)
	if err != nil {
		// if it cannot load activation key exit
		return nil, fmt.Errorf("cannot read activation key, %s\n", err)
	}
	return akInfo, nil
}

func requestAKey(clientKey userKeyInfo, options PilotOptions) (bool, error) {
	bearerToken := NewAKRequestBearerToken(clientKey, options)
	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: options.InsecureSkipVerify,
			},
		},
		Timeout: time.Second * 60,
	}
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/activation-key", clientKey.URI), nil)
	if err != nil {
		return false, fmt.Errorf("cannot create activation request: %s\n", err)
	}
	req.Header.Add("Authorization", bearerToken.String())
	req.Header.Add("Content-Type", "application/json")
	var resp *http.Response
	if IsDebug() {
		DebugLogger.Printf("requesting activation key from: %s/activation-key", clientKey.URI)
	}
	resp, err = c.Do(req)
	if err != nil {
		if IsDebug() {
			if resp != nil {
				var respBytes []byte
				respBytes, err = json.MarshalIndent(resp, "", " ")
				if err != nil {
					ErrorLogger.Printf("cannot marshall activation key response: %s", err)
				}
				DebugLogger.Println(string(respBytes[:]))
			}
		}
		return false, fmt.Errorf("cannot request activation key: %s\n", err)
	}
	if resp != nil && resp.StatusCode != http.StatusCreated {
		return false, fmt.Errorf("activation key request failed with code %d: %s\n", resp.StatusCode, resp.Status)
	}
	ak, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf("cannot read activation key from http response: %s\n", err)
	}
	err = os.WriteFile(AkFile(), ak, 0600)
	if err != nil {
		return false, fmt.Errorf("cannot write activation file: %s\n", err)
	}
	return true, nil
}
