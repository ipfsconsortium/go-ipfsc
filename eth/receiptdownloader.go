package eth

import (
	"bytes"
	"context"
	"errors"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	log "github.com/sirupsen/logrus"
)

var (
	errTxNotFound = errors.New("tx not found")
)

const (
	// retryDownloadTimes how much times tries to download a receipt
	retryDownloadTimes = 5
)

// receiptTask is a single receipt download task.
type ReceiptTask struct {
	Tx     common.Hash
	result chan error

	done    bool
	Err     error
	Receipt *types.Receipt
}

// receiptDownloader allows to download multiple receipts at the same time.
type ReceiptDownloader struct {
	sync.Mutex

	client *ethclient.Client

	concurrency int
	queue       []*ReceiptTask
	pending     map[common.Hash]*ReceiptTask

	nextch       chan interface{}
	terminatech  chan interface{}
	terminatedch chan interface{}
}

func NewReceiptDownloader(client *ethclient.Client, concurrency int) *ReceiptDownloader {

	return &ReceiptDownloader{
		concurrency:  concurrency,
		client:       client,
		queue:        []*ReceiptTask{},
		pending:      make(map[common.Hash]*ReceiptTask),
		nextch:       make(chan interface{}, concurrency),
		terminatech:  make(chan interface{}),
		terminatedch: make(chan interface{}),
	}

}

// Request to download a transaction.
func (r *ReceiptDownloader) Request(txid common.Hash) {
	r.Lock()
	r.queue = append(r.queue, &ReceiptTask{
		Tx:     txid,
		result: make(chan error),
	})
	r.Unlock()

	r.next()
}

// Get the requested transaction, if not still downloaded, it waits.
func (r *ReceiptDownloader) Get(txid common.Hash) (*types.Receipt, error) {

	// Get the task from the pending list, if not, look is is still queued.
	r.Lock()
	task, ok := r.pending[txid]
	if !ok {
		for _, v := range r.queue {
			if bytes.Compare(txid[:], v.Tx[:]) == 0 {
				task = v
				ok = true
				break
			}
		}
	}
	r.Unlock()

	if !ok {
		log.WithField("tx", txid.Hex()).Warn(errTxNotFound)
		return nil, errTxNotFound
	}

	if !task.done {
		task.Err = <-task.result
		task.done = true
	}

	return task.Receipt, task.Err
}

// Forget (deletes) an already downloaded transaction.
func (r *ReceiptDownloader) Forget(txid common.Hash) {
	r.Lock()
	if _, exists := r.pending[txid]; exists {
		delete(r.pending, txid)
	} else {
		log.WithField("txid", txid.Hex()).Warn("RDOWN cannot forget tx")
	}
	r.Unlock()
}

// Stats retrieves the status.
func (r *ReceiptDownloader) Stats() (queuelen, pendinglen int) {
	r.Lock()
	queuelen = len(r.queue)
	pendinglen = len(r.pending)
	r.Unlock()

	return
}

// next tryies to process next item to download, if possible.
func (r *ReceiptDownloader) next() {

	// Get the next receipt to download, if does not reach maximum concurrency.
	r.Lock()

	if len(r.queue) == 0 || len(r.pending) >= r.concurrency {
		r.Unlock()
		return
	}

	task := r.queue[0]
	r.queue = r.queue[1:]

	r.pending[task.Tx] = task

	r.Unlock()

	// Start the download task.
	go func() {

		log.WithField("txid", task.Tx.Hex()).Debug("RDOWN Downloading receipt")

		err := errors.New("")
		var receipt *types.Receipt

		for retry := retryDownloadTimes; err != nil && retry > 0; retry-- {

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			receipt, err = r.client.TransactionReceipt(ctx, task.Tx)
			if err != nil {
				log.WithFields(log.Fields{
					"txid":  task.Tx.Hex(),
					"retry": retry,
					"err":   err,
				}).Warn("RDOWN failed to download receipt")

				<-time.After(time.Duration(rand.Int63n(int64(time.Second))))
			}
		}

		r.Lock()
		task := r.pending[task.Tx]
		r.Unlock()

		task.Receipt = receipt
		task.result <- err

		r.nextch <- nil

	}()

}

// Stop processing requests
func (r *ReceiptDownloader) Stop() {
	go func() {
		r.terminatech <- nil
	}()
}

// Join waits until all background jobs stopped
func (r *ReceiptDownloader) Join() {
	<-r.terminatedch
}

// Start processing requests
func (r *ReceiptDownloader) Start() {

	go func() {
		for true {
			select {
			case <-r.terminatech:
				log.Debug("RDOWN terminatech")
				r.terminatedch <- nil
				return

			case <-r.nextch:
				r.next()

			case <-time.After(4 * time.Second):

			}
		}
		r.terminatedch <- nil
	}()
}

/*
func Test(cmd *cobra.Command, args []string) {

	client, err := ethclient.Dial("http://localhost:8545")
	must(err)
	download := service.NewReceiptDownloader(client, 20)
	download.Start()

	startblock := 4000000
	for blockno := startblock; blockno < startblock+5000; blockno++ {
		block, err := client.BlockByNumber(context.TODO(), big.NewInt(int64(blockno)))
		must(err)
		if len(block.Transactions()) == 0 {
			continue
		}
		for _, tx := range block.Transactions() {
			download.Request(tx.Hash())
		}
		for _, tx := range block.Transactions() {
			_, err := download.Get(tx.Hash())
			must(err)
			download.Forget(tx.Hash())
		}
		queuelen, pendinglen := download.Stats()
		log.Info("Downloaded block ", blockno, " ",
			block.Transactions().Len(),
			" len(queue)=", queuelen,
			" len(pending)=", pendinglen,
		)
	}
	download.QueryStop()
	download.WaitStopped()

}
*/
