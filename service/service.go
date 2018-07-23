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

func (s *Service) collectENS(expr, path string) (pinned, unpinned, errors int) {

	log.Info("Collecting[ens] " + path + ">" + expr)

	enskey, textkey, err := parseENSEntry(expr)
	if err != nil {
		log.WithError(err).Warn("Error parsing ens " + expr)
		return 0, 0, 1
	}

	// Parse an ENS entry
	if textkey != "" && textkey != DefaultManifestKey {
		// an IPFS hash stored in ENS
		expr, err := s.ipfsc.ENS().Text(enskey, textkey)
		if err != nil {
			log.WithError(err).Warn("Failed to get " + expr)
			return 0, 0, 1
		}
		return s.collect(expr, enskey+">"+path)
	}

	// Parse manifest entry
	manifest, err := s.ipfsc.Read(expr)
	if err != nil {
		log.WithError(err).Warn("Failed to get " + expr)
		return 0, 0, 1
	}

	pinned, unpinned, errors = 0, 0, 0

	switch v := manifest.(type) {

	case *ConsortiumManifest:
		for _, member := range v.Members {
			Δpinned, Δunpinned, Δerrros := s.collect(member.EnsName, path+">"+member.EnsName)
			pinned += Δpinned
			unpinned += Δunpinned
			errors += Δerrros

		}
		return pinned, unpinned, errors

	case *PinningManifest:
		for i, entry := range v.Pin {
			Δpinned, Δunpinned, Δerrros := s.collect(entry, fmt.Sprintf("%v/%v(#%v)", path, expr, i))
			pinned += Δpinned
			unpinned += Δunpinned
			errors += Δerrros

		}
		return pinned, unpinned, errors

	default:
		log.Warn("Unable to parse manifest " + expr)
		return 1, 0, 0
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

func (s *Service) collectIPFS(expr, path string) (pinned, unpinned, errors int) {
	log.Info("Collecting[ipfs] " + path + ">" + expr)

	pinned, unpinned, errors = 0, 0, 0

	// if information is available in local db, use it
	hentry, _ := s.storage.Hash(expr)
	if hentry != nil && !hentry.Dirty {
		if hentry.Mark {
			// if already marked, has been also already recursevlly marked
			return 0, 0, 0
		}
		hentry.Mark = true
		hentry.Dirty = false

		if err := s.storage.UpdateHash(expr, hentry); err != nil {
			log.WithError(err).Warn("Failed to update hash db")
			return 0, 0, 1
		}
		for _, linkhash := range hentry.Links {
			Δpinned, Δunpinned, Δerrros := s.collectIPFS(linkhash, path+">"+expr+"()")
			pinned += Δpinned
			unpinned += Δunpinned
			errors += Δerrros
		}
		return pinned, unpinned, errors
	}

	// object is not in the database, so get data from it
	start := time.Now()
	err := s.ipfsc.IPFS().Pin(expr, false)
	if err != nil {
		log.WithError(err).Warn("Unable to get object " + expr)
		return 0, 0, 1
	}

	ipfsObject, err := s.ipfsc.IPFS().ObjectGet(expr)
	if err != nil {
		log.WithError(err).Warn("Unable to get object " + expr)
		return 0, 0, 1
	}
	pinned++

	log.WithFields(log.Fields{
		"#links":    len(ipfsObject.Links),
		"len(data)": len(ipfsObject.Data),
	}).Info("Got Objectats in ", time.Since(start))

	var links []string
	if len(ipfsObject.Links) > 0 && ipfsObject.Links[0].Name != "" {
		links = make([]string, len(ipfsObject.Links))
		for i, link := range ipfsObject.Links {
			links[i] = link.Hash
			Δpinned, Δunpinned, Δerrors := s.collectIPFS(link.Hash, path+">"+expr+"("+link.Name+")")
			pinned += Δpinned
			unpinned += Δunpinned
			errors += Δerrors
		}
	}

	if err = s.storage.UpdateHash(expr, &sto.HashEntry{
		DataSize: 0,
		Links:    links,
		Mark:     true,
		Dirty:    false,
	}); err != nil {
		log.WithError(err).Warn("Failed to add hash " + expr)
		return pinned, unpinned, errors + 1
	}

	return pinned, unpinned, errors
}

func (s *Service) collect(expr, path string) (pinned, unpinned, errors int) {

	if strings.HasPrefix(expr, "/ipfs/") {
		return s.collectIPFS(expr, path)
	} else if strings.HasPrefix(expr, "0x") {
		// handle contract, TODO
	} else if strings.Contains(expr, ".eth") {
		return s.collectENS(expr, path)
	}
	log.Warn("Unable to find resolver to sync '" + expr + "'")
	return 0, 0, 1
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
	pinned, unpinned, errors = 0, 0, 0
	for _, expr := range ensnames {
		log.WithFields(log.Fields{
			"expr": expr,
		}).Info("Processing entries")

		Δpinned, Δunpinned, Δerrors := s.collect(expr, "")
		pinned += Δpinned
		unpinned += Δunpinned
		errors += Δerrors
	}

	if errors == 0 {
		/* No errors, unpin the unused hashes and mark as deleted */
		err = s.storage.HashUpdateIter(func(hash string, entry *sto.HashEntry) *sto.HashEntry {
			if !entry.Mark {
				if err := s.ipfsc.IPFS().Unpin(hash); err != nil {
					log.WithError(err).Warn("Failed to unpin " + hash)
				}
				unpinned++
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
