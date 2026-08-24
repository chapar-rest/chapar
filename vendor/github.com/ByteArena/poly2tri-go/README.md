# poly2tri-go

Golang port of poly2tri.js

The license of poly2tri-go is the same as poly2tri.js; consult it here https://github.com/r3mi/poly2tri.js/blob/master/LICENSE.txt

Usage example : (triangulation of "Dude with 2 holes", taken from here: http://r3mi.github.io/poly2tri.js/)

```golang
package main

import (
	"fmt"
	"github.com/netgusto/poly2tri-go"
)

func main() {
	contour := []*poly2tri.Point{
		poly2tri.NewPoint(280.35714, 648.79075),
		poly2tri.NewPoint(286.78571, 662.8979),
		poly2tri.NewPoint(263.28607, 661.17871),
		poly2tri.NewPoint(262.31092, 671.41548),
		poly2tri.NewPoint(250.53571, 677.00504),
		poly2tri.NewPoint(250.53571, 683.43361),
		poly2tri.NewPoint(256.42857, 685.21933),
		poly2tri.NewPoint(297.14286, 669.50504),
		poly2tri.NewPoint(289.28571, 649.50504),
		poly2tri.NewPoint(285.0, 631.6479),
		poly2tri.NewPoint(285.0, 608.79075),
		poly2tri.NewPoint(292.85714, 585.21932),
		poly2tri.NewPoint(306.42857, 563.79075),
		poly2tri.NewPoint(323.57143, 548.79075),
		poly2tri.NewPoint(339.28571, 545.21932),
		poly2tri.NewPoint(357.85714, 547.36218),
		poly2tri.NewPoint(375.0, 550.21932),
		poly2tri.NewPoint(391.42857, 568.07647),
		poly2tri.NewPoint(404.28571, 588.79075),
		poly2tri.NewPoint(413.57143, 612.36218),
		poly2tri.NewPoint(417.14286, 628.07647),
		poly2tri.NewPoint(438.57143, 619.1479),
		poly2tri.NewPoint(438.03572, 618.96932),
		poly2tri.NewPoint(437.5, 609.50504),
		poly2tri.NewPoint(426.96429, 609.86218),
		poly2tri.NewPoint(424.64286, 615.57647),
		poly2tri.NewPoint(419.82143, 615.04075),
		poly2tri.NewPoint(420.35714, 605.04075),
		poly2tri.NewPoint(428.39286, 598.43361),
		poly2tri.NewPoint(437.85714, 599.68361),
		poly2tri.NewPoint(443.57143, 613.79075),
		poly2tri.NewPoint(450.71429, 610.21933),
		poly2tri.NewPoint(431.42857, 575.21932),
		poly2tri.NewPoint(405.71429, 550.21932),
		poly2tri.NewPoint(372.85714, 534.50504),
		poly2tri.NewPoint(349.28571, 531.6479),
		poly2tri.NewPoint(346.42857, 521.6479),
		poly2tri.NewPoint(346.42857, 511.6479),
		poly2tri.NewPoint(350.71429, 496.6479),
		poly2tri.NewPoint(367.85714, 476.6479),
		poly2tri.NewPoint(377.14286, 460.93361),
		poly2tri.NewPoint(385.71429, 445.21932),
		poly2tri.NewPoint(388.57143, 404.50504),
		poly2tri.NewPoint(360.0, 352.36218),
		poly2tri.NewPoint(337.14286, 325.93361),
		poly2tri.NewPoint(330.71429, 334.50504),
		poly2tri.NewPoint(347.14286, 354.50504),
		poly2tri.NewPoint(337.85714, 370.21932),
		poly2tri.NewPoint(333.57143, 359.50504),
		poly2tri.NewPoint(319.28571, 353.07647),
		poly2tri.NewPoint(312.85714, 366.6479),
		poly2tri.NewPoint(350.71429, 387.36218),
		poly2tri.NewPoint(368.57143, 408.07647),
		poly2tri.NewPoint(375.71429, 431.6479),
		poly2tri.NewPoint(372.14286, 454.50504),
		poly2tri.NewPoint(366.42857, 462.36218),
		poly2tri.NewPoint(352.85714, 462.36218),
		poly2tri.NewPoint(336.42857, 456.6479),
		poly2tri.NewPoint(332.85714, 438.79075),
		poly2tri.NewPoint(338.57143, 423.79075),
		poly2tri.NewPoint(338.57143, 411.6479),
		poly2tri.NewPoint(327.85714, 405.93361),
		poly2tri.NewPoint(320.71429, 407.36218),
		poly2tri.NewPoint(315.71429, 423.07647),
		poly2tri.NewPoint(314.28571, 440.21932),
		poly2tri.NewPoint(325.0, 447.71932),
		poly2tri.NewPoint(324.82143, 460.93361),
		poly2tri.NewPoint(317.85714, 470.57647),
		poly2tri.NewPoint(304.28571, 483.79075),
		poly2tri.NewPoint(287.14286, 491.29075),
		poly2tri.NewPoint(263.03571, 498.61218),
		poly2tri.NewPoint(251.60714, 503.07647),
		poly2tri.NewPoint(251.25, 533.61218),
		poly2tri.NewPoint(260.71429, 533.61218),
		poly2tri.NewPoint(272.85714, 528.43361),
		poly2tri.NewPoint(286.07143, 518.61218),
		poly2tri.NewPoint(297.32143, 508.25504),
		poly2tri.NewPoint(297.85714, 507.36218),
		poly2tri.NewPoint(298.39286, 506.46932),
		poly2tri.NewPoint(307.14286, 496.6479),
		poly2tri.NewPoint(312.67857, 491.6479),
		poly2tri.NewPoint(317.32143, 503.07647),
		poly2tri.NewPoint(322.5, 514.1479),
		poly2tri.NewPoint(325.53571, 521.11218),
		poly2tri.NewPoint(327.14286, 525.75504),
		poly2tri.NewPoint(326.96429, 535.04075),
		poly2tri.NewPoint(311.78571, 540.04075),
		poly2tri.NewPoint(291.07143, 552.71932),
		poly2tri.NewPoint(274.82143, 568.43361),
		poly2tri.NewPoint(259.10714, 592.8979),
		poly2tri.NewPoint(254.28571, 604.50504),
		poly2tri.NewPoint(251.07143, 621.11218),
		poly2tri.NewPoint(250.53571, 649.1479),
		poly2tri.NewPoint(268.1955, 654.36208),
	}

	swctx := poly2tri.NewSweepContext(contour, false)
	swctx.AddHole([]*poly2tri.Point{
		poly2tri.NewPoint(325, 437),
		poly2tri.NewPoint(320, 423),
		poly2tri.NewPoint(329, 413),
		poly2tri.NewPoint(332, 423),
	})

	swctx.AddHole([]*poly2tri.Point{
		poly2tri.NewPoint(320.72342, 480),
		poly2tri.NewPoint(338.90617, 465.96863),
		poly2tri.NewPoint(347.99754, 480.61584),
		poly2tri.NewPoint(329.8148, 510.41534),
		poly2tri.NewPoint(339.91632, 480.11077),
		poly2tri.NewPoint(334.86556, 478.09046),
	})

	swctx.Triangulate()
	triangles := swctx.GetTriangles()

	fmt.Println(triangles)
}
```
