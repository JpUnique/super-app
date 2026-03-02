package router

import (
	"time"

	"github.com/JpUnique/super-app/internal/gateway/versioning"
)

func LoadVersionRouter() *versioning.VersionRouter {
	vr := versioning.NewVersionRouter("v1")

	vr.AddRoute("v1", versioning.V1Handler())
	vr.AddRoute("v2", versioning.V2Handler())

	vr.SetDeprecation(versioning.DeprecationRule{
		Version: "v1",
		Sunset:  time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		Message: "v1 will be removed soon. Use v2.",
	})

	return vr
}
