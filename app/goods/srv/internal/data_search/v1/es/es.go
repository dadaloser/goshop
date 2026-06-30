package es

import (
	v1 "goshop/app/goods/srv/internal/data_search/v1"
	"goshop/app/pkg/options"
	"goshop/pkg/db"
	"goshop/pkg/errors"
	"sync"

	"github.com/olivere/elastic/v7"
)

var (
	searchFactory v1.SearchFactory
	once          sync.Once
)

type esDataSearch struct {
	esClient *elastic.Client
}

func (ds *esDataSearch) Goods() v1.GoodsStore {
	return newGoods(ds)
}

func GetSearchFactoryOr(opts *options.EsOptions) (v1.SearchFactory, error) {
	if opts == nil && searchFactory == nil {
		return nil, errors.New("failed to get es client")
	}

	once.Do(func() {
		esOpt := db.EsOptions{
			Host:                  opts.Host,
			Port:                  opts.Port,
			Scheme:                opts.Scheme,
			Username:              opts.Username,
			Password:              opts.Password,
			Timeout:               opts.Timeout,
			UseSSL:                opts.UseSSL,
			SSLInsecureSkipVerify: opts.SSLInsecureSkipVerify,
			DisableHealthcheck:    opts.DisableHealthcheck,
		}
		esClient, err := db.NewEsClient(&esOpt)
		if err != nil {
			return
		}
		searchFactory = &esDataSearch{esClient: esClient}
	})
	if searchFactory == nil {
		return nil, errors.New("failed to get es client")
	}
	return searchFactory, nil
}
