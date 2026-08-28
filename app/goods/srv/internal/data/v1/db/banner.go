package db

import (
	"context"
	stderrors "errors"
	"goshop/app/pkg/errorcontract"

	"goshop/app/pkg/bizcode"
	"goshop/pkg/errors"

	v1 "goshop/app/goods/srv/internal/data/v1"
	"goshop/app/goods/srv/internal/domain/do"
	metav1 "goshop/pkg/common/meta/v1"

	"gorm.io/gorm"
)

type banners struct {
	db *gorm.DB
}

func newBanner(factory *mysqlFactory) *banners {
	banners := &banners{
		db: factory.db,
	}
	return banners
}

func (b *banners) List(ctx context.Context, opts metav1.ListMeta, orderBy []string) (*do.BannerList, error) {
	ret := &do.BannerList{}
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}

	countQuery := b.db.WithContext(ctx).Model(&do.BannerDO{})
	if err := countQuery.Count(&ret.TotalCount).Error; err != nil {
		return nil, errorcontract.WrapDatabase(err, "count banners")
	}

	query := b.db.WithContext(ctx).Model(&do.BannerDO{})
	query = applyOrderBy(query, orderBy, bannerOrderColumns)
	if opts.PageSize <= 0 {
		opts.PageSize = 10
	}
	if opts.Page > 0 {
		query = query.Offset((opts.Page - 1) * opts.PageSize)
	}
	tx := query.Limit(opts.PageSize).Find(&ret.Items)
	if tx.Error != nil {
		return nil, errorcontract.WrapDatabase(tx.Error, "list banners")
	}
	return ret, nil
}

func (b *banners) Create(ctx context.Context, txn *gorm.DB, banner *do.BannerDO) error {
	if banner == nil {
		return errors.NewCode(bizcode.ErrGoodsInvalid, "banner is required")
	}

	db := b.db
	if txn != nil {
		db = txn
	}
	if err := db.WithContext(ctx).Create(banner).Error; err != nil {
		return errorcontract.WrapDatabase(err, "create banner")
	}
	return nil
}

func (b *banners) Update(ctx context.Context, txn *gorm.DB, banner *do.BannerDO) error {
	if banner == nil || banner.ID <= 0 {
		return errors.NewCode(bizcode.ErrBannerNotFound, "banner not found")
	}

	db := b.db
	if txn != nil {
		db = txn
	}
	tx := db.WithContext(ctx).Model(&do.BannerDO{}).
		Where("id = ?", banner.ID).
		Updates(map[string]interface{}{
			"image": banner.Image,
			"url":   banner.Url,
			"index": banner.Index,
		})
	if tx.Error != nil {
		return errorcontract.WrapDatabase(tx.Error, "update banner")
	}
	if tx.RowsAffected == 0 {
		exists, err := b.exists(ctx, uint64(banner.ID))
		if err != nil {
			return err
		}
		if !exists {
			return errors.NewCode(bizcode.ErrBannerNotFound, "banner not found")
		}
	}
	return nil
}

func (b *banners) Delete(ctx context.Context, ID uint64) error {
	if ID == 0 {
		return errors.NewCode(bizcode.ErrBannerNotFound, "banner not found")
	}

	tx := b.db.WithContext(ctx).Where("id = ?", ID).Delete(&do.BannerDO{})
	if tx.Error != nil {
		return errorcontract.WrapDatabase(tx.Error, "delete banner")
	}
	if tx.RowsAffected == 0 {
		return errors.NewCode(bizcode.ErrBannerNotFound, "banner not found")
	}
	return nil
}

var _ v1.BannerStore = &banners{}

func (b *banners) exists(ctx context.Context, id uint64) (bool, error) {
	var banner do.BannerDO
	err := b.db.WithContext(ctx).Select("id").First(&banner, id).Error
	if err != nil {
		if stderrors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, errorcontract.WrapDatabase(err, "check banner")
	}
	return true, nil
}
