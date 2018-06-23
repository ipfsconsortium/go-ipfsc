package eth

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"

	log "github.com/sirupsen/logrus"
)

type EventHandlerFunc func(*types.Log, *ScanEventHandler) error

type SavePoint interface {
	Load() (lastBlock uint64, lastTxIndex, lastLogIndex uint, err error)
	Save(logevent *types.Log) error
	SkipTx(txid common.Hash) (bool, error)
}

type ScanEventHandler struct {
	Address   common.Address
	EventName string
	Topic     string
	Handler   EventHandlerFunc
	UserData  interface{}
}

type ScanEventDispatcher struct {
	sync.Mutex

	client *ethclient.Client

	eventHandlers []ScanEventHandler
	savepoint     SavePoint

	block    *types.Block
	receipts *ReceiptDownloader

	terminatech  chan interface{}
	terminatedch chan interface{}

	nextBlock    uint64
	nextTxIndex  uint
	nextLogIndex uint
}

func NewScanEventDispatcher(client *ethclient.Client, savepoint SavePoint) *ScanEventDispatcher {

	return &ScanEventDispatcher{
		client:    client,
		savepoint: savepoint,

		receipts: NewReceiptDownloader(client, 3),

		terminatech:  make(chan interface{}),
		terminatedch: make(chan interface{}),
	}
}

// Register registers a function to be called on event emission. Call NotifyUpdate if this is done
//   when everything started
func (e *ScanEventDispatcher) RegisterHandler(address common.Address, abi *abi.ABI, event string, handler EventHandlerFunc, userdata interface{}) {

	abievent, ok := abi.Events[event]
	if !ok {
		panic(fmt.Errorf("Event %v not found", event))
	}
	topicID := abievent.Id()

	eventHandler := ScanEventHandler{
		Address:   address,
		EventName: event,
		Topic:     "0x" + hex.EncodeToString(topicID[:]),
		Handler:   handler,
		UserData:  userdata,
	}

	e.Lock()
	e.eventHandlers = append(e.eventHandlers, eventHandler)
	e.Unlock()
}

func (e *ScanEventDispatcher) UnregisterHandler(address common.Address, event string) {

	e.Lock()
	defer e.Unlock()

	for i, handler := range e.eventHandlers {
		if bytes.Equal(handler.Address[:], address[:]) && handler.EventName == event {
			e.eventHandlers[i] = e.eventHandlers[len(e.eventHandlers)-1]
			e.eventHandlers = e.eventHandlers[:len(e.eventHandlers)-1]
			return
		}
	}

}

func (e *ScanEventDispatcher) runHandlerFor(logevent *types.Log) error {

	if !logevent.Removed {

		var handler *ScanEventHandler
		e.Lock()
		for _, v := range e.eventHandlers {
			if logevent.Address == v.Address && logevent.Topics[0].Hex() == v.Topic {
				handler = &v
				break
			}
		}
		e.Unlock()

		if handler != nil {
			log.WithField("event", handler.EventName).Debug("EVENT run handler ")
			return handler.Handler(logevent, handler)
		}
	}

	return nil
}

func (e *ScanEventDispatcher) process() (bool, error) {

	var err error

	// Retrieve the last processed log, this is only called in the first loop.
	if e.nextBlock == 0 {
		e.nextBlock, e.nextTxIndex, e.nextLogIndex, err = e.savepoint.Load()
		if err != nil {
			return false, err
		}
		e.nextLogIndex++
	}

	log.WithFields(log.Fields{
		"block/tx/log": fmt.Sprintf("%v/%v/%v", e.nextBlock, e.nextTxIndex, e.nextLogIndex),
	}).Info("EVENT scanning")

	// Check if e.block is valid, if not download it.
	if e.block == nil || e.block.NumberU64() < e.nextBlock {
		e.block, err = e.client.BlockByNumber(context.TODO(), big.NewInt(int64(e.nextBlock)))

		// Check if block is available, if is in the main chain.
		if err == ethereum.NotFound {
			return true, nil
		}
		if err != nil {
			return false, err
		}

		// Download all receipts, starting with the last processed one,
		//   transactions marked as skip, are not processed
		for index := e.nextTxIndex; index < uint(len(e.block.Transactions())); index++ {
			txid := e.block.Transactions()[index].Hash()
			if skip, err := e.savepoint.SkipTx(txid); err == nil {
				if !skip {
					e.receipts.Request(e.block.Transactions()[index].Hash())
				}
			} else {
				return false, err
			}
		}
	}

	var receipt *types.Receipt

	// Download the receipt that contains the log
	if e.nextTxIndex < uint(len(e.block.Transactions())) {
		txid := e.block.Transactions()[e.nextTxIndex].Hash()
		if skip, err := e.savepoint.SkipTx(txid); err == nil {
			if !skip {
				receipt, err = e.receipts.Get(txid)
				if err != nil {
					return false, err
				}
			}
		} else {
			return false, err
		}
	}

	// Process next log in this receipt
	if receipt != nil && e.nextLogIndex < uint(len(receipt.Logs)) {

		logevent := receipt.Logs[e.nextLogIndex]

		err = e.runHandlerFor(logevent)
		if err != nil {
			log.WithFields(log.Fields{
				"err":  err,
				"txid": receipt.TxHash.Hex(),
			}).Warn("EVENT Failed handling ")
			return false, err
		}

		e.savepoint.Save(logevent)
		e.nextLogIndex++
	}

	/* go to next */

	if receipt == nil || e.nextLogIndex >= uint(len(receipt.Logs)) {
		if receipt != nil {
			e.receipts.Forget(receipt.TxHash)
		}
		e.nextLogIndex = 0
		e.nextTxIndex++
	}

	if e.nextTxIndex >= uint(len(e.block.Transactions())) {
		e.nextLogIndex = 0
		e.nextTxIndex = 0
		e.nextBlock++
	}

	return false, nil
}

// Stop scanning the blockchain for events
func (e *ScanEventDispatcher) Stop() {
	go func() {
		e.terminatech <- nil
	}()
}

// Join waits all background jobs finished
func (e *ScanEventDispatcher) Join() {
	<-e.terminatedch
}

// Start scanning the blockchain for events
func (e *ScanEventDispatcher) Start() {

	go func() {
		e.receipts.Start()
		loop := true
		for loop {
			select {

			case <-e.terminatech:
				log.Debug("EVENT Dispatching terminatech")
				loop = false
				break

			default:
				wait, err := e.process()
				if err != nil {
					log.Error("EVENT Failed ", err)
					loop = false
				} else if wait {
					time.Sleep(4 * time.Second)
				}
			}
		}
		e.terminatedch <- nil
		e.receipts.Stop()
		e.receipts.Join()

	}()
}
