














package gcpcloud

import (
	"github.com/google/wire"
	"github.com/swaraj1802/CloudDevKit/genericstorage/gcsblob"
	"github.com/swaraj1802/CloudDevKit/gcp"
	"github.com/swaraj1802/CloudDevKit/server/sdserver"
)



var GCP = wire.NewSet(Services, gcp.DefaultIdentity)




var Services = wire.NewSet(
	gcp.DefaultTransport,
	gcp.NewHTTPClient,
	gcsblob.Set,
	sdserver.Set,
)
