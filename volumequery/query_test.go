/*
	Test functions for the udev API querying
*/

package volumequery

import (
	"testing"

	"fmt"
	. "gopkg.in/check.v1"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type QueryTestSuite struct{}

var _ = Suite(&QueryTestSuite{})

func (this *QueryTestSuite) TestGetDevicesByDevNode_FindingDisks(c *C) {
	// Run a query with some rules which generally work

	// We should always be able to find something looking a bit disk-like in
	// our test environment.
	rules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVNAME": "/dev/?d[a-z]",
				"DEVTYPE": "disk",
			},
		},
	}

	disks, err := GetDevicesByDevNode(rules)
	c.Assert(err, IsNil)                 // Check it works
	c.Assert(len(disks), Not(Equals), 0) // Check we can find disks with search logic
	fmt.Println("Found disks:", disks)
}

func (this *QueryTestSuite) TestGetDevicesByDevNode_WithComplexRules(c *C) {
	// Check that all the property fields work (don't expect to return any result
	rules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Subsystems: []string{"block"},
			Name:       []string{"sda"},
			Tag:        []string{"test"},
			Properties: map[string]string{
				"DEVNAME": "/dev/?d[a-z]",
				"DEVTYPE": "disk",
			},
			Attrs: map[string]string{
				"DEVNAME": "/dev/?d[a-z]",
				"DEVTYPE": "disk",
			},
		},
	}

	_, err := GetDevicesByDevNode(rules)
	c.Assert(err, IsNil) // Check it works
}
