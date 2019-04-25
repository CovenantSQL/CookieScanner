package ipv6

import (
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestIPv6(t *testing.T) {
	Convey("nil", t, func() {
		ips, _ := ToIPv6(nil)
		So(ips, ShouldHaveLength, 0)
	})
	Convey("error", t, func() {
		ips, err := ToIPv6([]byte("aa"))
		So(err, ShouldNotBeNil)
		So(ips, ShouldHaveLength, 0)
	})
	Convey("from to IPv6", t, func() {
		in := []byte("1234567812345678")
		ips, err := ToIPv6(in)
		So(err, ShouldBeNil)
		So(ips, ShouldHaveLength, 1)
		So(ips[0].String(), ShouldEqual, "3132:3334:3536:3738:3132:3334:3536:3738")

		out, err := FromIPv6(ips)
		So(err, ShouldBeNil)
		So(out, ShouldResemble, in)
	})
	Convey("from to IPv6", t, func() {
		in := []byte("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
		ips, err := ToIPv6(in)
		So(err, ShouldBeNil)
		So(ips, ShouldHaveLength, 2)
		So(ips[0].String(), ShouldEqual, "6161:6161:6161:6161:6161:6161:6161:6161")
		So(ips[1].String(), ShouldEqual, "6161:6161:6161:6161:6161:6161:6161:6161")

		out, err := FromIPv6(ips)
		So(err, ShouldBeNil)
		So(out, ShouldResemble, in)
	})
	Convey("from domain", t, func() {
		buf, err := FromDomain("zh.test.optool.net")
		So(err, ShouldBeNil)
		So(buf, ShouldResemble, []byte("从前有座山の里有座庙12"))
	})
}
