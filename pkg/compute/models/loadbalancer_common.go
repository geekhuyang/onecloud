// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package models

import (
	"context"

	"yunion.io/x/jsonutils"
	"yunion.io/x/sqlchemy"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/db"
	"yunion.io/x/onecloud/pkg/mcclient"
)

type SLoadbalancerStatus struct {
	RuntimeStatus string `width:"36" charset:"ascii" nullable:"false" default:"init" list:"user"`
}

type ILoadbalancerSubResourceManager interface {
	db.IModelManager

	// PreDeleteSubs is to be called by upper manager to PreDelete models managed by this one
	PreDeleteSubs(ctx context.Context, userCred mcclient.TokenCredential, q *sqlchemy.SQuery)
}

// TODO
// notify on post create/update/delete
type SLoadbalancerNotifier struct{}

func (n *SLoadbalancerNotifier) PostCreate(ctx context.Context, userCred mcclient.TokenCredential, ownerProjId string, query jsonutils.JSONObject, data jsonutils.JSONObject) {
	return
}

type SLoadbalancerLogSkipper struct{}

func (lls SLoadbalancerLogSkipper) skipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	data, ok := query.(*jsonutils.JSONDict)
	if !ok {
		return false
	}
	if val, _ := data.GetString(api.LBAGENT_QUERY_ORIG_KEY); val != api.LBAGENT_QUERY_ORIG_VAL {
		return false
	}
	return true
}

func (lls SLoadbalancerLogSkipper) ListSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return lls.skipLog(ctx, userCred, query)
}

func (lls SLoadbalancerLogSkipper) GetSkipLog(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) bool {
	return lls.skipLog(ctx, userCred, query)
}
