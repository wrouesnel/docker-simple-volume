package volumelabel

import (
	. "gopkg.in/check.v1"
	"testing"
	"math"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParserSuite struct{}

var _ = Suite(&ParserSuite{})

func (this *ParserSuite) TestRoundTrip(c *C) {
	// Test struct with all datatypes
	type S struct {
		Test1  int    `volumelabel:"test-int"`
		Test2  int8   `volumelabel:"test-int8"`
		Test3  int16  `volumelabel:"test-int16"`
		Test4  int32  `volumelabel:"test-int32"`
		Test5  int64  `volumelabel:"test-int64"`
		Test6  uint   `volumelabel:"test-uint"`
		Test7  uint8  `volumelabel:"test-uint8"`
		Test8  uint16 `volumelabel:"test-uint16"`
		Test9  uint32 `volumelabel:"test-uint32"`
		Test10 uint64 `volumelabel:"test-uint64"`
		Test11 string `volumelabel:"test-str"`
		Test12 bool   `volumelabel:"test-bool"`
	}

	// Do a 0 roundtrip
	var outbound string
	var err error
	var sout S
	var sin S

	sout = S{}
	outbound, err = MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("Zeroed Value: %s", outbound)

	sin = S{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	// Do structs match? (note: no DeepEquals because we only do "simple" structs)
	c.Check(sout, Equals, sin)

	// Test with maxed data types
	sout = S{
		Test1: int(^uint(0) >> 1),
		Test2: math.MaxInt8,
		Test3: math.MaxInt16,
		Test4: math.MaxInt32,
		Test5: math.MaxInt64,
		Test6: ^uint(0),
		Test7: math.MaxUint8,
		Test8: math.MaxUint16,
		Test9: math.MaxUint32,
		Test10: math.MaxUint64,

		Test11: "abcdefghijklmonpqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ---------",
		Test12: true,
	}

	sout = S{}
	outbound, err = MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("Maxed Value: %s", outbound)

	sin = S{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	// Do structs match? (note: no DeepEquals because we only do "simple" structs)
	c.Check(sout, Equals, sin)
}
