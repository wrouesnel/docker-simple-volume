/*
	the simple volume name query parser
*/

package main

import (
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"strconv"
	"strings"
)

const (
	PARSER_FIELDSEP = "_"
	PARSER_KVSEP    = "."
)

// Parses a docker volume name i.e "label.some_value" into an expanded
// volume query structure.
func ParseVolumeNameToQuery(volumeName string) (VolumeQuery, error) {
	q := VolumeQuery{}
	// Get fields
	fields := strings.Split(volumeName, PARSER_FIELDSEP)

	for _, field := range fields {
		// Split field into key value
		kv := strings.SplitN(field, PARSER_KVSEP, 1)

		key := kv[0]
		value := kv[1]

		switch key {
		case "label":
			q.Label = value
		case "own-hostname":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.OwnHostname = b
			}
		case "own-machine-id":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.OwnMachineId = b
			}
		case "initialized":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.Initialized = b
			}
		case "basename":
			q.Basename = value
		case "naming-style":
			switch value {
			case NamingNumeric, NamingUUID:
				q.NamingStyle = NamingType(value)
			default:
				return VolumeQuery{}, errors.New("Not a valid naming style")
			}
		case "exclusive":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.Exclusive = b
			}
		case "min-size":
			if bytes, err := humanize.ParseBytes(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.MinimumSizeBytes = bytes
			}
		case "max-size":
			if bytes, err := humanize.ParseBytes(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.MaximumSizeBytes = bytes
			}
		case "min-disks":
			if numDisks, err := strconv.ParseInt(value, 10, 32); err != nil {
				return VolumeQuery{}, err
			} else {
				q.MinDisks = numDisks
			}
		case "max-disks":
			if numDisks, err := strconv.ParseInt(value, 10, 32); err != nil {
				return VolumeQuery{}, err
			} else {
				q.MaxDisks = numDisks
			}
		case "dynamic-mounts":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.DynamicMounts = b
			}
		case "persist-numbering":
			if b, err := strconv.ParseBool(value); err != nil {
				return VolumeQuery{}, err
			} else {
				q.PersistNumbering = b
			}
		case "filesystem":
			q.Filesystem = value
		default:
			if strings.HasPrefix(key, "meta-") {
				metaKey := strings.TrimPrefix(key, "meta-")
				q.Metadata[metaKey] = value
			} else {
				return VolumeQuery{}, errors.New(fmt.Sprintf("Unknown query parameter found: %s", value))
			}
		}
	}

	return q, nil
}
