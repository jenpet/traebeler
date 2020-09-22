package froxlor

import (
	"errors"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestRefreshCache_shouldKeepValidEntriesAndReturnAListOfRequiredUpdates(t *testing.T) {
	cacheTests := []struct{
		name string
		initialCache []record
		domains []string
		expectedCache[]record
		expectedRequiredUpdates []record
		errorExpected bool
	}{
		{
			"cold cache requires all to be updated",
			[]record{},
			[]string{"foo.bar", "sub.jen.pet"},
			[]record{},
			[]record{{"foo.bar", "@", ""}, {"jen.pet", "sub", ""}},
			false,
		},
		{
			"incomplete cache requires some to be updated",
			[]record{{"foo.bar", "@", "127.0.0.1"}, {"jen.pet", "old", "127.0.0.1"}},
			[]string{"foo.bar", "new.foo.bar"},
			[]record{{"foo.bar", "@", "127.0.0.1"}},
			[]record{{"foo.bar", "new", ""}},
			false,
		},
		{
			"changed ip requires update",
			[]record{{"foo.bar", "@", "192.168.178.1"}},
			[]string{"foo.bar"},
			[]record{},
			[]record{{"foo.bar", "@", ""}},
			false,
		},
		{
			"malformed domains should result in error",
			[]record{{"foo.bar", "@", "192.168.178.1"}},
			[]string{"foo--"},
			[]record{{"foo.bar", "@", "192.168.178.1"}},
			[]record{},
			true,
		},
	}

	for _, tt := range cacheTests {
		t.Run(tt.name, func(t *testing.T) {
			p := Processor{cache: tt.initialCache}
			// use hardcoded ip an vary the given cache
			ru, err := p.refreshCache(tt.domains, "127.0.0.1")
			assert.Equal(t, tt.errorExpected, err != nil, "error expected: '%t' and received '%t'", tt.errorExpected, err != nil)
			assert.ElementsMatch(t, tt.expectedCache, p.cache, "expected and actual cache did not match")
			assert.ElementsMatch(t, tt.expectedRequiredUpdates, ru, "expected required updates and actual returned list did not match")
		})
	}
}

func TestUpdateRecord_whenRepositoryReturnsSingleValueHavingSameIP_shouldReturnRecord(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mr := mockRecordRepo{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "127.0.0.1"}}, nil
		},
	}
	updated, err := updateRecord(&mr, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when working on a single valid record")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated the values retrieved from the repo")
	assert.Equal(t, 1, mr.findInteractions, "expected only one find interaction")
	assert.Equal(t, 0, mr.deleteInteractions, "expected exactly one delete interaction")
	assert.Equal(t, 0, mr.addInteractions, "expected exactly one add interaction")
}

func TestUpdateRecord_whenRepositoryReturnsSingleValueHavingDifferentIP_shouldUpdateRecordInRepoAndReturnUpdated(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mr := mockRecordRepo{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "192.168.178.1"}}, nil
		},
	}
	updated, err := updateRecord(&mr, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when working on a single record and updating its value")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated when the repo contains a mismatch")
	assert.Equal(t, 1, mr.findInteractions, "expected exactly one find interaction")
	assert.Equal(t, 1, mr.deleteInteractions, "expected exactly one delete interaction")
	assert.Equal(t, 1, mr.addInteractions, "expected exactly one add interaction")
}

func TestUpdateRecord_whenRepositoryReturnsMultipleValues_shouldReturnErrorAndPerformNothing(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mr := mockRecordRepo{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "192.168.178.1"},
				{"98", "1338", "18000", "@", "A", "127.0.0.1"}}, nil
		},
	}
	updated, err := updateRecord(&mr, rec, "127.0.0.1")
	assert.NotNil(t, err, "an error should occur when multiple results are returned by the repository during lookup")
	assert.Equal(t, record{}, updated, "returned record should be blank when having multiple results during lookup")
	assert.Equal(t, 1, mr.findInteractions, "expected exactly one find interaction")
	assert.Equal(t, 0, mr.deleteInteractions, "expected exactly one delete interaction")
	assert.Equal(t, 0, mr.addInteractions, "expected exactly one add interaction")
}

func TestUpdateRecord_whenRepositoryReturnsNoValue_shouldAddRecordAndReturnUpdate(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mr := mockRecordRepo{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{}, nil
		},
	}
	updated, err := updateRecord(&mr, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when repo does not have an entry")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated when the repo contains no value at all")
	assert.Equal(t, 1, mr.findInteractions, "expected exactly one find interaction")
	assert.Equal(t, 0, mr.deleteInteractions, "expected exactly one delete interaction")
	assert.Equal(t, 1, mr.addInteractions, "expected exactly one add interaction")
}

func TestUpdateRecords_whenPartiallyFails_shouldReturnUpdatesAndErrors(t *testing.T) {
	recs := []record{{"foo.bar", "@", ""}, {"foo.bar", "sub", ""}}
	mr := mockRecordRepo {
		findMock: func(domain, record string) ([]zone, error) {
			if record == "sub" {
				return []zone{}, errors.New("Repo error")
			}
			return []zone{}, nil
		},
	}
	updates, errs := updateRecords(&mr, recs, "127.0.0.1")
	assert.Len(t, updates, 1, "at least one update should succeed")
	assert.Len(t, errs, 1, "at least one update should fail")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updates[0], "at least one update should be returned")
	assert.Equal(t, 2, mr.findInteractions, "expected two find interactions")
	assert.Equal(t, 1, mr.addInteractions, "expected exactly one add interaction")
}

func TestProcess_shouldEventuallyUpdateRepositoryAndCache(t *testing.T) {
	mr := mockRecordRepo{}
	p := Processor{}
	recs := []record{{"foo.bar", "@", "127.0.0.1"}, {"foo.bar", "sub", "127.0.0.1"}}
	assert.Nil(t, p.Init(), "initializing the processor should not result in an error")
	// actively overwrite the repository to not use traefik
	p.repo = &mr
	p.ip = mockIpProvider{}

	p.Process([]string{"foo.bar", "sub.foo.bar"})
	assert.Equal(t, 2, mr.findInteractions, "expected two find interactions")
	assert.Equal(t, 2, mr.addInteractions, "expected two add interactions")
	assert.ElementsMatch(t, p.cache, recs, "expected elements in cache are invalid")
}

func TestID_shouldReturnFroxlorProcessorID(t *testing.T) {
	assert.Equal(t, "froxlor", (&Processor{}).ID(), "processor ID does not match")
}

func TestProcess_whenIPLookupFails_shouldNotTriggerAnyInteraction(t *testing.T) {
	mr := mockRecordRepo{}
	p := Processor{
		ip: mockIpProvider{mockv4: func() (string, error) {
			return "", errors.New("ip lookup error")
		}},
		repo: &mr,
	}
	p.Process([]string{"foo.bar", "sub.foo.bar"})
	assert.Equal(t, 0, mr.findInteractions, "expected no find interactions")
}

func TestProcess_whenDomainsAreInvalid_shouldNotTriggerAnyInteraction(t *testing.T) {
	mr := mockRecordRepo{}
	p := Processor{
		ip: mockIpProvider{},
		repo: &mr,
	}
	p.Process([]string{"foo.bar", "sub--bar"})
	assert.Equal(t, 0, mr.findInteractions, "expected no find interactions")
}

type mockRecordRepo struct {
	findMock func(domain, record string) ([]zone, error)
	findInteractions int
	addMock func(domain, record, content, ttl, rtype string) error
	addInteractions int
	deleteMock func(domain, entryID string) error
	deleteInteractions int
}

func (mrr *mockRecordRepo) find(domain, record string) ([]zone, error) {
	mrr.findInteractions++
	if mrr.findMock != nil {
		return mrr.findMock(domain,record)
	}
	return []zone{}, nil
}

func (mrr *mockRecordRepo) add(domain, record, content, ttl, rtype string) error {
	mrr.addInteractions++
	if mrr.addMock != nil {
		return mrr.addMock(domain, record, content, ttl, rtype)
	}
	return nil
}

func (mrr *mockRecordRepo) delete(domain, entryID string) error {
	mrr.deleteInteractions++
	if mrr.deleteMock != nil {
		return mrr.deleteMock(domain, entryID)
	}
	return nil
}

type mockIpProvider struct {
	mockv4 func() (string,error)
}

func (mip mockIpProvider) ipv4() (string, error) {
	if mip.mockv4 != nil {
		return mip.mockv4()
	}
	return "127.0.0.1", nil
}