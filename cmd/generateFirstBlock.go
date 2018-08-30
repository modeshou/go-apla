package cmd

import (
	"encoding/hex"
	"io/ioutil"
	"time"

	"github.com/spf13/cobra"

	"path/filepath"

	"github.com/GenesisKernel/go-genesis/packages/blockchain"
	"github.com/GenesisKernel/go-genesis/packages/conf"
	"github.com/GenesisKernel/go-genesis/packages/consts"

	log "github.com/sirupsen/logrus"
	msgpack "gopkg.in/vmihailenco/msgpack.v2"
)

var stopNetworkBundleFilepath string

// generateFirstBlockCmd represents the generateFirstBlock command
var generateFirstBlockCmd = &cobra.Command{
	Use:    "generateFirstBlock",
	Short:  "First generation",
	PreRun: loadConfigWKey,
	Run: func(cmd *cobra.Command, args []string) {
		now := time.Now().Unix()

		header := &blockchain.BlockHeader{
			BlockID:      1,
			Time:         now,
			EcosystemID:  0,
			KeyID:        conf.Config.KeyID,
			NodePosition: 0,
			Version:      consts.BLOCK_VERSION,
		}

		decodeKeyFile := func(kName string) []byte {
			filepath := filepath.Join(conf.Config.KeysDir, kName)
			data, err := ioutil.ReadFile(filepath)
			if err != nil {
				log.WithError(err).WithFields(log.Fields{"key": kName, "filepath": filepath}).Fatal("Reading key data")
			}

			decodedKey, err := hex.DecodeString(string(data))
			if err != nil {
				log.WithError(err).Fatalf("converting %s from hex", kName)
			}

			return decodedKey
		}

		var stopNetworkCert []byte
		if len(stopNetworkBundleFilepath) > 0 {
			var err error
			fp := filepath.Join(conf.Config.KeysDir, stopNetworkBundleFilepath)
			if stopNetworkCert, err = ioutil.ReadFile(fp); err != nil {
				log.WithError(err).WithFields(log.Fields{"filepath": fp}).Fatal("Reading cert data")
			}
		}

		if len(stopNetworkCert) == 0 {
			log.Warn("the fullchain of certificates for a network stopping is not specified")
		}

		tx, err := msgpack.Marshal(
			&consts.FirstBlock{
				TxHeader: consts.TxHeader{
					Type:  consts.TxTypeFirstBlock,
					Time:  uint32(now),
					KeyID: conf.Config.KeyID,
				},
				PublicKey:             decodeKeyFile(consts.PublicKeyFilename),
				NodePublicKey:         decodeKeyFile(consts.NodePublicKeyFilename),
				StopNetworkCertBundle: stopNetworkCert,
			},
		)

		if err != nil {
			log.WithFields(log.Fields{"type": consts.MarshallingError, "error": err}).Fatal("first block body bin marshalling")
			return
		}
		tx = append([]byte{consts.TxTypeFirstBlock}, tx...)
		block := blockchain.Block{
			Header:       header,
			Transactions: [][]byte{tx},
		}

		blockBytes, err := block.Marshal()
		if err != nil {
			log.WithFields(log.Fields{"type": consts.MarshallingError, "error": err}).Fatal("first block marshalling")
			return
		}

		ioutil.WriteFile(conf.Config.FirstBlockPath, blockBytes, 0644)
		log.Info("first block generated")
	},
}

func init() {
	generateFirstBlockCmd.Flags().StringVar(&stopNetworkBundleFilepath, "stopNetworkCert", "", "Filepath to the fullchain of certificates for network stopping")
}
