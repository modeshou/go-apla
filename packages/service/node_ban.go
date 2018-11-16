package service

import (
	"sync"
	"time"

	"github.com/GenesisKernel/go-genesis/packages/conf"
	"github.com/GenesisKernel/go-genesis/packages/conf/syspar"
	"github.com/GenesisKernel/go-genesis/packages/consts"
	"github.com/GenesisKernel/go-genesis/packages/script"
	"github.com/GenesisKernel/go-genesis/packages/smart"
	"github.com/GenesisKernel/go-genesis/packages/utils"
	"github.com/GenesisKernel/go-genesis/packages/utils/tx"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type localBannedNode struct {
	FullNode       *syspar.FullNode
	LocalUnBanTime time.Time
}

type NodesBanService struct {
	localBannedNodes map[int64]localBannedNode
	fullNodes        []syspar.FullNode

	m *sync.Mutex
}

var nbs *NodesBanService

// GetNodesBanService is returning nodes ban service
func GetNodesBanService() *NodesBanService {
	return nbs
}

// InitNodesBanService initializing nodes ban storage
func InitNodesBanService() error {
	nbs = &NodesBanService{
		localBannedNodes: make(map[int64]localBannedNode),
		m:                &sync.Mutex{},
	}

	nbs.refreshNodes()
	return nil
}

// RegisterBadBlock is set node to local ban and saving bad block to global registry
func (nbs *NodesBanService) RegisterBadBlock(node syspar.FullNode, badBlockId, blockTime int64, reason string) error {
	if nbs.IsBanned(node) {
		return nil
	}

	nbs.localBan(node)

	err := nbs.newBadBlock(node, badBlockId, blockTime, reason)
	if err != nil {
		return err
	}

	return nil
}

// IsBanned is allows to check node ban (local or global)
func (nbs *NodesBanService) IsBanned(node syspar.FullNode) bool {
	nbs.refreshNodes()

	nbs.m.Lock()
	defer nbs.m.Unlock()

	// Searching for local ban
	now := time.Now()
	if fn, ok := nbs.localBannedNodes[node.KeyID]; ok {
		if now.Equal(fn.LocalUnBanTime) || now.After(fn.LocalUnBanTime) {
			delete(nbs.localBannedNodes, node.KeyID)
			return false
		}

		return true
	}

	// Searching for global ban.
	// Here we don't estimating global ban expiration. If ban time doesn't equal zero - we assuming
	// that node is still banned (even if `unban` time has already passed)
	for _, fn := range nbs.fullNodes {
		if fn.KeyID == node.KeyID {
			if !fn.UnbanTime.Equal(time.Unix(0, 0)) {
				return true
			} else {
				break
			}
		}
	}

	return false
}

func (nbs *NodesBanService) refreshNodes() {
	nbs.m.Lock()
	nbs.fullNodes = syspar.GetNodes()
	nbs.m.Unlock()
}

func (nbs *NodesBanService) localBan(node syspar.FullNode) {
	nbs.m.Lock()
	defer nbs.m.Unlock()

	nbs.localBannedNodes[node.KeyID] = localBannedNode{
		FullNode:       &node,
		LocalUnBanTime: time.Now().Add(syspar.GetLocalNodeBanTime()),
	}
}

func (nbs *NodesBanService) newBadBlock(producer syspar.FullNode, blockId, blockTime int64, reason string) error {
	nodePrivateKey, err := utils.GetNodePrivateKey()
	if err != nil || len(nodePrivateKey) < 1 {
		if err == nil {
			log.WithFields(log.Fields{"type": consts.EmptyObject}).Error("node private key is empty")
		}
		return err
	}

	var currentNode syspar.FullNode
	nbs.m.Lock()
	for _, fn := range nbs.fullNodes {
		if fn.KeyID == conf.Config.KeyID {
			currentNode = fn
			break
		}
	}
	nbs.m.Unlock()

	if currentNode.KeyID == 0 {
		return errors.New("cant find current node in full nodes list")
	}

	vm := smart.GetVM()
	contract := smart.VMGetContract(vm, "NewBadBlock", 1)
	info := contract.Block.Info.(*script.ContractInfo)

	sc := tx.SmartContract{
		Header: tx.Header{
			ID:          int(info.ID),
			Time:        time.Now().Unix(),
			EcosystemID: 1,
			KeyID:       conf.Config.KeyID,
		},
		Params: map[string]interface{}{
			"ProducerNodeID": producer.KeyID,
			"ConsumerNodeID": currentNode.KeyID,
			"BlockID":        blockId,
			"Timestamp":      blockTime,
			"Reason":         reason,
		},
	}

	txData, txHash, err := tx.NewInternalTransaction(sc, nodePrivateKey)
	if err != nil {
		return err
	}

	return tx.CreateTransaction(txData, txHash, conf.Config.KeyID)
}

func (nbs *NodesBanService) FilterBannedHosts(hosts []string) ([]string, error) {
	var goodHosts []string
	for _, h := range hosts {
		n, err := syspar.GetNodeByHost(h)
		if err != nil {
			log.WithFields(log.Fields{"error": err, "host": h}).Error("getting node by host")
			return nil, err
		}

		if !nbs.IsBanned(n) {
			goodHosts = append(goodHosts, n.TCPAddress)
		}
	}
	return goodHosts, nil
}