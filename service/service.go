package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sto "github.com/ipfsconsortium/go-ipfsc/storage"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ipfsc     *Ipfsc
	storage   *sto.Storage
	stats     ServiceStats
	laststats ServiceStats
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ipfsc *Ipfsc, storage *sto.Storage) *Service {
	return &Service{ipfsc: ipfsc, storage: storage}
}

func (s *Service) collectENS(expr, path string) {

	log.Info("Collecting[ens] " + path + ">" + expr)

	enskey, textkey, err := parseENSEntry(expr)
	if err != nil {
		log.WithError(err).Warn("Error parsing ens " + expr)
		s.stats.Errors++
		return
	}

	// Parse an ENS entry
	if textkey != "" && textkey != DefaultManifestKey {
		// an IPFS hash stored in ENS
		expr, err := s.ipfsc.ENS().Text(enskey, textkey)
		if err != nil {
			log.WithError(err).Warn("Failed to get " + expr)
			s.stats.Errors++
			return
		}
		s.collect(expr, enskey+">"+path)
	}

	// Parse manifest entry
	manifest, err := s.ipfsc.Read(expr)
	if err != nil {
		log.WithError(err).Warn("Failed to get " + expr)
		s.stats.Errors++
		return
	}

	switch v := manifest.(type) {

	case *ConsortiumManifest:
		for _, member := range v.Members {
			s.collect(member.EnsName, path+">"+member.EnsName)
		}
		return

	case *PinningManifest:
		for i, entry := range v.Pin {
			s.collect(entry, fmt.Sprintf("%v/%v(#%v)", path, expr, i))
		}

	default:
		log.Warn("Unable to parse manifest " + expr)
		s.stats.Errors++
	}

}

func parseENSEntry(expr string) (enskey, textkey string, err error) {
	if strings.Contains(expr, "[") {
		sp1 := strings.Split(expr, "[")
		if len(sp1) != 2 {
			return "", "", errors.New("Ill formed ENS name")
		}
		sp2 := strings.Split(sp1[1], "]")
		if len(sp2) != 2 || len(sp2[1]) != 0 {
			return "", "", errors.New("Ill formed ENS name")
		}
		enskey = sp2[0]
		textkey = sp1[0]
	} else {
		enskey = expr
		textkey = ""
	}
	return enskey, textkey, nil
}

func (s *Service) collectIPFS(expr, path string) {
	log.Info("Collecting[ipfs] " + path + ">" + expr)

	// if information is available in local db, use it
	hentry, _ := s.storage.Hash(expr)
	if hentry != nil && !hentry.Dirty {

		log.WithField("hash", expr).Info("Already cached")
		if hentry.Mark {
			// if already marked, has been also already recursevlly marked
			return
		}
		hentry.Mark = true
		hentry.Dirty = false

		if err := s.storage.UpdateHash(expr, hentry); err != nil {
			log.WithError(err).Warn("Failed to update hash db")
			s.stats.Errors++
			return
		}
		for _, linkhash := range hentry.Links {
			s.collectIPFS(linkhash, path+">"+expr+"()")
		}
		return
	}

	// object is not in the database, so get data from it
	start := time.Now()
	err := s.ipfsc.IPFS().Pin(expr, false)
	if err != nil {
		log.WithError(err).Warn("Unable to get object " + expr)
		s.stats.Errors++
		return
	}

	s.stats.Pinned++

	log.WithFields(log.Fields{
		"hash": expr,
		"time": time.Since(start),
	}).Info("Pinned object")

	ipfsObject, err := s.ipfsc.IPFS().ObjectGet(expr)
	if err != nil {
		log.WithError(err).Warn("Unable to get object " + expr)
		s.stats.Errors++
		return
	}

	var links []string
	if len(ipfsObject.Links) > 0 && ipfsObject.Links[0].Name != "" {
		links = make([]string, len(ipfsObject.Links))
		for i, link := range ipfsObject.Links {
			links[i] = link.Hash
			s.collectIPFS(link.Hash, path+">"+expr+"("+link.Name+")")
		}
	}

	if err = s.storage.UpdateHash(expr, &sto.HashEntry{
		DataSize: 0,
		Links:    links,
		Mark:     true,
		Dirty:    false,
	}); err != nil {
		log.WithError(err).Warn("Failed to add hash " + expr)
		s.stats.Errors++
		return
	}

	return
}

func (s *Service) collect(expr, path string) {

	s.stats.Count++

	if strings.HasPrefix(expr, "/ipfs/") {
		s.collectIPFS(expr, path)
		return
	} else if strings.HasPrefix(expr, "0x") {
		// handle contract, TODO
	} else if strings.Contains(expr, ".eth") {
		s.collectENS(expr, path)
		return
	}
	log.Warn("Unable to find resolver to sync '" + expr + "'")
	s.stats.Errors++
	return
}

func (s *Service) Sync(ensnames []string) (ServiceStats, error) {

	s.laststats = s.stats
	s.stats = ServiceStats{}

	var err error

	/* unmark all hashes */
	err = s.storage.HashUpdateIter(func(_ string, entry *sto.HashEntry) *sto.HashEntry {
		if entry.Mark {
			entry.Mark = false
			return entry
		}
		return nil
	})

	if err != nil {
		return s.stats, err
	}

	/* discover, and mark hashes that needs to be pinned */
	for _, expr := range ensnames {
		log.WithFields(log.Fields{
			"expr": expr,
		}).Info("Processing entries")

		s.collect(expr, "")
	}

	if s.stats.Errors == 0 {
		/* No errors, unpin the unused hashes and mark as deleted */
		err = s.storage.HashUpdateIter(func(hash string, entry *sto.HashEntry) *sto.HashEntry {
			if !entry.Mark {
				if err := s.ipfsc.IPFS().Unpin(hash); err != nil {
					log.WithError(err).Warn("Failed to unpin " + hash)
				}
				s.stats.Unpinned++
				entry.Dirty = true
				return entry
			}
			return nil
		})

		if err != nil {
			s.stats.Errors++
			return s.stats, err
		}

	}
	return s.stats, nil
}
