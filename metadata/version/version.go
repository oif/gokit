package version

import (
	"fmt"
	"math/big"
)

var (
	_versionIncrease = big.NewInt(1)
)

func Parse(raw string) (*big.Int, error) {
	if raw == "" {
		raw = "0"
	}
	ver := new(big.Int)
	_, err := fmt.Sscan(raw, ver)
	return ver, err
}

func IncreaseVersion(raw string) (string, error) {
	ver, err := Parse(raw)
	if err != nil {
		return "", err
	}
	ver = ver.Add(ver, _versionIncrease)
	return ver.String(), nil
}

//   -1 if x <  y
//    0 if x == y
//   +1 if x >  y
func CompareVersion(x, y string) (int, error) {
	xVer, err := Parse(x)
	if err != nil {
		return 0, err
	}
	yVer, err := Parse(y)
	if err != nil {
		return 0, err
	}
	return xVer.Cmp(yVer), nil
}
