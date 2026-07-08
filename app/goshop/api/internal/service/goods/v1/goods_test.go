package v1

import (
	"context"
	"testing"

	gpb "goshop/api/goods/v1"
	"goshop/app/goshop/api/internal/data"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestGoodsServiceListRejectsInvalidBoundary(t *testing.T) {
	tests := []struct {
		name string
		svc  GoodsSrv
		req  *gpb.GoodsFilterRequest
		code int
	}{
		{
			name: "nil service",
			svc:  (*goodsService)(nil),
			req:  &gpb.GoodsFilterRequest{},
			code: code.ErrConnectGRPC,
		},
		{
			name: "nil data factory",
			svc:  NewGoods(nil),
			req:  &gpb.GoodsFilterRequest{},
			code: code.ErrConnectGRPC,
		},
		{
			name: "nil request",
			svc:  NewGoods(&fakeGoodsDataFactory{goods: &fakeGoodsClient{listResp: &gpb.GoodsListResponse{}}}),
			req:  nil,
			code: code.ErrGoodsInvalid,
		},
		{
			name: "nil goods client",
			svc:  NewGoods(&fakeGoodsDataFactory{}),
			req:  &gpb.GoodsFilterRequest{},
			code: code.ErrConnectGRPC,
		},
		{
			name: "nil downstream response",
			svc:  NewGoods(&fakeGoodsDataFactory{goods: &fakeGoodsClient{}}),
			req:  &gpb.GoodsFilterRequest{},
			code: code.ErrConnectGRPC,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.svc.List(context.Background(), tt.req)
			if !errors.IsCode(err, tt.code) {
				t.Fatalf("List() error = %v, want code %d", err, tt.code)
			}
		})
	}
}

func TestGoodsServiceListCallsGoodsClient(t *testing.T) {
	client := &fakeGoodsClient{listResp: &gpb.GoodsListResponse{Total: 1}}
	svc := NewGoods(&fakeGoodsDataFactory{goods: client})

	resp, err := svc.List(context.Background(), &gpb.GoodsFilterRequest{Pages: 2})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if resp.GetTotal() != 1 {
		t.Fatalf("List() total = %d, want 1", resp.GetTotal())
	}
	if client.gotListRequest.GetPages() != 2 {
		t.Fatalf("GoodsList() pages = %d, want 2", client.gotListRequest.GetPages())
	}
}

type fakeGoodsDataFactory struct {
	goods gpb.GoodsClient
	users data.UserData
}

func (f *fakeGoodsDataFactory) Goods() gpb.GoodsClient {
	return f.goods
}

func (f *fakeGoodsDataFactory) Users() data.UserData {
	return f.users
}

type fakeGoodsClient struct {
	gpb.UnimplementedGoodsServer

	listResp       *gpb.GoodsListResponse
	listErr        error
	gotListRequest *gpb.GoodsFilterRequest
}

func (f *fakeGoodsClient) GoodsList(ctx context.Context, in *gpb.GoodsFilterRequest, opts ...grpc.CallOption) (*gpb.GoodsListResponse, error) {
	f.gotListRequest = in
	return f.listResp, f.listErr
}

func (f *fakeGoodsClient) BatchGetGoods(context.Context, *gpb.BatchGoodsIdInfo, ...grpc.CallOption) (*gpb.GoodsListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CreateGoods(context.Context, *gpb.CreateGoodsInfo, ...grpc.CallOption) (*gpb.GoodsInfoResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) DeleteGoods(context.Context, *gpb.DeleteGoodsInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) UpdateGoods(context.Context, *gpb.CreateGoodsInfo, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) GetGoodsDetail(context.Context, *gpb.GoodInfoRequest, ...grpc.CallOption) (*gpb.GoodsInfoResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) GetAllCategorysList(context.Context, *emptypb.Empty, ...grpc.CallOption) (*gpb.CategoryListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) GetSubCategory(context.Context, *gpb.CategoryListRequest, ...grpc.CallOption) (*gpb.SubCategoryListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CreateCategory(context.Context, *gpb.CategoryInfoRequest, ...grpc.CallOption) (*gpb.CategoryInfoResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) DeleteCategory(context.Context, *gpb.DeleteCategoryRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) UpdateCategory(context.Context, *gpb.CategoryInfoRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) BrandList(context.Context, *gpb.BrandFilterRequest, ...grpc.CallOption) (*gpb.BrandListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CreateBrand(context.Context, *gpb.BrandRequest, ...grpc.CallOption) (*gpb.BrandInfoResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) DeleteBrand(context.Context, *gpb.BrandRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) UpdateBrand(context.Context, *gpb.BrandRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) BannerList(context.Context, *emptypb.Empty, ...grpc.CallOption) (*gpb.BannerListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CreateBanner(context.Context, *gpb.BannerRequest, ...grpc.CallOption) (*gpb.BannerResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) DeleteBanner(context.Context, *gpb.BannerRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) UpdateBanner(context.Context, *gpb.BannerRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CategoryBrandList(context.Context, *gpb.CategoryBrandFilterRequest, ...grpc.CallOption) (*gpb.CategoryBrandListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) GetCategoryBrandList(context.Context, *gpb.CategoryInfoRequest, ...grpc.CallOption) (*gpb.BrandListResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) CreateCategoryBrand(context.Context, *gpb.CategoryBrandRequest, ...grpc.CallOption) (*gpb.CategoryBrandResponse, error) {
	return nil, nil
}

func (f *fakeGoodsClient) DeleteCategoryBrand(context.Context, *gpb.CategoryBrandRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}

func (f *fakeGoodsClient) UpdateCategoryBrand(context.Context, *gpb.CategoryBrandRequest, ...grpc.CallOption) (*emptypb.Empty, error) {
	return nil, nil
}
