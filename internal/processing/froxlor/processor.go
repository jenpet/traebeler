package froxlor

import (
	"fmt"
	"github.com/bobesa/go-domain-util/domainutil"
	"github.com/jenpet/traebeler/internal/log"
	"github.com/kelseyhightower/envconfig"
	"net/http"
	"sync"
)

// time to live of entries in the repository
const recordTTL = 18000

// Processor which can process domains for Froxlor.
type Processor struct {
	cfg   config
	repo  recordRepo
	cache []record
	ip    ipProvider
}

func (p *Processor) Process(domains []string) {
	log.Infof("Froxlor processor received domains %d (%v)", len(domains), domains)
	ip, err := p.ip.ipv4()
	if err != nil {
		log.Errorf("Failed to get IP v4 address from provider. Error: %v", err)
		return
	}
	requiredUpdates, err := p.refreshCache(domains, ip)
	if err != nil {
		log.Errorf("Failed to update cache based on domains. Error: %v", err)
		return
	}
	log.Infof("Identified %d records which require an update", len(requiredUpdates))
	p.updateRecordsAndCache(requiredUpdates, ip)
}


func (p *Processor) updateRecordsAndCache(recs []record, ip string) {
	updates, errs := updateRecords(p.repo, recs, ip)
	p.cache = append(p.cache, updates...)
	if len(errs) > 0 {
		log.Errorf("Multiple (%d) errors occurred during record update. Errors: '%+v'", len(errs), errs)
	}
}

// updateRecords performs multiple async calls towards a record repository to update a given set of records.
// The returned records array hold the successfully updated records, the errors array potential errors which
// occurred in one of the updates.
func updateRecords(repo recordRepo, recs []record, ip string) ([]record, []error) {
	var wg sync.WaitGroup
	var errs []error
	var updates []record
	for _, rec := range recs {
		wg.Add(1)
		rec := rec
		go func() {
			defer wg.Done()
			update, err := updateRecord(repo, rec, ip)
			if err != nil {
				errs = append(errs, err)
				return
			}
			updates = append(updates, update)
		}()
	}
	wg.Wait()
	return updates, errs
}

// updateRecord updates a given record in a record repository with a new ip in case it is differing.
// The entry will be looked up first assuming that there will only be one or none result at all. In case the lookup resulted in two
// records updateRecord will return an error.
// For a single result the record is deleted first and then re-added. For no result only the addition will be invoked.
// After a successful update in the repository the new record will be returned which can be used for caching.
//
// In case of any error during repository interactions the functions exits leaving a "dirty" state in the repository and returning an error.
func updateRecord(repo recordRepo, rec record, ip string) (record, error) {
	zones, err := repo.find(rec.tld, rec.subdomain)
	if err != nil {
		return record{}, err
	}

	// multiple entries of a record indicate that something went wrong in an earlier update process
	if len(zones) > 1 {
		log.Errorf("Looked up more than one result regarding records for domain '%s'. Please verify manually. Records: '%+v'", rec.name(), zones)
		return record{}, fmt.Errorf("multiple lookup results for existing record entries for domain '%s'", rec.name())
	}

	// if there is an existing entry check the content and delete if it its outdated / not matching
	if len(zones) == 1 {
		// same ip as the given one
		entry := zones[0]
		if entry.Content == ip {
			rec.ip = ip
			log.Infof("Present record %+v matches ip of entry with ID '%s' and domain ID '%s' for domain '%s'. No update required.", rec, entry.ID, entry.DomainID, rec.name())
			return rec, nil
		}

		// ip in the repo is different than the passed one so delete the repo entry and create a new one
		err = repo.delete(rec.tld, entry.ID)
		if err != nil {
			log.Errorf("Failed to delete record entry with ID '%s' for domain '%s'. Error: %s", entry.ID, rec.name(), err)
			return record{}, err
		}
	}

	err = repo.add(rec.tld, rec.subdomain, ip, fmt.Sprintf("%d", recordTTL), "A")
	if err != nil {
		log.Errorf("Failed to add record for domain '%s' with ip '%s'. Error: %s", rec.name(), ip, err)
		return record{}, err
	}
	log.Infof("Updated ip for domain '%s' to '%s' in repository.", rec.name(), ip)
	rec.ip = ip
	return rec, nil
}

// ID returns the identifier for the froxlor processor
func (p *Processor) ID() string {
	return "froxlor"
}

func (p *Processor) Init() error {
	err := envconfig.Process("traebeler_processor_froxlor", &p.cfg)
	if err != nil {
		return err
	}
	if p.cache == nil {
		p.cache = []record{}
	}
	p.repo = froxlorApi{uri: p.cfg.URI, key: p.cfg.Key, secret: p.cfg.Secret, action: http.Post}
	p.ip = ipifyApi{}
	return nil
}

// drops old cache entries which are not part of the domains array and returns new records which where not in the
// cache or require an update since the ip changed
func (p *Processor) refreshCache(domains []string, ip string) ([]record, error) {
	cleanedCache := []record{}
	updateRequired := []record{}

	// check every new incoming domain
	for _, domain := range domains {
		rec, err := domainToRecord(domain)
		if err != nil {
			log.Errorf("Failed converting domain string to record. Error %+v", err)
			return []record{}, err
		}
		requiresUpdate := true
		// search for the entry in the cache and whether the ip changed
		for _, entry := range p.cache {
			// if nothing changed add them to the cleaned up cache
			if domain == entry.name() && entry.ip == ip {
				cleanedCache = append(cleanedCache, entry)
				requiresUpdate = false
				break
			}
		}
		// requiresUpdate is true when the ip in the cache differed from the ip parameter or if there is no entry
		// for this domain (converted record) in the cache at all.
		if requiresUpdate {
			updateRequired = append(updateRequired, *rec)
		}
	}
	p.cache = cleanedCache
	return updateRequired, nil
}

func domainToRecord(domain string) (*record, error) {
	tld := domainutil.Domain(domain)

	// if there is no TLD assume that the url is malformed
	if tld == "" {
		return nil, fmt.Errorf("domain '%s' is malformed", domain)
	}

	r := record{
		tld:       tld,
		subdomain: "@",
		ip:        "",
	}

	if subdomain := domainutil.Subdomain(domain); subdomain != "" {
		r.subdomain = subdomain
	}
	return &r, nil
}

type record struct {
	tld       string
	subdomain string
	ip        string
}

func (r record) name() string {
	if r.subdomain == "@" || r.subdomain == "" {
		return r.tld
	}
	return fmt.Sprintf("%s.%s", r.subdomain, r.tld)
}

type recordRepo interface {
	find(domain, record string) ([]zone, error)
	add(domain, record, content, ttl, rtype string) error
	delete(domain, entryID string) error
}

type ipProvider interface {
	ipv4() (string, error)
}

type config struct {
	URI string
	Key string
	Secret string
}