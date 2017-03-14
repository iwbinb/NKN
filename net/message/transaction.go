package message

import (
	"GoOnchain/common"
	"GoOnchain/common/log"
	"GoOnchain/core/ledger"
	"GoOnchain/core/transaction"
	. "GoOnchain/net/protocol"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

type dataReq struct {
	msgHdr
	dataType common.InventoryType
	hash     common.Uint256
}

// Transaction message
type trn struct {
	msgHdr
	// TBD
	//txn []byte
	txn transaction.Transaction
	//hash common.Uint256
}

func (msg trn) Handle(node Noder) error {
	common.Trace()
	log.Debug("RX Transaction message")

	if !node.LocalNode().ExistedID(msg.txn.Hash()) {
		node.LocalNode().AppendTxnPool(&(msg.txn))
	}

	return nil
}

func reqTxnData(node Noder, hash common.Uint256) error {
	var msg dataReq
	msg.dataType = common.TRANSACTION
	// TODO handle the hash array case
	//msg.hash = hash

	buf, _ := msg.Serialization()
	go node.Tx(buf)
	return nil
}

func (msg dataReq) Serialization() ([]byte, error) {
	var buf bytes.Buffer

	//using serilization function
	err := binary.Write(&buf, binary.LittleEndian, msg)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), err
}

func (msg *dataReq) Deserialization(p []byte) error {
	// TODO
	return nil
}

func NewTxFromHash(hash common.Uint256) *transaction.Transaction {

	trx, _ := ledger.DefaultLedger.GetTransactionWithHash(hash)

	//var trx *transaction.Transaction
	return trx
}
func NewTx(trx *transaction.Transaction) ([]byte, error) {
	common.Trace()
	var msg trn

	msg.msgHdr.Magic = NETMAGIC
	cmd := "tx"
	copy(msg.msgHdr.CMD[0:len(cmd)], cmd)
	tmpBuffer := bytes.NewBuffer([]byte{})
	trx.Serialize(tmpBuffer)
	msg.txn = *trx
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.LittleEndian, tmpBuffer.Bytes())
	if err != nil {
		log.Error("Binary Write failed at new Msg")
		return nil, err
	}
	s := sha256.Sum256(b.Bytes())
	s2 := s[:]
	s = sha256.Sum256(s2)
	buf := bytes.NewBuffer(s[:4])
	binary.Read(buf, binary.LittleEndian, &(msg.msgHdr.Checksum))
	msg.msgHdr.Length = uint32(len(b.Bytes()))
	log.Info(fmt.Sprintf("The message payload length is %d", msg.msgHdr.Length))

	m, err := msg.Serialization()
	if err != nil {
		log.Error("Error Convert net message ", err.Error())
		return nil, err
	}

	return m, nil
}

func (msg trn) Serialization() ([]byte, error) {
	hdrBuf, err := msg.msgHdr.Serialization()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(hdrBuf)
	msg.txn.Serialize(buf)

	return buf.Bytes(), err
}

func (msg trn) DeSerialization(p []byte) error {
	buf := bytes.NewBuffer(p)
	err := binary.Read(buf, binary.LittleEndian, &(msg.msgHdr))
	err = msg.txn.Deserialize(buf)
	if err != nil {
		return err
	}

	return nil
}