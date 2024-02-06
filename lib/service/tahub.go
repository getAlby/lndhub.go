package service

import (
	"context"
	b64 "encoding/base64"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"

	"github.com/lightninglabs/taproot-assets/taprpc"
	"github.com/lightninglabs/taproot-assets/taprpc/universerpc"
)

type AssetRoot struct {
	AssetName  string `json:"asset_name"`
	AssetID    string `json:"asset_id"`
	GroupKey   string `json:"group_key"`  
}

func (svc *LndhubService) GetUniverseAssets(ctx context.Context) (okMsg string, isError bool) {
	req := universerpc.AssetRootRequest{}
	universeRoots, err := svc.TapdClient.GetUniverseAssets(ctx, &req)
	if err != nil {
		// TODO OK Relay-Compatible messages need a central location
		return "error: no assets found, possible disconnect.", false
	}
	var okSuccessMsg = "uniassets: "
	// since there can be two root entries per asset (one for issuance and one for transfer: https://lightning.engineering/api-docs/api/taproot-assets/universe/query-asset-roots#universerpcqueryrootresponse)
	// the observedAssetIds array helps us return something the user expects to see i.e. joins asset/transfer entry if both exist
	var observedAssetIds = []string{}
	// TODO confirm when the key may be the group key hash instead of the assetId
	for assetId, root := range universeRoots.UniverseRoots {
		rawAssetId := strings.Split(assetId, "-")[1]
		seen := slices.Contains(observedAssetIds, rawAssetId)

		if !seen {
			decoded, err := hex.DecodeString(rawAssetId)

			if err != nil {
				// TODO OK Relay-Compatible messages need a central location
				return "error: failed to parse assetID.", false				
			}

			final := b64.StdEncoding.EncodeToString(decoded)

			appendAsset := fmt.Sprintf("%s %s,", final, root.AssetName)
			okSuccessMsg = okSuccessMsg + appendAsset

			observedAssetIds = append(observedAssetIds, rawAssetId)
		}
	}

	return okSuccessMsg, true
}

func (svc *LndhubService) GetAddressByAssetId(ctx context.Context, assetId string, amt uint64) (okMsg string, isError bool) {
	decoded, err := b64.StdEncoding.DecodeString(assetId)
	if err != nil {
		// TODO OK Relay-Compatible messages need a central location
		return "error: failed to parse assetID.", false	
	}

	req := taprpc.NewAddrRequest{
		AssetId: decoded,
		Amt: amt,
	}
	newAddr, err := svc.TapdClient.NewAddress(ctx, &req)
	if err != nil {
		// TODO OK Relay-Compatible messages need a central location
		return "error: failed to create receive address.", false
	}
	return fmt.Sprintf("address: %s", newAddr.Encoded), true
}