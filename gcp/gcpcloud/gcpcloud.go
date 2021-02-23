














package gcpcloud

import (
	"github.com/google/wire"
	"genericstoragesdk/blob/gcsblob"
	"genericstoragesdk/gcp"
	"genericstoragesdk/server/sdserver"
)



var GCP = wire.NewSet(Services, gcp.DefaultIdentity)




var Services = wire.NewSet(
	gcp.DefaultTransport,
	gcp.NewHTTPClient,
	gcsblob.Set,
	sdserver.Set,
)
