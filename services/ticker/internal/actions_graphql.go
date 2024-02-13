package ticker

import (
	"github.com/sanjayhashcash/go/services/ticker/internal/gql"
	"github.com/sanjayhashcash/go/services/ticker/internal/tickerdb"
	hlog "github.com/sanjayhashcash/go/support/log"
)

func StartGraphQLServer(s *tickerdb.TickerSession, l *hlog.Entry, port string) {
	graphql := gql.New(s, l)

	graphql.Serve(port)
}
