package processors

import (
	"context"
	"io"

	"github.com/hcnet/go/ingest"
	"github.com/hcnet/go/support/errors"
)

func StreamChanges(
	ctx context.Context,
	changeProcessor ChangeProcessor,
	reader ingest.ChangeReader,
) error {
	for {
		change, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errors.Wrap(err, "could not read transaction")
		}

		if err = changeProcessor.ProcessChange(ctx, change); err != nil {
			return errors.Wrap(
				err,
				"could not process change",
			)
		}
	}
}
