// Apla Software includes an integrated development
// environment with a multi-level system for the management
// of access rights to data, interfaces, and Smart contracts. The
// technical characteristics of the Apla Software are indicated in
// Apla Technical Paper.

// Apla Users are granted a permission to deal in the Apla
// Software without restrictions, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense,
// and/or sell copies of Apla Software, and to permit persons
// to whom Apla Software is furnished to do so, subject to the
// following conditions:
// * the copyright notice of GenesisKernel and EGAAS S.A.
// and this permission notice shall be included in all copies or
// substantial portions of the software;
// * a result of the dealing in Apla Software cannot be
// implemented outside of the Apla Platform environment.

// THE APLA SOFTWARE IS PROVIDED “AS IS”, WITHOUT WARRANTY
// OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED
// TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A
// PARTICULAR PURPOSE, ERROR FREE AND NONINFRINGEMENT. IN
// NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
// LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR
// THE USE OR OTHER DEALINGS IN THE APLA SOFTWARE.

package contract

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/AplaProject/go-apla/packages/conf"
	"github.com/AplaProject/go-apla/packages/consts"
	"github.com/AplaProject/go-apla/packages/converter"
	"github.com/AplaProject/go-apla/packages/crypto"
	"github.com/AplaProject/go-apla/packages/utils"

	log "github.com/sirupsen/logrus"
)

const (
	headerAuthPrefix = "Bearer "
)

type authResult struct {
	UID   string `json:"uid,omitempty"`
	Token string `json:"token,omitempty"`
}

type contractResult struct {
	Hash string `json:"hash"`
	// These fields are used for OBS
	Message struct {
		Type  string `json:"type,omitempty"`
		Error string `json:"error,omitempty"`
	} `json:"errmsg,omitempty"`
	Result string `json:"result,omitempty"`
}

// NodeContract creates a transaction to execute the contract.
// The transaction is signed with a node key.
func NodeContract(Name string) (result contractResult, err error) {
	var (
		sign                          []byte
		ret                           authResult
		NodePrivateKey, NodePublicKey string
	)
	err = sendAPIRequest(`GET`, `getuid`, nil, &ret, ``)
	if err != nil {
		return
	}
	auth := ret.Token
	if len(ret.UID) == 0 {
		err = fmt.Errorf(`getuid has returned empty uid`)
		return
	}
	NodePrivateKey, NodePublicKey, err = utils.GetNodeKeys()
	if err != nil || len(NodePrivateKey) == 0 {
		if err == nil {
			log.WithFields(log.Fields{"type": consts.EmptyObject}).Error("node private key is empty")
			err = errors.New(`empty node private key`)
		}
		return
	}
	sign, err = crypto.SignString(NodePrivateKey, ret.UID)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.CryptoError, "error": err}).Error("signing node uid")
		return
	}
	form := url.Values{"pubkey": {NodePublicKey}, "signature": {hex.EncodeToString(sign)},
		`ecosystem`: {converter.Int64ToStr(1)}}
	var logret authResult
	err = sendAPIRequest(`POST`, `login`, &form, &logret, auth)
	if err != nil {
		return
	}
	auth = logret.Token
	form = url.Values{`obs`: {`true`}}
	err = sendAPIRequest(`POST`, `node/`+Name, &form, &result, auth)
	if err != nil {
		return
	}
	return
}

func sendAPIRequest(rtype, url string, form *url.Values, v interface{}, auth string) error {
	client := &http.Client{}
	var ioform io.Reader
	if form != nil {
		ioform = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequest(rtype, fmt.Sprintf(`http://%s:%d%s%s`, conf.Config.HTTP.Host,
		conf.Config.HTTP.Port, consts.ApiPath, url), ioform)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.NetworkError, "error": err}).Error("new api request")
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if len(auth) > 0 {
		req.Header.Set("Authorization", headerAuthPrefix+auth)
	}
	resp, err := client.Do(req)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.NetworkError, "error": err}).Error("api request")
		return err
	}

	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading api answer")
		return err
	}

	if resp.StatusCode != http.StatusOK {
		log.WithFields(log.Fields{"type": consts.NetworkError, "error": err}).Error("api status code")
		return fmt.Errorf(`%d %s`, resp.StatusCode, strings.TrimSpace(string(data)))
	}

	if err = json.Unmarshal(data, v); err != nil {
		log.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling api answer")
		return err
	}
	return nil
}
