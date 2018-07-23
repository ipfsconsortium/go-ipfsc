package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sto "github.com/ipfsconsortium/gipc/storage"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ipfsc   *Ipfsc
	storage *sto.Storage
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ipfsc *Ipfsc, storage *sto.Storage) *Service {
	return &Service{ipfsc, storage}
}

func (s *Service) collectENS(expr, path string) int {

	log.Info("Collecting[ens] " + path + ">" + expr)

	enskey, textkey, err := parseENSEntry(expr)
	if err != nil {
		log.WithError(err).Warn("Error parsing ens " + expr)
		return 1
	}

	// Parse an ENS entry
	if textkey != "" && textkey != DefaultManifestKey {
		// an IPFS hash stored in ENS
		expr, err := s.ipfsc.ENS().Text(enskey, textkey)
		if err != nil {
			log.WithError(err).Warn("Failed to get " + expr)
			return 1
		}
		return s.collect(expr, enskey+">"+path)
	}

	// Parse manifest entry
	manifest, err := s.ipfsc.Read(expr)
	if err != nil {
		log.WithError(err).Warn("Failed to get " + expr)
		return 1
	}

	switch v := manifest.(type) {

	case *ConsortiumManifest:
		errors := 0
		for _, member := range v.Members {
			errors += s.collect(member.EnsName, path+">"+member.EnsName)
		}
		return errors

	case *PinningManifest:
		errors := 0
		for i, entry := range v.Pin {
			errors += s.collect(entry, fmt.Sprintf("%v/%v(#%v)", path, expr, i))
		}
		return errors

	default:
		log.Warn("Unable to parse manifest " + expr)
		return 1
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
func (s *Service) collectIPFS(expr, path string) int {
	log.Info("Collecting[ipfs] " + path + ">" + expr)

	// if information is available in local db, use it
	hentry, _ := s.storage.Hash(expr)
	if hentry != nil {
		if hentry.Mark {
			// if already marked, has been also already recursevlly marked
			return 0
		}
		hentry.Mark = true
		hentry.Dirty = false

		if err := s.storage.UpdateHash(expr, hentry); err != nil {
			log.WithError(err).Warn("Failed to update hash db")
			return 1
		}
		errors := 0
		for _, linkhash := range hentry.Links {
			errors += s.collectIPFS(linkhash, path+">"+expr+"()")
		}
		return errors
	}

	// object is not in the database, so get data from it
	start := time.Now()
	ipfsObject, err := s.ipfsc.IPFS().ObjectGet(expr)
	if err != nil {
		log.WithError(err).Warn("Unable to get object " + expr)
		return 1
	}

	log.WithFields(log.Fields{
		"#links": len(ipfsObject.Links),
	}).Info("Got Objectats in ", time.Since(start))

	var links []string
	errors := 0
	// if is a list (large file), names are empty, if not,
	//  is a folder
	if len(ipfsObject.Links) > 0 && ipfsObject.Links[0].Name != "" {
		links = make([]string, len(ipfsObject.Links))
		for i, link := range ipfsObject.Links {
			links[i] = link.Hash
			errors += s.collectIPFS(link.Hash, path+">"+expr+"("+link.Name+")")
		}
	}

	if err = s.storage.UpdateHash(expr, &sto.HashEntry{
		DataSize: 0,
		Links:    links,
		Mark:     true,
		Pinned:   false,
		Dirty:    false,
	}); err != nil {
		log.WithError(err).Warn("Failed to add hash " + expr)
		return 1 + errors
	}

	return errors
}

func (s *Service) collect(expr, path string) int {

	if strings.HasPrefix(expr, "/ipfs/") {
		return s.collectIPFS(expr, path)
	} else if strings.HasPrefix(expr, "0x") {
		// handle contract, TODO
	} else if strings.Contains(expr, ".eth") {
		return s.collectENS(expr, path)
	}
	log.Warn("Unable to find resolver to sync '" + expr + "'")
	return 1
}

func (s *Service) Sync(ensnames []string) (pinned, unpinned, errors int, err error) {

	/* unmark all hashes */
	err = s.storage.HashUpdateIter(func(_ string, entry *sto.HashEntry) *sto.HashEntry {
		if entry.Mark {
			entry.Mark = false
			return entry
		}
		return nil
	})

	if err != nil {
		return 0, 0, 0, err
	}

	/* discover, and mark hashes that needs to be pinned */
	errors = 0
	for _, expr := range ensnames {
		log.WithFields(log.Fields{
			"expr": expr,
		}).Info("Processing entries")

		errors += s.collect(expr, "")
	}

	/* Pin all hashes that has Mark=true && Pinned=false */
	err = s.storage.HashUpdateIter(func(hash string, entry *sto.HashEntry) *sto.HashEntry {
		if entry.Mark && !entry.Pinned {
			if err := s.ipfsc.IPFS().Pin(hash, false); err != nil {
				errors++
				log.WithError(err).Warn("Failed to pin " + hash)
				return nil
			} else {
				pinned++
				entry.Pinned = true
			}
			return entry
		}
		return nil
	})

	if err != nil {
		return 0, 0, 0, err
	}

	if errors == 0 {
		/* No errors, unpin the unused hashes and mark as deleted */
		err = s.storage.HashUpdateIter(func(hash string, entry *sto.HashEntry) *sto.HashEntry {
			if !entry.Mark && entry.Pinned {
				if err := s.ipfsc.IPFS().Unpin(hash); err != nil {
					log.WithError(err).Warn("Failed to unpin " + hash)
				}
				unpinned++
				entry.Pinned = false
				entry.Dirty = true
				return entry
			}
			return nil
		})

		if err != nil {
			return 0, 0, 0, err
		}

	}
	return pinned, unpinned, errors, nil

}
