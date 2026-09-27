package external

import "target"

// Known limitations: these zero values are not detected. The test fails when
// detection improves, so that the README can be updated.
func KnownUndetectedZeroValues(m map[string]target.User, x any) (u target.User, err error) {
	_ = make([]target.User, 3)
	var a [3]target.User
	_ = a
	v := m["missing"]
	w, _ := x.(target.User)
	_, _ = v, w
	return
}
