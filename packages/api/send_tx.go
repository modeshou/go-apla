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

package api

import (
	"encoding/hex"
	"io/ioutil"
	"net/http"

	"github.com/AplaProject/go-apla/packages/block"
	"github.com/AplaProject/go-apla/packages/conf/syspar"
	"github.com/AplaProject/go-apla/packages/consts"

	log "github.com/sirupsen/logrus"
)

type sendTxResult struct {
	Hashes map[string]string `json:"hashes"`
}

func getTxData(r *http.Request, key string) ([]byte, error) {
	logger := getLogger(r)

	file, _, err := r.FormFile(key)
	if err != nil {
		logger.WithFields(log.Fields{"error": err}).Error("request.FormFile")
		return nil, err
	}
	defer file.Close()

	var txData []byte
	if txData, err = ioutil.ReadAll(file); err != nil {
		logger.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading multipart file")
		return nil, err
	}

	return txData, nil
}

func (m Mode) sendTxHandler(w http.ResponseWriter, r *http.Request) {
	client := getClient(r)

	if block.IsKeyBanned(client.KeyID) {
		errorResponse(w, errBannded.Errorf(block.BannedTill(client.KeyID)))
		return
	}

	err := r.ParseMultipartForm(multipartBuf)
	if err != nil {
		errorResponse(w, err, http.StatusBadRequest)
		return
	}

	result := &sendTxResult{Hashes: make(map[string]string)}
	for key := range r.MultipartForm.File {
		txData, err := getTxData(r, key)
		if err != nil {
			errorResponse(w, err)
			return
		}

		hash, err := txHandler(r, txData, m)
		if err != nil {
			errorResponse(w, err)
			return
		}
		result.Hashes[key] = hash
	}

	for key := range r.Form {
		txData, err := hex.DecodeString(r.FormValue(key))
		if err != nil {
			errorResponse(w, err)
			return
		}

		hash, err := txHandler(r, txData, m)
		if err != nil {
			errorResponse(w, err)
			return
		}
		result.Hashes[key] = hash
	}

	jsonResponse(w, result)
}

type contractResult struct {
	Hash string `json:"hash"`
	// These fields are used for OBS
	Message *txstatusError `json:"errmsg,omitempty"`
	Result  string         `json:"result,omitempty"`
}

func txHandler(r *http.Request, txData []byte, m Mode) (string, error) {
	client := getClient(r)
	logger := getLogger(r)

	if int64(len(txData)) > syspar.GetMaxTxSize() {
		logger.WithFields(log.Fields{"type": consts.ParameterExceeded, "max_size": syspar.GetMaxTxSize(), "size": len(txData)}).Error("transaction size exceeds max size")
		block.BadTxForBan(client.KeyID)
		return "", errLimitTxSize.Errorf(len(txData))
	}

	hash, err := m.ClientTxProcessor.ProcessClientTranstaction(txData, client.KeyID, logger)
	if err != nil {
		return "", err
	}

	return hash, nil
}
