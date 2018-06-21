package service

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"os/signal"
	"strconv"

	shell "github.com/adriamb/go-ipfs-api"
	cfg "github.com/ipfsconsortium/gipc/config"
	eth "github.com/ipfsconsortium/gipc/eth"
	sto "github.com/ipfsconsortium/gipc/storage"
	log "github.com/sirupsen/logrus"
)

type Service struct {
	persistLimit *big.Int
	ens          *eth.ENSClient
	ipfs         *shell.Shell
	storage      *sto.Storage
}

var (
	errVerifySmartcontract = errors.New("cannot verify deployed smartcontract")
	errReadPersistLimit    = errors.New("error reading current persistLimit")
	errReachedPersistLimit = errors.New("persistlimit reached")
)

func NewService(ens *eth.ENSClient, ipfs *shell.Shell, storage *sto.Storage) *Service {

	return &Service{
		ens:     ens,
		ipfs:    ipfs,
		storage: storage,
	}
}

func (s *Service) syncConsortiumManifest(member string, m *consortiumManifest, quotum *int) {

	// TODO: check quotum

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

func (s *Service) syncPinningManifest(member string, m *pinningManifest, quotum *int) error {

	for _, ipfshash := range m.Pin {
		s.addHash(member, ipfshash)
	}
}

func (s *Service) syncEnsName(ensname string, quotum *int) error {

	ipfshash, err := s.ens.GetText(ensname, "consortiumManifest")
	if err != nil {
		return err
	}
	reader, err := s.ipfs.Cat(ipfshash)
	if err != nil {
		return err
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return err
	}
	manifest, err := parseManifest(data)
	if err != nil {
		return err
	}

	switch v := manifest.(type) {
	case *consortiumManifest:
		s.syncConsortiumManifest(ensname, manifest.(*consortiumManifest), quotum)
	case *pinningManifest:
		s.syncPinningManifest(ensname, manifest.(*pinningManifest), quotum)
	default:
		return fmt.Errorf("Unknown manifest type")
	}
	return nil
}

func (s *Service) addHash(member, ipfsHash string) error {

	log.WithFields(log.Fields{
		"hash": ipfsHash,
	}).Info("SERVE addHash")

	// get the size & pin
	stats, err := s.ipfs.ObjectStat(ipfsHash)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"hash":     ipfsHash,
		"datasize": stats.DataSize,
	}).Debug("IPFS stats")

	globals, err := s.storage.Globals()
	if err != nil {
		return err
	}

	requieredLimit := uint64(globals.CurrentQuota) + uint64(stats.DataSize)
	if requieredLimit > s.persistLimit.Uint64() {
		log.WithFields(log.Fields{
			"hash":      ipfsHash,
			"current":   s.persistLimit,
			"requiered": requieredLimit,
		}).Error(errReachedPersistLimit)

		return errReachedPersistLimit
	}

	err = s.ipfs.Pin(ipfsHash, false)
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
		err = s.ipfs.Unpin(ipfsHash)
		if err != nil {
			log.Warn("IPFS says that hash ", ipfsHash, " is not pinned.")
		}
	}

	return nil
}

func (s *Service) serve(cancel chan os.Signal) {
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

func (s *Service) Serve() error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	s.serve(c)
	return nil
}
