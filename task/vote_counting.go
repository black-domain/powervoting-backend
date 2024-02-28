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

package task

import (
	"go.uber.org/zap"
	"powervoting-server/config"
	"powervoting-server/contract"
	"powervoting-server/db"
	"powervoting-server/model"
	"powervoting-server/utils"
	"time"
)

// VotingCountHandler vote count
func VotingCountHandler() {
	networkList := config.Client.Network
	for _, network := range networkList {
		ethClient, err := contract.GetClient(network.Id)
		if err != nil {
			zap.L().Error("get go-eth client error:", zap.Error(err))
			continue
		}
		go VotingCount(ethClient)
	}
}

func VotingCount(ethClient model.GoEthClient) {
	now, err := utils.GetTimestamp(ethClient)
	if err != nil {
		zap.L().Error("get timestamp on chain error: ", zap.Error(err))
		now = time.Now().Unix()
	}
	proposals, err := db.GetProposalList(ethClient.Id, now)
	if err != nil {
		zap.L().Error("get proposal from db error:", zap.Error(err))
		return
	}
	zap.L().Info("get proposal list success!", zap.Reflect("proposals", proposals))
	for _, proposal := range proposals {
		SyncVote(ethClient, proposal.ProposalId)
		voteInfos, err := db.GetVoteList(ethClient.Id, proposal.ProposalId)
		if err != nil {
			zap.L().Error("get vote info from db error:", zap.Error(err))
			continue
		}
		zap.L().Info("get vote list success!", zap.Reflect("voteInfos", voteInfos))
		var voteList []model.Vote4Counting
		for _, voteInfo := range voteInfos {
			list, err := utils.DecodeVoteList(voteInfo)
			if err != nil {
				zap.L().Error("get vote info from IPFS or decrypt error: ", zap.Error(err))
				return
			}
			voteList = append(voteList, list...)
		}
		zap.L().Info("decode vote list success!", zap.Reflect("voteList", voteList))
		var voteHistoryList []model.VoteHistory
		// vote counting
		var result = make(map[int64]float64, 5) // max 5 options
		for _, vote := range voteList {
			var votes float64
			balance, err := utils.GetWBTC(vote.Address, ethClient.Client)
			if err != nil {
				zap.L().Error("Get balance error", zap.Error(err))
				return
			}
			if vote.Votes != 0 {
				var votePercent = float64(vote.Votes) / 100
				votes = (float64(balance.Int64()) * votePercent) / 100000000
			}
			voteHistory := model.VoteHistory{
				ProposalId: proposal.ProposalId,
				OptionId:   vote.OptionId,
				Votes:      int64(votes),
				Address:    vote.Address,
				Network:    ethClient.Id,
			}
			voteHistoryList = append(voteHistoryList, voteHistory)
			if _, ok := result[vote.OptionId]; ok {
				result[vote.OptionId] += votes
			} else {
				result[vote.OptionId] = votes
			}
		}
		var voteResultList []model.VoteResult
		options, err := utils.GetOptions(proposal.Cid)
		if err != nil {
			zap.L().Error("get options error: ", zap.Error(err))
			continue
		}
		for i := 0; i < len(options); i++ {
			voteResult := model.VoteResult{
				ProposalId: proposal.ProposalId,
				OptionId:   int64(i),
				Votes:      int64(result[int64(i)]),
				Network:    ethClient.Id,
			}
			voteResultList = append(voteResultList, voteResult)
		}
		// Save vote history and vote result to database and update status
		db.VoteResult(proposal.Id, voteHistoryList, voteResultList)
	}
}
