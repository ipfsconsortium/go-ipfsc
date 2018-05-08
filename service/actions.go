package service

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

const (
	metaTypeHashAdded   = "HashAdded"
	metaTypeHashRemoved = "HashRemoved"
)

type MetadataEvent struct {
	Event     string `json:"event"`
	IpfsParam string `json:"ipfsParam"`
	Type      string `json:"type"`
}

type MetadataObject struct {
	Contract   string          `json:"contract"`
	Startblock string          `json:"startblock"`
	Ttl        string          `json:"ttl"`
	NetworkId  string          `json:"networkId"`
	Events     []MetadataEvent `json:"events"`
	Abi        interface{}     `json:"abi"`
}

type MetadataUserData struct {
	metadata  *MetadataObject
	event     *MetadataEvent
	abi       *abi.ABI
	networkid uint64
}

var (
	errEventParamNotFound  = errors.New("event parameter not found")
	errEventParamNotString = errors.New("event parameter cannot be cast to string")
)

func (s *Service) addHash(address common.Address, ipfsHash string, ttl uint) error {

	log.WithFields(log.Fields{
		"Hash": ipfsHash,
		"TTL":  ttl,
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
		address, ipfsHash,
		ttl, uint(stats.DataSize),
	)

	if err != nil {
		return err
	}

	return nil
}

func (s *Service) removeHash(address common.Address, ipfsHash string) error {

	log.WithFields(log.Fields{
		"Hash": ipfsHash,
	}).Info("SERVE removeHash")

	// remove from DB
	isLastHash, err := s.storage.RemoveHash(address, ipfsHash)
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

func (s *Service) getMetadataObject(hash string) (*MetadataObject, error) {

	reader, err := s.ipfs.Cat(hash)
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)

	var meta MetadataObject
	if err = json.Unmarshal(buf.Bytes(), &meta); err != nil {
		return nil, err
	}
	return &meta, nil
}

func (s *Service) registerMetadataHandler(hash string) error {

	log.WithFields(log.Fields{
		"hash": hash,
	}).Info("SERVE start listening meta")

	meta, err := s.getMetadataObject(hash)
	if err != nil {
		return err
	}

	fmt.Printf("%#v\n", meta)

	networkid, err := strconv.Atoi(meta.NetworkId)
	if err != nil {
		return err
	}

	fmt.Printf("---- got network ------\n")

	if _, ok := s.dispatchers[uint64(networkid)]; !ok {
		return fmt.Errorf("Metadata object '%s' contains unregistered network '%v'", hash, meta.NetworkId)
	}

	abijson, err := json.Marshal(meta.Abi)
	if err != nil {
		return err
	}

	abiobj, err := abi.JSON(bytes.NewReader(abijson))
	if err != nil {
		return err
	}

	for _, event := range meta.Events {

		s.dispatchers[uint64(networkid)].RegisterHandler(
			common.HexToAddress(meta.Contract),
			&abiobj,
			event.Event,
			nil,
			MetadataUserData{meta, &event, &abiobj, uint64(networkid)},
		)
	}

	return nil
}

func (s *Service) unregisterMetadataHandler(hash string) error {

	meta, err := s.getMetadataObject(hash)
	if err != nil {
		return err
	}

	networkid, err := strconv.Atoi(meta.NetworkId)
	if err != nil {
		return err
	}

	if _, ok := s.dispatchers[uint64(networkid)]; !ok {
		return fmt.Errorf("Metadata object '%s' contains unregistered network '%v'", hash, meta.NetworkId)
	}

	for _, event := range meta.Events {
		s.dispatchers[uint64(networkid)].UnregisterHandler(
			common.HexToAddress(meta.Contract),
			event.Event,
		)
	}

	return nil
}
