package db

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/nknorg/nkn/common/serialization"
	"github.com/nknorg/nkn/core/ledger"
	tx "github.com/nknorg/nkn/core/transaction"
	"github.com/nknorg/nkn/core/transaction/payload"
)

func (cs *ChainStore) Rollback(b *ledger.Block) error {
	if err := cs.st.NewBatch(); err != nil {
		return err
	}

	if b.Header.Height == 0 {
		return errors.New("the genesis block need not be rolled back.")
	}

	if err := cs.rollbackHeader(b); err != nil {
		return err
	}

	if err := cs.rollbackTransaction(b); err != nil {
		return err
	}

	if err := cs.rollbackBlockHash(b); err != nil {
		return err
	}

	if err := cs.rollbackCurrentBlockHash(b); err != nil {
		return err
	}

	if err := cs.rollbackNames(b); err != nil {
		return err
	}

	if err := cs.rollbackPubSub(b); err != nil {
		return err
	}

	if err := cs.st.BatchCommit(); err != nil {
		return err
	}

	return nil
}

func (cs *ChainStore) rollbackHeader(b *ledger.Block) error {
	blockHash := b.Hash()
	return cs.st.BatchDelete(append([]byte{byte(DATA_Header)}, blockHash[:]...))
}

func (cs *ChainStore) rollbackTransaction(b *ledger.Block) error {
	for _, txn := range b.Transactions {
		txHash := txn.Hash()
		if err := cs.st.BatchDelete(append([]byte{byte(DATA_Transaction)}, txHash[:]...)); err != nil {
			return err
		}
	}

	return nil
}

func (cs *ChainStore) rollbackBlockHash(b *ledger.Block) error {
	height := make([]byte, 4)
	binary.LittleEndian.PutUint32(height[:], b.Header.Height)
	return cs.st.BatchDelete(append([]byte{byte(DATA_BlockHash)}, height...))
}

func (cs *ChainStore) rollbackCurrentBlockHash(b *ledger.Block) error {
	value := new(bytes.Buffer)
	if _, err := b.Header.PrevBlockHash.Serialize(value); err != nil {
		return err
	}
	if err := serialization.WriteUint32(value, b.Header.Height-1); err != nil {
		return err
	}

	return cs.st.BatchPut([]byte{byte(SYS_CurrentBlock)}, value.Bytes())
}

func (cs *ChainStore) rollbackNames(b *ledger.Block) error {
	for _, txn := range b.Transactions {
		if txn.TxType == tx.RegisterName {
			registerNamePayload := txn.Payload.(*payload.RegisterName)
			err := cs.DeleteName(registerNamePayload.Registrant)
			if err != nil {
				return err
			}
		}
	}

	for _, txn := range b.Transactions {
		if txn.TxType == tx.DeleteName {
			version := txn.PayloadVersion
			if version > 0 { // can't rollback DeleteName tx with version 0
				deleteNamePayload := txn.Payload.(*payload.DeleteName)
				err := cs.SaveName(deleteNamePayload.Registrant, deleteNamePayload.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (cs *ChainStore) rollbackPubSub(b *ledger.Block) error {
	height := b.Header.Height

	for _, txn := range b.Transactions {
		if txn.TxType == tx.Subscribe {
			subscribePayload := txn.Payload.(*payload.Subscribe)
			err := cs.Unsubscribe(subscribePayload.Subscriber, subscribePayload.Identifier, subscribePayload.Topic, subscribePayload.Bucket, subscribePayload.Duration, height)
			if err != nil {
				return err
			}
		}
	}

	return nil
}