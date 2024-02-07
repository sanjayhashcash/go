//lint:file-ignore U1001 Ignore all unused code, staticcheck doesn't understand testify/suite

package ingest

import (
	"context"
	"io"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/hcnet/go/ingest"
	"github.com/hcnet/go/ingest/ledgerbackend"
	"github.com/hcnet/go/services/aurora/internal/test"
	"github.com/hcnet/go/xdr"
	"github.com/stretchr/testify/suite"
)

type memoryChangeReader xdr.LedgerEntryChanges

func loadChanges(path string) (*memoryChangeReader, error) {
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	entryChanges := xdr.LedgerEntryChanges{}
	if err := entryChanges.UnmarshalBinary(contents); err != nil {
		return nil, err
	}

	reader := memoryChangeReader(entryChanges)
	return &reader, nil
}

func (r *memoryChangeReader) Read() (ingest.Change, error) {
	entryChanges := *r
	if len(entryChanges) == 0 {
		return ingest.Change{}, io.EOF
	}

	change := entryChanges[0]
	*r = entryChanges[1:]
	return ingest.Change{
		Type: change.State.Data.Type,
		Post: change.State,
		Pre:  nil,
	}, nil
}

func (r *memoryChangeReader) Close() error {
	return nil
}

func TestDBTestSuite(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}

type DBTestSuite struct {
	suite.Suite
	ctx            context.Context
	sampleFile     string
	sequence       uint32
	ledgerBackend  *ledgerbackend.MockDatabaseBackend
	historyAdapter *mockHistoryArchiveAdapter
	system         *system
	tt             *test.T
}

func (s *DBTestSuite) SetupTest() {
	s.tt = test.Start(s.T())
	test.ResetAuroraDB(s.T(), s.tt.AuroraDB)

	// sample-changes.xdr is generated by sample_changes_test.go and is checked into
	// the testdata directory. To regenerate the file run:
	// go test -v -timeout 5m --tags=update  github.com/hcnet/go/services/aurora/internal/ingest -run "^(TestUpdateSampleChanges)$"
	// and commit the new file to the git repo.
	s.sampleFile = filepath.Join("testdata", "sample-changes.xdr")

	s.ledgerBackend = &ledgerbackend.MockDatabaseBackend{}
	s.historyAdapter = &mockHistoryArchiveAdapter{}
	var err error
	sIface, err := NewSystem(Config{
		HistorySession:           s.tt.AuroraSession(),
		HistoryArchiveURLs:       []string{"http://ignore.test"},
		DisableStateVerification: false,
		CheckpointFrequency:      64,
	})
	s.Assert().NoError(err)
	s.system = sIface.(*system)
	s.ctx = s.system.ctx

	s.sequence = uint32(28660351)
	s.setupMocksForBuildState()

	s.system.historyAdapter = s.historyAdapter
	s.system.ledgerBackend = s.ledgerBackend
	s.system.runner.SetHistoryAdapter(s.historyAdapter)
}

func (s *DBTestSuite) mockChangeReader() {
	changeReader, err := loadChanges(s.sampleFile)
	s.Assert().NoError(err)
	s.historyAdapter.On("GetState", s.ctx, s.sequence).
		Return(ingest.ChangeReader(changeReader), nil).Once()
}
func (s *DBTestSuite) setupMocksForBuildState() {
	checkpointHash := xdr.Hash{1, 2, 3}
	s.historyAdapter.On("GetLatestLedgerSequence").
		Return(s.sequence, nil).Once()
	s.mockChangeReader()
	s.historyAdapter.On("BucketListHash", s.sequence).
		Return(checkpointHash, nil).Once()

	s.ledgerBackend.On("IsPrepared", s.ctx, ledgerbackend.UnboundedRange(s.sequence)).Return(true, nil).Once()
	s.ledgerBackend.On("GetLedger", s.ctx, s.sequence).
		Return(
			xdr.LedgerCloseMeta{
				V0: &xdr.LedgerCloseMetaV0{
					LedgerHeader: xdr.LedgerHeaderHistoryEntry{
						Header: xdr.LedgerHeader{
							LedgerSeq:      xdr.Uint32(s.sequence),
							BucketListHash: checkpointHash,
						},
					},
				},
			},
			nil,
		).Once()
}

func (s *DBTestSuite) TearDownTest() {
	t := s.T()
	s.historyAdapter.AssertExpectations(t)
	s.ledgerBackend.AssertExpectations(t)
	s.tt.Finish()
}

func (s *DBTestSuite) TestBuildState() {
	next, err := startState{}.run(s.system)
	s.Assert().NoError(err)
	build := next.node.(buildState)
	s.Assert().Equal(s.sequence, build.checkpointLedger)

	next, err = build.run(s.system)
	s.Assert().NoError(err)
	resume := next.node.(resumeState)
	s.Assert().Equal(s.sequence, resume.latestSuccessfullyProcessedLedger)

	s.mockChangeReader()
	s.Assert().NoError(s.system.verifyState(false))
}

func (s *DBTestSuite) TestVersionMismatchTriggersRebuild() {
	s.TestBuildState()

	s.Assert().NoError(
		s.system.historyQ.UpdateIngestVersion(context.Background(), CurrentVersion-1),
	)

	s.setupMocksForBuildState()
	s.TestBuildState()
}
