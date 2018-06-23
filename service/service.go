package service

import (
	"errors"
	"fmt"
	"strconv"

	cfg "github.com/ipfsconsortium/gipc/config"
	ipfsc "github.com/ipfsconsortium/gipc/ipfsc"
	sto "github.com/ipfsconsortium/gipc/storage"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	ipfsc   *ipfsc.Ipfsc
	storage *sto.Storage
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ipfsc *ipfsc.Ipfsc, storage *sto.Storage) *Service {
	return &Service{ipfsc, storage}
}

func (s *Service) syncConsortiumManifest(member string, m *ipfsc.ConsortiumManifest, quotum *int) {

	for _, member := range m.Members {

		log.WithFields(log.Fields{
			"member": member.EnsName,
			"quotum": member.Quotum,
		}).Info("Processing member")

		quotum, err := strconv.Atoi(member.Quotum)
		if err != nil {
			log.WithFields(log.Fields{
				"member": member.EnsName,
			}).Error("Bad member quotum")
			continue
		}

		err = s.syncEnsName(member.EnsName, &quotum)
		if err != nil {
			log.WithFields(log.Fields{
				"member": member.EnsName,
				"err":    err,
			}).Error("Error processing member")
		}

	}
}

func (s *Service) syncPinningManifest(member string, m *ipfsc.PinningManifest, quotum *int) {

	if quotum == nil {
		localquotum, err := strconv.Atoi(m.Quotum)
		if err != nil {
			log.WithFields(log.Fields{
				"quotum": m.Quotum,
				"err":    err,
			}).Error("Invalid quota, unable to process")
			return
		}
		quotum = &localquotum
	}

	for i, ipfshash := range m.Pin {
		log.WithField("c", fmt.Sprintf("%v/%v", i, len(m.Pin))).Info("Processing manifest")
		if err := s.addHash(member, ipfshash, *quotum); err != nil {
			log.WithFields(log.Fields{
				"hash": ipfshash,
				"err":  err,
			}).Error("Failed adding hash")
		}
	}
}

func (s *Service) syncEnsName(ensname string, quotum *int) error {

	manifest, err := s.ipfsc.Read(ensname)
	if err != nil {
		return err
	}

	switch v := manifest.(type) {
	case *ipfsc.ConsortiumManifest:
		s.syncConsortiumManifest(ensname, v, quotum)
	case *ipfsc.PinningManifest:
		s.syncPinningManifest(ensname, v, quotum)
	default:
		return fmt.Errorf("Unknown manifest type")
	}
	return nil
}

func (s *Service) addHash(member, ipfsHash string, quotum int) error {

	log.WithFields(log.Fields{
		"hash": ipfsHash,
	}).Info("SERVE addHash")

	// get the size & pin

	stats, err := s.ipfsc.IPFS().ObjectStat(ipfsHash)
	if err != nil {
		log.WithError(err).Error("Failed to ipfs.ObjectStat")
		return err
	}

	log.WithFields(log.Fields{
		"hash":     ipfsHash,
		"datasize": stats.DataSize,
	}).Debug("IPFS stats")

	globals, err := s.storage.Globals()
	if err != nil {
		log.WithError(err).Error("Failed to get storage globals")
		return err
	}

	requieredLimit := int(globals.CurrentQuota) + int(stats.DataSize)
	if requieredLimit > quotum {
		log.WithFields(log.Fields{
			"hash":      ipfsHash,
			"current":   globals.CurrentQuota,
			"requiered": requieredLimit,
		}).Error(errReachedPersistLimit)

		return errReachedPersistLimit
	}
	err = s.ipfsc.IPFS().Pin(ipfsHash, false)
	if err != nil {
		return err
	}
	log.WithFields(log.Fields{
		"Hash": ipfsHash,
	}).Debug("IPFS pinning ok")

	// store in the DB
	err = s.storage.AddHash(
		member, ipfsHash,
		uint(stats.DataSize),
	)

	if err != nil {
		return err
	}

	return nil
}

func (s *Service) removeHash(member, ipfsHash string) error {

	log.WithFields(log.Fields{
		"hash": ipfsHash,
	}).Info("SERVE removeHash")

	// remove from DB
	isLastHash, err := s.storage.RemoveHash(member, ipfsHash)
	if err != nil {
		return err
	}

	if isLastHash {
		// is the last has registerer in a contract, unpin it
		s.ipfsc.IPFS().Unpin(ipfsHash)
		if err != nil {
			log.Warn("IPFS says that hash ", ipfsHash, " is not pinned.")
		}
	}

	return nil
}

func (s *Service) serve() {
	for _, name := range cfg.C.EnsNames.Remotes {
		log.WithFields(log.Fields{
			"ens":  name,
			"ipfs": name,
		}).Info("Processing ENS name")

		err := s.syncEnsName(name, nil)

		if err != nil {
			log.WithFields(log.Fields{
				"ens":   name,
				"error": err,
			}).Error("Cannot process name")
		}
	}
}

func (s *Service) Sync() error {
	s.serve()
	return nil
}
