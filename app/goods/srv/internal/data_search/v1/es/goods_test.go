package es

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	proto "goshop/api/goods/v1"
	searchv1 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/goods/srv/internal/domain/do"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
)

func TestGoodsWriteRejectsInvalidInput(t *testing.T) {
	store := &goods{}

	tests := []struct {
		name string
		run  func() error
		code int
	}{
		{
			name: "create nil goods",
			run: func() error {
				return store.Create(context.Background(), nil)
			},
			code: code.ErrGoodsInvalid,
		},
		{
			name: "create zero id",
			run: func() error {
				return store.Create(context.Background(), &do.GoodsSearchDO{})
			},
			code: code.ErrGoodsInvalid,
		},
		{
			name: "delete zero id",
			run: func() error {
				return store.Delete(context.Background(), 0)
			},
			code: code.ErrGoodsNotFound,
		},
		{
			name: "update nil goods",
			run: func() error {
				return store.Update(context.Background(), nil)
			},
			code: code.ErrGoodsInvalid,
		},
		{
			name: "update zero id",
			run: func() error {
				return store.Update(context.Background(), &do.GoodsSearchDO{})
			},
			code: code.ErrGoodsInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("error = %v, want code %d", err, tt.code)
			}
		})
	}
}

func TestGoodsSearchVisibilityAndSKUFilters(t *testing.T) {
	query := buildGoodsSearchQuery(&searchv1.GoodsFilterRequest{GoodsFilterRequest: &proto.GoodsFilterRequest{SpuCode: "spu-1", SkuCode: "sku-1"}})
	source, err := query.Source()
	if err != nil {
		t.Fatalf("query.Source() error = %v", err)
	}
	encoded, _ := json.Marshal(source)
	text := string(encoded)
	for _, want := range []string{`"on_sale"`, `"spu_code.keyword"`, `"sku_code.keyword"`} {
		if !strings.Contains(text, want) {
			t.Fatalf("query = %s, want %s", text, want)
		}
	}
}
