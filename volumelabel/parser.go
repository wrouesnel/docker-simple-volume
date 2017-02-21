/*
	the simple volume name query parser. Implements a struct encoder/decoder
	pattern for easy extension.

	The volumelabel parser is designed to be human friendly, so some
	go-humanizing happens automaticaly on Unmarshalling meaning it is not a
	perfect lens. Tests should do a 3-loop encode to check equivalence.
*/

package volumelabel

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

const (
	ParserFieldSep = "_"
	ParserKVSep    = "."
)

const StructTag = "volumelabel"

// TODO: allow human-readable specifiers i.e. "bytes" as extra tag info

// Structs which want to marshal/unmarshal volumelabels should implement
// this interface
//type VolumeLabelMarshaller interface {
//	// Marshals the given struct out to a string representing a compatible
//	// docker volume label
//	MarshalVolumeLabel() (string, error)
//	// Unmarshals the given struct from a compatible docker volume label
//	// spec.
//	UnmarshalVolumeLabel(l string) error
//}

var volumeFieldRegex *regexp.Regexp

func init() {
	// Precompile the volume field regex
	var err error
	volumeFieldRegex, err = regexp.Compile("[a-zA-Z0-9][a-zA-Z0-9-]")
	if err != nil {
		panic(fmt.Sprintf("BUG! %v", err))
	}
}

// Checks if a given string is a valid volume label key is valid
// Keys may not blank.
func VolumeFieldKeyValid(v string) bool {
	return volumeFieldRegex.MatchString(v)
}

// Checks if a given string is a valid volume label key is valid
// Values may be blank.
func VolumeFieldValueValid(v string) bool {
	if v == "" {
		return true
	}
	return volumeFieldRegex.MatchString(v)
}

// Marshals a value to a volume-field compatible string
func marshalType(v interface{}) (string, error) {
	switch v.(type) {
	case bool:
		if v.(bool) {
			return "true", nil
		} else {
			return "false", nil
		}
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%v", v), nil
	case string:
		s := v.(string)
		if VolumeFieldValueValid(v.(string)) {
			return s, nil
		}
		return "", fmt.Errorf("value does not parse field regex: %v", s)
	//case fmt.Stringer:
	//	s := v.(fmt.Stringer).String()
	//	if VolumeFieldValid(s) {
	//		return s, nil
	//	}
	//	return "", fmt.Errorf("value does not parse field regex: %v", s)
	default:
		return "", fmt.Errorf("value is not a marshallable type: %T", v)
	}
}

// Unmarshals v into the pointer provided by t
func unmarshalType(v string, t interface{}) error {
	switch t.(type) {
	case *bool:
		r, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*bool) = r
	case *int:
		r, err := strconv.ParseInt(v, 10, 0)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*int) = int(r)
	case *int8:
		r, err := strconv.ParseInt(v, 10, 8)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*int8) = int8(r)
	case *int16:
		r, err := strconv.ParseInt(v, 10, 16)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*int16) = int16(r)
	case *int32:
		r, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*int32) = int32(r)
	case *int64:
		r, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*int64) = int64(r)
	case *uint:
		r, err := strconv.ParseUint(v, 10, 0)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*uint) = uint(r)
	case *uint8:
		r, err := strconv.ParseUint(v, 10, 8)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*uint8) = uint8(r)
	case *uint16:
		r, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*uint16) = uint16(r)
	case *uint32:
		r, err := strconv.ParseUint(v, 10, 32)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*uint32) = uint32(r)
	case *uint64:
		r, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			return fmt.Errorf("could not unmarshal: %v %v", v, err.Error())
		}
		*t.(*uint64) = uint64(r)
	case *string:
		if !VolumeFieldValueValid(v) {
			return fmt.Errorf("value does not parse field regex: %v", v)
		}
		*t.(*string) = v
	}
	return nil
}

// MarshalVolumeLabel reads StructTag fields from a struct and turns them into
// a docker volume label compatible string accordng to our field style.
//
// Only integers, booleans and string types are supported.
//
// The encoder is domain constrained to valid docker local volume labels. As a
// result strings are filtered for non-conforming values and will raise an error
// if found.
func MarshalVolumeLabel(v interface{}) (string, error) {
	vtype := reflect.TypeOf(v)
	vvalue := reflect.ValueOf(v)

	if vvalue.Kind() != reflect.Struct {
		return "", fmt.Errorf("cannot marshal non-struct type")
	}

	keyValues := []string{}

	for i := 0; i < vtype.NumField(); i++ {
		keyName := vtype.Field(i).Tag.Get(StructTag)
		if keyName == "" {
			// Not a key member
			continue
		}

		if !VolumeFieldKeyValid(keyName) {
			return "", fmt.Errorf("key name does not parse field regex: %v", keyName)
		}

		// Key name is valid. Get the value.
		value, err := marshalType(vvalue.Field(i).Interface())
		if err != nil {
			return "", err
		}

		// Have key and value, join and append to string
		keyValues = append(keyValues, fmt.Sprintf("%s%s%s", keyName, ParserKVSep, value))
	}

	return strings.Join(keyValues, ParserFieldSep), nil
}

func UnmarshalVolumeLabel(l string, v interface{}) error {
	if v == nil {
		return fmt.Errorf("unmarshal target must be a non-nil struct pointer")
	}

	value := reflect.ValueOf(v)
	if value.Type().Kind() != reflect.Ptr {
		return fmt.Errorf("unmarshal target must be a non-nil struct pointer")
	}

	if value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("unmarshal target must be a non-nil struct pointer")
	}

	// Prechecks passed. Start expanding values into the raw map
	keyValues := strings.Split(l, ParserFieldSep)
	rawValues := make(map[string]string, len(keyValues))

	for _, kv := range keyValues {
		kvTuple := strings.Split(kv, ParserKVSep)
		if len(kvTuple) == 2 {
			rawValues[kvTuple[0]] = kvTuple[1]
		} else {
			rawValues[kvTuple[0]] = ""
		}
	}

	// Scan the struct and try and unmarshal matching keys
	for i := 0; i < value.Elem().NumField(); i++ {
		keyName := value.Type().Elem().Field(i).Tag.Get(StructTag)
		// TODO: should we recognize "-" and just ignore it?
		if keyName == "" {
			continue
		}
		// Print something helpful if the struct could never unmarshal
		if !VolumeFieldKeyValid(keyName) {
			return fmt.Errorf("key name does not parse field regex: %v %v", keyName, value.Type().Field(i).PkgPath)
		}

		// Okay, do we have this keyname?
		if rawstr, found := rawValues[keyName]; found {
			// Yes. Let's try and unmarshal it as the type
			if !value.Elem().Field(i).CanAddr() {
				return fmt.Errorf("key cannot be addressed and will never be unmarshalled: %v %v", keyName, value.Type().Field(i).PkgPath)
			}
			// Get a pointer to the field in the target struct
			target := value.Elem().Field(i).Addr().Interface()
			// Unmarshal straight into it
			err := unmarshalType(rawstr, target)
			if err != nil {
				return fmt.Errorf("Error while unmarshalling %v : %v : %v", keyName, rawstr, err)
			}
		}
		// TODO: how to handle unspecified fields (i.e. meta- vals)?
	}
	return nil
}

// Parses a docker volume name i.e "label.some_value" into an expanded
// volume query structure.
//func UnmarshalVolumeLabel(volumeName string, v interface{}) error {
//
//	q := volumequery.VolumeQuery{}
//	// Get fields
//	fields := strings.Split(volumeName, ParserFieldSep)
//
//	for _, field := range fields {
//		// Split field into key value
//		kv := strings.SplitN(field, ParserKVSep, 1)
//
//		key := kv[0]
//		value := kv[1]
//
//		switch key {
//		case "label":
//			q.Label = value
//		case "own-hostname":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.OwnHostname = b
//			}
//		case "own-machine-id":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.OwnMachineId = b
//			}
//		case "initialized":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.Initialized = b
//			}
//		case "basename":
//			q.Basename = value
//		case "naming-style":
//			switch value {
//			case volumequery.NamingNumeric, volumequery.NamingUUID:
//				q.NamingStyle = volumequery.NamingType(value)
//			default:
//				return volumequery.VolumeQuery{}, errors.New("Not a valid naming style")
//			}
//		case "exclusive":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.Exclusive = b
//			}
//		case "min-size":
//			if bytes, err := humanize.ParseBytes(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.MinimumSizeBytes = bytes
//			}
//		case "max-size":
//			if bytes, err := humanize.ParseBytes(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.MaximumSizeBytes = bytes
//			}
//		case "min-disks":
//			if numDisks, err := strconv.ParseInt(value, 10, 32); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.MinDisks = int32(numDisks)
//			}
//		case "max-disks":
//			if numDisks, err := strconv.ParseInt(value, 10, 32); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.MaxDisks = int32(numDisks)
//			}
//		case "dynamic-mounts":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.DynamicMounts = b
//			}
//		case "persist-numbering":
//			if b, err := strconv.ParseBool(value); err != nil {
//				return volumequery.VolumeQuery{}, err
//			} else {
//				q.PersistNumbering = b
//			}
//		case "filesystem":
//			q.Filesystem = value
//		default:
//			if strings.HasPrefix(key, "meta-") {
//				metaKey := strings.TrimPrefix(key, "meta-")
//				q.Metadata[metaKey] = value
//			} else {
//				return volumequery.VolumeQuery{}, errors.New(fmt.Sprintf("Unknown query parameter found: %s", value))
//			}
//		}
//	}
//
//	// Label is required.
//	if q.Label == "" {
//		return volumequery.VolumeQuery{}, errors.New("label is a required field.")
//	}
//
//	return q, nil
//}
