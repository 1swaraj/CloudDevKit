package awscloud

import (
	"genericstoragesdk/aws"
	"genericstoragesdk/blob/s3blob"
	"genericstoragesdk/server/xrayserver"
	"github.com/google/wire"
	"net/http"
)

var AWS = wire.NewSet(
	Services,
	aws.DefaultSession,
	wire.Value(http.DefaultClient),
)

var Services = wire.NewSet(
	s3blob.Set,
	xrayserver.Set,
)
