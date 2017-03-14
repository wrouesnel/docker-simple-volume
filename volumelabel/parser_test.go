package volumelabel

import (
	. "gopkg.in/check.v1"
	"math"
	"regexp"
	"testing"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParserSuite struct{}

var _ = Suite(&ParserSuite{})

const MaxUint = ^uint(0)
const MaxInt = int(MaxUint >> 1)
const MinInt = -MaxInt - 1

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

// Test struct with pointers of all datatypes
type PtrS struct {
	Test1  *int    `volumelabel:"ptrtest-int"`
	Test2  *int8   `volumelabel:"ptrtest-int8"`
	Test3  *int16  `volumelabel:"ptrtest-int16"`
	Test4  *int32  `volumelabel:"ptrtest-int32"`
	Test5  *int64  `volumelabel:"ptrtest-int64"`
	Test6  *uint   `volumelabel:"ptrtest-uint"`
	Test7  *uint8  `volumelabel:"ptrtest-uint8"`
	Test8  *uint16 `volumelabel:"ptrtest-uint16"`
	Test9  *uint32 `volumelabel:"ptrtest-uint32"`
	Test10 *uint64 `volumelabel:"ptrtest-uint64"`
	Test11 *string `volumelabel:"ptrtest-str"`
	Test12 *bool   `volumelabel:"ptrtest-bool"`
}

// Make a test struct which points to actual values.
func newInitializedPtrS() *PtrS {
	return &PtrS{
		Test1:  new(int),
		Test2:  new(int8),
		Test3:  new(int16),
		Test4:  new(int32),
		Test5:  new(int64),
		Test6:  new(uint),
		Test7:  new(uint8),
		Test8:  new(uint16),
		Test9:  new(uint32),
		Test10: new(uint64),
		Test11: new(string),
		Test12: new(bool),
	}
}

const SUnparseable string = "test-int.notInt_test-int8.notInt8_test-int16.notInt16_test-int32.notInt32_test-int64.notInt64_test-uint.notUint_test-uint8.notUint8_test-uint16.notUint16_test-uint32.notUint32_test-uint64.notUint64_test-str.Not$Parseable_test-bool.NotDistinctlyTrue"

func (this *ParserSuite) TestRoundTripWithZeroValues(c *C) {
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
}

func (this *ParserSuite) TestRoundTripWithNilPointerValues(c *C) {
	// Do a nil roundtrip
	var outbound string
	var err error
	var sout PtrS
	var sin PtrS

	sout = PtrS{}
	outbound, err = MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("Nil'd Value: %s", outbound)

	sin = PtrS{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	c.Check(sout, Equals, sin)
}

func (this *ParserSuite) TestRoundTripWithInitializedPointerValues(c *C) {
	// Do a nil roundtrip
	var outbound string
	var err error
	var sout PtrS
	var sin PtrS

	sout = *newInitializedPtrS()
	outbound, err = MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("initialized Values: %s", outbound)

	sin = PtrS{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	// Do structs match? (note: no DeepEquals because we only do "simple" structs)
	c.Check(*sout.Test1, Equals, *sin.Test1)
	c.Check(*sout.Test2, Equals, *sin.Test2)
	c.Check(*sout.Test3, Equals, *sin.Test3)
	c.Check(*sout.Test4, Equals, *sin.Test4)
	c.Check(*sout.Test5, Equals, *sin.Test5)
	c.Check(*sout.Test6, Equals, *sin.Test6)
	c.Check(*sout.Test7, Equals, *sin.Test7)
	c.Check(*sout.Test8, Equals, *sin.Test8)
	c.Check(*sout.Test9, Equals, *sin.Test9)
	c.Check(*sout.Test10, Equals, *sin.Test10)
	c.Check(*sout.Test11, Equals, *sin.Test11)
	c.Check(*sout.Test12, Equals, *sin.Test12)
}

func (this *ParserSuite) TestRoundTripWithMaxValues(c *C) {
	// Test with maxed data types
	sout := S{
		Test1:  MaxInt,
		Test2:  math.MaxInt8,
		Test3:  math.MaxInt16,
		Test4:  math.MaxInt32,
		Test5:  math.MaxInt64,
		Test6:  MaxUint,
		Test7:  math.MaxUint8,
		Test8:  math.MaxUint16,
		Test9:  math.MaxUint32,
		Test10: math.MaxUint64,

		Test11: "abcdefghijklmonpqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ---------",
		Test12: true,
	}

	outbound, err := MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("Maxed Value: %s", outbound)

	sin := S{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	// Do structs match? (note: no DeepEquals because we only do "simple" structs)
	c.Check(sout, Equals, sin)
}

func (this *ParserSuite) TestRoundTripWithMinValues(c *C) {
	// Test with maxed data types
	sout := S{
		Test1:  MinInt,
		Test2:  math.MinInt8,
		Test3:  math.MinInt16,
		Test4:  math.MinInt32,
		Test5:  math.MinInt64,
		Test6:  0,
		Test7:  0,
		Test8:  0,
		Test9:  0,
		Test10: 0,
		Test11: "",
		Test12: false,
	}

	outbound, err := MarshalVolumeLabel(sout)
	c.Check(err, IsNil)

	c.Logf("Minimum Value: %s", outbound)

	sin := S{}
	err = UnmarshalVolumeLabel(outbound, &sin)
	c.Check(err, IsNil)

	// Do structs match? (note: no DeepEquals because we only do "simple" structs)
	c.Check(sout, Equals, sin)
}

func (this *ParserSuite) TestVolumeFieldValidationFn(c *C) {
	// Check the regex actually compiles
	_, err := regexp.Compile(volumeFieldRegexExpr)
	c.Assert(err, IsNil)

	// Rules for Valid Key fields == not nil, [a-zA-Z0-9][a-zA-Z0-9-]
	// Values allow nil values.

	fns := []func(string) bool{VolumeFieldKeyValid, VolumeFieldValueValid}

	// Key valus only deviate by this at the moment
	c.Check(VolumeFieldKeyValid(""), Equals, false)

	for _, fn := range fns {
		// Some pathological cases
		c.Check(fn("-prefixhyphen"), Equals, false)
		c.Check(fn("."), Equals, false)
		c.Check(fn("afull.key"), Equals, false)
		c.Check(fn("another.full.key"), Equals, false)

		// A few cases which should pass
		c.Check(fn("justaname"), Equals, true)
		c.Check(fn("just-a-name"), Equals, true)
		c.Check(fn("just-a-trailing-hyphen-"), Equals, true)
	}
}

func (this *ParserSuite) TestMarshalTypeWithBadFieldValues(c *C) {
	needle := "this_is_not_allowed"
	out, err := marshalType(needle)
	c.Check(err, NotNil)
	// Check the fail behavior is correct.
	c.Check(out, Equals, "")
}

func (this *ParserSuite) TestMarshalTypeWithBadType(c *C) {
	needle := []string{"this_is_not_allowed"}
	out, err := marshalType(needle)
	c.Log(err)
	c.Check(err, NotNil)
	// Check the fail behavior is correct.
	c.Check(out, Equals, "")
}

func (this *ParserSuite) TestUnmarshalTypeWithBadType(c *C) {
	type testCase struct {
		t interface{}
		s string
	}

	var err error

	err = unmarshalType("not_distinctly_true", new(bool))
	c.Check(err, NotNil)

	err = unmarshalType("not_an_int", new(int))
	c.Check(err, NotNil)

	err = unmarshalType("not_int8", new(int8))
	c.Check(err, NotNil)

	err = unmarshalType("not_int16", new(int16))
	c.Check(err, NotNil)

	err = unmarshalType("not_int32", new(int32))
	c.Check(err, NotNil)

	err = unmarshalType("not_int64", new(int64))
	c.Check(err, NotNil)

	err = unmarshalType("not_uint", new(uint))
	c.Check(err, NotNil)

	err = unmarshalType("not_uint8", new(uint8))
	c.Check(err, NotNil)

	err = unmarshalType("not_uint16", new(uint16))
	c.Check(err, NotNil)

	err = unmarshalType("not_uint32", new(uint32))
	c.Check(err, NotNil)

	err = unmarshalType("not_uint64", new(uint64))
	c.Check(err, NotNil)

	err = unmarshalType("this_is_not_allowed", new(string))
	c.Check(err, NotNil)
}

func (this *ParserSuite) TestMarshalVolumeLabelFailsWithNoStruct(c *C) {
	c.Log("Test should when not a struct")
	var x interface{}
	x = nil
	out, err := MarshalVolumeLabel(x)
	c.Check(err, NotNil)
	c.Check(out, Equals, "")
}

func (this *ParserSuite) TestMarshalVolumeIgnoresUntaggedFields(c *C) {
	type s struct {
		Tagged   string `volumelabel:"tagged"`
		Untagged string
	}

	testcase := s{"this-is-the-tagged-field", "this-is-the-untagged-field"}
	expected := "tagged.this-is-the-tagged-field"

	out, err := MarshalVolumeLabel(testcase)
	c.Check(err, IsNil, Commentf("Error: %v", err))
	c.Check(out, Equals, expected)
}

func (this *ParserSuite) TestMarshalVolumeFailedOnBadKeyTag(c *C) {
	type s struct {
		Tagged string `volumelabel:"tagged_but_should_fail"`
	}

	testcase := s{"this-is-the-tagged-field"}

	out, err := MarshalVolumeLabel(testcase)
	c.Check(err, NotNil)
	c.Check(out, Equals, "")
}

func (this *ParserSuite) TestMarshalVolumeFailedOnBadStructValue(c *C) {
	type s struct {
		Tagged string `volumelabel:"tagged"`
	}

	testcase := s{"this_should_fail"}
	out, err := MarshalVolumeLabel(testcase)
	c.Check(err, NotNil)
	c.Check(out, Equals, "")
}

func (this *ParserSuite) TestUnMarshalVolumeLabelWithBadTypes(c *C) {
	var err error

	err = UnmarshalVolumeLabel("valid.label", nil)
	c.Check(err, NotNil, Commentf("Should've failed if we pass nil"))

	err = UnmarshalVolumeLabel("valid.label", struct{}{})
	c.Check(err, NotNil, Commentf("Should've failed with a non-pointer"))

	err = UnmarshalVolumeLabel("valid.label", new(int))
	c.Check(err, NotNil, Commentf("Should've failed because pointer does not point to a struct"))
}

func (this *ParserSuite) TestUnMarshalVolumeLabelWithEmptyField(c *C) {
	type s struct {
		Tagged string `volumelabel:"tagged"`
	}

	unmarshalled := s{}

	err := UnmarshalVolumeLabel("tagged", &unmarshalled)
	c.Check(err, IsNil, Commentf("Standalone key should parse okay"))
	c.Check(unmarshalled.Tagged, Equals, "")
}

func (this *ParserSuite) TestUnMarshalVolumeLabelWithEmptyKeyname(c *C) {
	type s struct {
		Tagged string `volumelabel:""`
	}

	unmarshalled := s{}

	err := UnmarshalVolumeLabel("tagged.a-value", &unmarshalled)
	c.Check(err, IsNil, Commentf("Blank key should be ignored"))
	c.Check(unmarshalled.Tagged, Equals, "")
}

func (this *ParserSuite) TestUnMarshalVolumeFailsOnBadKeyTag(c *C) {
	type s struct {
		Tagged string `volumelabel:"tagged_but_should_fail"`
	}

	testcase := s{}

	err := UnmarshalVolumeLabel("tagged_but_should_fail.a-value-we-cant-get", &testcase)
	c.Check(err, NotNil)
}

func (this *ParserSuite) TestUnMarshalVolumeFailsOnUnaddressableValue(c *C) {
	// TODO: can this *ever* happen with a struct?
}

func (this *ParserSuite) TestUnMarshalVolumeFailsOnBadValueWhileUnmarshalling(c *C) {
	testcase := S{}

	err := UnmarshalVolumeLabel(SUnparseable, &testcase)
	c.Check(err, NotNil)
}
