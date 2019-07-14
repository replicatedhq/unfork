package unforker

import (
	"math"

	"github.com/Masterminds/semver"
)

type ChartVersionSorter struct {
	Charts       []*PublishedChart
	ChartVersion string
	AppVersion   string
}

func (c ChartVersionSorter) Len() int {
	return len(c.Charts)
}

func (c ChartVersionSorter) Swap(i, j int) {
	c.Charts[i], c.Charts[j] = c.Charts[j], c.Charts[i]
}

func (c ChartVersionSorter) Less(i, j int) bool {
	targetVersion, _ := semver.NewVersion(c.ChartVersion)

	iVersion, _ := semver.NewVersion(c.Charts[i].ChartVersion)
	jVersion, _ := semver.NewVersion(c.Charts[j].ChartVersion)

	if iVersion.Major() != jVersion.Major() {
		iDiff := iVersion.Major() - targetVersion.Major()
		jDiff := jVersion.Major() - targetVersion.Major()

		return math.Abs(float64(iDiff)) < math.Abs(float64(jDiff))
	}

	if iVersion.Minor() != jVersion.Minor() {
		iDiff := iVersion.Minor() - targetVersion.Minor()
		jDiff := jVersion.Minor() - targetVersion.Minor()

		return math.Abs(float64(iDiff)) < math.Abs(float64(jDiff))
	}

	if iVersion.Patch() != jVersion.Patch() {
		iDiff := iVersion.Patch() - targetVersion.Patch()
		jDiff := jVersion.Patch() - targetVersion.Patch()

		return math.Abs(float64(iDiff)) < math.Abs(float64(jDiff))
	}

	return true
}
