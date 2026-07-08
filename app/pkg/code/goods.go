package code

const (
	// ErrGoodsNotFound - 404: Goods not found.
	ErrGoodsNotFound int = iota + 100501

	// ErrCategoryNotFound - 404: Category not found.
	ErrCategoryNotFound

	// ErrEsUnmarshal - 500: Es unmarshal error.
	ErrEsUnmarshal

	// ErrGoodsInvalid - 400: Goods request is invalid.
	ErrGoodsInvalid

	// ErrBrandNotFound - 404: Brand not found.
	ErrBrandNotFound

	// ErrBannerNotFound - 404: Banner not found.
	ErrBannerNotFound

	// ErrCategoryBrandNotFound - 404: Category brand relation not found.
	ErrCategoryBrandNotFound
)
