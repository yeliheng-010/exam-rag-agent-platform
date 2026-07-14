package types

import (
	"sync"
	"testing"

	"gorm.io/gorm/schema"
)

func TestQuestionGroupAssetBBoxColumnName(t *testing.T) {
	parsed, err := schema.Parse(&QuestionGroupAsset{}, &sync.Map{}, schema.NamingStrategy{})
	if err != nil {
		t.Fatalf("parse QuestionGroupAsset schema: %v", err)
	}

	field := parsed.LookUpField("BBox")
	if field == nil {
		t.Fatal("BBox field not found")
	}
	if field.DBName != "bbox" {
		t.Fatalf("expected BBox database column bbox, got %s", field.DBName)
	}
}
