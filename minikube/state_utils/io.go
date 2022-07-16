package state_utils

import "io/ioutil"

func ReadContents(path string) (string, error) {
	b, err := ioutil.ReadFile(path)
	return string(b), err
}
