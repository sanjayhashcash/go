package resourceadapter

import (
	"context"

	protocol "github.com/sanjayhashcash/go/protocols/aurora"
	"github.com/sanjayhashcash/go/xdr"
)

func PopulateAsset(ctx context.Context, dest *protocol.Asset, asset xdr.Asset) error {
	return asset.Extract(&dest.Type, &dest.Code, &dest.Issuer)
}
