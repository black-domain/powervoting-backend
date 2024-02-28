// Copyright (C) 2023-2024 StorSwift Inc.
// This file is part of the PowerVoting library.

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at:
// http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"context"
	"encoding/json"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"math/big"
	"powervoting-server/model"
	"strconv"
	"strings"
)

func GetWBTC(ethAddress string, client *ethclient.Client) (*big.Int, error) {
	abiJSON := `
	 [
	   {
		"inputs": [
		  {
			"internalType": "address",
			"name": "account",
			"type": "address"
		  }
		],
		"name": "balanceOf",
		"outputs": [
		  {
			"internalType": "uint256",
			"name": "",
			"type": "uint256"
		  }
		],
		"stateMutability": "view",
		"type": "function"
	  }
	 ]
	 `
	contractAbi, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		zap.L().Error("abi json error", zap.Error(err))
	}

	data, err := contractAbi.Pack("balanceOf", common.HexToAddress(ethAddress))
	if err != nil {
		zap.L().Error("abi pack error", zap.Error(err))
	}

	contractAddress := common.HexToAddress("0x2868d708e442A6a940670d26100036d426F1e16b")
	msg := ethereum.CallMsg{
		To:   &contractAddress,
		Data: data,
	}

	result, err := client.CallContract(context.Background(), msg, nil)
	if err != nil {
		zap.L().Error("Call contract error", zap.Error(err))
	}

	unpack, err := contractAbi.Unpack("balanceOf", result)
	if err != nil {
		zap.L().Error("Unpack return data to interface error", zap.Error(err))
	}
	return unpack[0].(*big.Int), nil
}

func GetTimestamp(client model.GoEthClient) (int64, error) {
	ctx := context.Background()
	number, err := client.Client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}
	block, err := client.Client.BlockByNumber(ctx, big.NewInt(int64(number)))
	if err != nil {
		return 0, err
	}
	now := int64(block.Time())
	return now, nil
}

// GetVote Get vote info
func GetVote(client model.GoEthClient, proposalId int64, voteId int64) (model.ContractVote, error) {
	data, err := client.PowerVotingAbi.Pack("proposalToVote", big.NewInt(proposalId), big.NewInt(voteId))
	if err != nil {
		zap.L().Error("Pack method and param error: ", zap.Error(err))
		return model.ContractVote{}, err
	}
	msg := ethereum.CallMsg{
		To:   &client.PowerVotingContract,
		Data: data,
	}
	result, err := client.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		zap.L().Error("Call contract error: ", zap.Error(err))
		return model.ContractVote{}, err
	}
	var voteInfo model.ContractVote
	err = client.PowerVotingAbi.UnpackIntoInterface(&voteInfo, "proposalToVote", result)
	if err != nil {
		zap.L().Error("Unpack return data to interface error: ", zap.Error(err))
		return model.ContractVote{}, err
	}
	return voteInfo, nil
}

// GetProposal Get proposal
func GetProposal(client model.GoEthClient, proposalId int64) (model.ContractProposal, error) {
	data, err := client.PowerVotingAbi.Pack("idToProposal", big.NewInt(proposalId))
	if err != nil {
		zap.L().Error("Pack method and param error: ", zap.Error(err))
		return model.ContractProposal{}, err
	}
	msg := ethereum.CallMsg{
		To:   &client.PowerVotingContract,
		Data: data,
	}
	result, err := client.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		zap.L().Error("Call contract error: ", zap.Error(err))
		return model.ContractProposal{}, err
	}
	var proposal model.ContractProposal
	err = client.PowerVotingAbi.UnpackIntoInterface(&proposal, "idToProposal", result)
	if err != nil {
		zap.L().Error("Unpack return data to interface error: ", zap.Error(err))
		return model.ContractProposal{}, err
	}
	return proposal, nil
}

func GetProposalLatestId(client model.GoEthClient) (int, error) {
	data, err := client.PowerVotingAbi.Pack("proposalId")
	if err != nil {
		zap.L().Error("Pack method and param error: ", zap.Error(err))
		return 0, err
	}
	msg := ethereum.CallMsg{
		To:   &client.PowerVotingContract,
		Data: data,
	}
	result, err := client.Client.CallContract(context.Background(), msg, nil)
	if err != nil {
		zap.L().Error("Call contract error: ", zap.Error(err))
		return 0, err
	}
	unpack, err := client.PowerVotingAbi.Unpack("proposalId", result)
	if err != nil {
		zap.L().Error("unpack proposal id error: ", zap.Error(err))
		return 0, err
	}
	idStr := unpack[0]
	marshal, err := json.Marshal(idStr)
	if err != nil {
		zap.L().Error("json marshal error: ", zap.Error(err))
		return 0, err
	}
	id, err := strconv.Atoi(string(marshal))
	if err != nil {
		zap.L().Error("string to int error: ", zap.Error(err))
		return 0, err
	}
	return id, nil
}
