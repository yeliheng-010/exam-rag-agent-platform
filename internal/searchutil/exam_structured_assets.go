package searchutil

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Tencent/WeKnora/internal/types"
)

func writeStructuredAssets(builder *strings.Builder, assets []*types.QuestionGroupAsset) {
	ordered := sortedStructuredAssets(assets)
	if builder == nil || len(ordered) == 0 {
		return
	}
	builder.WriteString("\n[Assets]\n")
	for index, asset := range ordered {
		label := strings.TrimSpace(asset.AltText)
		if label == "" {
			label = strings.TrimSpace(asset.AssetType)
		}
		writeStructuredLine(builder, fmt.Sprintf("Asset %d", index+1), label)
		writeStructuredLine(builder, "Asset URI", asset.StorageURI)
		writeStructuredLine(builder, "Asset source chunk", asset.SourceChunkID)
	}
}

func sortedStructuredAssets(assets []*types.QuestionGroupAsset) []*types.QuestionGroupAsset {
	out := make([]*types.QuestionGroupAsset, 0, len(assets))
	for _, asset := range assets {
		if asset != nil {
			out = append(out, asset)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SortOrder == out[j].SortOrder {
			return out[i].ID < out[j].ID
		}
		return out[i].SortOrder < out[j].SortOrder
	})
	return out
}
