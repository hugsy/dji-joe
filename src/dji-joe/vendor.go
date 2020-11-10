package djijoe

import (
	"bytes"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
)

type Vendor struct {
	Name               string
	MacAddressPrefixes [][]byte
}

/*
Checks if a MAC address prefix belongs to the vendor.
*/
func (v *Vendor) HasPrefix(mac []byte) bool {
	for _, prefix := range v.MacAddressPrefixes {
		if bytes.Compare(mac, prefix) == 0 {
			return true
		}
	}
	return false
}

var AlreadyExistingMacError = errors.New("The MAC address is already defined")

/*
Add a new prefix to the list of the vendor
*/
func (v *Vendor) AddPrefix(mac []byte) error {
	if v.HasPrefix(mac) {
		return AlreadyExistingMacError
	}

	v.MacAddressPrefixes = append(v.MacAddressPrefixes, mac)
	return nil
}

func (v Vendor) String() string {
	return fmt.Sprintf("<Vendor name='%s', prefix=%v>", v.Name, v.MacAddressPrefixes)
}

type Vendors []*Vendor

var AlreadyExistingVendorError = errors.New("The MAC address is already defined")

func (v *Vendors) GetOrCreateVendor(name string) *Vendor {
	for _, vendor := range *v {
		if name == vendor.Name {
			return vendor
		}
	}

	new_vendor := new(Vendor)
	new_vendor.Name = name
	*v = append(*v, new_vendor)
	return new_vendor
}

/*
Load the vendor MAC address prefixes
*/
func LoadVendorsInfoFromFile(filePath string) Vendors {
	file, err := os.Open(filePath)
	if err != nil {
		Log.FatalF("Failed to open '%s': %+v", filePath, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	lineno := 0
	nbPrefix := 0

	var v Vendors

	for {
		records, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			Log.FatalF("Error: %+v", err)
		}

		if len(records) < 2 || len(records[1]) != 6 {
			Log.ErrorF("Incorrect entry line %d, skipping...", lineno)
			lineno++
			continue
		}

		decoded, err := hex.DecodeString(records[1])
		if err != nil {
			Log.ErrorF("Incorrect entry line %d, skipping...", lineno)
			lineno++
			continue
		}

		new_vendor := v.GetOrCreateVendor(records[0])
		err = new_vendor.AddPrefix(decoded)
		if err != nil {
			Log.WarningF("Cannot add prefix: %+v", err)
		} else {
			Log.DebugF("Added prefix '%s' for vendor '%s'", records[1], records[0])
			nbPrefix++
		}
		lineno++
	}

	Log.InfoF("%d vendors loaded (%d MAC address prefixes)", len(v), nbPrefix)
	return v
}
