package service

import (
	"encoding/json"
	"fmt"
)

const (
	consortiumType = "consortium"
	manifestType   = "manifest"
)

type configBase struct {
	Type string `json:"type"`
}

type consortium struct {
	configBase
	Quotum  string `json:"quotum"`
	Members []struct {
		EnsName string `json:"ensname"`
		Quotum  string `json:"quotum"`
	}
}

type manifest struct {
	configBase
	Pin []string `json:"pin"`
}

func parseManifest(data []byte) (interface{}, error) {

	var cfg configBase
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	switch cfg.Type {
	case consortiumType:
		var t consortium
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		return &t, nil
	case manifestType:
		var t manifest
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}

	return nil, fmt.Errorf("Uknown manifest type %v", cfg.Type)
}
