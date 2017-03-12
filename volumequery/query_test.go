/*
	Test functions for the udev API querying
*/

package volumequery

import (
	"testing"
	. "gopkg.in/check.v1"
	"strings"
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

	diskDevices, err := getDevicesByDevNode(rules)
	disks := []string{}
	for _, v := range diskDevices {
		disks = append(disks, v.Devnode())
	}
	c.Assert(err, IsNil)                 // Check it works
	c.Assert(len(disks), Not(Equals), 0) // Check we can find disks with search logic
	c.Logf("Found disks: %s", strings.Join(disks," "))
}

func (this *QueryTestSuite) TestGetDevicesByDevNode_WithComplexRules(c *C) {
	// Check that all the property fields work (don't expect to return any result
	rules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Subsystems: []string{"block"},
			Name:       []string{"sda"},
			Tag:        []string{},
			Properties: map[string]string{
				"DEVNAME": "/dev/?d[a-z]",
				"DEVTYPE": "disk",
			},
			Attrs: map[string]string{},
		},
	}

	diskDevices, err := getDevicesByDevNode(rules)
	disks := []string{}
	for _, v := range diskDevices {
		disks = append(disks, v.Devnode())
	}
	c.Assert(err, IsNil) // Check it works
	c.Assert(len(disks), Not(Equals), 0) // Check we can find disks with search logic
	c.Logf("Found disks: %s", strings.Join(disks," "))
}

func (this *QueryTestSuite) TestGetPartitionsFromDiskPath(c *C) {
	// Find something disk like and get partitions from it.
	rules := []DeviceSelectionRule{
		DeviceSelectionRule{
			Properties: map[string]string{
				"DEVNAME": "/dev/?d[a-z]",
				"DEVTYPE": "disk",
			},
		},
	}

	diskDevices, err := getDevicesByDevNode(rules)
	disks := []string{}
	for _, v := range diskDevices {
		disks = append(disks, v.Devnode())
	}
	c.Assert(err, IsNil)
	c.Assert(len(disks), Not(Equals), 0, Commentf("Need at least 1 disk to pass this check."))

	allpartitions := []string{}

	for _, disk := range disks {
		partitions, err := GetPartitionDevicesFromDiskPath(disk)
		c.Assert(err, IsNil, Commentf("Error querying %v for partitions: %v", disk, err))
		allpartitions = append(allpartitions, partitions...)
	}

	c.Assert(len(allpartitions), Not(Equals), 0, Commentf("Need at least 1 partition on a disk to pass this check."))
	c.Logf("Found partitions: %s", strings.Join(allpartitions, " "))
}