package volumelabel

import (
	"testing"
	. "gopkg.in/check.v1"
	"github.com/wrouesnel/docker-simple-disk/volumequery"
)

// Hook up gocheck into the "go test" runner.
func Test(t *testing.T) { TestingT(t) }

type ParserSuite struct{}

var _ = Suite(&ParserSuite{})

// TestQuery holds a label to be parsed and a VolumeQuery which should be
// returned
type TestQuery struct {
	Query string
	Result volumequery.VolumeQuery
	ShouldFail bool
}

var TestQueries []TestQuery = []{
	// Blank
	TestQuery{
		Query: "",
		Result: volumequery.VolumeQuery{},
		ShouldFail: true,
	},
	// Simple label
	TestQuery{
		Query: "label.simplelabel",
		Result: volumequery.VolumeQuery{
			Label: "simplelabel",
		},
		ShouldFail: false,
	},
	// Fully loaded label.
	TestQuery{
		Query: "label.simplelabel_own-hostname.myhostname_own-machine-id_mymachineid_initialized.true_basename_mybasename",
		Result: volumequery.VolumeQuery{
			Label: "simplelabel",
		},
		ShouldFail: false,
	},
}

func (s *ParserSuite) TestVolumeQueries(c *C) {

}