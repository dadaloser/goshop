package v1

import (
	"context"
	"testing"

	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"
)

func TestGoodsServerRejectsNilRequests(t *testing.T) {
	server := &goodsServer{}

	tests := []struct {
		name string
		run  func() error
	}{
		{name: "goods list", run: func() error { _, err := server.GoodsList(context.Background(), nil); return err }},
		{name: "batch get goods", run: func() error { _, err := server.BatchGetGoods(context.Background(), nil); return err }},
		{name: "create goods", run: func() error { _, err := server.CreateGoods(context.Background(), nil); return err }},
		{name: "delete goods", run: func() error { _, err := server.DeleteGoods(context.Background(), nil); return err }},
		{name: "update goods", run: func() error { _, err := server.UpdateGoods(context.Background(), nil); return err }},
		{name: "goods detail", run: func() error { _, err := server.GetGoodsDetail(context.Background(), nil); return err }},
		{name: "sub category", run: func() error { _, err := server.GetSubCategory(context.Background(), nil); return err }},
		{name: "create category", run: func() error { _, err := server.CreateCategory(context.Background(), nil); return err }},
		{name: "delete category", run: func() error { _, err := server.DeleteCategory(context.Background(), nil); return err }},
		{name: "update category", run: func() error { _, err := server.UpdateCategory(context.Background(), nil); return err }},
		{name: "brand list", run: func() error { _, err := server.BrandList(context.Background(), nil); return err }},
		{name: "create brand", run: func() error { _, err := server.CreateBrand(context.Background(), nil); return err }},
		{name: "delete brand", run: func() error { _, err := server.DeleteBrand(context.Background(), nil); return err }},
		{name: "update brand", run: func() error { _, err := server.UpdateBrand(context.Background(), nil); return err }},
		{name: "create banner", run: func() error { _, err := server.CreateBanner(context.Background(), nil); return err }},
		{name: "delete banner", run: func() error { _, err := server.DeleteBanner(context.Background(), nil); return err }},
		{name: "update banner", run: func() error { _, err := server.UpdateBanner(context.Background(), nil); return err }},
		{name: "category brand list", run: func() error { _, err := server.CategoryBrandList(context.Background(), nil); return err }},
		{name: "category brand by category", run: func() error { _, err := server.GetCategoryBrandList(context.Background(), nil); return err }},
		{name: "create category brand", run: func() error { _, err := server.CreateCategoryBrand(context.Background(), nil); return err }},
		{name: "delete category brand", run: func() error { _, err := server.DeleteCategoryBrand(context.Background(), nil); return err }},
		{name: "update category brand", run: func() error { _, err := server.UpdateCategoryBrand(context.Background(), nil); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.run()
			if !errors.IsCode(err, code2.ErrValidation) {
				t.Fatalf("error = %v, want code %d", err, code2.ErrValidation)
			}
		})
	}
}
