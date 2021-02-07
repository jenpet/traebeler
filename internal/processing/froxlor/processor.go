// Package froxlor provides functionality to interact with the Froxlor "REST" API. All functions provided work in the context of a customer.
// Therefore the configuration (key, secret) used to initialize this package can only be used with customer credentials.
//
// Administrative interactions are not available at this point in time.
package froxlor

import (
	"errors"
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
	api   froxlorHandler
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
	updates, errs := updateRecords(p.api, recs, ip)
	p.cache = append(p.cache, updates...)
	if len(errs) > 0 {
		log.Errorf("Multiple (%d) errors occurred during record update. Errors: '%+v'", len(errs), errs)
	}
}

// updateRecords performs multiple async calls towards a record repository to update a given set of records.
// The returned records array hold the successfully updated records, the errors array potential errors which
// occurred in one of the updates.
func updateRecords(fh froxlorHandler, recs []record, ip string) ([]record, []error) {
	var wg sync.WaitGroup
	var errs []error
	var updates []record
	for _, rec := range recs {
		wg.Add(1)
		rec := rec
		go func() {
			defer wg.Done()
			if err := ensureDomainExistence(fh, rec); err != nil {
				log.Errorf("Failed ensuring domain existence of record '%s'. Error: %v", rec.fqn(), err)
				errs = append(errs, err)
				return
			}
			update, err := updateRecord(fh, rec, ip)
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
func updateRecord(rh recordHandler, rec record, ip string) (record, error) {
	zones, err := rh.findDomainZones(rec.tld, rec.subdomain)
	if err != nil {
		return record{}, err
	}

	// multiple entries of a record indicate that something went wrong in an earlier update process
	if len(zones) > 1 {
		log.Errorf("Looked up more than one result regarding records for domain '%s'. Please verify manually. Records: '%+v'", rec.fqn(), zones)
		return record{}, fmt.Errorf("multiple lookup results for existing record entries for domain '%s'", rec.fqn())
	}

	// if there is an existing entry check the content and deleteDomainZone if it its outdated / not matching
	if len(zones) == 1 {
		// same ip as the given one
		entry := zones[0]
		if entry.Content == ip {
			rec.ip = ip
			log.Infof("Present record %+v matches ip of entry with ID '%s' and domain ID '%s' for domain '%s'. No update required.", rec, entry.ID, entry.DomainID, rec.fqn())
			return rec, nil
		}

		// ip in the api is different than the passed one so deleteDomainZone the api entry and create a new one
		err = rh.deleteDomainZone(rec.tld, entry.ID)
		if err != nil {
			log.Errorf("Failed to deleteDomainZone record entry with ID '%s' for domain '%s'. Error: %s", entry.ID, rec.fqn(), err)
			return record{}, err
		}
	}

	err = rh.addDomainZone(rec.tld, rec.subdomain, ip, fmt.Sprintf("%d", recordTTL), "A")
	if err != nil {
		log.Errorf("Failed to addDomainZone record for domain '%s' with ip '%s'. Error: %s", rec.fqn(), ip, err)
		return record{}, err
	}
	log.Infof("Updated ip for domain '%s' to '%s' in repository.", rec.fqn(), ip)
	rec.ip = ip
	return rec, nil
}

// ensureDomainExistence ensures that a record exists in within froxlor for the customer.
//
// Since traebeler only operates on the behalf of a customer we can just ensure subdomains.
// A missing "main" domain has to be registered by an admin manually and will result in an error.
func ensureDomainExistence(dh domainHandler, rec record) error {
	exists, err := dh.domainExists(rec.fqn())
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if !rec.hasSubdomain() {
		return errors.New("record does not have a subdomain that can be used for creation")
	}
	return dh.addDomain(rec.tld, rec.subdomain)
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
	p.api = froxlorApi{uri: p.cfg.URI, key: p.cfg.Key, secret: p.cfg.Secret, action: http.Post}
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
			// if nothing changed addDomainZone them to the cleaned up cache
			if domain == entry.fqn() && entry.ip == ip {
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

func (r record) fqn() string {
	if !r.hasSubdomain() {
		return r.tld
	}
	return fmt.Sprintf("%s.%s", r.subdomain, r.tld)
}

func (r record) hasSubdomain() bool {
	return len(r.subdomain) > 0 && r.subdomain != "@"
}

type froxlorHandler interface {
	recordHandler
	domainHandler
}

type recordHandler interface {
	findDomainZones(domain, record string) ([]zone, error)
	addDomainZone(domain, record, content, ttl, rtype string) error
	deleteDomainZone(domain, entryID string) error
}

type domainHandler interface {
	domainExists(fqn string) (bool, error)
	addDomain(domain, subdomain string) error
}

type ipProvider interface {
	ipv4() (string, error)
}

type config struct {
	URI string
	Key string
	Secret string
}