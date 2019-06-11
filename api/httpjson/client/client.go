package client

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/nknorg/nkn/api/common"
	. "github.com/nknorg/nkn/common"
	"github.com/nknorg/nkn/crypto"
)

// Call sends RPC request to server
func Call(address string, method string, id interface{}, params map[string]interface{}) ([]byte, error) {
	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     id,
		"params": params,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Marshal JSON request: %v\n", err)
		return nil, err
	}
	resp, err := http.Post(address, "application/json", strings.NewReader(string(data)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "POST request: %v\n", err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "GET response: %v\n", err)
		return nil, err
	}

	return body, nil
}

func GetRemoteBlkHeight(remote string) (uint32, error) {
	resp, err := Call(remote, "getlatestblockheight", 0, map[string]interface{}{})
	if err != nil {
		return 0, err
	}

	var ret struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      uint   `json:"id"`
		Result  uint32 `json:"result"`
	}
	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return 0, err
	}

	return ret.Result, nil
}

func FindSuccessorAddrs(remote string, key []byte) ([]string, error) {
	resp, err := Call(remote, "findsuccessoraddrs", 0, map[string]interface{}{
		"key": hex.EncodeToString(key),
	})
	if err != nil {
		return nil, err
	}
	log.Printf("FindSuccessorAddrs: %s\n", string(resp))

	var ret struct {
		Result []string `json:"result"`
	}
	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return nil, err
	}
	log.Printf("Successor Address: %v\n", ret)

	return ret.Result, nil
}

func GetMyExtIP(remote string, ip []byte) (string, error) {
	resp, err := Call(remote, "getmyextip", 0, map[string]interface{}{
		"RemoteAddr": ip,
	})
	if err != nil {
		return "", err
	}
	log.Printf("GetMyExtIP got resp: %v from %s\n", string(resp), remote)

	var ret struct {
		Result struct{ RemoteAddr string } `json:"result"`
		Err    map[string]interface{}      `json:"error"`
	}
	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return "", err
	}
	if len(ret.Err) != 0 { // resp.error NOT empty
		return "", fmt.Errorf("GetMyExtIP(%s) resp error: %v", remote, string(resp))
	}
	log.Printf("From %s got myself ExtIP: %s\n", remote, string(resp))

	return ret.Result.RemoteAddr, nil
}

func GetID(remote string, publicKey []byte) ([]byte, error) {
	resp, err := Call(remote, "getid", 0, map[string]interface{}{
		"publickey": BytesToHexString(publicKey),
	})
	if err != nil {
		return nil, err
	}
	log.Printf("GetID got resp: %v from %s\n", string(resp), remote)

	var ret struct {
		Result struct{ Id string }    `json:"result"`
		Err    map[string]interface{} `json:"error"`
	}
	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return nil, err
	}
	if len(ret.Err) != 0 { // resp.error NOT empty
		code, ok := ret.Err["code"].(float64)
		if !ok {
			return nil, fmt.Errorf("GetID resp error,interface conversion faild")
		}
		if int64(code) == -int64(common.ErrZeroID) {
			return crypto.Sha256ZeroHash, nil
		}

		return nil, fmt.Errorf("GetID(%s) resp error: %v", remote, string(resp))
	}

	idSlice, err := HexStringToBytes(ret.Result.Id)
	if err != nil {
		return nil, err
	}

	return idSlice, nil
}

func CreateID(remote string, genIdTxn string) (string, error) {
	params := map[string]interface{}{
		"tx": genIdTxn,
	}

	resp, err := Call(remote, "sendrawtransaction", 0, params)
	if err != nil {
		return "", err
	}

	log.Printf("CreateID got resp: %v from %s\n", string(resp), remote)

	var ret struct {
		Result string                 `json:"result"`
		Err    map[string]interface{} `json:"error"`
	}

	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return "", err
	}
	if len(ret.Err) != 0 { // resp.error NOT empty
		code, ok := ret.Err["code"].(float64)
		if !ok {
			return "", fmt.Errorf("CreateID resp error,interface conversion faild")
		}

		if int64(code) == -int64(common.ErrDuplicatedTx) {
			return "", nil
		}
		return "", fmt.Errorf("CreateID(%s) resp error: %v", remote, string(resp))
	}

	return ret.Result, nil
}

func GetNonceByAddr(remote string, addr string) (uint64, error) {
	params := map[string]interface{}{
		"address": addr,
	}

	resp, err := Call(remote, "getnoncebyaddr", 0, params)
	if err != nil {
		return 0, err
	}

	log.Printf("GetNonceByAddr got resp: %v from %s\n", string(resp), remote)

	var ret struct {
		Result struct {
			nonce         uint64
			nonceInTxPool uint64
			currentHeight uint32
		} `json:"result"`
		Err map[string]interface{} `json:"error"`
	}

	if err := json.Unmarshal(resp, &ret); err != nil {
		log.Println(err)
		return 0, err
	}
	if len(ret.Err) != 0 { // resp.error NOT empty
		return 0, fmt.Errorf("GetNonceByAddr(%s) resp error: %v", remote, string(resp))
	}

	return ret.Result.nonceInTxPool, nil
}
