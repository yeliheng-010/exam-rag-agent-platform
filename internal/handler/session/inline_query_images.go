package session

import (
	"path"
	"strings"

	"github.com/Tencent/WeKnora/internal/infrastructure/docparser"
	secutils "github.com/Tencent/WeKnora/internal/utils"
)

const maxInlineQueryImages = 8

func appendInlineQueryImages(query string, tenantID uint64, images []ImageAttachment) []ImageAttachment {
	if tenantID == 0 {
		return images
	}
	seen := make(map[string]struct{}, len(images))
	for _, image := range images {
		if image.URL != "" {
			seen[image.URL] = struct{}{}
		}
	}

	added := 0
	for _, imagePath := range docparser.ExtractMarkdownImagePaths(query) {
		if added >= maxInlineQueryImages {
			break
		}
		if !isSupportedInlineImage(imagePath) || secutils.ValidateStoragePathTenant(imagePath, tenantID) != nil {
			continue
		}
		if _, exists := seen[imagePath]; exists {
			continue
		}
		seen[imagePath] = struct{}{}
		images = append(images, ImageAttachment{URL: imagePath})
		added++
	}
	return images
}

func isSupportedInlineImage(imagePath string) bool {
	if !strings.HasPrefix(imagePath, "local://") {
		return false
	}
	switch strings.ToLower(path.Ext(imagePath)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return true
	default:
		return false
	}
}
