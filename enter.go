package pan_client

import (
	"github.com/hefeiyu2025/pan-client/common"
	"github.com/hefeiyu2025/pan-client/pan"
	_ "github.com/hefeiyu2025/pan-client/pan/driver"
)

func GracefulExist() {
	common.Exit()
}
func GetClient(driverType pan.DriverType) (pan.Driver, error) {
	return pan.GetDriver(pan.Cloudreve)
}
