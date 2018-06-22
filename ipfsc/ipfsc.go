package ipfsc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"

	shell "github.com/adriamb/go-ipfs-api"
	ens "github.com/ipfsconsortium/gipc/ens"
	log "github.com/sirupsen/logrus"
)

const (
	consortiumType = "consortium"
	pinningType    = "manifest"
)

type configBase struct {
	Type string `json:"type"`
}

type ConsortiumManifest struct {
	configBase
	Quotum  string `json:"quotum"`
	Members []struct {
		EnsName string `json:"ensname"`
		Quotum  string `json:"quotum"`
	}
}

type PinningManifest struct {
	configBase
	Quotum string   `json:"quotum"`
	Pin    []string `json:"pin"`
}

const (
	manifestKey = "consortiumManifest"
)

type Ipfsc struct {
	ipfs *shell.Shell
	ens  *ens.ENSClient
}

func parse(data []byte) (interface{}, error) {

	var cfg configBase
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	switch cfg.Type {

	case consortiumType:
		var t ConsortiumManifest
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		return &t, nil

	case pinningType:
		var t PinningManifest
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}

	return nil, fmt.Errorf("Uknown manifest type %v", cfg.Type)
}

func New(ipfs *shell.Shell, ens *ens.ENSClient) *Ipfsc {
	return &Ipfsc{ipfs, ens}
}

func (i *Ipfsc) Read(ensname string) (interface{}, error) {

	log.WithField("ensname", ensname).Info("Reading IPFS key from ENS")
	ipfshash, err := i.ens.Text(ensname, manifestKey)
	if err != nil {
		return nil, err
	}

	log.WithField("hash", ipfshash).Info("Downloading manifest")
	reader, err := i.ipfs.Cat(ipfshash)
	if err != nil {
		return nil, err
	}
	data, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	log.WithField("hash", ipfshash).Debug("Manifest downloaded")
	manifest, err := parse(data)
	if err != nil {
		return nil, err
	}

	return manifest, nil

}

func (i *Ipfsc) Write(ensname string, manifest *PinningManifest) error {

	manifest.Type = pinningType
	encoded, err := json.Marshal(manifest)
	if err != nil {
		return err
	}

	log.Info("Adding manifest to IPFS")
	ipfshash, err := i.ipfs.Add(bytes.NewReader(encoded))
	if err != nil {
		return err
	}

	log.WithField("hash", ipfshash).Info("Writing manifest IPFS to ENS")
	err = i.ens.SetText(ensname, manifestKey, ipfshash)
	if err != nil {
		return err
	}
	return nil

}

func (i *Ipfsc) IPFS() *shell.Shell {
	return i.ipfs
}

func (i *Ipfsc) ENS() *ens.ENSClient {
	return i.ens
}
