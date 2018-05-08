package service

import (
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	log "github.com/sirupsen/logrus"
)

func (s *Service) registerBasicHandlers(networkid uint64, address common.Address) {

	log.WithFields(log.Fields{
		"network":  networkid,
		"contract": address.Hex(),
	}).Info("SERVE start listening hashAdd/hashRemove")

	s.dispatchers[networkid].RegisterHandler(address, s.proxy.Abi(), "HashAdded", s.handleHashAdded, nil)
	s.dispatchers[networkid].RegisterHandler(address, s.proxy.Abi(), "HashRemoved", s.handleHashRemoved, nil)
	s.dispatchers[networkid].RegisterHandler(address, s.proxy.Abi(), "MetadataObjectAdded", s.handleMetadataObjectAdded, nil)
	s.dispatchers[networkid].RegisterHandler(address, s.proxy.Abi(), "MetadataObjectRemoved", s.handleMetadataObjectRemoved, nil)

}

func (s *Service) handleHashAdded(eventlog *types.Log, _ *ScanEventHandler) error {

	log.Info("HashAdded EVENT")

	type HashAdded struct {
		Hash string
		Ttl  *big.Int
	}

	var event HashAdded
	err := s.proxy.Abi().Unpack(&event, "HashAdded", eventlog.Data)
	if err != nil {
		return err
	}

	return s.addHash(eventlog.Address, event.Hash, uint(event.Ttl.Int64()))

}

func (s *Service) handleHashRemoved(eventlog *types.Log, _ *ScanEventHandler) error {

	type HashRemoved struct {
		Hash string
	}

	var event HashRemoved
	err := s.proxy.Abi().Unpack(&event, "HashRemoved", eventlog.Data)
	if err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"Hash": event.Hash,
	}).Info("SERVE HashRemoved")

	return s.removeHash(eventlog.Address, event.Hash)

}

func (s *Service) handleMetadataObjectAdded(eventlog *types.Log, _ *ScanEventHandler) error {

	var hash string
	if err := s.proxy.Abi().Unpack(&hash, "MetadataObjectAdded", eventlog.Data); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"Hash": hash,
	}).Info("SERVE MetadataObjectAdded")

	err := s.storage.AddMetadata(hash)
	if err != nil {
		err = s.registerMetadataHandler(hash)
	}

	return err
}

func (s *Service) handleMetadataObjectRemoved(eventlog *types.Log, _ *ScanEventHandler) error {

	var hash string
	if err := s.proxy.Abi().Unpack(&hash, "MetadataObjectRemoved", eventlog.Data); err != nil {
		return err
	}

	log.WithFields(log.Fields{
		"Hash": hash,
	}).Info("SERVE MetadataObjectAdded")

	err := s.storage.RemoveMetadata(hash)
	if err != nil {
		err = s.unregisterMetadataHandler(hash)
	}

	return err
}

func (s *Service) handleMetadataEvent(eventlog *types.Log, scanHanlder *ScanEventHandler) error {

	userData, _ := scanHanlder.UserData.(MetadataUserData)
	arguments := userData.abi.Events[userData.event.Event].Inputs

	paramIndex := -1
	for index, arg := range arguments.NonIndexed() {
		if arg.Name == userData.event.IpfsParam {
			paramIndex = index
		}
	}
	if paramIndex == -1 {
		return errEventParamNotFound
	}

	values, err := arguments.UnpackValues(eventlog.Data)
	if err != nil {
		return err
	}

	ipfsHash, success := values[paramIndex].(string)
	if !success {
		return errEventParamNotString
	}

	ttl, err := strconv.Atoi(userData.metadata.Ttl)
	if err != nil {
		return err
	}

	switch userData.event.Type {
	case metaTypeHashAdded:
		err = s.addHash(eventlog.Address, ipfsHash, uint(ttl))
	case metaTypeHashRemoved:
		err = s.removeHash(eventlog.Address, ipfsHash)
	}

	return err
}
