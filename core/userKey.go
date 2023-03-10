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
	"strconv"
	"strings"
	"time"
)

type userKey struct {
	Key       userKeyString `json:"uk"`
	Signature string        `json:"s"`
}

type userKeyString string

type userKeyInfo struct {
	Username string
	URI      string
	IV       []byte
	SK       []byte
	Expiry   *time.Time
}

func loadUserKey(path string) (*userKey, error) {
	defer TRA(CE())
	if len(path) == 0 {
		path = ".userkey"
	}
	path = Abs(path)
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read user key file: %s\n", err)
	}
	d, err := hex.DecodeString(string(b[:]))
	if err != nil {
		return nil, fmt.Errorf("cannot decode user key: %s\n", err)
	}
	key := new(userKey)
	err = json.Unmarshal(d, key)
	if err != nil {
		return nil, fmt.Errorf("cannot unmarshal user key: %s\n", err)
	}
	return key, nil
}

// readUserKey read the content of an encoded user key and verifies its digital signature
func readUserKey(key userKey) (*userKeyInfo, error) {
	defer TRA(CE())
	// check the validity of the key's digital signature
	if valid, err := verify(string(key.Key), key.Signature); !valid {
		return nil, fmt.Errorf("invalid user key signature: %s\n", err)
	}
	// decrypt the key information
	d, err := decrypt(sk, string(key.Key), iv)
	if err != nil {
		return nil, fmt.Errorf("cannot decrypt user key data: %s\n", err)
	}
	db, err := hex.DecodeString(d[:])
	if err != nil {
		return nil, fmt.Errorf("cannot decode user key data: %s\n", err)
	}
	parts := strings.Split(string(db[44:]), ",")
	var expiry *time.Time
	// parse the expiry days
	days, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve expiry days from user key: %s\n", err)
	}
	// if days is not zero then create a date stamp
	if days > 0 {
		expiryDate := time.Now().Add(time.Hour * 24 * time.Duration(days))
		expiry = &expiryDate
	}
	return &userKeyInfo{
		Username: parts[0],
		URI:      parts[1],
		IV:       db[:12],
		SK:       db[12:44],
		Expiry:   expiry,
	}, nil
}
