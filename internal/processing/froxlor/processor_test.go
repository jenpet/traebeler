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
	mrh := mockRecordHandler{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "127.0.0.1"}}, nil
		},
	}
	updated, err := updateRecord(&mrh, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when working on a single valid record")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated the values retrieved from the api")
	assert.Equal(t, 1, mrh.findInteractions, "expected only one findDomainZones interaction")
	assert.Equal(t, 0, mrh.deleteInteractions, "expected exactly one deleteDomainZone interaction")
	assert.Equal(t, 0, mrh.addInteractions, "expected exactly one addDomainZone interaction")
}

func TestUpdateRecord_whenRepositoryReturnsSingleValueHavingDifferentIP_shouldUpdateRecordInRepoAndReturnUpdated(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mrh := mockRecordHandler{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "192.168.178.1"}}, nil
		},
	}
	updated, err := updateRecord(&mrh, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when working on a single record and updating its value")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated when the api contains a mismatch")
	assert.Equal(t, 1, mrh.findInteractions, "expected exactly one findDomainZones interaction")
	assert.Equal(t, 1, mrh.deleteInteractions, "expected exactly one deleteDomainZone interaction")
	assert.Equal(t, 1, mrh.addInteractions, "expected exactly one addDomainZone interaction")
}

func TestUpdateRecord_whenRepositoryReturnsMultipleValues_shouldReturnErrorAndPerformNothing(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mrh := mockRecordHandler{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{{"98", "1337", "18000", "@", "A", "192.168.178.1"},
				{"98", "1338", "18000", "@", "A", "127.0.0.1"}}, nil
		},
	}
	updated, err := updateRecord(&mrh, rec, "127.0.0.1")
	assert.NotNil(t, err, "an error should occur when multiple results are returned by the repository during lookup")
	assert.Equal(t, record{}, updated, "returned record should be blank when having multiple results during lookup")
	assert.Equal(t, 1, mrh.findInteractions, "expected exactly one findDomainZones interaction")
	assert.Equal(t, 0, mrh.deleteInteractions, "expected exactly one deleteDomainZone interaction")
	assert.Equal(t, 0, mrh.addInteractions, "expected exactly one addDomainZone interaction")
}

func TestUpdateRecord_whenRepositoryReturnsNoValue_shouldAddRecordAndReturnUpdate(t *testing.T) {
	rec := record{"foo.bar", "@", ""}
	mrh := mockRecordHandler{
		findMock: func(domain, record string) ([]zone, error) {
			return []zone{}, nil
		},
	}
	updated, err := updateRecord(&mrh, rec, "127.0.0.1")
	assert.Nil(t, err, "no error should occur when api does not have an entry")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updated, "record should be updated when the api contains no value at all")
	assert.Equal(t, 1, mrh.findInteractions, "expected exactly one findDomainZones interaction")
	assert.Equal(t, 0, mrh.deleteInteractions, "expected exactly one deleteDomainZone interaction")
	assert.Equal(t, 1, mrh.addInteractions, "expected exactly one addDomainZone interaction")
}

func TestUpdateRecords_whenPartiallyFails_shouldReturnUpdatesAndErrors(t *testing.T) {
	recs := []record{{"foo.bar", "@", ""}, {"foo.bar", "sub", ""}, {"example.com", "@", ""}}
	mrh := mockRecordHandler{
		findMock: func(domain, record string) ([]zone, error) {
			if record == "sub" {
				return []zone{}, errors.New("Repo error")
			}
			return []zone{}, nil
		},
	}
	mdh := mockDomainHandler{
		existsMock: func(fqn string) (bool, error) {
			// domain example.com should be identified as not present
			return fqn != "example.com", nil
		},
	}
	mfh := mockFroxlorHandler{
		mockRecordHandler: mrh,
		mockDomainHandler: mdh,
	}
	updates, errs := updateRecords(&mfh, recs, "127.0.0.1")
	assert.Len(t, updates, 1, "at least one update should succeed")
	assert.Len(t, errs, 2, "at least two updates should fail")
	assert.Equal(t, record{"foo.bar", "@", "127.0.0.1"}, updates[0], "at least one update should be returned")
	assert.Equal(t, 2, mfh.findInteractions, "expected two findDomainZones interactions")
	assert.Equal(t, 1, mfh.addInteractions, "expected exactly one addDomainZone interaction")
}

func TestProcess_shouldEventuallyUpdateRepositoryAndCache(t *testing.T) {
	mfh := mockFroxlorHandler{}
	p := Processor{}
	recs := []record{{"foo.bar", "@", "127.0.0.1"}, {"foo.bar", "sub", "127.0.0.1"}}
	assert.Nil(t, p.Init(), "initializing the processor should not result in an error")
	// actively overwrite the repository to not use traefik
	p.api = &mfh
	p.ip = mockIpProvider{}

	p.Process([]string{"foo.bar", "sub.foo.bar"})
	assert.Equal(t, 2, mfh.findInteractions, "expected two findDomainZones interactions")
	assert.Equal(t, 2, mfh.addInteractions, "expected two addDomainZone interactions")
	assert.ElementsMatch(t, p.cache, recs, "expected elements in cache are invalid")
}

func TestID_shouldReturnFroxlorProcessorID(t *testing.T) {
	assert.Equal(t, "froxlor", (&Processor{}).ID(), "processor ID does not match")
}

func TestProcess_whenIPLookupFails_shouldNotTriggerAnyInteraction(t *testing.T) {
	mfh := mockFroxlorHandler{}
	p := Processor{
		ip: mockIpProvider{mockv4: func() (string, error) {
			return "", errors.New("ip lookup error")
		}},
		api: &mfh,
	}
	p.Process([]string{"foo.bar", "sub.foo.bar"})
	assert.Equal(t, 0, mfh.findInteractions, "expected no findDomainZones interactions")
}

func TestProcess_whenDomainsAreInvalid_shouldNotTriggerAnyInteraction(t *testing.T) {
	mfh := mockFroxlorHandler{}
	p := Processor{
		ip:  mockIpProvider{},
		api: &mfh,
	}
	p.Process([]string{"foo.bar", "sub--bar"})
	assert.Equal(t, 0, mfh.findInteractions, "expected no findDomainZones interactions")
}

func TestEnsureDomainExistence_shouldAddMissingOnes(t *testing.T) {
	var domainExistenceTests = []struct{
		name string
		rec record
		existMock func(fqn string) (bool, error)
		addMock func() error
		expectedInteractions int
		errExpected bool
	}{
		{
			"subdomain exists",
			record{ tld: "foo.bar", subdomain: "sub"},
			func(fqn string) (bool, error) { return true, nil},
			nil,
			1,
			false,
		},
		{
			"non-existent subdomain",
			record{ tld: "foo.bar", subdomain: "missing"},
			func(fqn string) (bool, error) { return false, nil},
			func() error { return nil },
			2,
			false,
		},
		{
			"domain missing",
			record{ tld: "foo.bar", subdomain: ""},
			func(fqn string) (bool, error) { return false, nil},
			nil,
			1,
			true,
		},
		{
			"api error",
			record{ tld: "foo.bar", subdomain: ""},
			func(fqn string) (bool, error) { return false, errors.New("error")},
			nil,
			1,
			true,
		},
	}

	for _, tt := range domainExistenceTests {
		t.Run(tt.name, func(t *testing.T) {
			mdr := mockDomainHandler{
				existsMock:   tt.existMock,
				addMock:      tt.addMock,
			}
			assert.Equal(t, tt.errExpected, ensureDomainExistence(&mdr, tt.rec) != nil, "error expectation mismatch")
			assert.Equal(t, tt.expectedInteractions, mdr.interactions, "interaction amount with froxlor api not matching")
		})
	}
}

type mockRecordHandler struct {
	findMock func(domain, record string) ([]zone, error)
	findInteractions int
	addMock func(domain, record, content, ttl, rtype string) error
	addInteractions int
	deleteMock func(domain, entryID string) error
	deleteInteractions int
}

func (mrh *mockRecordHandler) findDomainZones(domain, record string) ([]zone, error) {
	mrh.findInteractions++
	if mrh.findMock != nil {
		return mrh.findMock(domain,record)
	}
	return []zone{}, nil
}

func (mrh *mockRecordHandler) addDomainZone(domain, record, content, ttl, rtype string) error {
	mrh.addInteractions++
	if mrh.addMock != nil {
		return mrh.addMock(domain, record, content, ttl, rtype)
	}
	return nil
}

func (mrh *mockRecordHandler) deleteDomainZone(domain, entryID string) error {
	mrh.deleteInteractions++
	if mrh.deleteMock != nil {
		return mrh.deleteMock(domain, entryID)
	}
	return nil
}

type mockDomainHandler struct {
	interactions int
	existsMock func(fqn string)(bool, error)
	addMock func() error
}

func (mdh *mockDomainHandler) domainExists(fqn string) (bool, error) {
	mdh.interactions++
	if mdh.existsMock != nil {
		return mdh.existsMock(fqn)
	}
	return true, nil
}

func (mdh *mockDomainHandler) addDomain(_, _ string) error {
	mdh.interactions++
	if mdh.addMock != nil {
		return mdh.addMock()
	}
	return nil
}

type mockFroxlorHandler struct {
	mockRecordHandler
	mockDomainHandler
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