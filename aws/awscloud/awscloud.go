package awscloud

import (
	"github.com/google/wire"
	"github.com/swaraj1802/CloudDevKit/aws"
	"github.com/swaraj1802/CloudDevKit/genericstorage/s3blob"
	"github.com/swaraj1802/CloudDevKit/server/xrayserver"
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
